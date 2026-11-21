// Package graphfixtures lists source-project fixtures shared by graph providers.
package graphfixtures

import (
	"os"
	"path/filepath"
	"sort"
)

const ExpectedGraphFile = "expected-graph.json"

type Fixture struct {
	Name        string
	Path        string
	HasExpected bool
}

// List returns directories under root that contain projectFile, ordered by
// name. A missing fixtures root is an empty collection; other read errors are
// returned to the serving owner.
func List(root, projectFile string) ([]Fixture, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []Fixture{}, nil
		}
		return nil, err
	}

	fixtures := make([]Fixture, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, projectFile)); err != nil {
			continue
		}
		_, expectedErr := os.Stat(filepath.Join(dir, ExpectedGraphFile))
		fixtures = append(fixtures, Fixture{
			Name:        entry.Name(),
			Path:        filepath.ToSlash(dir),
			HasExpected: expectedErr == nil,
		})
	}
	sort.Slice(fixtures, func(i, j int) bool { return fixtures[i].Name < fixtures[j].Name })
	return fixtures, nil
}
