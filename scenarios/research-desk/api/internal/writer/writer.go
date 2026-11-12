// Package writer is the desk's only way to change a file. Each method names
// one allow-listed file of FILE-CONTRACT.md §1; there is no general write.
// Only the lifecycle-managed live instance writes, and only while it holds
// the lock; every other instance is read-only and says so (rulings A10, F7).
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
func New(researchDir, stateDir string) *Writer {
	return &Writer{researchDir: researchDir, stateDir: stateDir}
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
// the file; it writes nothing and needs no writer role (rulings P5).
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

// AppendVerdict appends one verdict to reviews/<id>/VERDICTS.md.
func (w *Writer) AppendVerdict(reviewID string, e home.Entry) (string, error) {
	path, title, err := w.verdictFile(reviewID, e)
	if err != nil {
		return "", err
	}
	return path, w.appendEntry(path, title, e)
}

// RenderVerdict returns the exact text AppendVerdict would append now, and
// the file; it writes nothing and needs no writer role (rulings P5).
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

// EnableSteering appends the Steering section to an older charter that does
// not carry the Steering sentence (rulings B4). It changes nothing when the
// sentence is already there and reports whether it wrote.
func (w *Writer) EnableSteering(commissionID string) (string, bool, error) {
	dir, err := w.folder("cohorts", commissionID)
	if err != nil {
		return "", false, err
	}
	path := filepath.Join(dir, home.CharterFile)
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.held(); err != nil {
		return path, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return path, false, fmt.Errorf("read the charter %s: %w", path, err)
	}
	if bytes.Contains(data, []byte(home.SteeringSentence)) {
		return path, false, nil
	}
	return path, true, appendBytes(path, separator(data)+"## Steering\n\n"+home.SteeringSentence+"\n")
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
	if err := home.CheckEntryText(body, w.researchDir); err != nil {
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
