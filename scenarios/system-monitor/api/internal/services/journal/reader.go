// Package journal centralizes journalctl access for system-monitor.
//
// Each scenario owns its own journal reader (per interop steer "no shared
// Go module across scenarios"). This is a deliberate mirror of the autoheal
// reader, not an import — keeping the scenarios independently buildable and
// avoiding cross-scenario coupling. Drift is mitigated by both implementations
// being thin and well-tested.
package journal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// CommandExecutor abstracts shelling out for tests. system-monitor defines
// its own minimal interface here rather than importing autoheal's checks
// package.
type CommandExecutor interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

// LogEntry is one structured journal record.
type LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Realtime   int64     `json:"realtime"` // microseconds since unix epoch
	Priority   int       `json:"priority"` // 0=emerg .. 7=debug; -1 if unknown
	Unit       string    `json:"unit,omitempty"`
	UserUnit   string    `json:"userUnit,omitempty"`
	Identifier string    `json:"identifier,omitempty"`
	Hostname   string    `json:"hostname,omitempty"`
	PID        int       `json:"pid,omitempty"`
	BootID     string    `json:"bootId,omitempty"`
	Cursor     string    `json:"cursor,omitempty"`
	Message    string    `json:"message"`
	Raw        string    `json:"raw,omitempty"`
}

// BootRecord describes one entry from `journalctl --list-boots`.
type BootRecord struct {
	Index      int       `json:"index"`
	BootID     string    `json:"bootId"`
	FirstEntry time.Time `json:"firstEntry"`
	LastEntry  time.Time `json:"lastEntry"`
}

// QueryOpts is a structured journalctl query.
type QueryOpts struct {
	Unit        []string
	UserUnit    []string
	Kernel      bool
	Since       string
	Until       string
	Boot        string
	Priority    string
	Grep        string
	Tail        int
	Reverse     bool
	AfterCursor string // --after-cursor
	Cursor      string // --cursor (start at)
}

// Reader is the only sanctioned entry point for journalctl access.
type Reader struct {
	exec CommandExecutor
}

// NewReader builds a Reader using the given executor.
func NewReader(exec CommandExecutor) *Reader {
	return &Reader{exec: exec}
}

// Available reports whether journalctl is callable on this host.
func (r *Reader) Available(ctx context.Context) bool {
	if r == nil || r.exec == nil {
		return false
	}
	_, err := r.exec.CombinedOutput(ctx, "journalctl", "--version")
	return err == nil
}

// QueryLogs runs a structured query and parses the JSON output.
func (r *Reader) QueryLogs(ctx context.Context, opts QueryOpts) ([]LogEntry, error) {
	if r == nil || r.exec == nil {
		return nil, errors.New("journal: nil reader")
	}

	// journalctl rejects --since with a cursor. Resolve the same systemd time
	// expression and enforce its inclusive lower bound on the cursor page.
	var lowerBound *int64
	if opts.Since != "" && (opts.AfterCursor != "" || opts.Cursor != "") {
		bound, err := r.resolveJournalTime(ctx, opts.Since)
		if err != nil {
			return nil, err
		}
		lowerBound = &bound
		opts.Since = ""
	}
	args := buildArgs(opts, true)
	out, err := r.exec.CombinedOutput(ctx, "journalctl", args...)
	if err != nil {
		// journalctl's grep mode returns status 1 with no output when no
		// entries match. Preserve diagnostic or other exit failures as errors.
		var exitStatus interface{ ExitCode() int }
		if opts.Grep != "" && len(bytes.TrimSpace(out)) == 0 && errors.As(err, &exitStatus) && exitStatus.ExitCode() == 1 {
			return []LogEntry{}, nil
		}
		return nil, fmt.Errorf("journalctl %s: %w (output: %s)",
			strings.Join(args, " "), err, truncate(string(out), 200))
	}

	entries, jsonErr := parseJSON(out)
	if jsonErr == nil {
		return filterSince(entries, lowerBound)
	}

	textArgs := buildArgs(opts, false)
	textOut, textErr := r.exec.CombinedOutput(ctx, "journalctl", textArgs...)
	if textErr != nil {
		return nil, fmt.Errorf("journalctl text fallback failed: %w", textErr)
	}
	return filterSince(parseText(textOut), lowerBound)
}

func (r *Reader) resolveJournalTime(ctx context.Context, expression string) (int64, error) {
	out, err := r.exec.CombinedOutput(ctx, "systemd-analyze", "timestamp", "--", expression)
	if err != nil {
		return 0, fmt.Errorf("resolve journal since expression: %w (output: %s)", err, truncate(string(out), 200))
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(line, "UNIX seconds: @"); ok {
			seconds, parseErr := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if parseErr == nil && !math.IsNaN(seconds) && !math.IsInf(seconds, 0) {
				return int64(math.Round(seconds * 1e6)), nil
			}
		}
	}
	return 0, errors.New("could not resolve journal since expression")
}

func filterSince(entries []LogEntry, lowerBound *int64) ([]LogEntry, error) {
	if lowerBound == nil {
		return entries, nil
	}
	filtered := make([]LogEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Realtime == 0 {
			return nil, errors.New("journal entry has no timestamp to enforce since bound")
		}
		if entry.Realtime >= *lowerBound {
			filtered = append(filtered, entry)
		}
	}
	return filtered, nil
}

// ListBoots returns parsed `journalctl --list-boots` output.
func (r *Reader) ListBoots(ctx context.Context) ([]BootRecord, error) {
	if r == nil || r.exec == nil {
		return nil, errors.New("journal: nil reader")
	}

	out, err := r.exec.CombinedOutput(ctx, "journalctl", "--list-boots", "--no-pager", "-o", "json")
	if err == nil {
		if boots, parseErr := parseBootsJSON(out); parseErr == nil {
			return boots, nil
		}
	}

	textOut, textErr := r.exec.CombinedOutput(ctx, "journalctl", "--list-boots", "--no-pager")
	if textErr != nil {
		return nil, fmt.Errorf("journalctl --list-boots: %w", textErr)
	}
	return parseBootsText(textOut), nil
}

// buildArgs renders QueryOpts to a deterministic argv slice.
func buildArgs(opts QueryOpts, jsonFormat bool) []string {
	args := []string{"--no-pager"}

	if jsonFormat {
		// Without --all journalctl replaces fields over 4096 bytes with null.
		args = append(args, "--all", "-o", "json")
	} else {
		args = append(args, "-o", "short-iso")
	}

	for _, u := range opts.Unit {
		args = append(args, "-u", u)
	}
	for _, u := range opts.UserUnit {
		args = append(args, "--user-unit", u)
	}
	if opts.Kernel {
		args = append(args, "-k")
	}
	if opts.Boot != "" {
		args = append(args, "-b", opts.Boot)
	}
	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}
	if opts.Until != "" {
		args = append(args, "--until", opts.Until)
	}
	if opts.Priority != "" {
		args = append(args, "-p", opts.Priority)
	}
	if opts.Grep != "" {
		args = append(args, "-g", opts.Grep)
	}
	if opts.AfterCursor != "" {
		args = append(args, "--after-cursor", opts.AfterCursor)
	}
	if opts.Cursor != "" {
		args = append(args, "--cursor", opts.Cursor)
	}
	if opts.Tail > 0 {
		args = append(args, "-n", strconv.Itoa(opts.Tail))
	}
	if opts.Reverse {
		args = append(args, "-r")
	}
	return args
}

type rawJournalEntry struct {
	Realtime    json.RawMessage `json:"__REALTIME_TIMESTAMP"`
	Cursor      json.RawMessage `json:"__CURSOR"`
	BootID      json.RawMessage `json:"_BOOT_ID"`
	Hostname    json.RawMessage `json:"_HOSTNAME"`
	Unit        json.RawMessage `json:"_SYSTEMD_UNIT"`
	UserUnit    json.RawMessage `json:"_SYSTEMD_USER_UNIT"`
	Identifier  json.RawMessage `json:"SYSLOG_IDENTIFIER"`
	PID         json.RawMessage `json:"_PID"`
	Priority    json.RawMessage `json:"PRIORITY"`
	MessageJSON json.RawMessage `json:"MESSAGE"`
}

func parseJSON(out []byte) ([]LogEntry, error) {
	if len(bytes.TrimSpace(out)) == 0 {
		return nil, nil
	}
	var entries []LogEntry
	parsedAny := false
	// The command output is already in memory. Split it without a Scanner
	// token limit so a long journal record remains structured and complete.
	for _, line := range bytes.Split(out, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var raw rawJournalEntry
		if err := json.Unmarshal(line, &raw); err != nil {
			entries = append(entries, LogEntry{Raw: string(line), Priority: -1})
			continue
		}
		parsedAny = true
		entry := rawToEntry(raw)
		// Preserve the complete journal record, including fields not mapped to
		// the compact list, for the operator's read-only evidence workspace.
		entry.Raw = string(line)
		entries = append(entries, entry)
	}
	if !parsedAny && len(entries) > 0 {
		return nil, errors.New("journal: no entries parsed as JSON")
	}
	return entries, nil
}

func rawToEntry(r rawJournalEntry) LogEntry {
	e := LogEntry{
		Unit: journalString(r.Unit), UserUnit: journalString(r.UserUnit), Identifier: journalString(r.Identifier),
		Hostname: journalString(r.Hostname), BootID: journalString(r.BootID), Cursor: journalString(r.Cursor),
		Priority: -1,
	}
	if realtime := journalString(r.Realtime); realtime != "" {
		if us, err := strconv.ParseInt(realtime, 10, 64); err == nil {
			e.Realtime = us
			e.Timestamp = time.Unix(us/1_000_000, (us%1_000_000)*1_000).UTC()
		}
	}
	if pidText := journalString(r.PID); pidText != "" {
		if pid, err := strconv.Atoi(pidText); err == nil {
			e.PID = pid
		}
	}
	if priority := journalString(r.Priority); priority != "" {
		if p, err := strconv.Atoi(priority); err == nil {
			e.Priority = p
		}
	}
	e.Message = decodeMessage(r.MessageJSON)
	return e
}

// A repeated or binary journal field is represented as an array. Keep its
// original value in Raw rather than selecting an arbitrary compact value.
func journalString(raw json.RawMessage) string {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func decodeMessage(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var bytesArr []int
	if err := json.Unmarshal(raw, &bytesArr); err == nil {
		buf := make([]byte, len(bytesArr))
		for i, b := range bytesArr {
			buf[i] = byte(b)
		}
		return string(buf)
	}
	return string(raw)
}

func parseText(out []byte) []LogEntry {
	var entries []LogEntry
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "-- ") || strings.HasPrefix(line, "Hint: ") {
			continue
		}
		entries = append(entries, parseTextLine(line))
	}
	return entries
}

func parseTextLine(line string) LogEntry {
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return LogEntry{Raw: line, Priority: -1}
	}
	ts, err := time.Parse("2006-01-02T15:04:05-0700", parts[0])
	if err != nil {
		return LogEntry{Raw: line, Priority: -1}
	}
	rest := parts[2]
	identMsg := strings.SplitN(rest, ": ", 2)
	identifier := strings.TrimSpace(identMsg[0])
	pid := 0
	if i := strings.LastIndex(identifier, "["); i > 0 && strings.HasSuffix(identifier, "]") {
		pidStr := identifier[i+1 : len(identifier)-1]
		if v, perr := strconv.Atoi(pidStr); perr == nil {
			pid = v
			identifier = identifier[:i]
		}
	}
	msg := ""
	if len(identMsg) == 2 {
		msg = identMsg[1]
	}
	return LogEntry{
		Timestamp:  ts.UTC(),
		Realtime:   ts.UnixMicro(),
		Hostname:   parts[1],
		Identifier: identifier,
		PID:        pid,
		Priority:   -1,
		Message:    msg,
	}
}

type rawBootJSON struct {
	Index         int    `json:"index"`
	BootID        string `json:"boot_id"`
	FirstEntry    int64  `json:"first_entry"`
	LastEntry     int64  `json:"last_entry"`
	FirstEntryStr string `json:"first_entry_str"`
	LastEntryStr  string `json:"last_entry_str"`
}

func parseBootsJSON(out []byte) ([]BootRecord, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}
	var raws []rawBootJSON
	if err := json.Unmarshal(trimmed, &raws); err != nil {
		return nil, err
	}
	if len(raws) == 0 {
		return nil, errors.New("journal: empty boot list")
	}
	out2 := make([]BootRecord, 0, len(raws))
	for _, r := range raws {
		b := BootRecord{Index: r.Index, BootID: r.BootID}
		if r.FirstEntry > 0 {
			b.FirstEntry = time.Unix(r.FirstEntry/1_000_000, (r.FirstEntry%1_000_000)*1_000).UTC()
		} else if r.FirstEntryStr != "" {
			if t, err := time.Parse(time.RFC3339, r.FirstEntryStr); err == nil {
				b.FirstEntry = t.UTC()
			}
		}
		if r.LastEntry > 0 {
			b.LastEntry = time.Unix(r.LastEntry/1_000_000, (r.LastEntry%1_000_000)*1_000).UTC()
		} else if r.LastEntryStr != "" {
			if t, err := time.Parse(time.RFC3339, r.LastEntryStr); err == nil {
				b.LastEntry = t.UTC()
			}
		}
		out2 = append(out2, b)
	}
	return out2, nil
}

func parseBootsText(out []byte) []BootRecord {
	var records []BootRecord
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 0, 8*1024), 1*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		idx, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		records = append(records, BootRecord{
			Index:  idx,
			BootID: fields[1],
		})
	}
	return records
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
