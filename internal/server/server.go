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
package server

import (
	"context"

	"github.com/tcpaddock/shiplot/internal/config"
	"github.com/tcpaddock/shiplot/internal/sower"
	"golang.org/x/exp/slog"
)

type Server struct {
	cfg   config.Config
	sower *sower.Sower
}

func NewServer(cfg config.Config) (s *Server, err error) {
	s = new(Server)

	s.cfg = cfg
	s.sower, err = sower.NewSower(cfg)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) (err error) {
	if s.cfg.Server.Enabled {
		slog.Default().Info("Starting server")
		ts := sower.NewTcpServer(s.cfg, s.sower)

		err = ts.Run(ctx)
		if err != nil {
			return err
		}
	}

	if len(s.cfg.StagingPaths) > 0 {
		slog.Default().Info("Starting file system watcher")
		fw, err := sower.NewFsWatcher(s.cfg, s.sower)
		if err != nil {
			return err
		}

		err = fw.Run(ctx)
		if err != nil {
			return err
		}
	}

	for {
		select {
		case <-make(chan struct{}):
		case <-ctx.Done():
			return nil
		}
	}
}
