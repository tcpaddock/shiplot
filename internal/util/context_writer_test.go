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
package util

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewContextWriter(t *testing.T) {
	// Arrange
	c := context.Background()
	w := &bytes.Buffer{}
	expected := &contextWriter{
		ctx:    c,
		writer: w,
	}

	// Act
	actual := NewContextWriter(c, w)

	// Assert
	require.Equal(t, expected, actual)
}

func TestWrite(t *testing.T) {
	// Arrange
	c := context.Background()
	b := bytes.Buffer{}
	cw := &contextWriter{
		ctx:    c,
		writer: &b,
	}
	expected := "test"

	// Act
	sr := strings.NewReader("test")
	_, _ = sr.WriteTo(cw)
	actual := b.String()

	// Assert
	require.Equal(t, expected, actual)
}

func TestWriteContext(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	b := bytes.Buffer{}
	cw := &contextWriter{
		ctx:    ctx,
		writer: &b,
	}

	// Act
	cancel()
	sr := strings.NewReader("test")
	_, err := sr.WriteTo(cw)

	// Assert
	require.Error(t, err)
}
