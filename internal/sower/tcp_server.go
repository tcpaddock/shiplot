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
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/tcpaddock/shiplot/internal/config"
	"github.com/tcpaddock/shiplot/internal/util"
	"golang.org/x/exp/slog"
)

type TcpServer struct {
	cfg   config.Config
	sower *Sower
}

func NewTcpServer(cfg config.Config, sower *Sower) (s *TcpServer) {
	s = new(TcpServer)

	s.cfg = cfg
	s.sower = sower

	return s
}

func (s *TcpServer) Run(ctx context.Context) (err error) {
	endpoint := fmt.Sprintf("%s:%d", s.cfg.Server.Ip, s.cfg.Server.Port)
	slog.Default().Info(fmt.Sprintf("Starting TCP server on %s", endpoint))
	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		return err
	}

	defer listener.Close()

	s.sower.wg.Add(1)
	go s.runLoop(ctx, listener)

	for {
		select {
		case <-make(chan struct{}):
		case <-ctx.Done():
			listener.Close()
			return nil
		}
	}
}

func (s *TcpServer) runLoop(ctx context.Context, listener net.Listener) {
	defer s.sower.wg.Done()

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Default().Error("Incoming connection failed", err)
		}

		go s.handleRequest(ctx, conn)
	}
}

func (s *TcpServer) handleRequest(ctx context.Context, conn net.Conn) {
	fileName, err := s.readFileName(ctx, conn)
	if err != nil {
		slog.Default().Error("Failed to read file name from request", err)
		_, _ = writeFail(ctx, conn)
		return
	}

	fileSize, err := s.readFileSize(ctx, conn)
	if err != nil {
		slog.Default().Error("Failed to read file size from request", err)
		_, _ = writeFail(ctx, conn)
		return
	}

	err = s.sower.enqueuePlotDownload(ctx, fileName, fileSize, conn, conn)
	if err != nil {
		slog.Default().Error("Failed to add plot download to queue", err, slog.String("name", fileName))
		_, _ = writeFail(ctx, conn)
		return
	}

	_, err = writeSuccess(ctx, conn)
	if err != nil {
		slog.Default().Error("Failed to send success status", err)
		return
	}
}

func (s *TcpServer) readFileName(ctx context.Context, conn net.Conn) (name string, err error) {
	cr := util.NewContextReader(ctx, conn)

	fileNameSizeBytes := make([]byte, 1)
	_, err = io.ReadFull(cr, fileNameSizeBytes)
	if err != nil {
		return "", err
	}

	fileNameBytes := make([]byte, int(fileNameSizeBytes[0]))
	_, err = io.ReadFull(cr, fileNameBytes)
	if err != nil {
		return "", err
	}

	fileName := string(fileNameBytes)
	if !strings.HasSuffix(fileName, ".plot") {
		return "", fmt.Errorf("request provided incorrect file name %s", fileName)
	}

	return fileName, nil
}

func (s *TcpServer) readFileSize(ctx context.Context, reader io.Reader) (size uint64, err error) {
	cr := util.NewContextReader(ctx, reader)

	fileSizeBytes := make([]byte, 8)
	_, err = io.ReadFull(cr, fileSizeBytes)
	if err != nil {
		return 0, err
	}

	fileSize := binary.LittleEndian.Uint64(fileSizeBytes)

	return fileSize, nil
}
