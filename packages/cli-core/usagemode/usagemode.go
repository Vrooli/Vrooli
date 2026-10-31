// Package usagemode reads, writes and applies usage modes: expiring operator
// policies that lower what new coding-agent work runs on. See
// docs/configuration/usage-modes.md. It uses only the standard library so every
// module that imports cli-core can depend on it without cost, and every read
// fails open: a missing, corrupt or expired file means no mode.
package usagemode

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"
)

// SchemaVersion is the only mode.json schema this package reads or writes.
const SchemaVersion = 1

// EnvMode names the mode applied to a launch.
const EnvMode = "VROOLI_USAGE_MODE"

// Until kinds.
const (
	UntilWindowReset = "window_reset"
	UntilTime        = "time"
	UntilOff         = "off"
)

// DefaultExemptRoles keep their model unless a mode is scoped to that role.
var DefaultExemptRoles = []string{"code.fallback", "code.review", "judgment.supervision", "code.supervision"}

type Scope struct {
	Runner  string `json:"runner"`
	Role    string `json:"role"`
	Profile string `json:"profile"`
}

type Until struct {
	Kind          string `json:"kind"`
	Provider      string `json:"provider,omitempty"`
	WindowMinutes int    `json:"window_minutes,omitempty"`
}

// RunnerSettings are concrete values computed when the mode starts.
type RunnerSettings struct {
	ModelMap           map[string]string `json:"model_map,omitempty"`
	SessionModel       string            `json:"session_model,omitempty"`
	Effort             string            `json:"effort,omitempty"`
	ContextLimitTokens int               `json:"context_limit_tokens,omitempty"`

	extra map[string]json.RawMessage
}

type Mode struct {
	ID          string                    `json:"id"`
	Label       string                    `json:"label,omitempty"`
	Kind        string                    `json:"kind,omitempty"`
	Scope       Scope                     `json:"scope"`
	StartedAt   time.Time                 `json:"started_at"`
	StartedBy   string                    `json:"started_by,omitempty"`
	Source      string                    `json:"source,omitempty"`
	Until       Until                     `json:"until"`
	ExpiresAt   *time.Time                `json:"expires_at,omitempty"`
	ExemptRoles []string                  `json:"exempt_roles,omitempty"`
	Runners     map[string]RunnerSettings `json:"runners"`
	// Learn is set on a learn-kind mode: a controlled comparison, not a saving.
	Learn *Learn `json:"learn,omitempty"`

	extra map[string]json.RawMessage
}

type File struct {
	SchemaVersion int    `json:"schema_version"`
	Modes         []Mode `json:"modes"`

	extra map[string]json.RawMessage
}

// Fields a newer writer may add survive a rewrite by this version: File, Mode,
// Learn and RunnerSettings keep any JSON key they do not know and write it back.

func (f *File) UnmarshalJSON(b []byte) error {
	type plain File
	return decodeKeepingExtra(b, (*plain)(f), &f.extra)
}
func (f File) MarshalJSON() ([]byte, error) {
	type plain File
	return encodeWithExtra(plain(f), f.extra)
}
func (m *Mode) UnmarshalJSON(b []byte) error {
	type plain Mode
	return decodeKeepingExtra(b, (*plain)(m), &m.extra)
}
func (m Mode) MarshalJSON() ([]byte, error) {
	type plain Mode
	return encodeWithExtra(plain(m), m.extra)
}
func (l *Learn) UnmarshalJSON(b []byte) error {
	type plain Learn
	return decodeKeepingExtra(b, (*plain)(l), &l.extra)
}
func (l Learn) MarshalJSON() ([]byte, error) {
	type plain Learn
	return encodeWithExtra(plain(l), l.extra)
}
func (r *RunnerSettings) UnmarshalJSON(b []byte) error {
	type plain RunnerSettings
	return decodeKeepingExtra(b, (*plain)(r), &r.extra)
}
func (r RunnerSettings) MarshalJSON() ([]byte, error) {
	type plain RunnerSettings
	return encodeWithExtra(plain(r), r.extra)
}

func jsonKeys(v any) map[string]bool {
	keys := map[string]bool{}
	t := reflect.TypeOf(v).Elem()
	for i := 0; i < t.NumField(); i++ {
		if name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ","); name != "" && name != "-" {
			keys[name] = true
		}
	}
	return keys
}

func decodeKeepingExtra(b []byte, dst any, extra *map[string]json.RawMessage) error {
	if err := json.Unmarshal(b, dst); err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if json.Unmarshal(b, &all) != nil {
		return nil
	}
	known := jsonKeys(dst)
	*extra = nil
	for key, value := range all {
		if !known[key] {
			if *extra == nil {
				*extra = map[string]json.RawMessage{}
			}
			(*extra)[key] = value
		}
	}
	return nil
}

func encodeWithExtra(v any, extra map[string]json.RawMessage) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil || len(extra) == 0 {
		return raw, err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, err
	}
	for key, value := range extra {
		if _, own := all[key]; !own {
			all[key] = value
		}
	}
	return json.Marshal(all)
}

// Path is the mode file, resolved through VROOLI_HOME when set.
func Path() string {
	home := strings.TrimSpace(os.Getenv("VROOLI_HOME"))
	if home == "" {
		user, err := os.UserHomeDir()
		if err != nil || user == "" {
			return ""
		}
		home = filepath.Join(user, ".vrooli")
	}
	return filepath.Join(home, "state", "vrooli", "usage-mode", "mode.json")
}

// Active reports whether the mode is in force at now. An off mode has no
// expiry; every other kind needs one, and it is exclusive.
func (m Mode) Active(now time.Time) bool {
	if m.ID == "" || m.StartedAt.After(now) {
		return false
	}
	if m.Until.Kind == UntilOff {
		return true
	}
	return m.ExpiresAt != nil && now.Before(*m.ExpiresAt)
}

// Read returns the unexpired modes in the default file. It never fails.
func Read(now time.Time) []Mode {
	path := Path()
	if path == "" {
		return nil
	}
	return ReadFile(path, now)
}

// MaxFileBytes caps what a read accepts. A real mode file is a few KiB; a
// larger one is not ours and is ignored rather than slurped into a launch.
const MaxFileBytes = 1 << 20

// readRegular reads a regular file of at most max bytes. The open is
// non-blocking and the type is checked on the opened descriptor, so a FIFO or
// device swapped in at the path can never block a launch.
func readRegular(path string, max int64) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if info.Size() > max {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, max)
	}
	raw, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > max {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, max)
	}
	return raw, nil
}

// ReadFile is Read for an explicit path. A file that is not a regular file, is
// over MaxFileBytes, is corrupt or has another schema means no modes.
func ReadFile(path string, now time.Time) []Mode {
	raw, err := readRegular(path, MaxFileBytes)
	if err != nil {
		return nil
	}
	var file File
	if json.Unmarshal(raw, &file) != nil || file.SchemaVersion != SchemaVersion {
		return nil
	}
	var active []Mode
	for _, mode := range file.Modes {
		if mode.Active(now) {
			active = append(active, mode)
		}
	}
	return active
}

// Prune drops modes that are not active at now.
func (f *File) Prune(now time.Time) {
	kept := f.Modes[:0:0]
	for _, mode := range f.Modes {
		if mode.Active(now) {
			kept = append(kept, mode)
		}
	}
	f.Modes = kept
}

// Update is UpdateFile for the default file.
func Update(fn func(*File) error) error {
	path := Path()
	if path == "" {
		return os.ErrNotExist
	}
	return UpdateFile(path, fn)
}

// UpdateFile is the only safe way to change the file: it holds an exclusive
// lock across the read, fn and the atomic write, so two writers (Agent Manager,
// Web Console, a second process) cannot overwrite each other's change. fn sees
// every stored mode, expired ones included (call Prune to drop them), with any
// fields this version does not know preserved. A missing or corrupt file starts
// empty; a file of another schema version, or one that is not a regular file, is
// an error rather than being overwritten. An error from fn writes nothing.
func UpdateFile(path string, fn func(*File) error) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	unlock, err := lockPath(path + ".lock")
	if err != nil {
		return err
	}
	defer unlock()
	file := File{SchemaVersion: SchemaVersion}
	raw, err := readRegular(path, MaxFileBytes)
	switch {
	case err == nil:
		var parsed File
		if json.Unmarshal(raw, &parsed) == nil {
			if parsed.SchemaVersion != SchemaVersion {
				return fmt.Errorf("usage mode file has schema version %d; this build writes %d", parsed.SchemaVersion, SchemaVersion)
			}
			file = parsed
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return err
	}
	if err := fn(&file); err != nil {
		return err
	}
	return WriteFile(path, file)
}

// Write atomically replaces the default file.
func Write(file File) error {
	path := Path()
	if path == "" {
		return os.ErrNotExist
	}
	return WriteFile(path, file)
}

// WriteFile atomically replaces path (temp file, mode 0600, rename).
func WriteFile(path string, file File) error {
	file.SchemaVersion = SchemaVersion
	if file.Modes == nil {
		file.Modes = []Mode{}
	}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".mode-*.tmp")
	if err != nil {
		return err
	}
	name := temp.Name()
	cleanup := func() { _ = os.Remove(name) }
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		cleanup()
		return err
	}
	if _, err := temp.Write(append(raw, '\n')); err != nil {
		_ = temp.Close()
		cleanup()
		return err
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(name, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

// Query selects the mode for one launch.
type Query struct {
	Runner  string
	Role    string
	Profile string
	Now     time.Time
}

// NormalizeRunner maps resource and command names onto one runner key.
func NormalizeRunner(runner string) string {
	runner = strings.ToLower(strings.TrimSpace(runner))
	if runner == "claude-code" || runner == "claude_code" {
		return "claude"
	}
	return runner
}

// SettingsFor returns the mode's settings for a runner.
func (m Mode) SettingsFor(runner string) (RunnerSettings, bool) {
	want := NormalizeRunner(runner)
	for key, settings := range m.Runners {
		if NormalizeRunner(key) == want {
			return settings, true
		}
	}
	return RunnerSettings{}, false
}

func (m Mode) exempts(role string) bool {
	if role == "" || m.Scope.Role == role {
		return false // a mode scoped to a role lists it explicitly
	}
	for _, exempt := range m.ExemptRoles {
		if exempt == role {
			return true
		}
	}
	for _, exempt := range DefaultExemptRoles {
		if exempt == role {
			return true
		}
	}
	return false
}

func (m Mode) specificity(q Query) int {
	if m.Scope.Profile != "" {
		if m.Scope.Profile != q.Profile {
			return -1
		}
	}
	if m.Scope.Role != "" && m.Scope.Role != q.Role {
		return -1
	}
	if m.Scope.Runner != "" && NormalizeRunner(m.Scope.Runner) != NormalizeRunner(q.Runner) {
		return -1
	}
	switch {
	case m.Scope.Profile != "":
		return 4
	case m.Scope.Role != "":
		return 3
	case m.Scope.Runner != "":
		return 2
	}
	return 1
}

// Resolve returns the most specific active mode that applies to the query and
// carries settings for its runner: profile, then role, then runner, then
// global; among equals the latest start wins. Exempt roles are skipped.
func Resolve(q Query) (Mode, bool) {
	return resolveFrom(Read(q.Now), q)
}

func resolveFrom(modes []Mode, q Query) (Mode, bool) {
	var best Mode
	bestRank, found := -1, false
	for _, mode := range modes {
		// A learn mode assigns arms; it never lowers everything like a save mode.
		if !mode.Active(q.Now) || mode.IsLearn() || mode.exempts(q.Role) {
			continue
		}
		if _, ok := mode.SettingsFor(q.Runner); !ok {
			continue
		}
		rank := mode.specificity(q)
		if rank < 0 {
			continue
		}
		if rank > bestRank || (rank == bestRank && mode.StartedAt.After(best.StartedAt)) {
			best, bestRank, found = mode, rank, true
		}
	}
	return best, found
}
