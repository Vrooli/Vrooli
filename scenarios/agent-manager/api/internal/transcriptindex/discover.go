package transcriptindex

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	corestorage "github.com/vrooli/api-core/storage"
	"github.com/vrooli/cli-core/agenttranscript"
)

// RootConfig names the directories discovery starts from.
type RootConfig struct {
	// Home is the operator's OS home, holding the default Codex and Claude
	// harness homes (~/.codex, ~/.claude).
	Home string
	// GrokSessionsRoot overrides Grok's default ~/.grok/sessions directory
	// while retaining the default root for existing operator sessions.
	GrokSessionsRoot string
	// WebConsoleCodexRoot is the parent of web-console's per-session
	// CODEX_HOME directories (ResolveWebConsoleCodexRoot); empty skips them.
	WebConsoleCodexRoot string
	RunStateRoot        string
	OpenCodeArchiveDB   string
	OpenCodeArchiveRoot string
	CursorStateRoot     string
	ExtraRoots          []string
}

// ResolveWebConsoleCodexRoot resolves the parent of web-console's per-session
// CODEX_HOME directories; each child's sessions/ subtree holds that session's
// rollouts. It is the path web-console itself lays those homes out under:
// sessions/codex in the web-console owner's storage state directory, so a
// custom VROOLI_STORAGE_ROOT moves writer and reader together. cfg carries
// the resolver's seams.
func ResolveWebConsoleCodexRoot(cfg corestorage.ResolverConfig) (string, error) {
	resolver, err := corestorage.NewResolver(cfg)
	if err != nil {
		return "", err
	}
	return resolver.Path(corestorage.Options{ScenarioID: "web-console"}, corestorage.ClassState, filepath.Join("sessions", "codex"))
}

// Roots lists every transcript root on this host: web-console Codex homes,
// the default Codex and Claude homes, Agent Manager run homes, and configured
// extra roots. An extra root equal to or inside another root is dropped so no
// file is discovered twice.
func Roots(cfg RootConfig) []Root {
	var roots []Root
	if cfg.WebConsoleCodexRoot != "" {
		entries, _ := os.ReadDir(cfg.WebConsoleCodexRoot)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			for _, format := range agenttranscript.Formats() {
				harness := Family(format.Harness)
				for _, rel := range format.WebConsoleRoots {
					dir := filepath.Join(cfg.WebConsoleCodexRoot, entry.Name(), filepath.FromSlash(rel))
					if isDir(dir) {
						roots = append(roots, Root{Path: dir, Kind: RootWebConsoleCodex, Harness: harness, WebConsoleSessionID: entry.Name()})
					}
				}
			}
		}
	}
	for _, format := range agenttranscript.Formats() {
		harness := Family(format.Harness)
		for _, rel := range format.HomeRoots {
			kind := RootCodexHome
			switch harness {
			case HarnessClaude:
				kind = RootClaudeHome
			case HarnessGrok:
				kind = RootGrokHome
			case "gemini-cli":
				kind = RootGeminiCLI
			case "qwen-code":
				kind = RootQwenCode
			case "cursor":
				kind = RootCursor
			}
			roots = append(roots, Root{Path: filepath.Join(cfg.Home, filepath.FromSlash(rel)), Kind: kind, Harness: harness})
		}
		if harness == HarnessGrok && cfg.GrokSessionsRoot != "" {
			custom := filepath.Clean(cfg.GrokSessionsRoot)
			defaultRoot := filepath.Join(cfg.Home, ".grok", "sessions")
			if custom != filepath.Clean(defaultRoot) {
				roots = append(roots, Root{Path: custom, Kind: RootGrokHome, Harness: harness})
			}
		}
	}
	if cfg.RunStateRoot != "" {
		if entries, err := os.ReadDir(cfg.RunStateRoot); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				for _, format := range agenttranscript.Formats() {
					harness := Family(format.Harness)
					for _, rel := range format.RunRoots {
						dir := filepath.Join(cfg.RunStateRoot, entry.Name(), filepath.FromSlash(rel))
						if isDir(dir) {
							roots = append(roots, Root{Path: dir, Kind: RootRunHome, Harness: harness, RunID: entry.Name()})
						}
					}
				}
			}
		}
	}
	if cfg.OpenCodeArchiveRoot != "" && isDir(cfg.OpenCodeArchiveRoot) {
		roots = append(roots, Root{Path: cfg.OpenCodeArchiveRoot, Kind: RootOpenCodeArchive, Harness: "opencode"})
	}
	if cfg.CursorStateRoot != "" && isDir(cfg.CursorStateRoot) {
		duplicate := false
		for _, root := range roots {
			if root.Kind == RootCursor && filepath.Clean(root.Path) == filepath.Clean(cfg.CursorStateRoot) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			roots = append(roots, Root{Path: cfg.CursorStateRoot, Kind: RootCursor, Harness: "cursor"})
		}
	}
	var covered []string
	for _, format := range agenttranscript.Formats() {
		for _, rel := range format.HomeRoots {
			covered = append(covered, filepath.Join(cfg.Home, filepath.FromSlash(rel)))
		}
	}
	if cfg.WebConsoleCodexRoot != "" {
		covered = append(covered, filepath.Clean(cfg.WebConsoleCodexRoot))
	}
	if cfg.RunStateRoot != "" {
		covered = append(covered, filepath.Clean(cfg.RunStateRoot))
	}
	for _, extra := range cfg.ExtraRoots {
		cleaned := filepath.Clean(ExpandHome(strings.TrimSpace(extra), cfg.Home))
		if strings.TrimSpace(extra) == "" || pathCoveredBy(cleaned, covered) {
			continue
		}
		roots = append(roots, Root{Path: cleaned, Kind: RootConfigured, Harness: HarnessConfigured})
		covered = append(covered, cleaned)
	}
	return roots
}

// Family reduces a harness label ("codex", "resource:codex/sessions",
// "claude", "claude-code") to the transcript format it names.
func Family(harness string) string {
	family := agenttranscript.HarnessFamily(harness)
	if family == "claude" {
		return HarnessClaude
	}
	return family
}

// ExpandHome replaces a leading ~ with home.
func ExpandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func pathCoveredBy(cleaned string, covered []string) bool {
	for _, c := range covered {
		if cleaned == c || strings.HasPrefix(cleaned, c+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Candidate is one source conversation found by discovery. Most sources are
// one transcript file; multi-conversation stores can supply a stable key and
// session ID for each logical conversation in the source.
type Candidate struct {
	Root Root
	Path string
	Key  string
	// SessionID identifies one logical conversation within a multi-session source.
	SessionID     string
	Size          int64
	ContentHash   string
	MtimeUnixNano int64
}

// Walk lists the transcript files (any transcriptEncodings suffix) under root. A root that does not exist yields
// nothing; any other failure is returned so coverage can name it.
func Walk(root Root) ([]Candidate, error) {
	var out []Candidate
	err := agenttranscript.WalkNativeFiles(context.Background(), root.Path, agenttranscript.NativeWalkOptions{
		VisitNonRegularEntries: true,
		Errors:                 agenttranscript.NativeWalkSkipUnreadableSubdirectories,
	}, func(path string, entry os.DirEntry, _ os.FileInfo) error {
		if root.Kind == RootCursor {
			if entry.Name() != "state.vscdb" {
				return nil
			}
			found, err := discoverCursorDB(path, root)
			if err != nil {
				return err
			}
			out = append(out, found...)
			return nil
		}
		if (!IsTranscriptName(entry.Name()) && !agenttranscript.IsNativeArchiveFile(root.Harness, path)) || isTestRollout(root, entry.Name()) {
			return nil
		}
		if candidate, ok := Identify(root, path); ok {
			out = append(out, candidate)
		}
		return nil
	})
	return out, err
}

// isTestRollout reports a web-console rollout that Codex never wrote. Codex
// names every rollout rollout-<timestamp>-<thread uuid>.jsonl; web-console's
// tailer tests drop fixtures such as rollout-test123.jsonl into the sessions
// folder, and those are not transcripts.
func isTestRollout(root Root, name string) bool {
	return root.Kind == RootWebConsoleCodex && !uuidPattern.MatchString(name)
}

// Identify stats one file into a Candidate keyed by device and inode. Size is
// the decoded length, the unit of every offset the indexer keeps.
func Identify(root Root, path string) (Candidate, bool) {
	if root.Kind == RootCursor {
		return Candidate{}, false // Cursor candidates are keyed by composer in discoverCursorDB.
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return Candidate{}, false
	}
	key := fileKey(path, info)
	size, err := transcriptSize(path, key, info)
	if err != nil {
		return Candidate{}, false
	}
	return Candidate{Root: root, Path: path, Key: key, Size: size, MtimeUnixNano: info.ModTime().UnixNano()}, true
}

// fileKey is a file's identity: device and inode, so a rename keeps it.
func fileKey(path string, info os.FileInfo) string {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return fmt.Sprintf("%d:%d", uint64(stat.Dev), stat.Ino) //nolint:unconvert // Dev width differs by platform
	}
	return "path:" + path
}

var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// Identity is what a file's name, location and header say about its session.
type Identity struct {
	Harness         string
	SessionID       string
	HeaderSessionID string
	ParentSessionID string
	Sidechain       bool
	// ParentPath is the parent session file a Claude sidechain belongs to.
	ParentPath string
}

// ReadIdentity keeps the indexer's public shape while delegating native
// identity and lineage parsing to the shared transcript package.
func ReadIdentity(path, harness string) Identity {
	shared := agenttranscript.ReadIdentity(path, harness, OpenTranscript)
	return Identity{
		Harness: Family(shared.Harness), SessionID: shared.SessionID,
		HeaderSessionID: shared.HeaderSessionID, ParentSessionID: shared.ParentSessionID,
		Sidechain: shared.Sidechain, ParentPath: shared.ParentPath,
	}
}

// headHashBytes is how much of a file's already-read prefix is fingerprinted.
const headHashBytes = 4096

// HeadHash fingerprints the first min(4 KiB, readOffset) decoded bytes of a
// transcript so a rewrite in place (same inode, different content) re-indexes
// from offset 0. The length is derived from the stored offset, so a hash
// written after a tick is checked against exactly the same prefix on the next
// one. An error means the bytes could not be read, which says nothing about
// whether they changed: callers retry rather than treat it as a rewrite.
func HeadHash(path string, readOffset int64) (string, error) {
	file, err := OpenTranscript(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	head := make([]byte, min(int64(headHashBytes), readOffset))
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", err
	}
	return agenttranscript.SHA256BytesPrefix(head[:n], headHashBytes), nil
}

// FullHash fingerprints a complete native document whose shape can change
// after the cursor's first 4 KiB, unlike line-oriented append-only formats.
func FullHash(path string) (string, error) {
	return agenttranscript.SHA256File(path)
}
