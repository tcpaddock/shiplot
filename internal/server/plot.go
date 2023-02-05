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
	"io"
	"os"
	"time"
)

type plot struct {
	Name string
	File os.FileInfo
}

func NewPlot(name string) (p *plot) {
	p = new(plot)
	p.Name = name
	p.File, _ = os.Stat(name)
	return p
}

func (p *plot) Move(name string) (file os.FileInfo, written int64, duration time.Duration, err error) {
	src, err := os.Open(p.Name)
	if err != nil {
		src.Close()
		return nil, 0, 0, err
	}

	dst, err := os.Create(name)
	if err != nil {
		src.Close()
		dst.Close()
		return nil, 0, 0, err
	}

	start := time.Now()
	written, err = io.Copy(dst, src)
	duration = time.Since(start)
	if err != nil {
		src.Close()
		dst.Close()
		return nil, 0, 0, err
	}

	file, err = dst.Stat()
	if err != nil {
		src.Close()
		dst.Close()
		return nil, 0, 0, err
	}

	err = src.Close()
	if err != nil {
		return nil, 0, 0, err
	}

	err = dst.Close()
	if err != nil {
		return nil, 0, 0, err
	}

	// copy succeeded, delete source
	err = os.Remove(src.Name())
	if err != nil {
		return nil, 0, 0, err
	}

	return file, written, duration, nil
}
