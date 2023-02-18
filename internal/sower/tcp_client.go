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

func NewClient(cfg config.Config) (c *TcpClient) {
	c = new(TcpClient)

	c.cfg = cfg

	return c
}

func (c *TcpClient) WritePlot(ctx context.Context, name string, size uint64, reader io.Reader) (err error) {
	conn, err := c.connect()
	if err != nil {
		return err
	}

	defer conn.Close()

	err = writeFileName(name, conn)
	if err != nil {
		return err
	}

	err = writeFileSize(size, conn)
	if err != nil {
		return err
	}

	_, err = writePlot(ctx, reader, conn)
	if err != nil {
		return err
	}

	ok := readResult(conn)
	if !ok {
		return fmt.Errorf("server returned failure")
	}

	return nil
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

func writeFileName(name string, writer io.Writer) (err error) {
	fileNameSizeByte := byte(len(name))
	fileNameBytes := []byte(name)
	_, err = writer.Write([]byte{fileNameSizeByte})
	if err != nil {
		return err
	}

	_, err = writer.Write(fileNameBytes)
	if err != nil {
		return err
	}

	return nil
}

func writeFileSize(size uint64, writer io.Writer) (err error) {
	fileSizeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(fileSizeBytes, size)
	_, err = writer.Write(fileSizeBytes)
	if err != nil {
		return err
	}

	return nil
}

func writePlot(ctx context.Context, reader io.Reader, writer io.Writer) (written int64, err error) {
	cr := util.NewContextReader(ctx, reader)
	cw := util.NewContextWriter(ctx, writer)

	return io.Copy(cw, cr)
}

func readResult(reader io.Reader) (ok bool) {
	result := make([]byte, 1)
	reader.Read(result)

	return result[0] == 1
}
