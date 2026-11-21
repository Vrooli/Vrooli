package agentharness

import (
	"bufio"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A coding agent's transcript records what its tools did, so it names some of
// the files the session created, though never all of them. Agent Manager runs,
// Web Console terminal sessions and the coding-agent resources all leave the
// same harness transcripts, in the harness home the session ran under, so one
// reader serves every route; only locating the file differs.

// CreatedPath is one path a transcript says the session created.
type CreatedPath struct {
	Path      string
	Certainty Certainty
	// Dir is true for a folder made by mkdir.
	Dir bool
	How string
}

// maxTranscriptLine skips a single enormous line instead of failing the file.
const maxTranscriptLine = 32 << 20

// ReadTranscriptCreations reads one harness transcript (Claude Code or Codex
// JSON lines) and returns what it records as created and the time of its first
// entry. A creating tool result is medium certainty; a mkdir or touch command
// is low, since the command may have failed or run somewhere else.
func ReadTranscriptCreations(r io.Reader) (map[string]CreatedPath, time.Time) {
	created := map[string]CreatedPath{}
	var first time.Time
	reader := bufio.NewReaderSize(r, 1<<20)
	for {
		line, err := readLongLine(reader)
		if len(line) > 0 {
			noteTranscriptLine(line, created, &first)
		}
		if err != nil {
			break
		}
	}
	return created, first
}

func readLongLine(reader *bufio.Reader) ([]byte, error) {
	var line []byte
	for {
		chunk, isPrefix, err := reader.ReadLine()
		if err != nil {
			return line, err
		}
		if len(line)+len(chunk) <= maxTranscriptLine {
			line = append(line, chunk...)
		}
		if !isPrefix {
			return line, nil
		}
	}
}

func noteTranscriptLine(line []byte, created map[string]CreatedPath, first *time.Time) {
	var entry struct {
		Timestamp     string          `json:"timestamp"`
		Cwd           string          `json:"cwd"`
		Type          string          `json:"type"`
		ToolUseResult json.RawMessage `json:"toolUseResult"`
		Message       json.RawMessage `json:"message"`
		Payload       json.RawMessage `json:"payload"`
	}
	if json.Unmarshal(line, &entry) != nil {
		return
	}
	if entry.Timestamp != "" && first.IsZero() {
		if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
			*first = t.UTC()
		}
	}
	// Claude Code: a tool result with type "create" is a file that did not exist.
	if len(entry.ToolUseResult) > 0 {
		var result struct {
			Type     string `json:"type"`
			FilePath string `json:"filePath"`
		}
		if json.Unmarshal(entry.ToolUseResult, &result) == nil && result.Type == "create" && filepath.IsAbs(result.FilePath) {
			addCreated(created, CreatedPath{Path: result.FilePath, Certainty: CertaintyMedium, How: "a Write that created the file"})
		}
	}
	// Claude Code: the Bash tool's command, run in the entry's folder.
	if len(entry.Message) > 0 {
		var message struct {
			Content json.RawMessage `json:"content"`
		}
		if json.Unmarshal(entry.Message, &message) == nil {
			var blocks []struct {
				Type  string `json:"type"`
				Name  string `json:"name"`
				Input struct {
					Command string `json:"command"`
				} `json:"input"`
			}
			if json.Unmarshal(message.Content, &blocks) == nil {
				for _, block := range blocks {
					if block.Type == "tool_use" && block.Name == "Bash" && block.Input.Command != "" {
						noteShellCreations(block.Input.Command, entry.Cwd, created)
					}
				}
			}
		}
	}
	// Codex: custom tool calls (apply_patch, exec) and function calls carry
	// their input as text.
	if len(entry.Payload) > 0 {
		var payload struct {
			Type      string `json:"type"`
			Name      string `json:"name"`
			Input     string `json:"input"`
			Arguments string `json:"arguments"`
			Cwd       string `json:"cwd"`
			Timestamp string `json:"timestamp"`
		}
		if json.Unmarshal(entry.Payload, &payload) != nil {
			return
		}
		if payload.Timestamp != "" && first.IsZero() {
			if t, err := time.Parse(time.RFC3339Nano, payload.Timestamp); err == nil {
				*first = t.UTC()
			}
		}
		if payload.Type == "custom_tool_call" || payload.Type == "function_call" {
			text := payload.Input + "\n" + payload.Arguments
			noteAddedFiles(text, payload.Cwd, created)
			noteShellCreations(commandText(text), payload.Cwd, created)
		}
	}
}

// noteAddedFiles reads apply_patch "*** Add File: <path>" lines.
func noteAddedFiles(text, cwd string, created map[string]CreatedPath) {
	for _, raw := range strings.Split(strings.ReplaceAll(text, `\n`, "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if rest, ok := strings.CutPrefix(line, "*** Add File:"); ok {
			path := strings.Trim(strings.TrimSpace(rest), `"'`)
			if path == "" {
				continue
			}
			if !filepath.IsAbs(path) && cwd != "" {
				path = filepath.Join(cwd, path)
			}
			if filepath.IsAbs(path) {
				addCreated(created, CreatedPath{Path: path, Certainty: CertaintyMedium, How: "an apply_patch that added the file"})
			}
		}
	}
}

// commandText pulls a shell command out of a Codex tool call: the `cmd` of an
// exec_command call, or the text itself.
func commandText(text string) string {
	var args struct {
		Cmd     string   `json:"cmd"`
		Command []string `json:"command"`
	}
	if json.Unmarshal([]byte(strings.TrimSpace(text)), &args) == nil {
		if args.Cmd != "" {
			return args.Cmd
		}
		if len(args.Command) > 0 {
			return strings.Join(args.Command, " ")
		}
	}
	return text
}

func noteShellCreations(command, cwd string, created map[string]CreatedPath) {
	if !strings.Contains(command, "mkdir") && !strings.Contains(command, "touch") {
		return
	}
	env := removalEnv{WorkingDirectory: cwd, Lookup: func(string) (string, bool) { return "", false }}
	if home, err := os.UserHomeDir(); err == nil {
		env.Home = home
	}
	for _, target := range parseCreations(command, dialectPOSIX, env) {
		addCreated(created, CreatedPath{Path: target.path, Certainty: CertaintyLow, Dir: target.dir, How: "a mkdir or touch command"})
	}
}

func addCreated(created map[string]CreatedPath, path CreatedPath) {
	path.Path = filepath.Clean(path.Path)
	if existing, ok := created[path.Path]; ok && existing.Certainty >= path.Certainty {
		return
	}
	created[path.Path] = path
}

// TranscriptCreations finds the session's transcripts, in the order the harness
// names them, and returns everything they record as created plus the earliest
// time they show. A session with no findable transcript returns nothing.
func TranscriptCreations(id RunIdentity) (map[string]CreatedPath, time.Time) {
	all := map[string]CreatedPath{}
	var earliest time.Time
	for _, file := range LocateTranscripts(id) {
		handle, err := os.Open(file)
		if err != nil {
			continue
		}
		created, first := ReadTranscriptCreations(handle)
		_ = handle.Close()
		for _, path := range created {
			addCreated(all, path)
		}
		if !first.IsZero() && (earliest.IsZero() || first.Before(earliest)) {
			earliest = first
		}
	}
	return all, earliest
}

// LocateTranscripts lists the transcript files of the session: the one the
// hook payload named, then the harness homes the session runs under. A Web
// Console terminal session or an isolated Agent Manager run sets CODEX_HOME or
// CLAUDE_CONFIG_DIR to its own home, which is searched before the user's.
func LocateTranscripts(id RunIdentity) []string {
	var files []string
	seen := map[string]bool{}
	add := func(path string) {
		if path != "" && !seen[path] {
			if _, err := os.Stat(path); err == nil {
				seen[path] = true
				files = append(files, path)
			}
		}
	}
	add(id.TranscriptPath)
	session := id.HarnessSession
	if session == "" || strings.ContainsAny(session, `/\*?[`) {
		return files
	}
	home, _ := os.UserHomeDir()
	var claudeHomes, codexHomes []string
	if id.ClaudeHome != "" {
		claudeHomes = append(claudeHomes, id.ClaudeHome)
	}
	if id.CodexHome != "" {
		codexHomes = append(codexHomes, id.CodexHome)
	}
	if home != "" {
		claudeHomes = append(claudeHomes, filepath.Join(home, ".claude"))
		codexHomes = append(codexHomes, filepath.Join(home, ".codex"))
	}
	for _, h := range claudeHomes {
		matches, _ := filepath.Glob(filepath.Join(h, "projects", "*", session+".jsonl"))
		for _, m := range matches {
			add(m)
		}
	}
	for _, h := range codexHomes {
		root := filepath.Join(h, "sessions")
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return fs.SkipDir
			}
			if !d.IsDir() && strings.HasPrefix(d.Name(), "rollout-") && strings.HasSuffix(d.Name(), "-"+session+".jsonl") {
				add(path)
			}
			return nil
		})
	}
	return files
}

// TranscriptResolver vouches for paths a transcript records as created. It
// cannot see what the agent made by other means, so a path it does not list is
// not vouched.
type TranscriptResolver struct {
	created map[string]CreatedPath
}

// NewTranscriptResolver wraps the creations a transcript recorded.
func NewTranscriptResolver(created map[string]CreatedPath) TranscriptResolver {
	return TranscriptResolver{created: created}
}

func (TranscriptResolver) Name() string { return "transcript" }

// Vouch says yes for a listed path. For a recursive query the path must be a
// folder the session made, and every file below it must be listed too.
func (r TranscriptResolver) Vouch(q CreationQuery) (Vouch, bool) {
	path := filepath.Clean(q.Path)
	entry, listed := r.created[path]
	info, err := os.Lstat(path)
	if err != nil {
		return Vouch{}, false
	}
	if !info.IsDir() {
		if !listed {
			return Vouch{}, false
		}
		return Vouch{Certainty: entry.Certainty, Source: "transcript", Detail: "the transcript shows " + entry.How}, true
	}
	if !q.Recursive {
		if !listed || !entry.Dir {
			return Vouch{}, false
		}
		return Vouch{Certainty: entry.Certainty, Source: "transcript", Detail: "the transcript shows " + entry.How}, true
	}
	if !listed || !entry.Dir {
		return Vouch{}, false // a folder the session did not make may predate it
	}
	certainty := entry.Certainty
	budget, files, complete := treeBudget, 0, true
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			complete = false
			return fs.SkipAll
		}
		if budget--; budget < 0 {
			complete = false
			return fs.SkipAll
		}
		if d.IsDir() {
			return nil
		}
		child, ok := r.created[filepath.Clean(p)]
		if !ok {
			complete = false
			return fs.SkipAll
		}
		if child.Certainty < certainty {
			certainty = child.Certainty
		}
		files++
		return nil
	})
	if !complete {
		return Vouch{}, false
	}
	return Vouch{Certainty: certainty, Source: "transcript", Detail: "the transcript lists the folder and every file in it"}, true
}
