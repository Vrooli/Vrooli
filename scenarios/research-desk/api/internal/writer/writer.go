// Package writer is the desk's only way to change a file. Each method names
// one allow-listed file of FILE-CONTRACT.md §1; there is no general write.
// Only the lifecycle-managed live instance writes, and only while it holds
// the lock; every other instance is read-only and says so.
package writer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"research-desk/internal/home"
)

// LockFile is the instance lock inside the state folder.
const LockFile = "instance.lock"

// ErrReadOnly is returned by every write of an instance that is not the writer.
var ErrReadOnly = home.Cause(home.ErrRefused, errors.New("this research-desk instance is read-only"))

// Writer appends operator entries, enables steering and writes desk records.
type Writer struct {
	researchDir string
	stateDir    string
	// entryChars is limits.json's entry_chars_max, read at each write.
	entryChars func() (int, error)
	// notWriter is why this instance never writes; empty for the writer.
	notWriter string

	mu   sync.Mutex
	lock *os.File
}

// DefaultStateDir is the desk's own folder, outside the research home:
// ~/.vrooli/state/vrooli/research-desk. It holds the lock, look snapshots and
// courier switches; deleting it only resets "since" and turns couriers back
// to their defaults.
func DefaultStateDir(userHome string) string {
	return filepath.Join(userHome, ".vrooli", "state", "vrooli", "research-desk")
}

// New returns a Writer. Nothing is touched until the first write.
// entryChars bounds one entry's text (limits.json entry_chars_max).
func New(researchDir, stateDir string, entryChars func() (int, error)) *Writer {
	return &Writer{researchDir: researchDir, stateDir: stateDir, entryChars: entryChars}
}

// EntryChars is the most characters one entry may hold, 0 when limits.json
// cannot be read (a write then says why).
func (w *Writer) EntryChars() int {
	max, err := w.entryChars()
	if err != nil {
		return 0
	}
	return max
}

// FitEntry makes text from elsewhere (an answer note typed on the phone) fit
// an entry; when the limit cannot be read the text is unchanged and the
// append then says why.
func (w *Writer) FitEntry(text string) string {
	if max, err := w.entryChars(); err == nil {
		return home.FitEntryText(text, max)
	}
	return text
}

// ReadOnly makes this instance a reader for good: it never takes the lock or
// writes, and every write says why.
func (w *Writer) ReadOnly(reason string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.notWriter = reason
}

// Role reports whether this instance is the writer without taking the lock:
// nil for the writer, else why not. Reads use it; only a write takes the lock.
func (w *Writer) Role() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.notWriter != "" {
		return fmt.Errorf("%w: %s", ErrReadOnly, w.notWriter)
	}
	return nil
}

// ResearchDir is the research home the writer writes in.
func (w *Writer) ResearchDir() string { return w.researchDir }

// StateDir is the desk's own folder.
func (w *Writer) StateDir() string { return w.stateDir }

// Held takes the lock if it is free and reports whether this instance holds
// it. It retries on every call, so a second instance becomes the writer once
// the first exits.
func (w *Writer) Held() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.held()
}

func (w *Writer) held() error {
	if w.notWriter != "" {
		return fmt.Errorf("%w: %s", ErrReadOnly, w.notWriter)
	}
	if w.lock != nil {
		return nil
	}
	if err := os.MkdirAll(w.stateDir, 0o755); err != nil {
		return fmt.Errorf("create the desk state folder %s: %w", w.stateDir, err)
	}
	path := filepath.Join(w.stateDir, LockFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open the lock %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return fmt.Errorf("%w: another instance holds the write lock %s; stop it or use it", ErrReadOnly, path)
		}
		return fmt.Errorf("lock %s: %w", path, err)
	}
	w.lock = f
	return nil
}

// Close releases the lock.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lock == nil {
		return nil
	}
	err := w.lock.Close()
	w.lock = nil
	return err
}

// AppendOperator appends one entry to cohorts/<id>/OPERATOR.md, creating it
// with its title when absent.
func (w *Writer) AppendOperator(commissionID string, e home.Entry) (string, error) {
	path, title, err := w.operatorFile(commissionID, e)
	if err != nil {
		return "", err
	}
	return path, w.appendEntry(path, title, e)
}

// RenderOperator returns the exact text AppendOperator would append now, and
// the file; it writes nothing and needs no writer role.
func (w *Writer) RenderOperator(commissionID string, e home.Entry) (path, text string, err error) {
	path, title, err := w.operatorFile(commissionID, e)
	if err != nil {
		return "", "", err
	}
	text, err = w.render(path, title, e)
	return path, text, err
}

func (w *Writer) operatorFile(commissionID string, e home.Entry) (path, title string, err error) {
	switch e.Kind {
	case home.EntryDirection, home.EntryWindDown, home.EntryAnswer, home.EntryDelivered:
	default:
		return "", "", fmt.Errorf("%w: OPERATOR.md takes direction, wind-down, answer and delivered entries, not %q", home.ErrInvalid, e.Kind)
	}
	dir, err := w.folder("cohorts", commissionID)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(dir, home.OperatorFile), "# Operator — " + commissionID, nil
}

// RenderStewardFinding returns an attributed finding for the commission's
// lower-priority STEWARD.md channel without writing it.
func (w *Writer) RenderStewardFinding(commissionID, runID, body string, at time.Time) (string, string, error) {
	dir, err := w.folder("cohorts", commissionID)
	if err != nil {
		return "", "", err
	}
	path := filepath.Join(dir, "STEWARD.md")
	text, err := w.stewardFindingText(path, runID, body, at)
	return path, text, err
}

// AppendStewardFinding writes a dated finding attributed to its verified
// agent run. It never writes the operator's higher-priority OPERATOR.md.
func (w *Writer) AppendStewardFinding(commissionID, runID, body string, at time.Time) (string, string, error) {
	path, text, err := w.RenderStewardFinding(commissionID, runID, body, at)
	if err != nil {
		return "", "", err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.held(); err != nil {
		return "", "", err
	}
	if _, err := w.enableStewardLocked(commissionID); err != nil {
		return "", "", err
	}
	text, err = w.stewardFindingText(path, runID, body, at)
	if err != nil {
		return "", "", err
	}
	return path, text, appendBytes(path, text)
}

func (w *Writer) enableStewardLocked(commissionID string) (string, error) {
	dir, err := w.folder("cohorts", commissionID)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, home.CharterFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read the charter %s: %w", path, err)
	}
	if bytes.Contains(data, []byte(home.StewardSentence)) {
		return "", nil
	}
	text := separator(data) + home.StewardSentence + "\n"
	return text, appendBytes(path, text)
}

func (w *Writer) stewardFindingText(path, runID, body string, at time.Time) (string, error) {
	body = strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n"))
	if runID == "" || at.IsZero() || body == "" {
		return "", fmt.Errorf("%w: a steward finding needs an agent run, time and text", home.ErrInvalid)
	}
	max, err := w.entryChars()
	if err != nil {
		return "", fmt.Errorf("%w: the entry limit is unknown: %v", home.ErrUnavailable, err)
	}
	if err := home.CheckEntryText(body, w.researchDir, max); err != nil {
		return "", fmt.Errorf("%w: %v", home.ErrInvalid, err)
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "### ") {
			return "", fmt.Errorf("%w: a steward finding cannot contain a heading line", home.ErrInvalid)
		}
	}
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	prefix := separator(existing)
	if len(existing) == 0 {
		prefix = "# Steward findings\n\n"
	}
	entry := fmt.Sprintf("### %s · finding · by agent run %s\n%s\n", at.UTC().Format(home.EntryTimeLayout), runID, body)
	return prefix + entry, nil
}

// AppendInbox appends one entry `### <UTC> · from <from> · <kind>` to a
// level's INBOX.md (docs/agent-system/LEVELS.md): topics/<topic>/INBOX.md, or
// the research home's (the supervisor's) when topic is empty. It creates the
// file when absent, and writes nothing when the inbox or its drained history
// already holds marker, which the body must name; wrote is false then. A body
// line that would start an entry is escaped, so the operator's words never
// split the file.
func (w *Writer) AppendInbox(topic, marker string, at time.Time, from, kind, body string) (path string, wrote bool, err error) {
	path, text, err := w.RenderInbox(topic, marker, at, from, kind, body)
	if err != nil {
		return "", false, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.held(); err != nil {
		return "", false, err
	}
	for _, file := range []string{path, filepath.Join(filepath.Dir(path), "history", home.InboxFile)} {
		existing, err := os.ReadFile(file)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", false, fmt.Errorf("read %s: %w", file, err)
		}
		if bytes.Contains(existing, []byte(marker)) {
			return path, false, nil
		}
	}
	existing, _ := os.ReadFile(path)
	prefix := separator(existing)
	if len(existing) == 0 {
		prefix = "# Inbox — " + filepath.Base(filepath.Dir(path)) + "\n\n"
	}
	return path, true, appendBytes(path, prefix+text)
}

// RenderInbox returns the inbox AppendInbox writes to and the entry it
// appends; it writes nothing and needs no writer role.
func (w *Writer) RenderInbox(topic, marker string, at time.Time, from, kind, body string) (path, text string, err error) {
	body = w.FitEntry(body)
	if marker == "" || !strings.Contains(body, marker) {
		return "", "", fmt.Errorf("%w: an inbox entry names its marker %q, which keeps it from being written twice", home.ErrInvalid, marker)
	}
	dir := w.researchDir
	if topic != "" {
		if dir, err = w.folder("topics", topic); err != nil {
			return "", "", err
		}
	}
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n")), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "### ") {
			lines[i] = `\` + line
		}
	}
	heading := "### " + at.UTC().Format(home.EntryTimeLayout) + " · from " + from + " · " + kind
	return filepath.Join(dir, home.InboxFile), heading + "\n" + strings.Join(lines, "\n") + "\n", nil
}

// AppendVerdict appends one verdict to reviews/<id>/VERDICTS.md.
func (w *Writer) AppendVerdict(reviewID string, e home.Entry) (string, error) {
	path, title, err := w.verdictFile(reviewID, e)
	if err != nil {
		return "", err
	}
	return path, w.appendEntry(path, title, e)
}

// RenderVerdict returns the exact text AppendVerdict would append now, and
// the file; it writes nothing and needs no writer role.
func (w *Writer) RenderVerdict(reviewID string, e home.Entry) (path, text string, err error) {
	path, title, err := w.verdictFile(reviewID, e)
	if err != nil {
		return "", "", err
	}
	text, err = w.render(path, title, e)
	return path, text, err
}

func (w *Writer) verdictFile(reviewID string, e home.Entry) (path, title string, err error) {
	if e.Kind != home.EntryVerdict {
		return "", "", fmt.Errorf("%w: VERDICTS.md takes verdict entries only", home.ErrInvalid)
	}
	dir, err := w.folder(home.ReviewsDir, reviewID)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(dir, home.VerdictsFile), "# Verdicts — " + reviewID, nil
}

// RenderSteering returns the missing operator and steward guidance for an
// older charter; it changes nothing when both authority clauses are present.
func (w *Writer) RenderSteering(commissionID string) (path, text string, err error) {
	dir, err := w.folder("cohorts", commissionID)
	if err != nil {
		return "", "", err
	}
	path = filepath.Join(dir, home.CharterFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return path, "", fmt.Errorf("read the charter %s: %w", path, err)
	}
	var additions strings.Builder
	if !bytes.Contains(data, []byte(home.SteeringSentence)) {
		additions.WriteString("## Steering\n\n" + home.SteeringSentence + "\n")
	}
	if !bytes.Contains(data, []byte(home.StewardSentence)) {
		additions.WriteString(home.StewardSentence + "\n")
	}
	if additions.Len() == 0 {
		return path, "", nil
	}
	return path, separator(data) + additions.String(), nil
}

// EnableSteering appends the Steering section to an older charter that does
// not carry the Steering sentence: the text RenderSteering returns, rendered
// again under the lock. It changes nothing when the sentence is already
// there.
func (w *Writer) EnableSteering(commissionID string) (path, text string, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.held(); err != nil {
		return "", "", err
	}
	if path, text, err = w.RenderSteering(commissionID); err != nil || text == "" {
		return path, "", err
	}
	return path, text, appendBytes(path, text)
}

// WriteLaunch replaces cohorts/<id>/LAUNCH.json atomically.
func (w *Writer) WriteLaunch(commissionID string, record *home.LaunchRecord) (string, error) {
	if record == nil || record.Schema != home.LaunchSchema || record.CommissionID != commissionID {
		return "", fmt.Errorf("%w: a launch record needs schema %s and commission_id %s", home.ErrInvalid, home.LaunchSchema, commissionID)
	}
	dir, err := w.folder("cohorts", commissionID)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, home.LaunchFile)
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.held(); err != nil {
		return path, err
	}
	return path, writeJSON(path, record)
}

// WriteHistoricalSettlement creates one immutable, evidence-backed historical
// disposition. It never edits LAUNCH.json or changes the observed worker state.
func (w *Writer) RenderHistoricalSettlement(record *home.HistoricalSettlement) (string, string, error) {
	path, err := w.validateHistoricalSettlement(record)
	if err != nil {
		return path, "", err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	return path, string(data) + "\n", err
}

func (w *Writer) validateHistoricalSettlement(record *home.HistoricalSettlement) (string, error) {
	if record == nil || record.Schema != home.SettlementSchema || record.CommissionID == "" || !home.HistoricalStatus(record.Status).Valid() || record.AsOf.IsZero() || record.RecordedAt.IsZero() || record.RecordedBy == "" || strings.TrimSpace(record.Reason) == "" || len(record.Evidence) == 0 {
		return "", fmt.Errorf("%w: settlement needs a valid status, times, reason and evidence", home.ErrInvalid)
	}
	if (record.ActivityStart == nil) != (record.ActivityEnd == nil) || (record.ActivityStart != nil && record.ActivityEnd.Before(*record.ActivityStart)) || (record.Status != string(home.HistoricalPrepared) && record.ActivityStart == nil) {
		return "", fmt.Errorf("%w: completed dispositions need a complete, ordered activity interval", home.ErrInvalid)
	}
	dir, err := w.folder("cohorts", record.CommissionID)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, home.SettlementFile)
	if err := home.ValidateHistoricalEvidence(w.researchDir, record); err != nil {
		return path, err
	}
	return path, nil
}

// WriteHistoricalSettlement creates an immutable historical disposition.
func (w *Writer) WriteHistoricalSettlement(record *home.HistoricalSettlement) (string, error) {
	path, err := w.validateHistoricalSettlement(record)
	if err != nil {
		return path, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.held(); err != nil {
		return path, err
	}
	if _, err := os.Lstat(path); err == nil {
		return path, fmt.Errorf("%w: %s already has an immutable historical settlement", home.ErrRefused, record.CommissionID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return path, err
	}
	if err := writeJSON(path, record); err != nil {
		return path, err
	}
	return path, nil
}

// WriteState replaces one JSON file in the desk state folder atomically.
// name is a slash-separated path inside it, such as "courier.json" or
// "looks/commission-x.json".
func (w *Writer) WriteState(name string, v any) (string, error) {
	path, err := w.statePath(name)
	if err != nil {
		return "", err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.held(); err != nil {
		return path, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, err
	}
	return path, writeJSON(path, v)
}

// RemoveState removes one JSON file of the desk state folder that a newer
// file replaced; a missing file is not an error.
func (w *Writer) RemoveState(name string) error {
	path, err := w.statePath(name)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.held(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// ReadState decodes one JSON file of the desk state folder; found is false
// when it does not exist.
func (w *Writer) ReadState(name string, into any) (found bool, err error) {
	path, err := w.statePath(name)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(data, into); err != nil {
		return true, fmt.Errorf("%s is not valid JSON (delete it to reset): %w", path, err)
	}
	return true, nil
}

func (w *Writer) statePath(name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") || filepath.Ext(clean) != ".json" {
		return "", fmt.Errorf("%w: state file %q must be a relative .json path inside %s", home.ErrInvalid, name, w.stateDir)
	}
	return filepath.Join(w.stateDir, clean), nil
}

// folder returns <research home>/<kind>/<id> when it is an existing folder.
func (w *Writer) folder(kind, id string) (string, error) {
	if !home.FolderID.MatchString(id) {
		return "", fmt.Errorf("%w: %q is not a folder name under %s/", home.ErrInvalid, id, kind)
	}
	dir := filepath.Join(w.researchDir, kind, id)
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("%w: no folder %s", home.ErrInvalid, dir)
	}
	return dir, nil
}

// appendEntry appends one entry as a single O_APPEND write.
func (w *Writer) appendEntry(path, title string, e home.Entry) error {
	if err := w.checkEntry(e); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.held(); err != nil {
		return err
	}
	text, err := w.text(path, title, e)
	if err != nil {
		return err
	}
	return appendBytes(path, text)
}

// render is the text appendEntry would append to path now.
func (w *Writer) render(path, title string, e home.Entry) (string, error) {
	if err := w.checkEntry(e); err != nil {
		return "", err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.text(path, title, e)
}

// checkEntry refuses an entry the file grammar cannot hold.
func (w *Writer) checkEntry(e home.Entry) error {
	body := entryBody(e)
	if body == "" {
		return fmt.Errorf("%w: an entry needs words", home.ErrInvalid)
	}
	max, err := w.entryChars()
	if err != nil {
		return fmt.Errorf("%w: the entry limit is unknown: %v", home.ErrUnavailable, err)
	}
	if err := home.CheckEntryText(body, w.researchDir, max); err != nil {
		return fmt.Errorf("%w: %v", home.ErrInvalid, err)
	}
	for _, line := range strings.Split(body, "\n") {
		if home.IsEntryHeading(line) {
			return fmt.Errorf("%w: the line %q would start a new entry; reword it", home.ErrInvalid, line)
		}
	}
	if e.At.IsZero() {
		return fmt.Errorf("%w: an entry needs its time", home.ErrInvalid)
	}
	return nil
}

func entryBody(e home.Entry) string {
	return strings.TrimSpace(strings.ReplaceAll(e.Body, "\r\n", "\n"))
}

// text is the exact text appended to path for e: the title for a new file or
// a separator, the entry heading and the body.
func (w *Writer) text(path, title string, e home.Entry) (string, error) {
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	prefix := separator(existing)
	if len(existing) == 0 {
		prefix = title + "\n\n"
	}
	return prefix + e.Heading() + "\n" + entryBody(e) + "\n", nil
}

// separator is what goes before appended text so it starts after a blank line.
func separator(existing []byte) string {
	switch {
	case len(existing) == 0, bytes.HasSuffix(existing, []byte("\n\n")):
		return ""
	case bytes.HasSuffix(existing, []byte("\n")):
		return "\n"
	default:
		return "\n\n"
	}
}

func appendBytes(path, text string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	if _, err := f.Write([]byte(text)); err != nil {
		f.Close()
		return fmt.Errorf("append to %s: %w", path, err)
	}
	return f.Close()
}

// writeJSON writes a temp file beside path and renames it over path.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
