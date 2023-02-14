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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLen(t *testing.T) {
	var tests = []struct {
		inputPaths    []string
		expectedCount int
	}{
		{inputPaths: []string{"/test1"}, expectedCount: 1},
		{inputPaths: []string{"/test1", "/test2"}, expectedCount: 2},
		{inputPaths: []string{"/test1", "/test2", "/test3"}, expectedCount: 3},
	}

	for _, test := range tests {
		pl := new(pathList)

		pl.Populate(test.inputPaths)

		require.Equal(t, test.expectedCount, pl.Len())
	}
}
