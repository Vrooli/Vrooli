package runreport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// CatalogResolution is the conservative result of resolving an executable
// against the read-only CLI Health-compatible manifest index. Unknown is a
// first-class outcome: no shell text is promoted to project ownership.
type CatalogResolution struct {
	Owner, Command, Snapshot, State string
}

type manifest struct {
	Name   string          `json:"name"`
	Groups []manifestGroup `json:"groups"`
}
type manifestGroup struct {
	Name     string            `json:"name"`
	Commands []manifestCommand `json:"commands"`
	Groups   []manifestGroup   `json:"groups"`
}
type manifestCommand struct {
	Name string `json:"name"`
}

var catalogCache struct {
	sync.Mutex
	root     string
	owners   map[string]map[string]bool
	snapshot string
}

func resolveCatalog(command string) CatalogResolution {
	command = unwrapShellCommand(command)
	tokens, ok := safeTokens(command)
	if !ok || len(tokens) == 0 {
		return CatalogResolution{State: "unknown"}
	}
	root, owners, snapshot := loadCatalog()
	_ = root
	paths, known := owners[tokens[0]]
	if !known {
		return CatalogResolution{State: "external", Snapshot: snapshot}
	}
	// A root invocation is still catalog-backed; subcommand precision is only
	// claimed where the manifest has a matching path.
	if len(tokens) == 1 {
		return CatalogResolution{Owner: tokens[0], Command: tokens[0], Snapshot: snapshot, State: "resolved"}
	}
	for n := len(tokens); n > 1; n-- {
		candidate := strings.Join(tokens[:n], " ")
		if paths[candidate] {
			return CatalogResolution{Owner: tokens[0], Command: candidate, Snapshot: snapshot, State: "resolved"}
		}
	}
	return CatalogResolution{Owner: tokens[0], Snapshot: snapshot, State: "unknown"}
}

// unwrapShellCommand recognizes only the literal non-interpolated wrapper
// emitted by supported runners. It never evaluates shell syntax: anything
// compound, unquoted, or containing a second quote remains unresolved.
func unwrapShellCommand(command string) string {
	command = strings.TrimSpace(command)
	for _, prefix := range []string{"/bin/bash -lc ", "bash -lc ", "/bin/sh -c ", "sh -c "} {
		if !strings.HasPrefix(command, prefix) {
			continue
		}
		inner := strings.TrimSpace(strings.TrimPrefix(command, prefix))
		if len(inner) < 2 || inner[0] != '\'' || inner[len(inner)-1] != '\'' {
			return command
		}
		inner = inner[1 : len(inner)-1]
		if strings.Contains(inner, "'") {
			return command
		}
		if _, ok := safeTokens(inner); ok {
			return inner
		}
		return command
	}
	return command
}

func safeTokens(command string) ([]string, bool) {
	command = strings.TrimSpace(command)
	if command == "" || strings.ContainsAny(command, "|;&`$\n\r") {
		return nil, false
	}
	return strings.Fields(command), true
}

func loadCatalog() (string, map[string]map[string]bool, string) {
	root := projectRoot()
	catalogCache.Lock()
	defer catalogCache.Unlock()
	if catalogCache.root == root && catalogCache.owners != nil {
		return root, catalogCache.owners, catalogCache.snapshot
	}
	owners := map[string]map[string]bool{}
	hash := sha256.New()
	entries, _ := filepath.Glob(filepath.Join(root, "scenarios", "*", "cli", "manifest.json"))
	sort.Strings(entries)
	for _, path := range entries {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		_, _ = hash.Write(body)
		var value manifest
		if json.Unmarshal(body, &value) != nil || value.Name == "" {
			continue
		}
		paths := map[string]bool{value.Name: true}
		for _, group := range value.Groups {
			collectCommands(value.Name, nil, group, paths)
		}
		owners[value.Name] = paths
	}
	catalogCache.root, catalogCache.owners = root, owners
	catalogCache.snapshot = "manifest-index:" + hex.EncodeToString(hash.Sum(nil))[:16]
	return root, owners, catalogCache.snapshot
}

func collectCommands(owner string, prefix []string, group manifestGroup, paths map[string]bool) {
	next := append(append([]string{}, prefix...), group.Name)
	for _, command := range group.Commands {
		if command.Name != "" {
			paths[strings.Join(append(append([]string{owner}, next...), command.Name), " ")] = true
		}
	}
	for _, child := range group.Groups {
		collectCommands(owner, next, child, paths)
	}
}

func projectRoot() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_PROJECT_ROOT")); root != "" {
		return root
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		// Agent Manager's API template includes a nested scenarios fixture;
		// only a root containing the real fleet marker is the project root.
		if info, err := os.Stat(filepath.Join(dir, "scenarios", "agent-manager")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}
