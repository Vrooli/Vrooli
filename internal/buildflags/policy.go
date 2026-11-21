// Package buildflags loads the repository's compile-affecting Go flag policy.
package buildflags

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Policy struct {
	Develop      []string `json:"develop"`
	Distribution []string `json:"distribution"`
	Scenario     []string `json:"scenario"`
}

func Load(root string) (Policy, error) {
	data, err := os.ReadFile(filepath.Join(root, ".vrooli", "repo-contract.json"))
	if err != nil {
		return Policy{}, fmt.Errorf("read build flag policy: %w", err)
	}
	var doc struct {
		BuildGoFlags Policy `json:"build.go_flags"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return Policy{}, fmt.Errorf("decode build flag policy: %w", err)
	}
	for name, flags := range map[string][]string{"develop": doc.BuildGoFlags.Develop, "distribution": doc.BuildGoFlags.Distribution, "scenario": doc.BuildGoFlags.Scenario} {
		for _, flag := range flags {
			if !validFlag(flag) {
				return Policy{}, fmt.Errorf("build.go_flags.%s contains unsupported flag %q", name, flag)
			}
		}
	}
	return doc.BuildGoFlags, nil
}

func (p Policy) For(channel string) []string {
	var flags []string
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "develop":
		flags = p.Develop
	case "distribution":
		flags = p.Distribution
	case "scenario":
		flags = p.Scenario
	}
	return slices.Clone(flags)
}

func validFlag(flag string) bool {
	return flag == "-trimpath" || flag == "-buildvcs=false"
}
