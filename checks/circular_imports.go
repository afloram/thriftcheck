// Copyright 2025 Pinterest
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package checks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pinterest/thriftcheck"
	"go.uber.org/thriftrw/ast"
)

type C struct {
	adjList      map[int][]int
	edgeMeta     map[int]map[int]*ast.Include
	inDegrees    map[int]int
	filenameToId map[string]int
	idToFilename map[int]string
}

func getRelPath(f string) (string, error) {
	wd, err := os.Getwd()

	if err != nil {
		return "", errors.New("could not get current working directory")
	}

	absP, err := filepath.Abs(f)

	if err != nil {
		return "", fmt.Errorf("could not get absolute path for %s", f)
	}

	return filepath.Rel(wd, absP)
}

func getFilenameId(c *C, f string) int {
	if _, exists := c.filenameToId[f]; !exists {
		nextId := len(c.filenameToId) + 1

		c.filenameToId[f] = nextId
		c.idToFilename[nextId] = f
	}

	return c.filenameToId[f]
}

// CheckCircularImport returns a thriftcheck.Check that reports an error
// if there is a circular import.
func CheckCircularImport() *thriftcheck.Check {
	fn := func(c *thriftcheck.C, cc *C, i *ast.Include) {
		importer, err := getRelPath(c.Filename)

		if err != nil {
			return
		}

		importee, err := getRelPath(i.Path)

		if err != nil {
			return
		}

		// a imports b
		a := getFilenameId(cc, importer)
		b := getFilenameId(cc, importee)

		for _, v := range []int{a, b} {
			if _, exists := cc.adjList[v]; !exists {
				cc.inDegrees[v] = 0
				cc.adjList[v] = []int{}
			}
		}

		cc.inDegrees[b] += 1
		cc.adjList[a] = append(cc.adjList[a], b)

		if _, exists := cc.edgeMeta[a]; !exists {
			cc.edgeMeta[a] = make(map[int]*ast.Include)
			cc.edgeMeta[a][b] = i
		}
	}

	circularImportCtx := &C{
		adjList:      make(map[int][]int),
		edgeMeta:     make(map[int]map[int]*ast.Include),
		inDegrees:    make(map[int]int),
		filenameToId: make(map[string]int),
		idToFilename: make(map[int]string),
	}

	return thriftcheck.NewMultiFileCheck("import.cycle.disallowed", fn, circularImportCtx, func(cc *C) {
		imports, cycle := lookForCycle(cc.adjList, cc.inDegrees)

		if cycle {
			fmt.Println("Cycle detected:")

			for i, im := range imports {
				inc := cc.edgeMeta[im][imports[(i+1)%len(imports)]]
				fmt.Printf(
					"%s -> %s\n"+
						"\tIncluded as: %s\n"+
						"\tAt: %s:%d:%d\n\n",
					filepath.Base(cc.idToFilename[im]), filepath.Base(inc.Path),
					inc.Path,
					cc.idToFilename[im], inc.Line, inc.Column,
				)
			}
		}
	})
}

// Topological processing
// https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm
func lookForCycle(adjList map[int][]int, inDegrees map[int]int) ([]int, bool) {
	count := 0
	sources := []int{}

	for v := range adjList {
		if inDegrees[v] == 0 {
			count += 1
			sources = append(sources, v)
		}
	}

	for len(sources) != 0 {
		newSources := []int{}

		for _, source := range sources {
			for _, v := range adjList[source] {
				inDegrees[v] -= 1
				if inDegrees[v] == 0 {
					count += 1
					newSources = append(sources, v)
				}
			}
		}

		sources = newSources
	}

	// there is at least one cycle,
	// so find the vertices of any of them
	if count != len(adjList) {
		return findCycleVertices(adjList), true
	}

	return nil, false
}

func findCycleVertices(adjList map[int][]int) []int {
	vis := make(map[int]bool)

	for v := range adjList {
		if vs := dfs(v, adjList, []int{}, make(map[int]bool), vis); vs != nil {
			return vs
		}
	}

	panic("unreachable (expected a cycle to exist)")
}

// Returns all of the vertices of a cycle if found, otherwise returns nil.
func dfs(cur int, adjList map[int][]int, vertices []int, vis map[int]bool, globalVis map[int]bool) []int {
	if vis[cur] {
		// return just the cycle (remove the vertices leading to it)
		for i, v := range vertices {
			if v == cur {
				return vertices[i:]
			}
		}
	}

	// path already explored
	if globalVis[cur] {
		return nil
	}

	vis[cur], globalVis[cur] = true, true

	for _, v := range adjList[cur] {
		if vs := dfs(v, adjList, append(vertices, cur), vis, globalVis); vs != nil {
			return vs
		}
	}

	return nil
}
