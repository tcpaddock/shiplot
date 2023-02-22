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

	"github.com/ricochet2200/go-disk-usage/du"
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
		// Arrange
		pl := new(pathList)
		for _, p := range test.inputPaths {
			*pl = append(*pl, &path{name: p, usage: nil, available: true})
		}
		expected := test.expectedCount

		// Act
		actual := pl.Len()

		// Assert
		require.Equal(t, expected, actual)
	}
}

func TestSwap(t *testing.T) {
	// Arrange
	pl := new(pathList)
	*pl = append(*pl, &path{name: "/test1", usage: nil, available: true})
	*pl = append(*pl, &path{name: "/test2", usage: nil, available: true})
	expected := pathList{(*pl)[1], (*pl)[0]}

	// Act
	pl.Swap(0, 1)
	actual := *pl

	// Assert
	require.Equal(t, expected, actual)
}

func TestLess(t *testing.T) {
	// Arrange
	du1 := MockDiskUsage{}
	du1.On("Free").Return(uint64(2))
	du2 := MockDiskUsage{}
	du2.On("Free").Return(uint64(1))
	pl := new(pathList)
	*pl = append(*pl, &path{name: "/test1", usage: &du1, available: true})
	*pl = append(*pl, &path{name: "/test2", usage: &du2, available: true})
	expected := true

	// Act
	actual := pl.Less(0, 1)

	// Assert
	require.Equal(t, expected, actual)
}

func TestPopulate(t *testing.T) {
	// Arrange
	dirs := []string{"/test1", "/test2"}
	actual := new(pathList)
	expected := &pathList{
		&path{"/test1", du.NewDiskUsage("blank"), true},
		&path{"/test2", du.NewDiskUsage("blank"), true},
	}

	// Act
	actual.Populate(dirs)

	// Assert
	require.Equal(t, expected, actual)
}

func TestFirstAvailable(t *testing.T) {
	// Arrange
	du1 := MockDiskUsage{}
	du1.On("Free").Return(uint64(1))
	du2 := MockDiskUsage{}
	du2.On("Free").Return(uint64(2))
	du3 := MockDiskUsage{}
	du3.On("Free").Return(uint64(3))
	pl := &pathList{
		&path{name: "/test1", usage: &du1, available: true},
		&path{name: "/test2", usage: &du2, available: true},
		&path{name: "/test3", usage: &du3, available: false},
	}
	expected := (*pl)[1]

	// Act
	actual := pl.FirstAvailable()

	// Assert
	require.Equal(t, expected, actual)
}

func TestSetAvailable(t *testing.T) {
	// Arrange
	pl := &pathList{
		&path{name: "/test1", usage: du.NewDiskUsage("blank"), available: true},
	}
	expected := false

	// Act
	pl.SetAvailable((*pl)[0], false)
	actual := (*pl)[0].available

	// Assert
	require.Equal(t, expected, actual)
}

func TestRemove(t *testing.T) {
	// Arrange
	actual := &pathList{
		&path{name: "/test1", usage: du.NewDiskUsage("blank"), available: true},
		&path{name: "/test2", usage: du.NewDiskUsage("blank"), available: true},
		&path{name: "/test3", usage: du.NewDiskUsage("blank"), available: false},
	}
	expected := &pathList{
		&path{name: "/test1", usage: du.NewDiskUsage("blank"), available: true},
		&path{name: "/test3", usage: du.NewDiskUsage("blank"), available: false},
	}

	// Act
	actual.Remove((*actual)[1])

	// Assert
	require.Equal(t, expected, actual)
}
