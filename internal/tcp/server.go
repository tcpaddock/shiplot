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
package tcp

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"

	"github.com/tcpaddock/shiplot/internal/config"
	"github.com/tcpaddock/shiplot/internal/sower"
	"golang.org/x/exp/slog"
)

type Server struct {
	ctx      context.Context
	cancel   context.CancelFunc
	cfg      config.Config
	sower    *sower.Sower
	listener net.Listener
}

func NewServer(ctx context.Context, cfg config.Config, sower *sower.Sower) (s *Server) {
	s = new(Server)

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.cfg = cfg
	s.sower = sower

	return s
}

func (s *Server) Run() (err error) {
	endpoint := fmt.Sprintf("%s:%d", s.cfg.Server.Ip, s.cfg.Server.Port)
	slog.Default().Info(fmt.Sprintf("Starting TCP server on %s", endpoint))
	s.listener, err = net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}

	go s.runLoop()

	for {
		select {
		case <-make(chan struct{}):
		case <-s.ctx.Done():
			s.listener.Close()
			return nil
		}
	}
}

func (s *Server) runLoop() {
	defer s.listener.Close()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			slog.Default().Error("Request failed", err)
		}

		go s.handleRequest(conn)
	}
}

func (s *Server) handleRequest(conn net.Conn) {
	fileName, err := readFileName(conn)
	if err != nil {
		slog.Default().Error("Failed to read file name from request", err)
		conn.Write([]byte{0})
		return
	}

	fileSize, err := readFileSize(conn)
	if err != nil {
		slog.Default().Error("Failed to read file size from request", err)
		conn.Write([]byte{0})
		return
	}

	err = s.sower.SavePlot(fileName, fileSize, conn)
	if err != nil {
		slog.Default().Error("Failed to read file from request", err)
		conn.Write([]byte{0})
		return
	}

	conn.Write([]byte{1})
}

func readFileName(conn net.Conn) (name string, err error) {
	fileNameBytes := make([]byte, 256)
	_, err = conn.Read(fileNameBytes)
	if err != nil {
		return "", err
	}

	fileName := string(fileNameBytes)
	if !strings.HasSuffix(fileName, ".plot.tmp") {
		return "", fmt.Errorf("request provided incorrect file name %s", fileName)
	}

	return fileName, nil
}

func readFileSize(conn net.Conn) (size uint64, err error) {
	fileSizeBytes := make([]byte, 8)
	_, err = conn.Read(fileSizeBytes)
	if err != nil {
		return 0, err
	}

	fileSize := binary.LittleEndian.Uint64(fileSizeBytes)

	return fileSize, nil
}
