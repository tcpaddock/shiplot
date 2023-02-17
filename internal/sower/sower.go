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
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/panjf2000/ants/v2"
	"github.com/tcpaddock/shiplot/internal/config"
	"golang.org/x/exp/slog"
)

type Sower struct {
	ctx     context.Context
	cancel  context.CancelFunc
	cfg     config.Config
	paths   *pathList
	pool    *ants.Pool
	watcher *fsnotify.Watcher
	wg      sync.WaitGroup
}

func NewSower(ctx context.Context, cfg config.Config) (s *Sower, err error) {
	s = new(Sower)

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.cfg = cfg

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

func (s *Sower) Run() (err error) {
	// Create filesystem watcher
	s.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Start running watcher
	s.wg.Add(1)
	go s.runLoop()

	// Add staging path to watcher
	var stagePaths []string

	for _, stagePath := range s.cfg.StagingPaths {
		if strings.Contains(stagePath, "*") {
			globPaths, err := filepath.Glob(stagePath)
			if err != nil {
				return err
			}

			stagePaths = append(stagePaths, globPaths...)
		} else {
			stagePaths = append(stagePaths, stagePath)
		}
	}

	for _, stagingPath := range stagePaths {
		// Add staging path to watcher
		slog.Default().Info(fmt.Sprintf("Starting watcher on %s", stagingPath))
		err = s.watcher.Add(stagingPath)
		if err != nil {
			s.Close()
			return err
		}

		// Move existing plots
		files, err := os.ReadDir(stagingPath)
		if err != nil {
			return err
		}

		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".plot") {
				err := s.SavePlot(file.Name(), 0, nil)
				if err != nil {
					slog.Default().Error(fmt.Sprintf("failed to move %s", file.Name()), err)
				}
			}
		}
	}

	return nil
}

func (s *Sower) Close() {
	s.cancel()
	s.wg.Wait()
}

func (s *Sower) runLoop() {
	defer s.wg.Done()
	for {
		select {
		// Read from Errors
		case err, ok := <-s.watcher.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called)
				return
			}
			slog.Default().Error("File watcher error", err)
		// Read from Events
		case e, ok := <-s.watcher.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called)
				return
			}

			if e.Op.Has(fsnotify.Create) && strings.HasSuffix(e.Name, ".plot") {
				err := s.SavePlot(e.Name, 0, nil)
				if err != nil {
					slog.Default().Error(fmt.Sprintf("failed to save %s", e.Name), err)
				}
			}
		// Read from context for closing
		case <-s.ctx.Done():
			s.watcher.Close()
			s.pool.Release()
			return
		}
	}
}

func (s *Sower) SavePlot(name string, size uint64, conn net.Conn) (err error) {
	s.wg.Add(1)
	err = s.pool.Submit(func() {
		defer s.wg.Done()

		if conn != nil {
			err := s.savePlot(name, size, conn)
			if err != nil {
				conn.Write([]byte{0})
				conn.Close()
				slog.Default().Error(fmt.Sprintf("failed to save %s", name), err)
			}
		} else {
			err := s.movePlot(name)
			if err != nil {
				slog.Default().Error(fmt.Sprintf("failed to move %s", name), err)
			}
		}
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Sower) savePlot(name string, size uint64, reader io.Reader) (err error) {
	// Find the best destination path
	dstPath := s.getDestinationPath(size)

	var (
		dstDir      = dstPath.name
		dstFullName = filepath.Join(dstDir, name)
	)

	slog.Default().Info(fmt.Sprintf("Moving %s to %s", name, dstDir))

	// Open destination file
	dst, err := os.Create(dstFullName + ".tmp")
	if err != nil {
		dst.Close()
		s.paths.Update(dstPath, true)
		return err
	}

	start := time.Now()

	// Copy plot to temporary file
	written, err := io.Copy(dst, reader)
	if err != nil {
		dst.Close()
		s.paths.Update(dstPath, true)
		return err
	}

	if uint64(written) != size {
		dst.Close()
		os.Remove(dstFullName + ".tmp")
		s.paths.Update(dstPath, true)
		return fmt.Errorf("file size mismatch on %s", name)
	}

	// Rename temporary file
	err = os.Rename(dstFullName+".tmp", dstFullName)
	if err != nil {
		dst.Close()
		s.paths.Update(dstPath, true)
		return err
	}

	duration := time.Since(start)

	// Close destination file
	err = dst.Close()
	if err != nil {
		s.paths.Update(dstPath, true)
		return err
	}

	// Update available paths
	s.paths.Update(dstPath, true)

	slog.Default().Info(fmt.Sprintf("Moved %s to %s", name, dstDir), slog.Int64("written", written), slog.Duration("time", duration))

	return nil
}

func (s *Sower) openFile(name string) (info fs.FileInfo, file *os.File, err error) {
	// Open source file
	file, err = os.Open(name)
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	// Get source file size
	info, err = file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	return info, file, err
}

func (s *Sower) movePlot(name string) (err error) {
	info, file, err := s.openFile(name)
	defer file.Close()
	if err != nil {
		return err
	}

	err = s.savePlot(info.Name(), uint64(info.Size()), file)
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
