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

	"github.com/tcpaddock/shiplot/internal/config"
	"github.com/tcpaddock/shiplot/internal/util"
)

type TcpClient struct {
	cfg config.Config
}

func NewTcpClient(cfg config.Config) (c *TcpClient) {
	c = new(TcpClient)

	c.cfg = cfg

	return c
}

func (c *TcpClient) WritePlot(ctx context.Context, name string, size uint64, reader io.Reader) (written int64, err error) {
	conn, err := c.connect()
	if err != nil {
		return 0, err
	}

	defer conn.Close()

	_, err = c.writeFileName(ctx, name, conn)
	if err != nil {
		return 0, err
	}

	_, err = c.writeFileSize(ctx, size, conn)
	if err != nil {
		return 0, err
	}

	written, err = c.writePlot(ctx, reader, conn)
	if err != nil {
		return 0, err
	}

	ok := c.readResult(ctx, conn)
	if !ok {
		return 0, fmt.Errorf("server returned failure")
	}

	return written, nil
}

func (c *TcpClient) connect() (conn *net.TCPConn, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", c.cfg.Client.ServerIp, c.cfg.Client.ServerPort))
	if err != nil {
		return nil, err
	}

	conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (c *TcpClient) writeFileName(ctx context.Context, name string, writer io.Writer) (written int, err error) {
	cw := util.NewContextWriter(ctx, writer)

	fileNameSizeByte := byte(len(name))
	fileNameBytes := []byte(name)
	w1, err := cw.Write([]byte{fileNameSizeByte})
	if err != nil {
		return 0, err
	}

	w2, err := cw.Write(fileNameBytes)
	if err != nil {
		return 0, err
	}

	return w1 + w2, nil
}

func (c *TcpClient) writeFileSize(ctx context.Context, size uint64, writer io.Writer) (written int, err error) {
	cw := util.NewContextWriter(ctx, writer)

	fileSizeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(fileSizeBytes, size)
	written, err = cw.Write(fileSizeBytes)
	if err != nil {
		return 0, err
	}

	return
}

func (c *TcpClient) writePlot(ctx context.Context, reader io.Reader, writer io.Writer) (written int64, err error) {
	cr := util.NewContextReader(ctx, reader)
	cw := util.NewContextWriter(ctx, writer)

	return io.Copy(cw, cr)
}

func (c *TcpClient) readResult(ctx context.Context, reader io.Reader) (ok bool) {
	cr := util.NewContextReader(ctx, reader)
	result := make([]byte, 1)
	cr.Read(result)

	return result[0] == 1
}
