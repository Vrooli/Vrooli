package home

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite" // read-only access to Codex goals_1.sqlite
)

// Goal is one row of a Codex home's goals_1.sqlite thread_goals table.
type Goal struct {
	SessionID string // the Codex home folder name = Web Console session id
	ThreadID  string
	Objective string
	// ObjectiveFile is set when Codex stored a long objective as an
	// attachment; Objective then holds that file's text.
	ObjectiveFile string
	Status        string
	Created       time.Time
	Updated       time.Time
	Source        string
}

// Prompt is one history.jsonl entry that names a commission folder.
type Prompt struct {
	SessionID string
	ThreadID  string
	At        time.Time
	IsGoal    bool // the prompt is a "/goal …" command
	Text      string
	Source    string
}

var (
	// commissionRef matches a commission folder named in any prompt or goal.
	commissionRef = regexp.MustCompile(`cohorts/([A-Za-z0-9._-]+)`)
	// objectiveAttachment is the objective Codex stores in place of a long goal.
	objectiveAttachment = regexp.MustCompile(`^Read the Codex goal objective file at (\S+/attachments/[^/\s]+/goal-objective\.md)`)
)

// maxObjectiveFile caps how much of an objective attachment is read.
const maxObjectiveFile = 64 * 1024

// codexIndex caches the parsed goals and prompts of every Codex home, keyed
// by file path, size and modification time, so a refresh rereads only the
// files that changed. The cache can be dropped at any time (charter I2).
type codexIndex struct {
	mu    sync.Mutex
	files map[string]cachedCodexFile
}

type cachedCodexFile struct {
	size    int64
	mod     time.Time
	goals   []Goal
	prompts []Prompt
}

// goalReadTime bounds one goals_1.sqlite read. Reads are detached from the
// request: a cut request must not leave a database looking unreadable.
const goalReadTime = 5 * time.Second

// CodexRecords is one consistent read of every Codex home.
type CodexRecords struct {
	Goals   []Goal
	Prompts []Prompt
	// Unreadable lists files that exist but could not be read.
	Unreadable []string
	Known      bool
	Reason     string
}

func newCodexIndex() *codexIndex {
	return &codexIndex{files: map[string]cachedCodexFile{}}
}

// read returns the goals and commission-naming prompts of every home under
// codexDir.
func (x *codexIndex) read(ctx context.Context, codexDir string) CodexRecords {
	entries, err := os.ReadDir(codexDir)
	if err != nil {
		return CodexRecords{Reason: "Codex session records unreadable: " + err.Error()}
	}
	x.mu.Lock()
	defer x.mu.Unlock()
	out := CodexRecords{Known: true}
	seen := make(map[string]bool, len(entries)*2)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if ctx.Err() != nil {
			out.Known, out.Reason = false, "reading Codex session records was cancelled"
			return out
		}
		sessionID := entry.Name()
		for _, name := range []string{"goals_1.sqlite", "history.jsonl"} {
			path := filepath.Join(codexDir, sessionID, name)
			size, mod, ok := fileKey(path)
			if !ok {
				continue
			}
			seen[path] = true
			cached, ok := x.files[path]
			if !ok || cached.size != size || !cached.mod.Equal(mod) {
				// An error is never cached: the next read tries again.
				cached = cachedCodexFile{size: size, mod: mod}
				var err error
				if name == "goals_1.sqlite" {
					cached.goals, err = readGoals(ctx, path, sessionID)
				} else {
					cached.prompts, err = readPrompts(path, sessionID)
				}
				if err != nil {
					delete(x.files, path)
					out.Unreadable = append(out.Unreadable, path+": "+err.Error())
					continue
				}
				x.files[path] = cached
			}
			out.Goals = append(out.Goals, cached.goals...)
			out.Prompts = append(out.Prompts, cached.prompts...)
		}
	}
	for path := range x.files {
		if !seen[path] {
			delete(x.files, path)
		}
	}
	return out
}

// OtherGoals are the goals of the Codex homes outside Web Console's session
// folder: the operator's own ~/.codex and Agent Manager's run homes. They are
// read only to prove that no goal names a prepared packet (rulings F8).
type OtherGoals struct {
	Goals []Goal
	// Homes is how many Codex homes were read.
	Homes int
	// Unreadable lists homes whose goal database could not be read; while
	// any is, absence cannot be claimed.
	Unreadable []string
}

// others reads goals_1.sqlite in every Codex home the patterns match (glob
// patterns, each naming a Codex home folder).
func (x *codexIndex) others(ctx context.Context, patterns []string) OtherGoals {
	var out OtherGoals
	for _, pattern := range patterns {
		homes, err := filepath.Glob(pattern)
		if err != nil {
			out.Unreadable = append(out.Unreadable, pattern+": "+err.Error())
			continue
		}
		for _, codexHome := range homes {
			path := filepath.Join(codexHome, "goals_1.sqlite")
			size, mod, ok := fileKey(path)
			if !ok {
				continue
			}
			out.Homes++
			x.mu.Lock()
			cached, hit := x.files[path]
			x.mu.Unlock()
			if !hit || cached.size != size || !cached.mod.Equal(mod) {
				goals, err := readGoals(ctx, path, filepath.Base(codexHome))
				if err != nil {
					out.Unreadable = append(out.Unreadable, path+": "+err.Error())
					x.mu.Lock()
					delete(x.files, path)
					x.mu.Unlock()
					continue
				}
				cached = cachedCodexFile{size: size, mod: mod, goals: goals}
				x.mu.Lock()
				x.files[path] = cached
				x.mu.Unlock()
			}
			out.Goals = append(out.Goals, cached.goals...)
		}
	}
	return out
}

// fileKey identifies a file's content by size and modification time,
// counting a SQLite write-ahead log beside it, whose writes do not touch the
// main file until a checkpoint.
func fileKey(path string) (int64, time.Time, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}, false
	}
	size, mod := info.Size(), info.ModTime()
	if wal, err := os.Stat(path + "-wal"); err == nil {
		size += wal.Size()
		if wal.ModTime().After(mod) {
			mod = wal.ModTime()
		}
	}
	return size, mod, true
}

// readGoals reads thread_goals read-only. Codex owns the database; the desk
// never writes it.
func readGoals(ctx context.Context, path, sessionID string) ([]Goal, error) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), goalReadTime)
	defer cancel()
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT thread_id, objective, status, created_at_ms, updated_at_ms FROM thread_goals`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var goals []Goal
	for rows.Next() {
		var goal Goal
		var created, updated int64
		if err := rows.Scan(&goal.ThreadID, &goal.Objective, &goal.Status, &created, &updated); err != nil {
			return nil, err
		}
		goal.SessionID = sessionID
		if err := resolveObjectiveFile(&goal, filepath.Dir(path)); err != nil {
			return nil, err
		}
		goal.Created = time.UnixMilli(created)
		goal.Updated = time.UnixMilli(updated)
		goal.Source = path
		goals = append(goals, goal)
	}
	return goals, rows.Err()
}

// resolveObjectiveFile replaces an attachment pointer with the attachment's
// text when the attachment lies inside the same Codex home. An attachment
// that exists but cannot be read is an error: the goal's words are unknown
// (rulings E1).
func resolveObjectiveFile(goal *Goal, codexHome string) error {
	sub := objectiveAttachment.FindStringSubmatch(strings.TrimSpace(goal.Objective))
	if sub == nil {
		return nil
	}
	path := filepath.Clean(sub[1])
	if !strings.HasPrefix(path, filepath.Join(codexHome, "attachments")+string(filepath.Separator)) {
		return nil
	}
	file, err := os.Open(path)
	if absent(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("goal objective attachment: %w", err)
	}
	defer file.Close()
	text, err := io.ReadAll(io.LimitReader(file, maxObjectiveFile))
	if err != nil {
		return fmt.Errorf("goal objective attachment %s: %w", path, err)
	}
	goal.Objective, goal.ObjectiveFile = string(text), path
	return nil
}

// readPrompts keeps only the history entries that name a commission folder.
func readPrompts(path, sessionID string) ([]Prompt, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReaderSize(file, 256*1024)
	var prompts []Prompt
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 && strings.Contains(string(line), "cohorts/") {
			var entry struct {
				SessionID string `json:"session_id"`
				TS        int64  `json:"ts"`
				Text      string `json:"text"`
			}
			if json.Unmarshal(line, &entry) == nil && commissionRef.MatchString(entry.Text) {
				prompts = append(prompts, Prompt{
					SessionID: sessionID,
					ThreadID:  entry.SessionID,
					At:        time.Unix(entry.TS, 0),
					IsGoal:    strings.HasPrefix(strings.TrimSpace(entry.Text), "/goal "),
					Text:      entry.Text,
					Source:    path,
				})
			}
		}
		if errors.Is(err, io.EOF) {
			return prompts, nil
		}
		if err != nil {
			return prompts, err
		}
	}
}

// rolloutFor returns the transcript file of one thread in a Codex home.
// Codex names it rollout-<start>-<thread>.jsonl under sessions/YYYY/MM/DD.
func rolloutFor(codexDir, sessionID, threadID string) (FileFact, bool) {
	if threadID == "" {
		return FileFact{}, false
	}
	matches, _ := filepath.Glob(filepath.Join(codexDir, sessionID, "sessions", "*", "*", "*", "rollout-*-"+threadID+".jsonl"))
	var best FileFact
	found := false
	for _, path := range matches {
		if fact, ok := stat(path); ok && (!found || fact.Modified.After(best.Modified)) {
			best, found = fact, true
		}
	}
	return best, found
}

// CodexHome is the Codex home of one Web Console session.
func (r *Reader) CodexHome(sessionID string) string {
	return filepath.Join(r.cfg.CodexDir, sessionID)
}

// SessionGoals reads one session's goals, read-only. A session whose Codex
// has not written a goal yet has none.
func (r *Reader) SessionGoals(ctx context.Context, sessionID string) ([]Goal, error) {
	path := filepath.Join(r.CodexHome(sessionID), "goals_1.sqlite")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return readGoals(ctx, path, sessionID)
}

// SessionProcesses lists the running Codex processes of one session's home,
// whether or not they belong to the session (Process.OfSession).
func (r *Reader) SessionProcesses(sessionID string) ([]Process, error) {
	processes, known, reason := r.deps.Procs.Codex()
	if !known {
		return nil, errors.New(reason)
	}
	home := filepath.Clean(r.CodexHome(sessionID))
	var out []Process
	for _, p := range processes {
		if p.CodexHome == home {
			out = append(out, p)
		}
	}
	return out, nil
}

// OfSession reports whether the process belongs to Web Console session id: it
// carries the session's id and still descends from the session's terminal.
// A process that only shares the Codex home is never the session's.
func (p Process) OfSession(id string) bool { return p.WebConsoleSession == id && p.InTerminal }
