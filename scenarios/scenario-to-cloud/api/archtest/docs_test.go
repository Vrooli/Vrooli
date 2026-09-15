package archtest

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// commandTree is the subset of a CLI manifest the documentation check reads.
type commandTree struct {
	Name     string        `json:"name"`
	Flat     bool          `json:"flat"`
	Commands []commandLeaf `json:"commands"`
	Groups   []commandTree `json:"groups"`
}

type commandLeaf struct {
	Name string `json:"name"`
}

type cliManifest struct {
	Groups []commandTree `json:"groups"`
}

// commandPaths returns every command path of a manifest as space-joined
// words ("deployment recovery-points list"). Flat groups contribute no
// segment, the same rule clidoc and scopecatalog apply.
func commandPaths(m cliManifest) map[string]struct{} {
	out := map[string]struct{}{}
	var walk func(g commandTree, parents []string)
	walk = func(g commandTree, parents []string) {
		path := parents
		if !g.Flat {
			path = append(append([]string(nil), parents...), g.Name)
		}
		for _, c := range g.Commands {
			out[strings.Join(append(append([]string(nil), path...), c.Name), " ")] = struct{}{}
		}
		for _, child := range g.Groups {
			walk(child, path)
		}
	}
	for _, g := range m.Groups {
		walk(g, nil)
	}
	return out
}

func loadManifest(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m cliManifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	paths := commandPaths(m)
	if len(paths) == 0 {
		t.Fatalf("%s declares no commands", path)
	}
	return paths
}

// documentedInvocation is one fenced command line in a markdown document.
type documentedInvocation struct {
	File  string
	Line  int
	Words []string
}

var (
	fenceLine = regexp.MustCompile("^\\s*(```|~~~)")
	wordToken = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
)

// fencedInvocations scans every markdown file under root for fenced lines
// that invoke the binary and returns the words that follow it.
func fencedInvocations(t *testing.T, root, binary string) []documentedInvocation {
	t.Helper()
	var out []documentedInvocation
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		rel, _ := filepath.Rel(root, path)
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 1<<20), 1<<20)
		inFence := false
		line := 0
		for scanner.Scan() {
			line++
			text := scanner.Text()
			if fenceLine.MatchString(text) {
				inFence = !inFence
				continue
			}
			if !inFence {
				continue
			}
			trimmed := strings.TrimSpace(text)
			trimmed = strings.TrimPrefix(trimmed, "$ ")
			if trimmed == binary || strings.HasPrefix(trimmed, binary+" ") {
				fields := strings.Fields(strings.TrimPrefix(trimmed, binary))
				out = append(out, documentedInvocation{File: filepath.ToSlash(rel), Line: line, Words: fields})
			}
		}
		return scanner.Err()
	})
	if err != nil {
		t.Fatalf("scan %s: %v", root, err)
	}
	return out
}

// commandWords returns the leading words of an invocation that can be part
// of a command path: lower-case identifiers before the first flag,
// placeholder or literal argument.
func commandWords(words []string) []string {
	var out []string
	for _, w := range words {
		if !wordToken.MatchString(w) {
			break
		}
		out = append(out, w)
	}
	return out
}

// resolveCommand finds the longest documented prefix that is a manifest
// command. An invocation whose words never reach a command is undocumented.
func resolveCommand(paths map[string]struct{}, words []string) (string, bool) {
	for n := len(words); n > 0; n-- {
		candidate := strings.Join(words[:n], " ")
		if _, ok := paths[candidate]; ok {
			return candidate, true
		}
	}
	return "", false
}

// nearest returns the manifest command closest to the words by edit
// distance, so a failure names the likely intended command.
func nearest(paths map[string]struct{}, words []string) string {
	target := strings.Join(words, " ")
	best, bestDistance := "", -1
	keys := make([]string, 0, len(paths))
	for k := range paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		d := editDistance(target, k)
		if bestDistance < 0 || d < bestDistance {
			best, bestDistance = k, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// documentedBinaries names each binary documentation may invoke inside a
// fence and the manifest that declares its commands.
func documentedBinaries(t *testing.T) map[string]map[string]struct{} {
	t.Helper()
	scenario := scenarioRoot(t)
	repo := filepath.Dir(filepath.Dir(scenario))
	return map[string]map[string]struct{}{
		"scenario-to-cloud":   loadManifest(t, filepath.Join(scenario, "cli", "manifest.json")),
		"vrooli cloud-target": prefixed(loadManifest(t, filepath.Join(repo, "cli", "manifest.json")), "cloud-target "),
	}
}

// prefixed keeps the command paths under one group and strips the group.
func prefixed(paths map[string]struct{}, prefix string) map[string]struct{} {
	out := map[string]struct{}{}
	for k := range paths {
		if strings.HasPrefix(k, prefix) {
			out[strings.TrimPrefix(k, prefix)] = struct{}{}
		}
	}
	return out
}

// TestDocumentedCommandsExistInTheManifest [REQ:STC-P0-043] scans every
// fenced `scenario-to-cloud …` and `vrooli cloud-target …` invocation under
// docs/ and fails, naming file:line and the nearest command, when the
// command path is not declared by the owning CLI manifest.
func TestDocumentedCommandsExistInTheManifest(t *testing.T) {
	docs := filepath.Join(scenarioRoot(t), "docs")
	total := 0
	for binary, paths := range documentedBinaries(t) {
		invocations := fencedInvocations(t, docs, binary)
		total += len(invocations)
		for _, inv := range invocations {
			words := commandWords(inv.Words)
			if len(words) == 0 {
				t.Errorf("docs/%s:%d: `%s` with no command", inv.File, inv.Line, binary)
				continue
			}
			if _, ok := resolveCommand(paths, words); ok {
				continue
			}
			t.Errorf("docs/%s:%d: `%s %s` is not a manifest command; nearest is `%s %s`", inv.File, inv.Line, binary, strings.Join(words, " "), binary, nearest(paths, words))
		}
	}
	if total < 50 {
		t.Fatalf("scanned only %d fenced invocations under docs/; the scan is broken", total)
	}
}
