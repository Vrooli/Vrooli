package home

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// ThreadView is what one session's Codex transcripts say about a thread
// before the desk types into the session (SYSTEM.md §6).
type ThreadView struct {
	// Shown is the newest top-level thread of the session: the one its
	// terminal shows. Sub-agent threads are never shown.
	Shown string
	// TurnRunning is true when the thread's last turn started and has not
	// completed or been aborted.
	TurnRunning bool
}

// transcriptTail is how much of a transcript's end is searched for the last
// turn boundary. An idle thread writes nothing after its last boundary.
const transcriptTail = 1 << 20

// firstLineLimit bounds a transcript's first line (session metadata with the
// base instructions).
const firstLineLimit = 4 << 20

// ThreadView reads the session's transcripts. It fails when it cannot tell:
// the desk then types nothing.
func (r *Reader) ThreadView(sessionID, threadID string) (ThreadView, error) {
	paths, err := filepath.Glob(filepath.Join(r.CodexHome(sessionID), "sessions", "*", "*", "*", "rollout-*.jsonl"))
	if err != nil || len(paths) == 0 {
		return ThreadView{}, fmt.Errorf("no Codex transcript in session %s", sessionID)
	}
	type rollout struct {
		path string
		mod  int64
	}
	list := make([]rollout, 0, len(paths))
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			list = append(list, rollout{path, info.ModTime().UnixNano()})
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].mod > list[j].mod })
	var view ThreadView
	own := ""
	for _, ro := range list {
		id, topLevel, err := transcriptMeta(ro.path)
		if err != nil {
			return ThreadView{}, err
		}
		if id == threadID && own == "" {
			own = ro.path
		}
		if topLevel && view.Shown == "" {
			view.Shown = id
		}
		if view.Shown != "" && own != "" {
			break
		}
	}
	if own == "" {
		return ThreadView{}, fmt.Errorf("no transcript of thread %s in session %s", threadID, sessionID)
	}
	running, err := turnRunning(own)
	if err != nil {
		return ThreadView{}, err
	}
	view.TurnRunning = running
	return view, nil
}

// transcriptMeta reads a transcript's session_meta line: the thread id and
// whether it is a top-level thread (its source is not a sub-agent spawn).
func transcriptMeta(path string) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	line, err := bufio.NewReaderSize(io.LimitReader(file, firstLineLimit), 64*1024).ReadBytes('\n')
	if err != nil && err != io.EOF {
		return "", false, fmt.Errorf("read %s: %w", path, err)
	}
	var meta struct {
		Type    string `json:"type"`
		Payload struct {
			ID     string          `json:"id"`
			Source json.RawMessage `json:"source"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(line, &meta); err != nil || meta.Type != "session_meta" || meta.Payload.ID == "" {
		return "", false, fmt.Errorf("%s does not start with Codex session metadata", path)
	}
	return meta.Payload.ID, !bytes.Contains(meta.Payload.Source, []byte(`"subagent"`)), nil
}

// turnRunning finds the last turn boundary in a transcript's tail.
func turnRunning(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return false, err
	}
	start := max(0, info.Size()-transcriptTail)
	tail := make([]byte, info.Size()-start)
	if _, err := file.ReadAt(tail, start); err != nil && err != io.EOF {
		return false, err
	}
	lines := bytes.Split(tail, []byte("\n"))
	if start > 0 {
		lines = lines[1:] // the first line is cut
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if !bytes.Contains(lines[i], []byte(`"event_msg"`)) {
			continue
		}
		var event struct {
			Type    string `json:"type"`
			Payload struct {
				Type string `json:"type"`
			} `json:"payload"`
		}
		if json.Unmarshal(lines[i], &event) != nil || event.Type != "event_msg" {
			continue
		}
		switch event.Payload.Type {
		case "task_started":
			return true, nil
		case "task_complete", "turn_aborted":
			return false, nil
		}
	}
	return false, fmt.Errorf("no turn boundary in the last %d KiB of %s", transcriptTail>>10, path)
}
