package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"web-console/internal/sessionstore"
)

// Claude Code is the only supported agent that cannot name its own transcript.
// Codex writes a session_meta record and Grok writes a summary.json, so their
// tailers recover the pane-to-transcript binding by reading the file they are
// already reading. Claude writes neither, so for a long time the binding had
// exactly one source: the Stop hook. When that hook broke, sessions stayed
// anonymous, their transcripts were never opened, and the Messages view showed
// an empty conversation that was indistinguishable from a new one.
//
// This resolver supplies the missing second path. It pairs an unidentified pane
// with a transcript using two facts the operating system can confirm without
// the agent's cooperation:
//
//   - which pane an agent process belongs to, read from the environment
//     variable every Web Console terminal exports; and
//   - when that process started, which bounds the transcripts it could have
//     written — a process cannot have produced a message before it existed.
//
// Attribution follows the doctrine the OpenCode watcher already established:
// an edge is adopted only when it is mutually unique. Two Claude sessions
// started in the same directory within seconds of each other produce ambiguous
// edges and are both left alone. That is deliberate. An unidentified session
// reports a clear, self-resolving state; a misidentified one silently shows
// somebody else's conversation, which is far worse than showing none.

const (
	// claudeIdentitySlack absorbs clock and filesystem jitter between a process
	// start time and the first timestamp its transcript records. Both come from
	// the same machine, so this only needs to cover scheduling noise — a wide
	// window would blur concurrent sessions into ambiguity.
	claudeIdentitySlack = 5 * time.Second
	// claudeIdentityHeaderLines bounds how far into a transcript to look for
	// the first timestamped entry. Claude writes a handful of untimestamped
	// preamble records (mode, permission-mode, bridge-session) before the first
	// message, so a small window is enough and keeps the scan O(1) per file.
	claudeIdentityHeaderLines = 64
)

// claudeIdentityResolver adopts transcripts for Claude panes that have not been
// identified by the Stop hook.
type claudeIdentityResolver struct {
	store    sessionstore.Store
	home     string
	discover func() ([]agentProcess, error)
	logger   *log.Logger

	// reported remembers the ambiguity already announced per session. Resolve
	// runs on the transcript poller's cadence, so logging every unresolved pass
	// would emit the same line every couple of seconds forever — noise that
	// buries the one-off lines an operator actually needs to see.
	mu       sync.Mutex
	reported map[string]string
}

func newClaudeIdentityResolver(store sessionstore.Store) *claudeIdentityResolver {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	return &claudeIdentityResolver{
		store:    store,
		home:     home,
		discover: func() ([]agentProcess, error) { return discoverAgentProcesses() },
		logger:   log.Default(),
		reported: map[string]string{},
	}
}

// transcriptCandidate is one unclaimed transcript file and the moment its first
// message was recorded.
type transcriptCandidate struct {
	path           string
	agentSessionID string
	firstEntryAt   time.Time
	modifiedAt     time.Time
}

// Resolve runs one attribution pass and returns the number of panes adopted.
// It is safe to call on every discovery tick: identified panes are filtered out
// before any file is touched, so the steady-state cost is one store list.
func (r *claudeIdentityResolver) Resolve(ctx context.Context) int {
	if r == nil || r.store == nil || r.home == "" {
		return 0
	}
	rows, err := r.store.List(ctx)
	if err != nil {
		return 0
	}

	// A transcript already bound to any session — live, dismissed or archived —
	// is off the table. Restricting this to live rows would let a pane adopt
	// the transcript of a session the operator closed minutes ago.
	claimed := make(map[string]bool, len(rows))
	var unidentified []sessionstore.Metadata
	for _, row := range rows {
		if row.AgentSessionID != "" {
			claimed[row.AgentSessionID] = true
			continue
		}
		if row.Status == sessionstore.StatusLive && row.AgentType == sessionstore.AgentClaude {
			unidentified = append(unidentified, row)
		}
	}
	if len(unidentified) == 0 {
		return 0
	}

	processes, err := r.discover()
	if err != nil || len(processes) == 0 {
		return 0
	}
	processBySession := make(map[string]agentProcess, len(processes))
	for _, proc := range processes {
		if proc.StartedAt.IsZero() {
			continue
		}
		// When several processes report the same session, the EARLIEST is the
		// pane's interactive agent. Anything started later inherited the same
		// environment from inside that agent — a nested `claude -p` call, a
		// subagent, an agent launched by a script the agent itself ran — and
		// its working directory is wherever that command happened to run.
		// Preferring the newest looks reasonable and is wrong: it hands the
		// pane a nested invocation's directory and the transcript search then
		// looks in a project folder the pane has nothing to do with.
		existing, seen := processBySession[proc.SessionID]
		if !seen || proc.StartedAt.Before(existing.StartedAt) {
			processBySession[proc.SessionID] = proc
		}
	}

	// Group the work by directory: every pane in one directory competes for the
	// same pool of transcripts, and resolving them together is what makes the
	// uniqueness test meaningful.
	byDirectory := make(map[string][]paneProcess)
	for _, pane := range unidentified {
		proc, ok := processBySession[pane.ID]
		if !ok || proc.StartedAt.IsZero() {
			continue // no running agent to attribute, or no usable start time
		}
		dir := strings.TrimSpace(pane.CWD)
		if dir == "" {
			dir = proc.WorkingDir
		}
		if dir == "" {
			continue
		}
		byDirectory[dir] = append(byDirectory[dir], paneProcess{pane: pane, proc: proc, workingDir: dir})
	}

	adopted := 0
	for dir, panes := range byDirectory {
		adopted += r.resolveDirectory(ctx, dir, panes, claimed)
	}
	return adopted
}

// paneProcess pairs an unidentified pane with the agent process running in it.
type paneProcess struct {
	pane       sessionstore.Metadata
	proc       agentProcess
	workingDir string
}

// resolveDirectory attributes the transcripts in one project directory. Each
// transcript nominates exactly one owner — the newest process that had already
// started when the transcript recorded its first message — and a pane is
// adopted only when exactly one transcript nominates it.
func (r *claudeIdentityResolver) resolveDirectory(ctx context.Context, dir string, panes []paneProcess, claimed map[string]bool) int {
	candidates := r.candidatesForDirectory(dir, claimed)
	if len(candidates) == 0 {
		return 0
	}

	// Newest first, so the first process that could have written a transcript
	// is also the closest one before it.
	ordered := append([]paneProcess(nil), panes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].proc.StartedAt.After(ordered[j].proc.StartedAt) })

	nominations := make(map[string][]transcriptCandidate)
	for _, candidate := range candidates {
		for _, pp := range ordered {
			if !transcriptCouldBelongTo(candidate, pp.proc) {
				continue
			}
			nominations[pp.pane.ID] = append(nominations[pp.pane.ID], candidate)
			break // the newest eligible process is the only nominee
		}
	}

	adopted := 0
	for _, pp := range ordered {
		matches := nominations[pp.pane.ID]
		if len(matches) != 1 {
			if len(matches) > 1 {
				r.reportAmbiguity(pp.pane.ID, dir, matches)
			}
			continue
		}
		if r.adopt(ctx, pp, matches[0]) {
			claimed[matches[0].agentSessionID] = true
			adopted++
			r.clearReported(pp.pane.ID)
		}
	}
	return adopted
}

// reportAmbiguity logs an unresolvable session once, and again only when the
// set of candidates actually changes — a new transcript appearing is worth
// saying; the same standoff on the next poll is not.
func (r *claudeIdentityResolver) reportAmbiguity(sessionID, dir string, matches []transcriptCandidate) {
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match.agentSessionID)
	}
	sort.Strings(ids)
	fingerprint := dir + "\x00" + strings.Join(ids, ",")

	r.mu.Lock()
	if r.reported == nil {
		r.reported = map[string]string{}
	}
	unchanged := r.reported[sessionID] == fingerprint
	r.reported[sessionID] = fingerprint
	r.mu.Unlock()
	if unchanged {
		return
	}
	r.printf("session %s: %d candidate transcripts in %s, leaving unidentified rather than guessing",
		sanitizeID(sessionID), len(matches), dir)
}

func (r *claudeIdentityResolver) clearReported(sessionID string) {
	r.mu.Lock()
	delete(r.reported, sessionID)
	r.mu.Unlock()
}

// transcriptCouldBelongTo is the per-edge predicate. Both halves are necessary:
// the first message cannot predate the process, and a transcript untouched
// since the process started cannot be the one it is currently writing.
func transcriptCouldBelongTo(candidate transcriptCandidate, proc agentProcess) bool {
	if candidate.firstEntryAt.Before(proc.StartedAt.Add(-claudeIdentitySlack)) {
		return false
	}
	return !candidate.modifiedAt.Before(proc.StartedAt)
}

// candidatesForDirectory lists the unclaimed transcripts Claude would write for
// a given working directory.
func (r *claudeIdentityResolver) candidatesForDirectory(dir string, claimed map[string]bool) []transcriptCandidate {
	projectDir := claudeProjectDir(r.home, dir)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil
	}
	candidates := make([]transcriptCandidate, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		agentSessionID := strings.TrimSuffix(entry.Name(), ".jsonl")
		if agentSessionID == "" || claimed[agentSessionID] {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		path := filepath.Join(projectDir, entry.Name())
		firstEntryAt, ok := claudeTranscriptFirstEntryTime(path)
		if !ok {
			// No message recorded yet. There is nothing to attribute and
			// nothing to show, and the next pass will reconsider it.
			continue
		}
		candidates = append(candidates, transcriptCandidate{
			path:           path,
			agentSessionID: agentSessionID,
			firstEntryAt:   firstEntryAt,
			modifiedAt:     info.ModTime(),
		})
	}
	return candidates
}

// adopt persists a resolved binding. The working directory is written alongside
// the session id because both are required to locate the transcript again, and
// a pane that reached here may have had neither.
func (r *claudeIdentityResolver) adopt(ctx context.Context, pp paneProcess, candidate transcriptCandidate) bool {
	err := r.store.UpdateAgentInfo(ctx, pp.pane.ID, sessionstore.AgentInfo{
		AgentType:       sessionstore.AgentClaude,
		AgentSessionID:  candidate.agentSessionID,
		CWD:             pp.workingDir,
		LastRolloutPath: candidate.path,
		LastActivityAt:  time.Now(),
	})
	if err != nil {
		r.printf("session %s: could not persist identity: %v", sanitizeID(pp.pane.ID), err)
		return false
	}
	r.printf("identified session %s as Claude session %s (pid %d, %s)",
		sanitizeID(pp.pane.ID), sanitizeID(candidate.agentSessionID), pp.proc.PID, pp.workingDir)
	return true
}

func (r *claudeIdentityResolver) printf(format string, args ...any) {
	if r.logger != nil {
		r.logger.Printf("claude-identity: "+format, args...)
	}
}

// claudeProjectDir returns the directory Claude Code writes a working
// directory's transcripts into. It is the encoding half of
// claudeTranscriptPath, kept separate so discovery can list a directory it
// cannot yet name a file inside.
func claudeProjectDir(home, cwd string) string {
	key := strings.ReplaceAll(filepath.Clean(cwd), string(filepath.Separator), "-")
	return filepath.Join(home, ".claude", "projects", key)
}

// claudeTranscriptFirstEntryTime reports when a transcript recorded its first
// timestamped entry. Claude precedes the first message with several records
// that carry no timestamp, so the scan looks past them but stays bounded.
func claudeTranscriptFirstEntryTime(path string) (time.Time, bool) {
	file, err := os.Open(path)
	if err != nil {
		return time.Time{}, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for line := 0; line < claudeIdentityHeaderLines && scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var entry struct {
			Timestamp string `json:"timestamp"`
		}
		if json.Unmarshal([]byte(raw), &entry) != nil || entry.Timestamp == "" {
			continue
		}
		parsed, parseErr := time.Parse(time.RFC3339, entry.Timestamp)
		if parseErr != nil {
			continue
		}
		return parsed, true
	}
	return time.Time{}, false
}
