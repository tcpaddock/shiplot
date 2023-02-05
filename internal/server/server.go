/*
Copyright © 2023 Taylor Paddock <tcpaddock@gmail.com>

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
package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/YSZhuoyang/go-dispatcher/dispatcher"
	"github.com/fsnotify/fsnotify"
	"github.com/ricochet2200/go-disk-usage/du"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/tcpaddock/shiplot/internal/config"
	"golang.org/x/exp/slog"
)

type destPath struct {
	name  string
	usage du.DiskUsage
	count int
}

type Server struct {
	cfg        config.Config
	wg         sync.WaitGroup
	dispatcher dispatcher.Dispatcher
	destPaths  []destPath
	dpMutex    sync.Mutex
}

func NewServer(cfg config.Config) (s *Server) {
	s = new(Server)

	s.cfg = cfg

	return s
}

func (srv *Server) Start() (err error) {
	slog.Default().Info("Starting server...")

	// Populate available paths
	srv.dpMutex.Lock()
	for _, path := range srv.cfg.DestinationPaths {
		usage := du.NewDiskUsage(path)
		srv.destPaths = append(srv.destPaths, destPath{name: path, usage: *usage, count: 0})
	}
	srv.dpMutex.Unlock()

	// Setup dispatcher threads
	if srv.dispatcher == nil {
		srv.dispatcher, err = dispatcher.NewDispatcher(srv.cfg.Threads)
		if err != nil {
			return err
		}
	}

	// Setup watcher thread
	srv.wg.Add(1)
	go func() {
		defer srv.wg.Done()
		srv.runWatcher()
	}()

	// Copy existing plots
	files, err := os.ReadDir(srv.cfg.StagingPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".plot") {
			p := NewPlot(filepath.Join(srv.cfg.StagingPath, file.Name()))

			dest := filepath.Join(srv.findDestinationPath(), filepath.Base(p.Name))

			srv.dispatcher.Dispatch(&moveJob{plot: p, dest: dest})
		}
	}

	// Wait for everything to close
	srv.wg.Wait()

	// Wait for all jobs
	srv.dispatcher.Finalize()

	return nil
}

func (srv *Server) runWatcher() {
	// Create a new watcher
	fw, err := fsnotify.NewWatcher()
	cobra.CheckErr(err)
	defer fw.Close()

	// Start listening for events
	go srv.watchLoop(fw)

	// Add staging path to watcher
	err = fw.Add(srv.cfg.StagingPath)
	cobra.CheckErr(err)

	<-make(chan struct{}) // Block forever
}

func (srv *Server) watchLoop(fw *fsnotify.Watcher) {
	for {
		select {
		// Read from Errors
		case err, ok := <-fw.Errors:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called)
				return
			}
			fmt.Printf("ERROR: %s\n", err)
		// Read from Events
		case e, ok := <-fw.Events:
			if !ok { // Channel was closed (i.e. Watcher.Close() was called)
				return
			}

			if e.Op.Has(fsnotify.Create) && strings.HasSuffix(e.Name, ".plot") {
				p := NewPlot(e.Name)

				dest := filepath.Join(srv.findDestinationPath(), filepath.Base(p.Name))

				srv.dispatcher.Dispatch(&moveJob{plot: p, dest: dest})
			}
		}
	}
}

func (srv *Server) findDestinationPath() (path string) {
	srv.dpMutex.Lock()

	// Find all paths that have no transfers
	var available []destPath
	for {
		available = lo.Filter(srv.destPaths, func(path destPath, index int) bool {
			return path.count == 0
		})

		if len(available) > 0 {
			break
		}
	}

	// Sort paths by free space
	sort.Slice(available, func(i, j int) bool {
		return available[i].usage.Free() < available[j].usage.Free()
	})

	path = available[0].name

	srv.dpMutex.Unlock()

	return path
}

type moveJob struct {
	plot *plot
	dest string
}

func (job *moveJob) Do() {
	slog.Default().Info(fmt.Sprintf("Moving %s to %s", job.plot.Name, job.dest))

	_, written, duration, err := job.plot.Move(job.dest)
	if err != nil {
		slog.Default().Error(fmt.Sprintf("Failed to move %s to %s", job.plot.Name, job.dest), err)
	}

	slog.Default().Info(fmt.Sprintf("Moved %s to %s", job.plot.Name, job.dest), slog.Int64("written", written), slog.Duration("time", duration))
}
