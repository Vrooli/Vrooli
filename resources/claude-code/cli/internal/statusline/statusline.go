// Package statusline owns Claude Code's provider quota feed. Claude Code pipes
// its statusline JSON (which carries the subscription's five-hour and seven-day
// rate-limit usage) to `resource-claude-code statusline run`; the command
// appends a bounded NDJSON sample that Agent Manager ingests as quota
// observations, then prints a status line. Install reconciles settings.json's
// statusLine to that command and keeps any previous command, which the feed
// chains so the operator's own status line keeps rendering.
package statusline

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/agentharness"
)

const (
	// SamplesEnv overrides the sample file path (tests, out-of-tree runs).
	SamplesEnv = "VROOLI_CLAUDE_QUOTA_SAMPLES"
	// CommandEnv overrides the executable written into settings.json.
	CommandEnv = "VROOLI_CLAUDE_STATUSLINE_COMMAND"
	// MaxSampleBytes bounds the live sample file; a full file rotates to
	// <file>.1, replacing the previous generation.
	MaxSampleBytes = 4 << 20
	// Heartbeat re-records an unchanged reading so readers can tell a quiet
	// feed from a stale one.
	Heartbeat = 5 * time.Minute
	// chainTimeout bounds a chained user status line command.
	chainTimeout  = 5 * time.Second
	stateFileName = "statusline.json"
)

// Input is the subset of Claude Code's statusline JSON the feed records.
// rate_limits appears only for Pro/Max subscribers after the first API
// response, and each window may be absent.
type Input struct {
	SessionID string `json:"session_id"`
	Model     struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"model"`
	RateLimits struct {
		FiveHour *Window `json:"five_hour"`
		SevenDay *Window `json:"seven_day"`
	} `json:"rate_limits"`
}

// Window is one rate-limit window: percent used (0-100) and its reset time
// in Unix epoch seconds.
type Window struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       int64    `json:"resets_at"`
}

// Sample is one NDJSON line of the quota feed. Agent Manager reads this shape.
type Sample struct {
	ObservedAt time.Time `json:"observed_at"`
	SessionID  string    `json:"session_id,omitempty"`
	Model      string    `json:"model,omitempty"`
	FiveHour   *Window   `json:"five_hour,omitempty"`
	SevenDay   *Window   `json:"seven_day,omitempty"`
}

// DefaultSamplesPath is ~/.vrooli/state/vrooli/claude-code/quota-samples.ndjson
// unless SamplesEnv overrides it.
func DefaultSamplesPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv(SamplesEnv)); override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".vrooli", "state", "vrooli", "claude-code", "quota-samples.ndjson"), nil
}

// Feed runs one statusline invocation: record a sample when the reading
// changed (or the heartbeat elapsed), then print the chained previous status
// line or, without one, a concise quota line. Recording failures never break
// the status line.
type Feed struct {
	SamplesPath string
	StatePath   string
	Now         func() time.Time
	Stderr      io.Writer
}

func (f Feed) Run(stdin io.Reader, stdout io.Writer) error {
	raw, err := io.ReadAll(io.LimitReader(stdin, 1<<20))
	if err != nil {
		return err
	}
	var in Input
	if err := json.Unmarshal(raw, &in); err == nil {
		if recErr := f.record(in); recErr != nil && f.Stderr != nil {
			_, _ = fmt.Fprintf(f.Stderr, "statusline: record quota sample: %v\n", recErr)
		}
	}
	if previous := loadState(f.StatePath).previousCommand(); previous != "" {
		ctx, cancel := context.WithTimeout(context.Background(), chainTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "sh", "-c", previous)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(raw), stdout, f.Stderr
		return cmd.Run()
	}
	_, err = fmt.Fprintln(stdout, render(in))
	return err
}

func render(in Input) string {
	parts := []string{}
	if name := strings.TrimSpace(in.Model.DisplayName); name != "" {
		parts = append(parts, name)
	}
	for _, w := range []struct {
		label  string
		window *Window
	}{{"5h", in.RateLimits.FiveHour}, {"7d", in.RateLimits.SevenDay}} {
		if w.window == nil || w.window.UsedPercentage == nil {
			continue
		}
		segment := fmt.Sprintf("%s %.0f%%", w.label, *w.window.UsedPercentage)
		if w.window.ResetsAt > 0 {
			segment += " ↻" + time.Unix(w.window.ResetsAt, 0).Local().Format("Jan 2 15:04")
		}
		parts = append(parts, segment)
	}
	if len(parts) == 0 {
		return "claude"
	}
	return strings.Join(parts, " · ")
}

func (f Feed) record(in Input) error {
	five, seven := usable(in.RateLimits.FiveHour), usable(in.RateLimits.SevenDay)
	if five == nil && seven == nil || f.SamplesPath == "" {
		return nil
	}
	now := time.Now
	if f.Now != nil {
		now = f.Now
	}
	sample := Sample{ObservedAt: now().UTC(), SessionID: in.SessionID, Model: firstNonEmpty(in.Model.ID, in.Model.DisplayName), FiveHour: five, SevenDay: seven}
	key := readingKey(sample)
	lastPath := f.SamplesPath + ".last"
	if last, err := os.ReadFile(lastPath); err == nil {
		var prior struct {
			Key string    `json:"key"`
			At  time.Time `json:"at"`
		}
		if json.Unmarshal(last, &prior) == nil && prior.Key == key && sample.ObservedAt.Sub(prior.At) < Heartbeat {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(f.SamplesPath), 0o755); err != nil {
		return err
	}
	if info, err := os.Stat(f.SamplesPath); err == nil && info.Size() >= MaxSampleBytes {
		if err := os.Rename(f.SamplesPath, f.SamplesPath+".1"); err != nil {
			return err
		}
	}
	line, err := json.Marshal(sample)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(f.SamplesPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(line, '\n')); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	marker, _ := json.Marshal(map[string]any{"key": key, "at": sample.ObservedAt})
	return os.WriteFile(lastPath, marker, 0o644)
}

func usable(w *Window) *Window {
	if w == nil || w.UsedPercentage == nil || *w.UsedPercentage < 0 || *w.UsedPercentage > 100 {
		return nil
	}
	return w
}

func readingKey(s Sample) string {
	key := ""
	for _, w := range []*Window{s.FiveHour, s.SevenDay} {
		if w == nil {
			key += "-|"
			continue
		}
		key += fmt.Sprintf("%g@%d|", *w.UsedPercentage, w.ResetsAt)
	}
	return key
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// state is the resource's record of what it installed: its own command and
// the statusLine object it displaced (restored verbatim on uninstall).
type state struct {
	Command  string          `json:"command"`
	Previous json.RawMessage `json:"previous,omitempty"`
}

func (s state) previousCommand() string {
	var previous struct {
		Command string `json:"command"`
	}
	if len(s.Previous) == 0 || json.Unmarshal(s.Previous, &previous) != nil {
		return ""
	}
	return strings.TrimSpace(previous.Command)
}

func loadState(path string) state {
	var s state
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	return s
}

// Settings binds one Claude settings.json and the resource state beside it.
type Settings struct {
	SettingsPath string
	StatePath    string
}

// DefaultSettings is ~/.claude/settings.json with state in ~/.claude/.vrooli-hooks.
func DefaultSettings(hookStateDir, settingsPath string) Settings {
	return Settings{SettingsPath: settingsPath, StatePath: filepath.Join(hookStateDir, stateFileName)}
}

// SelfCommand is the statusLine command: the running binary (or CommandEnv)
// followed by `statusline run`.
func SelfCommand() string {
	exe := strings.TrimSpace(os.Getenv(CommandEnv))
	if exe == "" {
		if path, err := os.Executable(); err == nil {
			if resolved, rErr := filepath.EvalSymlinks(path); rErr == nil {
				path = resolved
			}
			exe = path
		} else {
			exe = "resource-claude-code"
		}
	}
	return shellQuote(exe) + " statusline run"
}

func shellQuote(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !(r == '/' || r == '.' || r == '-' || r == '_' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	}) < 0 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

// Result reports what a reconcile or removal changed.
type Result struct {
	Status   string `json:"status"` // installed | unchanged | removed | absent
	Command  string `json:"command,omitempty"`
	Chained  string `json:"chained,omitempty"`
	Settings string `json:"settingsPath"`
}

// Install reconciles statusLine to command, idempotently. A statusLine that is
// not already ours is kept in the state file and chained by the feed.
func (s Settings) Install(command string) (Result, error) {
	result := Result{Command: command, Settings: s.SettingsPath}
	err := s.edit(func(top map[string]json.RawMessage) (bool, error) {
		current := top["statusLine"]
		st := loadState(s.StatePath)
		if statusCommand(current) == command {
			result.Status, result.Chained = "unchanged", st.previousCommand()
			return false, nil
		}
		if isOurs(current, st) {
			// Our earlier command at another path: keep the recorded previous.
		} else {
			st.Previous = append(json.RawMessage(nil), current...)
		}
		st.Command = command
		if err := writeJSON(s.StatePath, st); err != nil {
			return false, err
		}
		encoded, err := json.Marshal(map[string]any{"type": "command", "command": command, "padding": 0})
		if err != nil {
			return false, err
		}
		top["statusLine"] = encoded
		result.Status, result.Chained = "installed", st.previousCommand()
		return true, nil
	})
	return result, err
}

// Uninstall restores the displaced statusLine (or removes the key) when the
// current one is ours, and deletes the state file. A statusLine the operator
// changed since install is left alone.
func (s Settings) Uninstall() (Result, error) {
	result := Result{Settings: s.SettingsPath, Status: "absent"}
	err := s.edit(func(top map[string]json.RawMessage) (bool, error) {
		st := loadState(s.StatePath)
		if !isOurs(top["statusLine"], st) {
			return false, removeIfExists(s.StatePath)
		}
		if len(st.Previous) > 0 && string(st.Previous) != "null" {
			top["statusLine"] = st.Previous
		} else {
			delete(top, "statusLine")
		}
		result.Status, result.Chained = "removed", st.previousCommand()
		return true, removeIfExists(s.StatePath)
	})
	return result, err
}

// Installed reports the current statusLine command and whether it is ours.
func (s Settings) Installed() (string, bool, error) {
	top, err := readSettings(s.SettingsPath)
	if err != nil {
		return "", false, err
	}
	return statusCommand(top["statusLine"]), isOurs(top["statusLine"], loadState(s.StatePath)), nil
}

func isOurs(current json.RawMessage, st state) bool {
	command := statusCommand(current)
	return command != "" && (command == st.Command || strings.HasSuffix(command, " statusline run"))
}

func statusCommand(raw json.RawMessage) string {
	var line struct {
		Command string `json:"command"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &line) != nil {
		return ""
	}
	return strings.TrimSpace(line.Command)
}

func (s Settings) edit(fn func(map[string]json.RawMessage) (bool, error)) error {
	if err := os.MkdirAll(filepath.Dir(s.SettingsPath), 0o755); err != nil {
		return err
	}
	return agentharness.NewHookBroker().WithLock(s.SettingsPath, func() error {
		top, err := readSettings(s.SettingsPath)
		if err != nil {
			return err
		}
		changed, err := fn(top)
		if err != nil || !changed {
			return err
		}
		return writeJSON(s.SettingsPath, top)
	})
}

func readSettings(path string) (map[string]json.RawMessage, error) {
	top := map[string]json.RawMessage{}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) || err == nil && len(bytes.TrimSpace(data)) == 0 {
		return top, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &top); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return top, nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
