package runsignal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/availability"
)

// CatalogResolution conservatively resolves an executable against the read-only manifest index.
type CatalogResolution struct {
	Owner, Command, Snapshot string
	// Group is a resolved path the manifest declares subcommands under: the
	// next word of the line still selects the operation (`test-genie runs
	// wait`) even when the manifest does not list it.
	Group            bool
	State            availability.State
	Reason           string
	SemanticsKind    string
	SemanticsVerdict string
}

const (
	OwnershipResolved           = "resolved"
	OwnershipExternal           = "external"
	OwnershipNotACommand        = "not-a-command"
	OwnershipCompoundUnresolved = "compound-unresolved"
	OwnershipUnparseable        = "unparseable"
)

type manifest struct {
	Name       string              `json:"name"`
	Groups     []manifestGroup     `json:"groups"`
	Exceptions []manifestException `json:"exceptions"`
}

// manifestException is a command a CLI serves outside its manifest bindings
// (a durable run, a stream): it is a command path like any other, and it may
// declare semantics (`test-genie runs wait` maps the run's verdict to its exit
// code).
type manifestException struct {
	Command   string             `json:"command"`
	Semantics *manifestSemantics `json:"semantics,omitempty"`
}
type manifestGroup struct {
	Name     string            `json:"name"`
	Flat     bool              `json:"flat,omitempty"`
	Commands []manifestCommand `json:"commands"`
	Groups   []manifestGroup   `json:"groups"`
}
type manifestCommand struct {
	Name      string             `json:"name"`
	Semantics *manifestSemantics `json:"semantics,omitempty"`
}
type manifestSemantics struct {
	Kind    string           `json:"kind"`
	Verdict *manifestVerdict `json:"verdict,omitempty"`
	Blocks  *manifestBlocks  `json:"blocks,omitempty"`
}

// manifestBlocks declares that a command blocks until its target changes:
// only with Flag when one is named, and only on a target one of After acted
// on first (`vrooli scenario wait` blocks on an in-flight start or restart;
// with none it returns at once). Replaces are the reads an agent polling one
// target can replace with one call of it.
type manifestBlocks struct {
	Flag     string   `json:"flag,omitempty"`
	After    []string `json:"after,omitempty"`
	Replaces []string `json:"replaces,omitempty"`
}
type manifestVerdict struct {
	Path string `json:"path,omitempty"`
	Pass string `json:"pass,omitempty"`
	Fail string `json:"fail,omitempty"`
}

// catalog is the manifest index parsed once: every declared command path by
// owner, what a command's exit means, the commands declared blocking by path,
// and those that replace polling each read, in declaration order. Snapshot
// names the manifest bytes it was built from.
type catalog struct {
	root, snapshot, stamp string
	checked               time.Time
	owners                map[string]map[string]bool
	semantics             map[string]manifestSemantics
	blocks                map[string]Blocking
	replacing             map[string][]Blocking
}

// catalogRecheck bounds how often the manifests are stat'ed for a change: a
// resolution runs per command call, a manifest changes when a CLI is edited.
var catalogRecheck = 5 * time.Second

var catalogCache struct {
	sync.Mutex
	current *catalog
}

// ResolveCatalog returns the conservative, manifest-backed classification of
// a shell line that runs one command (wrappers such as `bash -lc '…'`, env
// assignments, redirections and `cd` prefixes are not commands). A line that
// runs several commands, or cannot be lexed, is unknown.
func ResolveCatalog(command string) CatalogResolution {
	units, ok := shellUnits(command)
	if !ok || len(units) != 1 || units[0].Kind != unitCommand || dynamicProgram(units[0].Tokens[0]) {
		return CatalogResolution{State: availability.Unknown, Reason: "command is not one simple command"}
	}
	return resolveCatalogTokens(units[0].Tokens)
}

func resolveCatalogTokens(tokens []string) CatalogResolution {
	c := currentCatalog()
	snapshot := c.snapshot
	paths, known := c.owners[tokens[0]]
	if !known {
		// A safely tokenized command has a factual executable even when it is
		// not catalog-owned. Compound or unparseable shell remains unknown.
		executable := filepath.Base(tokens[0])
		return CatalogResolution{Owner: executable, Command: executable, State: availability.External, Snapshot: snapshot}
	}
	// A root invocation is still catalog-backed; subcommand precision is only
	// claimed where the manifest has a matching path. Harnesses may place
	// boolean control flags before the command path, so compare both the
	// literal argv prefix and a flag-free path view. Flags after the path are
	// naturally ignored as the longest declared prefix wins.
	pathTokens := tokens
	if filtered := stripLeadingCommandFlags(tokens); len(filtered) > 0 {
		pathTokens = filtered
	}
	for n := len(pathTokens); n > 1; n-- {
		candidate := strings.Join(pathTokens[:n], " ")
		if paths[candidate] {
			return c.resolution(tokens[0], candidate, paths)
		}
	}
	if paths[tokens[0]] {
		return c.resolution(tokens[0], tokens[0], paths)
	}
	return CatalogResolution{Owner: tokens[0], Snapshot: snapshot, State: availability.Unknown, Reason: "catalog executable has no matching command path"}
}

func stripLeadingCommandFlags(tokens []string) []string {
	if len(tokens) < 2 {
		return tokens
	}
	filtered := []string{tokens[0]}
	for _, token := range tokens[1:] {
		if strings.HasPrefix(token, "-") {
			continue
		}
		filtered = append(filtered, token)
	}
	return filtered
}

func (c *catalog) resolution(owner, command string, paths map[string]bool) CatalogResolution {
	// command is the full catalog path, owner first ("agent-manager effort lint").
	semantics := c.semantics[command]
	result := CatalogResolution{Owner: owner, Command: command, Snapshot: c.snapshot, State: availability.Resolved, SemanticsKind: semantics.Kind}
	for path := range paths {
		if strings.HasPrefix(path, command+" ") {
			result.Group = true
			break
		}
	}
	if semantics.Verdict != nil {
		result.SemanticsVerdict = semantics.Verdict.Path + "|" + semantics.Verdict.Pass + "|" + semantics.Verdict.Fail
	}
	return result
}

// CurrentCatalogSnapshot returns the manifest-index digest used at import time.
func CurrentCatalogSnapshot() string {
	return currentCatalog().snapshot
}

// manifestPaths are every scenario CLI's manifest and the control plane's
// (cli/manifest.json, the `vrooli` CLI), in the order the snapshot hashes them.
func manifestPaths(root string) []string {
	entries, _ := filepath.Glob(filepath.Join(root, "scenarios", "*", "cli", "manifest.json"))
	sort.Strings(entries)
	return append(entries, filepath.Join(root, "cli", "manifest.json"))
}

// manifestStamp names the manifests' paths, sizes and modification times: a
// changed stamp means the index must be rebuilt.
func manifestStamp(paths []string) string {
	var b strings.Builder
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			fmt.Fprintf(&b, "%s %d %d\n", path, info.Size(), info.ModTime().UnixNano())
		}
	}
	return b.String()
}

// currentCatalog is the index for the project root, rebuilt when a manifest
// was added, removed or changed since it was built (checked at most every
// catalogRecheck), so the snapshot a fact records names the manifests its
// resolution used.
func currentCatalog() *catalog {
	root := projectRoot()
	catalogCache.Lock()
	defer catalogCache.Unlock()
	c := catalogCache.current
	if c != nil && c.root == root && time.Since(c.checked) < catalogRecheck {
		return c
	}
	paths := manifestPaths(root)
	stamp := manifestStamp(paths)
	if c != nil && c.root == root && c.stamp == stamp {
		c.checked = time.Now()
		return c
	}
	c = buildCatalog(root, paths)
	c.stamp, c.checked = stamp, time.Now()
	catalogCache.current = c
	return c
}

func buildCatalog(root string, entries []string) *catalog {
	c := &catalog{root: root, owners: map[string]map[string]bool{}, semantics: map[string]manifestSemantics{}, blocks: map[string]Blocking{}, replacing: map[string][]Blocking{}}
	hash := sha256.New()
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
			collectCommands(value.Name, nil, group, paths, c)
		}
		for _, exception := range value.Exceptions {
			if command := strings.Join(strings.Fields(exception.Command), " "); command != "" {
				path := value.Name + " " + command
				paths[path] = true
				c.note(path, exception.Semantics)
			}
		}
		c.owners[value.Name] = paths
	}
	c.snapshot = "manifest-index:" + hex.EncodeToString(hash.Sum(nil))[:16]
	return c
}

// note records what the manifest declares for one command path.
func (c *catalog) note(path string, semantics *manifestSemantics) {
	if semantics == nil {
		return
	}
	c.semantics[path] = *semantics
	if b := semantics.Blocks; b != nil {
		c.blocks[path] = Blocking{Path: path, Flag: b.Flag, After: b.After}
		for _, read := range b.Replaces {
			c.replacing[read] = append(c.replacing[read], c.blocks[path])
		}
	}
}

func collectCommands(owner string, prefix []string, group manifestGroup, paths map[string]bool, c *catalog) {
	next := append([]string{}, prefix...)
	if !group.Flat {
		next = append(next, group.Name)
		if group.Name != "" {
			paths[strings.Join(append([]string{owner}, next...), " ")] = true
		}
	}
	for _, command := range group.Commands {
		if command.Name != "" {
			path := strings.Join(append(append([]string{owner}, next...), command.Name), " ")
			paths[path] = true
			c.note(path, command.Semantics)
		}
	}
	for _, child := range group.Groups {
		collectCommands(owner, next, child, paths, c)
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

// Blocking is a command its manifest declares blocks until its target changes
// (semantics.blocks): always, with Flag when one is named, and only on a target
// one of After (command paths as facts record them) acted on first.
type Blocking struct {
	Path, Flag string
	After      []string
}

// Command is the blocking call as an agent types it: `agent-manager run
// events --follow`.
func (b Blocking) Command() string {
	return strings.TrimSpace(b.Path + " " + b.Flag)
}

// BlockingSiblings are the commands the owning CLI's manifest (commands and
// declared exceptions alike, and the control plane's cli/manifest.json)
// declares replace polling commandPath, in declaration order: `vrooli scenario
// wait` for `vrooli scenario status`. A verb is never evidence (`agent-manager
// run await-result` re-reads a parked result and blocks on nothing), and a
// listing or a read of something else has none.
func BlockingSiblings(commandPath string) []Blocking {
	return currentCatalog().replacing[commandPath]
}

// Blocks reports whether a call of commandPath with words blocks by its
// manifest's declaration: the command blocks, and its flag (when it names
// one) is among words.
func Blocks(commandPath string, words []string) bool {
	b, ok := currentCatalog().blocks[commandPath]
	return ok && (b.Flag == "" || slices.ContainsFunc(words, func(w string) bool { return w == b.Flag || strings.HasPrefix(w, b.Flag+"=") }))
}
