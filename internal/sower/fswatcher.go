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
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/tcpaddock/shiplot/internal/config"
	"golang.org/x/exp/slog"
)

type FsWatcher struct {
	cfg     config.Config
	sower   *Sower
	watcher *fsnotify.Watcher
}

func NewFsWatcher(cfg config.Config, sower *Sower) (w *FsWatcher, err error) {
	w = new(FsWatcher)

	w.cfg = cfg
	w.sower = sower

	// Create filesystem watcher
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *FsWatcher) Run(ctx context.Context) (err error) {
	// Start running watcher
	w.sower.wg.Add(1)
	go w.runLoop(ctx)

	// Add staging paths to watcher
	var stagePaths []string

	for _, stagePath := range w.cfg.StagingPaths {
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
		slog.Default().Info("Path added to watcher", slog.String("name", stagingPath))
		err = w.watcher.Add(stagingPath)
		if err != nil {
			return err
		}

		// Move existing plots
		files, err := os.ReadDir(stagingPath)
		if err != nil {
			return err
		}

		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".plot") {
				fullName := filepath.Join(stagingPath, file.Name())

				if w.cfg.Client.Enabled {
					err = w.sower.enqueuePlotUpload(ctx, fullName)
					if err != nil {
						slog.Default().Error("failed to add plot upload to queue", err, slog.String("name", file.Name()))
					}
				} else {
					err = w.sower.enqueuePlotMove(ctx, fullName)
					if err != nil {
						slog.Default().Error("failed to add plot move to queue", err, slog.String("name", file.Name()))
					}
				}
			}
		}
	}

	for {
		select {
		case <-make(chan struct{}):
		case <-ctx.Done():
			w.watcher.Close()
			return nil
		}
	}
}

func (w *FsWatcher) runLoop(ctx context.Context) {
	defer w.sower.wg.Done()
	for {
		select {
		// Read from Errors
		case err, ok := <-w.watcher.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called)
				return
			}
			slog.Default().Error("File watcher error", err)
		// Read from Events
		case e, ok := <-w.watcher.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called)
				return
			}

			if e.Op.Has(fsnotify.Create) && strings.HasSuffix(e.Name, ".plot") {
				if w.cfg.Client.Enabled {
					err := w.sower.enqueuePlotUpload(ctx, e.Name)
					if err != nil {
						slog.Default().Error("failed to add plot upload to queue", err, slog.String("name", e.Name))
					}
				} else {
					err := w.sower.enqueuePlotMove(ctx, e.Name)
					if err != nil {
						slog.Default().Error("failed to add plot move to queue", err, slog.String("name", e.Name))
					}
				}
			}
		// Read from context for closing
		case <-ctx.Done():
			return
		}
	}
}
