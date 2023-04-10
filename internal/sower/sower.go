/*
Copyright © 2023 Taylor Paddock

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package sower

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/tcpaddock/shiplot/internal/config"
	"github.com/tcpaddock/shiplot/internal/util"
	"golang.org/x/exp/slog"
)

type Sower struct {
	cfg    config.Config
	paths  *pathList
	pool   *ants.Pool
	wg     sync.WaitGroup
	client *TcpClient
}

func NewSower(cfg config.Config) (s *Sower, err error) {
	s = new(Sower)

	s.cfg = cfg
	s.client = NewTcpClient(cfg)

	// Fill list of available destination paths
	var destPaths []string

	for _, destPath := range s.cfg.DestinationPaths {
		if strings.Contains(destPath, "*") {
			globPaths, err := filepath.Glob(destPath)
			if err != nil {
				return nil, err
			}

			destPaths = append(destPaths, globPaths...)
		} else {
			destPaths = append(destPaths, destPath)
		}
	}
	s.paths = new(pathList)
	s.paths.Populate(destPaths)

	// Create worker pool for moving plots from stream
	size := s.getPoolSize()
	slog.Default().Info(fmt.Sprintf("Creating worker pool with max size %d", size))
	s.pool, err = ants.NewPool(size)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Sower) enqueuePlotMove(ctx context.Context, name string) (err error) {
	s.wg.Add(1)
	err = s.pool.Submit(func() {
		defer s.wg.Done()

		// Open source file
		src, err := os.Open(name)
		if err != nil {
			src.Close()
			slog.Default().Error(fmt.Sprintf("failed to open %s", name), err)
			return
		}

		// Get source file size
		info, err := src.Stat()
		if err != nil {
			slog.Default().Error(fmt.Sprintf("failed to get file info %s", name), err)
			return
		}

		// Find the best destination path
		dstPath := s.getDestinationPath(uint64(info.Size()))
		defer s.paths.SetAvailable(dstPath, true)

		var (
			dstDir      = dstPath.name
			dstFullName = filepath.Join(dstDir, filepath.Base(src.Name()))
		)

		slog.Default().Info(fmt.Sprintf("Moving %s to %s", filepath.Base(src.Name()), dstDir))

		// Create destination file
		dst, err := os.Create(dstFullName + ".tmp")
		if err != nil {
			slog.Default().Error(fmt.Sprintf("failed to create temp destination file %s", dstFullName+".tmp"), err)
		}

		start := time.Now()

		// Copy plot
		cr := util.NewContextReader(ctx, src)
		cw := util.NewContextWriter(ctx, dst)
		written, err := io.Copy(cw, cr)
		if err != nil {
			slog.Default().Error(fmt.Sprintf("failed to copy %s to %s", name, filepath.Base(dst.Name())), err)
		}

		if uint64(written) != uint64(info.Size()) {
			os.Remove(dstFullName + ".tmp")
			slog.Default().Error(fmt.Sprintf("failed to copy %s to %s", name, filepath.Base(dst.Name())), fmt.Errorf("file size mismatch"))
		}

		// Windows requires closing files before rename
		src.Close()
		dst.Close()

		// Rename temporary file
		err = os.Rename(dstFullName+".tmp", dstFullName)
		if err != nil {
			slog.Default().Error(fmt.Sprintf("failed to rename temp file %s", dstFullName+".tmp"), err)
		}

		duration := time.Since(start)

		// Delete source file
		err = os.Remove(src.Name())
		if err != nil {
			slog.Default().Error(fmt.Sprintf("failed to delete file %s", src.Name()), err)
		}

		// Update available paths
		s.paths.SetAvailable(dstPath, true)

		slog.Default().Info(fmt.Sprintf("Moved %s to %s", name, dstDir), slog.Int64("written", written), slog.Duration("time", duration))
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Sower) enqueuePlotDownload(ctx context.Context, name string, size uint64, reader io.Reader, writer io.Writer) (err error) {
	s.wg.Add(1)
	err = s.pool.Submit(func() {
		defer s.wg.Done()

		// Find the best destination path
		dstPath := s.getDestinationPath(size)
		defer s.paths.SetAvailable(dstPath, true)

		var (
			dstDir      = dstPath.name
			dstFullName = filepath.Join(dstDir, name)
		)

		slog.Default().Info("Downloading plot", slog.String("name", name), slog.String("destination", dstDir))

		// Create destination file
		dst, err := os.Create(dstFullName + ".tmp")
		if err != nil {
			_, _ = writeFail(ctx, writer)
			dst.Close()
			slog.Default().Error(fmt.Sprintf("failed to create temp destination file %s", dstFullName+".tmp"), err)
			return
		}

		start := time.Now()

		// Download plot to temporary file
		cr := util.NewContextReader(ctx, reader)
		cw := util.NewContextWriter(ctx, dst)
		written, err := io.Copy(cw, cr)
		if err != nil {
			_, _ = writeFail(ctx, writer)
			slog.Default().Error(fmt.Sprintf("failed to download %s", name), err)
			return
		}

		if uint64(written) != size {
			_, _ = writeFail(ctx, writer)
			os.Remove(dstFullName + ".tmp")
			slog.Default().Error(fmt.Sprintf("failed to download %s", name), fmt.Errorf("file size mismatch"))
			return
		}

		_, _ = writeSuccess(ctx, writer)

		// Windows requires closing files before rename
		dst.Close()

		// Rename temporary file
		err = os.Rename(dstFullName+".tmp", dstFullName)
		if err != nil {
			_, _ = writeFail(ctx, writer)
			slog.Default().Error(fmt.Sprintf("failed to rename temp file %s", dstFullName+".tmp"), err)
			return
		}

		duration := time.Since(start)

		// Update available paths
		s.paths.SetAvailable(dstPath, true)

		slog.Default().Info(fmt.Sprintf("Downloaded %s to %s", name, dstDir), slog.Int64("written", written), slog.Duration("time", duration))
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Sower) enqueuePlotUpload(ctx context.Context, name string) (err error) {
	s.wg.Add(1)
	err = s.pool.Submit(func() {
		defer s.wg.Done()
		slog.Default().Info(fmt.Sprintf("Uploading %s", name))

		// Open source file
		src, err := os.Open(name)
		if err != nil {
			src.Close()
			slog.Default().Error(fmt.Sprintf("failed to open %s", name), err)
			return
		}

		// Get source file size
		info, err := src.Stat()
		if err != nil {
			slog.Default().Error(fmt.Sprintf("failed to get file info for %s", name), err)
			return
		}

		start := time.Now()

		// Upload plot file
		cr := util.NewContextReader(ctx, src)
		written, err := s.client.WritePlot(ctx, filepath.Base(name), uint64(info.Size()), cr)
		if err != nil {
			slog.Default().Error(fmt.Sprintf("failed to upload %s", name), err)
			return
		}

		duration := time.Since(start)

		// Windows requires closing files before deleting
		src.Close()

		// Delete source file
		err = os.Remove(name)
		if err != nil {
			slog.Default().Error(fmt.Sprintf("failed to delete file %s", src.Name()), err)
		}

		slog.Default().Info(fmt.Sprintf("Successfully uploaded %s", name), slog.Int64("written", written), slog.Duration("time", duration))
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Sower) getDestinationPath(fileSize uint64) (destinationPath *path) {
	// Find the best destination path
	var dstPath *path

	for {
		// Get the lowest sized first path and mark it unavailable
		dstPath = s.paths.FirstAvailable()

		// Wait for 10 seconds if no available destination
		if dstPath == nil {
			time.Sleep(time.Second * 10)
			continue
		}

		// Ensure destination path has enough space
		if uint64(fileSize) < dstPath.usage.Free() {
			break
		} else {
			// Remove path if space is too low
			s.paths.Remove(dstPath)

			// Adjust move pool
			size := s.getPoolSize()
			if s.pool.Cap() != size {
				slog.Default().Info(fmt.Sprintf("Adjusting worker pool max size to %d", size))
				s.pool.Tune(size)
			}
			continue
		}
	}

	return dstPath
}

func (s *Sower) getPoolSize() (size int) {
	poolSize := int(s.cfg.MaxThreads)

	if s.cfg.Client.Enabled {
		return poolSize
	}

	if poolSize == 0 {
		poolSize = s.paths.Len()
	} else {
		if s.paths.Len() < poolSize {
			poolSize = s.paths.Len()
		}
	}

	if poolSize == 0 {
		poolSize = 1
	}

	return poolSize
}

func writeSuccess(ctx context.Context, writer io.Writer) (written int, err error) {
	cw := util.NewContextWriter(ctx, writer)

	return cw.Write([]byte{1})
}

func writeFail(ctx context.Context, writer io.Writer) (written int, err error) {
	cw := util.NewContextWriter(ctx, writer)

	return cw.Write([]byte{0})
}
