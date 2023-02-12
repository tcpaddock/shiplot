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
	"sort"
	"sync"

	"github.com/ricochet2200/go-disk-usage/du"
	"golang.org/x/exp/slices"
)

type path struct {
	name      string
	usage     *du.DiskUsage
	available bool
}

type pathList []*path

var pathListMutex sync.Mutex

func (pl *pathList) Len() int { return len(*pl) }

func (pl *pathList) Swap(i, j int) { (*pl)[i], (*pl)[j] = (*pl)[j], (*pl)[i] }

func (pl *pathList) Less(i, j int) bool { return (*pl)[i].usage.Free() > (*pl)[j].usage.Free() }

func (pl *pathList) Populate(paths []string) {
	pathListMutex.Lock()

	for _, p := range paths {
		usage := du.NewDiskUsage(p)
		*pl = append(*pl, &path{name: p, usage: usage, available: true})
	}

	pathListMutex.Unlock()
}

func (pl *pathList) FirstAvailable() (path *path) {
	pathListMutex.Lock()

	sort.Sort(pl)

	for _, p := range *pl {
		if p.available {
			path = p
			p.available = false
			break
		}
	}

	pathListMutex.Unlock()

	return
}

func (pl *pathList) Update(path *path, available bool) {
	pathListMutex.Lock()

	index := slices.Index(*pl, path)

	p := (*pl)[index]
	p.available = available
	p.usage = du.NewDiskUsage(p.name)
	sort.Sort(pl)

	pathListMutex.Unlock()
}

func (pl *pathList) Remove(path *path) {
	pathListMutex.Lock()

	var newList pathList

	copy(newList, *pl)

	index := slices.Index(newList, path)

	if index >= 0 {
		newList = append(newList[:index], newList[index+1:]...)
	}

	pl = &newList

	pathListMutex.Unlock()
}
