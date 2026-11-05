package transcriptindex

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// RootConfig names the directories discovery starts from.
type RootConfig struct {
	Home         string
	RunStateRoot string
	ExtraRoots   []string
}

// WebConsoleCodexBase is the parent of web-console's per-session CODEX_HOME
// directories; each child's sessions/ subtree holds that session's rollouts.
func WebConsoleCodexBase(home string) string {
	return filepath.Join(home, ".vrooli", "state", "vrooli", "web-console", "sessions", "codex")
}

// runHomeTranscriptDirs are the harness homes an Agent Manager run directory
// can hold (codec-pipe and interactive runs).
var runHomeTranscriptDirs = []struct{ rel, harness string }{
	{filepath.Join("runtime", "codex", "sessions"), HarnessCodex},
	{filepath.Join("codex", "sessions"), HarnessCodex},
	{filepath.Join("runtime", "claude", "projects"), HarnessClaude},
	{filepath.Join("claude", "projects"), HarnessClaude},
}

// Roots lists every transcript root on this host: web-console Codex homes,
// the default Codex and Claude homes, Agent Manager run homes, and configured
// extra roots. An extra root equal to or inside another root is dropped so no
// file is discovered twice.
func Roots(cfg RootConfig) []Root {
	var roots []Root
	if entries, err := os.ReadDir(WebConsoleCodexBase(cfg.Home)); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(WebConsoleCodexBase(cfg.Home), entry.Name(), "sessions")
			if isDir(dir) {
				roots = append(roots, Root{Path: dir, Kind: RootWebConsoleCodex, Harness: HarnessCodex, WebConsoleSessionID: entry.Name()})
			}
		}
	}
	roots = append(roots,
		Root{Path: filepath.Join(cfg.Home, ".codex", "sessions"), Kind: RootCodexHome, Harness: HarnessCodex},
		Root{Path: filepath.Join(cfg.Home, ".claude", "projects"), Kind: RootClaudeHome, Harness: HarnessClaude},
	)
	if cfg.RunStateRoot != "" {
		if entries, err := os.ReadDir(cfg.RunStateRoot); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				for _, sub := range runHomeTranscriptDirs {
					dir := filepath.Join(cfg.RunStateRoot, entry.Name(), sub.rel)
					if isDir(dir) {
						roots = append(roots, Root{Path: dir, Kind: RootRunHome, Harness: sub.harness, RunID: entry.Name()})
					}
				}
			}
		}
	}
	covered := []string{filepath.Join(cfg.Home, ".codex", "sessions"), filepath.Join(cfg.Home, ".claude", "projects"), WebConsoleCodexBase(cfg.Home)}
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
	lowered := strings.ToLower(harness)
	switch {
	case strings.Contains(lowered, "codex"):
		return HarnessCodex
	case strings.Contains(lowered, "claude"):
		return HarnessClaude
	}
	return HarnessConfigured
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

// Candidate is one transcript file found by a walk, with its identity.
type Candidate struct {
	Root          Root
	Path          string
	Key           string
	Size          int64
	MtimeUnixNano int64
}

// Walk lists the .jsonl files under root. A root that does not exist yields
// nothing; any other failure is returned so coverage can name it.
func Walk(root Root) ([]Candidate, error) {
	if _, err := os.Stat(root.Path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Candidate
	err := filepath.WalkDir(root.Path, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if entry != nil && entry.IsDir() && path != root.Path {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") || isTestRollout(root, entry.Name()) {
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

// Identify stats one file into a Candidate keyed by device and inode.
func Identify(root Root, path string) (Candidate, bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return Candidate{}, false
	}
	key := "path:" + path
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		key = fmt.Sprintf("%d:%d", uint64(stat.Dev), stat.Ino) //nolint:unconvert // Dev width differs by platform
	}
	return Candidate{Root: root, Path: path, Key: key, Size: info.Size(), MtimeUnixNano: info.ModTime().UnixNano()}, true
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

// ReadIdentity derives a file's session identity. Codex fork rollouts are
// named rollout-<ts>-<parent>_<child>.jsonl and carry the parent thread in
// their header; Claude sidechains live at <project>/<session>/subagents/.
func ReadIdentity(path, harness string) Identity {
	id := Identity{Harness: harness}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	id.HeaderSessionID = headerSessionID(path)
	family := Family(harness)
	if family == HarnessConfigured {
		family = sniffHarness(path)
		id.Harness = family
	}
	switch family {
	case HarnessClaude:
		id.SessionID = stem
		if filepath.Base(filepath.Dir(path)) == "subagents" {
			sessionDir := filepath.Dir(filepath.Dir(path))
			id.Sidechain = true
			id.ParentSessionID = filepath.Base(sessionDir)
			id.ParentPath = filepath.Join(filepath.Dir(sessionDir), id.ParentSessionID+".jsonl")
		}
	default:
		ids := uuidPattern.FindAllString(stem, -1)
		if strings.Contains(stem, "_") && len(ids) >= 2 {
			id.ParentSessionID, id.SessionID = ids[len(ids)-2], ids[len(ids)-1]
		} else if id.HeaderSessionID != "" {
			id.SessionID = id.HeaderSessionID
		} else if len(ids) > 0 {
			id.SessionID = ids[len(ids)-1]
		} else {
			id.SessionID = stem
		}
	}
	if id.HeaderSessionID == "" {
		id.HeaderSessionID = id.SessionID
		if id.Sidechain {
			id.HeaderSessionID = id.ParentSessionID
		}
	}
	return id
}

// headerSessionID reads the session id from the first lines of a transcript:
// a Codex session_meta id, or a session_id/sessionId/thread_id field.
func headerSessionID(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	reader := bufio.NewReaderSize(file, 64*1024)
	for lines := 0; lines < 40; lines++ {
		line, err := reader.ReadSlice('\n')
		if len(line) > 0 && err != bufio.ErrBufferFull {
			var header struct {
				Type    string `json:"type"`
				Payload struct {
					ID string `json:"id"`
				} `json:"payload"`
				SessionID  string `json:"session_id"`
				SessionID2 string `json:"sessionId"`
				ThreadID   string `json:"thread_id"`
			}
			if json.Unmarshal(line, &header) == nil {
				switch {
				case header.Type == "session_meta" && header.Payload.ID != "":
					return header.Payload.ID
				case header.SessionID2 != "":
					return header.SessionID2
				case header.SessionID != "":
					return header.SessionID
				case header.ThreadID != "":
					return header.ThreadID
				}
			}
		}
		if err == bufio.ErrBufferFull {
			// An oversized header line (inline instructions) is skipped whole.
			for err == bufio.ErrBufferFull {
				_, err = reader.ReadSlice('\n')
			}
			continue
		}
		if err != nil {
			return ""
		}
	}
	return ""
}

// sniffHarness tells a Codex rollout from a Claude project file for a
// configured root by their header shapes.
func sniffHarness(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return HarnessConfigured
	}
	defer file.Close()
	head := make([]byte, 8192)
	n, _ := io.ReadFull(file, head)
	text := string(head[:n])
	switch {
	case strings.Contains(text, `"type":"session_meta"`) || strings.Contains(text, `"type":"thread.started"`):
		return HarnessCodex
	case strings.Contains(text, `"sessionId"`) || strings.Contains(text, `"type":"system"`):
		return HarnessClaude
	}
	return HarnessConfigured
}

// headHashBytes is how much of a file's already-read prefix is fingerprinted.
const headHashBytes = 4096

// HeadHash fingerprints the first min(4 KiB, readOffset) bytes of a file so
// a rewrite in place (same inode, different content) re-indexes from offset
// 0. The length is derived from the stored offset, so a hash written after a
// tick is checked against exactly the same prefix on the next one.
func HeadHash(path string, readOffset int64) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	head := make([]byte, min(int64(headHashBytes), readOffset))
	n, _ := io.ReadFull(file, head)
	return hashHead(head[:n])
}

func hashHead(data []byte) string {
	sum := sha256.Sum256(data[:min(len(data), headHashBytes)])
	return hex.EncodeToString(sum[:])
}
