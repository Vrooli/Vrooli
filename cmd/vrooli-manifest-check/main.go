// Command vrooli-manifest-check verifies the root CLI manifest against the
// leaf paths registered by the root command tree.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

const (
	mndMainNumberValue2 = 2
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	root, err := buildinfo.ResolveSourceRoot()
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "cli", "manifest.json"))
	if err != nil {
		return fmt.Errorf("read root CLI manifest: %w", err)
	}
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		return err
	}
	want := manifestCommandPaths(manifest)
	got := rootcli.WalkCommandTree()

	var differences []string
	for _, path := range difference(got, want) {
		differences = append(differences, "tree-only: "+path)
	}
	for _, path := range difference(want, got) {
		differences = append(differences, "manifest-only: "+path)
	}
	if len(differences) > 0 {
		sort.Strings(differences)
		return fmt.Errorf("root CLI manifest drift:\n%s", strings.Join(differences, "\n"))
	}
	return nil
}

func manifestCommandPaths(manifest *cliapp.Manifest) []string {
	paths := make([]string, 0)
	var walk func([]cliapp.ManifestGroup, []string)
	walk = func(groups []cliapp.ManifestGroup, parents []string) {
		for _, group := range groups {
			if group.Name == "top-level" {
				// The top-level group is a governance catalog of parent commands,
				// intentionally excluded from the invocable leaf comparison.
				continue
			}
			prefix := append([]string(nil), parents...)
			if !group.Flat {
				// The scenario management group is a governance partition for
				// the root's flat scenario handler; its children are registered
				// directly beneath `vrooli scenario`.
				if !(len(parents) > 0 && parents[0] == "scenario" && group.Name == "management") {
					prefix = append(prefix, strings.TrimSpace(group.Name))
				}
			}
			for _, command := range group.Commands {
				path := append(append([]string(nil), prefix...), strings.TrimSpace(command.Name))
				paths = append(paths, strings.Join(path, " "))
			}
			walk(group.Groups, prefix)
		}
	}
	walk(manifest.Groups, nil)
	sort.Strings(paths)
	return unique(paths)
}

func difference(left, right []string) []string {
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		rightSet[value] = struct{}{}
	}
	out := make([]string, 0)
	for _, value := range left {
		if _, ok := rightSet[value]; !ok {
			out = append(out, value)
		}
	}
	return unique(out)
}

func unique(values []string) []string {
	if len(values) < mndMainNumberValue2 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}
