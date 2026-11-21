package audit

import (
	"context"
	"os"
	"path/filepath"
	"sort"
)

// DirScenarioLister returns every directory under <repoRoot>/scenarios/
// as a candidate scenario name. Hidden entries (".") are skipped. Used
// by Service.RunAll in production.
type DirScenarioLister struct {
	repoRoot string
}

// NewDirScenarioLister constructs the production ScenarioLister.
func NewDirScenarioLister(repoRoot string) *DirScenarioLister {
	return &DirScenarioLister{repoRoot: repoRoot}
}

// List returns sorted scenario directory names.
func (l *DirScenarioLister) List(_ context.Context) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(l.repoRoot, "scenarios"))
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "" || name[0] == '.' {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
}
