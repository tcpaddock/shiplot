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
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tcpaddock/shiplot/internal/config"
)

func TestNewTcpClient(t *testing.T) {
	// Arrange
	expected := &TcpClient{
		cfg: config.Config{},
	}

	// Act
	actual := NewTcpClient(config.Config{})

	// Assert
	require.Equal(t, expected, actual)
}

func TestWriteFileName(t *testing.T) {
	// Arrange
	client := &TcpClient{
		cfg: config.Config{},
	}
	b := bytes.Buffer{}
	expected := []byte{0x04, 0x74, 0x65, 0x73, 0x74}

	// Act
	_, _ = client.writeFileName(context.Background(), "test", &b)

	// Assert
	require.Equal(t, expected, b.Bytes())
}

func TestWriteFileSize(t *testing.T) {
	// Arrange
	client := &TcpClient{
		cfg: config.Config{},
	}
	b := bytes.Buffer{}
	expected := []byte{0x33, 0x33, 0x33, 0x33, 0x1b, 0x00, 0x00, 0x00}

	// Act
	_, _ = client.writeFileSize(context.Background(), 116823110451, &b)

	// Assert
	require.Equal(t, expected, b.Bytes())
}

func TestReadResult(t *testing.T) {
	// Arrange
	client := &TcpClient{
		cfg: config.Config{},
	}
	b := bytes.Buffer{}
	b.Write([]byte{1})
	expected := true

	// Act
	actual := client.readResult(context.Background(), &b)

	// Assert
	require.Equal(t, expected, actual)
}
