package runsignal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"agent-manager/internal/availability"
)

// CatalogResolution conservatively resolves an executable against the read-only manifest index.
type CatalogResolution struct {
	Owner, Command, Snapshot string
	State                    availability.State
	Reason                   string
}

const (
	OwnershipResolved           = "resolved"
	OwnershipExternal           = "external"
	OwnershipNotACommand        = "not-a-command"
	OwnershipCompoundUnresolved = "compound-unresolved"
	OwnershipUnparseable        = "unparseable"
)

// shellSegments is deliberately a non-evaluating lexer. It recognizes only
// the separators Agent Manager can classify safely; command substitution,
// process substitution, malformed quotes, and other shell grammar remain
// unparseable evidence rather than being guessed at.
func shellSegments(command string) ([]string, bool, bool) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, false, false
	}
	segments := []string{}
	start := 0
	var quote rune
	escaped := false
	compound := false
	for i, r := range command {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		// The first byte of &&/|| advances start past the second byte;
		// ignore that second byte when range visits it.
		if (r == '|' || r == '&') && i < start {
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '`':
			return nil, false, true
		case '$':
			// `$(` and `${` can change the command or arguments. The
			// latter is also rejected because expanding it requires shell.
			return nil, false, true
		case '<', '>':
			// Process substitution is explicitly unparseable. Ordinary
			// redirection is retained as part of the segment.
			if i+1 < len(command) && command[i+1] == '(' {
				return nil, false, true
			}
		case ';', '|', '&':
			if r == '&' && i+1 >= len(command) {
				return nil, false, true
			}
			if r == '|' && i+1 < len(command) && command[i+1] == '|' || r == '&' && i+1 < len(command) && command[i+1] == '&' {
				// The range index is a byte offset for ASCII operators.
				compound = true
			}
			if r == ';' || r == '|' || r == '&' {
				part := strings.TrimSpace(command[start:i])
				if part == "" {
					return nil, false, true
				}
				segments = append(segments, part)
				start = i + 1
				if (r == '|' || r == '&') && start < len(command) && command[start] == byte(r) {
					start++
				}
			}
		}
	}
	if quote != 0 || escaped {
		return nil, false, true
	}
	part := strings.TrimSpace(command[start:])
	if part == "" {
		return nil, false, true
	}
	segments = append(segments, part)
	return segments, compound || len(segments) > 1, false
}

// SegmentShell returns literal shell segments and whether the source was a
// compound command. It never executes or expands the input.
func SegmentShell(command string) ([]string, bool, string) {
	segments, compound, unparseable := shellSegments(command)
	if unparseable {
		return nil, false, "shell syntax requires evaluation"
	}
	return segments, compound, ""
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

// ResolveCatalog returns the conservative, manifest-backed classification for a command.
func ResolveCatalog(command string) CatalogResolution {
	command = unwrapShellCommand(command)
	tokens, ok := safeTokens(command)
	if !ok || len(tokens) == 0 {
		return CatalogResolution{State: availability.Unknown, Reason: "command cannot be safely tokenized"}
	}
	_, owners, snapshot := loadCatalog()
	paths, known := owners[tokens[0]]
	if !known {
		// A safely tokenized command has a factual executable even when it is
		// not catalog-owned. Compound or unparseable shell remains unknown.
		executable := filepath.Base(tokens[0])
		return CatalogResolution{Owner: executable, Command: executable, State: availability.External, Snapshot: snapshot}
	}
	// A root invocation is still catalog-backed; subcommand precision is only
	// claimed where the manifest has a matching path.
	if len(tokens) == 1 {
		return CatalogResolution{Owner: tokens[0], Command: tokens[0], Snapshot: snapshot, State: availability.Resolved}
	}
	for n := len(tokens); n > 1; n-- {
		candidate := strings.Join(tokens[:n], " ")
		if paths[candidate] {
			return CatalogResolution{Owner: tokens[0], Command: candidate, Snapshot: snapshot, State: availability.Resolved}
		}
	}
	return CatalogResolution{Owner: tokens[0], Snapshot: snapshot, State: availability.Unknown, Reason: "catalog executable has no matching command path"}
}

// CurrentCatalogSnapshot returns the manifest-index digest used at import time.
func CurrentCatalogSnapshot() string {
	_, _, snapshot := loadCatalog()
	return snapshot
}

// unwrapShellCommand accepts only literal runner wrappers and never evaluates shell syntax.
func unwrapShellCommand(command string) string {
	command = strings.TrimSpace(command)
	for _, prefix := range []string{"/bin/bash -lc ", "bash -lc ", "/bin/sh -c ", "sh -c "} {
		if !strings.HasPrefix(command, prefix) {
			continue
		}
		inner := strings.TrimSpace(strings.TrimPrefix(command, prefix))
		if len(inner) < 2 || (inner[0] != '\'' && inner[0] != '"') || inner[len(inner)-1] != inner[0] {
			return command
		}
		quote := inner[0]
		inner = inner[1 : len(inner)-1]
		if strings.ContainsRune(inner, rune(quote)) {
			return command
		}
		return inner
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
	root, _ := os.LookupEnv("VROOLI_PROJECT_ROOT")
	if root = strings.TrimSpace(root); root != "" {
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
