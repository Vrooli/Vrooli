package destinationreadiness

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

const recoveryJournalVersion = "dbm-volume-recovery-v1"

// ErrRecoveryNotFound distinguishes an absent journal from storage or JSON
// corruption so API clients can show a useful retry/reference message.
var ErrRecoveryNotFound = errors.New("recovery journal not found")

// RecoveryState is the durable state of a multi-step volume recovery.
type RecoveryState string

const (
	RecoveryPending   RecoveryState = "pending"
	RecoveryRunning   RecoveryState = "running"
	RecoveryPaused    RecoveryState = "paused"
	RecoveryCompleted RecoveryState = "completed"
	RecoveryFailed    RecoveryState = "failed"
)

// RecoveryStep records one identity-bound action. A step is only marked
// completed after the control plane reports a satisfied end state and the
// journal has been durably saved.
type RecoveryStep struct {
	Plan     Plan   `json:"plan"`
	Status   string `json:"status"`
	Attempts int    `json:"attempts"`
	Detail   string `json:"detail,omitempty"`
}

// RecoveryJournal is the crash-resumable record for the ordered recovery
// sequence unmount → check → repair (when required) → mount-rw. It contains no
// credentials and never stores raw tool output beyond the bounded detail
// returned by the control plane.
type RecoveryJournal struct {
	Version  string `json:"version"`
	ID       string `json:"id"`
	Location string `json:"location"`
	// RelativePath is the volume-relative destination locator, when the
	// observed identity had a mountpoint. Location remains the requested mount
	// root used by the host remediation plan.
	RelativePath string         `json:"relative_path,omitempty"`
	Identity     DeviceIdentity `json:"identity"`
	Steps        []RecoveryStep `json:"steps"`
	Current      int            `json:"current"`
	State        RecoveryState  `json:"state"`
	LastError    string         `json:"last_error,omitempty"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// RecoveryStore is the persistence seam. Tests use an in-memory fake; the
// production implementation below uses an atomic, fsync-backed JSON file.
type RecoveryStore interface {
	Load(ctx context.Context, id string) (RecoveryJournal, error)
	Save(ctx context.Context, journal RecoveryJournal) error
}

// NewRecoveryJournal validates that all steps target the same physical volume
// and creates a pending journal. Only remediation actions are accepted.
func NewRecoveryJournal(location string, identity DeviceIdentity, plans []Plan, now time.Time) (RecoveryJournal, error) {
	if strings.TrimSpace(location) == "" || !filepath.IsAbs(location) {
		return RecoveryJournal{}, errors.New("recovery location must be absolute")
	}
	if !identity.StrongIdentity() || strings.TrimSpace(identity.DevicePath) == "" {
		return RecoveryJournal{}, errors.New("recovery requires a device path and UUID or serial")
	}
	if len(plans) == 0 {
		return RecoveryJournal{}, errors.New("recovery requires at least one step")
	}
	steps := make([]RecoveryStep, len(plans))
	for n, plan := range plans {
		if !plan.Action.IsRemediation() {
			return RecoveryJournal{}, fmt.Errorf("recovery step %d is not host remediation", n+1)
		}
		if !plan.Identity.StrongIdentity() || strings.TrimSpace(plan.Identity.DevicePath) == "" {
			return RecoveryJournal{}, fmt.Errorf("recovery step %d is missing a device path and UUID or serial", n+1)
		}
		if !plan.Identity.MatchesDevice(identity) {
			return RecoveryJournal{}, fmt.Errorf("recovery step %d targets a different device", n+1)
		}
		if filepath.Clean(strings.TrimSpace(plan.Location)) != filepath.Clean(location) {
			return RecoveryJournal{}, fmt.Errorf("recovery step %d targets a different mount location", n+1)
		}
		steps[n] = RecoveryStep{Plan: plan, Status: "pending"}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	relativePath := ""
	if identity.Mountpoint != "" {
		if derived, err := RelativePathUnderMount(location, identity.Mountpoint); err == nil {
			relativePath = derived
		}
	}
	return RecoveryJournal{Version: recoveryJournalVersion, ID: recoveryID(), Location: filepath.Clean(location), RelativePath: relativePath, Identity: identity, Steps: steps, State: RecoveryPending, UpdatedAt: now.UTC()}, nil
}

// ResumeRecovery continues at the first incomplete step. Every transition is
// saved before and after execution. A process crash before the result is
// recorded leaves an in-flight marker and pauses the sequence for inspection;
// it never blindly repeats an ambiguous repair. Dry-runs execute all
// validation but never advance the durable cursor.
func (s *Service) ResumeRecovery(ctx context.Context, store RecoveryStore, journal RecoveryJournal, confirmations map[int]string, acknowledgeDataLoss bool, dryRun bool) (RecoveryJournal, error) {
	if store == nil {
		return journal, errors.New("recovery store is required")
	}
	if err := validateJournal(journal); err != nil {
		return journal, err
	}
	if journal.State == RecoveryCompleted {
		return journal, nil
	}
	journal.State = RecoveryRunning
	journal.LastError = ""
	if err := store.Save(ctx, journal); err != nil {
		return journal, fmt.Errorf("persist recovery start: %w", err)
	}
	cursor := journal.Current
	for cursor < len(journal.Steps) {
		idx := cursor
		step := &journal.Steps[idx]
		if step.Status == "in_flight" {
			// The previous process may have completed the host action and
			// crashed before recording the result. Re-running a repair in that
			// ambiguous window could modify the filesystem twice. Pause and make
			// the operator re-inspect/re-plan explicitly.
			journal.State = RecoveryPaused
			journal.LastError = "previous recovery action may have completed; inspect the volume before retrying this step"
			if saveErr := store.Save(ctx, journal); saveErr != nil {
				return journal, fmt.Errorf("persist recovery ambiguity: %w", saveErr)
			}
			return journal, errors.New(journal.LastError)
		}
		step.Attempts++
		step.Status = "in_flight"
		if err := store.Save(ctx, journal); err != nil {
			return journal, fmt.Errorf("persist recovery attempt: %w", err)
		}
		result, err := s.ExecutePreparation(ctx, ExecuteInput{
			Plan:                step.Plan,
			Confirmation:        confirmations[idx],
			DryRun:              dryRun,
			AcknowledgeDataLoss: acknowledgeDataLoss && step.Plan.Destructive,
		})
		if err != nil {
			journal.State = RecoveryPaused
			journal.LastError = err.Error()
			step.Status = "blocked"
			step.Detail = err.Error()
			if saveErr := store.Save(ctx, journal); saveErr != nil {
				return journal, fmt.Errorf("recovery paused (%v), and pause could not be persisted: %w", err, saveErr)
			}
			return journal, err
		}
		step.Detail = result.Detail
		if dryRun {
			step.Status = "validated"
			cursor++
			continue
		}
		step.Status = "completed"
		journal.Current++
		cursor = journal.Current
		if journal.Current == len(journal.Steps) {
			journal.State = RecoveryCompleted
		}
		journal.UpdatedAt = time.Now().UTC()
		if err := store.Save(ctx, journal); err != nil {
			return journal, fmt.Errorf("persist recovery progress: %w", err)
		}
	}
	if dryRun {
		journal.State = RecoveryPending
		journal.UpdatedAt = time.Now().UTC()
		if err := store.Save(ctx, journal); err != nil {
			return journal, fmt.Errorf("persist recovery dry-run: %w", err)
		}
	}
	return journal, nil
}

func validateJournal(j RecoveryJournal) error {
	if j.Version != recoveryJournalVersion || strings.TrimSpace(j.ID) == "" {
		return errors.New("unsupported or incomplete recovery journal")
	}
	if strings.TrimSpace(j.Location) == "" || !filepath.IsAbs(j.Location) {
		return errors.New("recovery journal has an invalid location")
	}
	if j.RelativePath != "" {
		relative := filepath.Clean(strings.TrimSpace(j.RelativePath))
		if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return errors.New("recovery journal has a relative path that escapes its volume")
		}
	}
	if !j.Identity.StrongIdentity() || strings.TrimSpace(j.Identity.DevicePath) == "" {
		return errors.New("recovery journal has an incomplete device identity")
	}
	if j.Current < 0 || j.Current > len(j.Steps) || len(j.Steps) == 0 {
		return errors.New("invalid recovery cursor")
	}
	for _, step := range j.Steps {
		if !step.Plan.Action.IsRemediation() || !step.Plan.Identity.StrongIdentity() ||
			strings.TrimSpace(step.Plan.Identity.DevicePath) == "" ||
			!step.Plan.Identity.MatchesDevice(j.Identity) ||
			filepath.Clean(strings.TrimSpace(step.Plan.Location)) != filepath.Clean(j.Location) {
			return errors.New("recovery journal contains an identity-mismatched step")
		}
	}
	for index, step := range j.Steps {
		if index < j.Current && step.Status != "completed" {
			return errors.New("recovery journal cursor skips an incomplete step")
		}
		if index >= j.Current && step.Status == "completed" {
			return errors.New("recovery journal marks a future step completed")
		}
	}
	if j.State == RecoveryCompleted && j.Current != len(j.Steps) {
		return errors.New("completed recovery journal has an incomplete cursor")
	}
	return nil
}

func recoveryID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("recovery-%d", time.Now().UnixNano())
}

// FileRecoveryStore atomically persists one journal per recovery ID. It is
// intentionally process-local and conservative: writes are serialized while a
// journal is being replaced, and the final file is always mode 0600.
type FileRecoveryStore struct {
	Root string
	mu   sync.Mutex
}

func (s *FileRecoveryStore) path(id string) (string, error) {
	if s == nil || !filepath.IsAbs(s.Root) || strings.TrimSpace(id) == "" || strings.ContainsAny(id, `/\\`) {
		return "", errors.New("invalid recovery store or journal id")
	}
	return filepath.Join(filepath.Clean(s.Root), id+".json"), nil
}

func (s *FileRecoveryStore) Save(ctx context.Context, journal RecoveryJournal) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateJournal(journal); err != nil {
		return err
	}
	p, err := s.path(journal.ID)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(journal, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".recovery-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0600); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(tmpName, p); err != nil {
		return err
	}
	// Rename is atomic, but on POSIX a crash can still lose the directory
	// entry unless the parent directory is synced. Windows does not expose a
	// portable directory fsync, so the atomic replacement remains the strongest
	// available guarantee there.
	if err := syncRecoveryDirectory(filepath.Dir(p)); err != nil {
		return err
	}
	return nil
}

func syncRecoveryDirectory(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	f, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

func (s *FileRecoveryStore) Load(ctx context.Context, id string) (RecoveryJournal, error) {
	if err := ctx.Err(); err != nil {
		return RecoveryJournal{}, err
	}
	p, err := s.path(id)
	if err != nil {
		return RecoveryJournal{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return RecoveryJournal{}, ErrRecoveryNotFound
		}
		return RecoveryJournal{}, err
	}
	if len(data) > 1<<20 {
		return RecoveryJournal{}, errors.New("recovery journal exceeds size limit")
	}
	var journal RecoveryJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		return RecoveryJournal{}, err
	}
	if journal.ID != id {
		return RecoveryJournal{}, errors.New("recovery journal id does not match its filename")
	}
	if err := validateJournal(journal); err != nil {
		return RecoveryJournal{}, err
	}
	return journal, nil
}
