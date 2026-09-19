package cloudtarget

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/packages/cloudrelease"
)

// SchemaVersion is stamped on every receipt, fence and pointer file.
const SchemaVersion = 1

const (
	deploymentsRelPath   = "cloud/deployments"
	releasesDirName      = "releases"
	operationsDirName    = "operations"
	activeReleaseFile    = "active-release.json"
	activationIntentFile = "activation-intent.json"
	fenceFile            = "fence.json"
	incompleteMarker     = ".incomplete"
	releaseManifestFile  = "release-manifest.json"
	stagingInfix         = ".staging-"
	dirMode              = 0o755
	fileMode             = 0o644
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	digestPattern     = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// Outcome is the receipt verdict vocabulary.
type Outcome string

const (
	OutcomeSucceeded Outcome = "succeeded"
	OutcomeFailed    Outcome = "failed"
	OutcomeUnchanged Outcome = "unchanged"
)

// Receipt is the durable record of one (operation, step) on this target.
type Receipt struct {
	SchemaVersion int            `json:"schema_version"`
	DeploymentID  string         `json:"deployment_id"`
	OperationID   string         `json:"operation_id"`
	Step          string         `json:"step"`
	Fence         uint64         `json:"fence"`
	Verb          string         `json:"verb"`
	Outcome       Outcome        `json:"outcome"`
	InputDigest   string         `json:"input_digest"`
	StartedAt     string         `json:"started_at"`
	CompletedAt   string         `json:"completed_at"`
	Details       map[string]any `json:"details"`
	Error         *ReceiptError  `json:"error,omitempty"`
	// Replayed is not persisted: it marks a receipt returned from disk.
	Replayed bool `json:"replayed,omitempty"`
}

// ReceiptError is the persisted failure shape.
type ReceiptError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Fence is the highest fence this deployment has accepted.
type Fence struct {
	SchemaVersion int    `json:"schema_version"`
	Current       uint64 `json:"current"`
	OperationID   string `json:"operation_id"`
	UpdatedAt     string `json:"updated_at"`
}

// EffectRequest identifies one effectful step.
type EffectRequest struct {
	DeploymentID string
	OperationID  string
	Step         string
	Fence        uint64
	Verb         string
	// Input is digested into the receipt so a replay with different inputs is
	// refused instead of silently returning the old receipt.
	Input any
}

// EffectResult is what every effectful verb returns.
type EffectResult struct {
	Receipt  Receipt `json:"receipt"`
	Replayed bool    `json:"replayed"`
}

// Hooks are test seams. Fault is consulted at named points inside multi-step
// verbs so a crash between boundaries can be simulated deterministically.
type Hooks struct {
	Fault func(point string) error
}

// Store owns the deployment layout beneath one root directory.
type Store struct {
	Root  string
	Hooks Hooks
	Now   func() time.Time
}

// DefaultStore resolves the store beneath the operator runtime home.
func DefaultStore() (*Store, error) {
	home, err := config.VrooliHome()
	if err != nil {
		return nil, fail(CodeStoreIO, "resolve runtime home: %v", err)
	}
	return NewStore(filepath.Join(home, filepath.FromSlash(deploymentsRelPath))), nil
}

// NewStore returns a store rooted at root (the deployments directory).
func NewStore(root string) *Store {
	return &Store{Root: filepath.Clean(root), Now: time.Now}
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Store) fault(point string) error {
	if s.Hooks.Fault == nil {
		return nil
	}
	return s.Hooks.Fault(point)
}

func validIdentifier(kind, value string) error {
	if !identifierPattern.MatchString(value) {
		return refuse(CodeInvalidArgument, "%s %q is not a valid identifier", kind, value)
	}
	return nil
}

func validDigest(value string) error {
	if !digestPattern.MatchString(value) {
		return refuse(CodeInvalidArgument, "release digest %q is not a lowercase sha256 hex string", value)
	}
	return nil
}

// DeploymentDir returns the deployment directory after validating its id.
func (s *Store) DeploymentDir(deploymentID string) (string, error) {
	if err := validIdentifier("deployment id", deploymentID); err != nil {
		return "", err
	}
	return filepath.Join(s.Root, deploymentID), nil
}

func (s *Store) releasesDir(deploymentID string) (string, error) {
	dir, err := s.DeploymentDir(deploymentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, releasesDirName), nil
}

// ReleaseDir returns the complete-release directory for a digest.
func (s *Store) ReleaseDir(deploymentID, digest string) (string, error) {
	if err := validDigest(digest); err != nil {
		return "", err
	}
	releases, err := s.releasesDir(deploymentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(releases, digest), nil
}

func (s *Store) stagingDir(deploymentID, digest, operationID string) (string, error) {
	releaseDir, err := s.ReleaseDir(deploymentID, digest)
	if err != nil {
		return "", err
	}
	return releaseDir + stagingInfix + operationID, nil
}

func (s *Store) receiptPath(deploymentID, operationID, step string) (string, error) {
	dir, err := s.DeploymentDir(deploymentID)
	if err != nil {
		return "", err
	}
	if err := validIdentifier("operation id", operationID); err != nil {
		return "", err
	}
	if err := validIdentifier("step", step); err != nil {
		return "", err
	}
	return filepath.Join(dir, operationsDirName, operationID, step+".json"), nil
}

func (s *Store) fencePath(deploymentID string) (string, error) {
	dir, err := s.DeploymentDir(deploymentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fenceFile), nil
}

func (s *Store) activePath(deploymentID string) (string, error) {
	dir, err := s.DeploymentDir(deploymentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, activeReleaseFile), nil
}

func (s *Store) intentPath(deploymentID string) (string, error) {
	dir, err := s.DeploymentDir(deploymentID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, activationIntentFile), nil
}

// ReadFence returns the current fence, zero when none has been accepted.
func (s *Store) ReadFence(deploymentID string) (Fence, error) {
	path, err := s.fencePath(deploymentID)
	if err != nil {
		return Fence{}, err
	}
	var fence Fence
	if err := readJSON(path, &fence); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Fence{SchemaVersion: SchemaVersion}, nil
		}
		return Fence{}, fail(CodeStoreIO, "read fence: %v", err)
	}
	return fence, nil
}

// ReadReceipt returns one receipt, or a receipt_not_found refusal.
func (s *Store) ReadReceipt(deploymentID, operationID, step string) (Receipt, error) {
	path, err := s.receiptPath(deploymentID, operationID, step)
	if err != nil {
		return Receipt{}, err
	}
	var receipt Receipt
	if err := readJSON(path, &receipt); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Receipt{}, refuse(CodeReceiptNotFound, "no receipt for operation %s step %s", operationID, step)
		}
		return Receipt{}, fail(CodeStoreIO, "read receipt: %v", err)
	}
	return receipt, nil
}

// RunEffect enforces the fence, replays an existing receipt, and otherwise
// runs the verb body and persists its receipt. The body's error (if any) is
// returned alongside the failed receipt so callers can exit with its code.
func (s *Store) RunEffect(ctx context.Context, req EffectRequest, body func(ctx context.Context) (map[string]any, Outcome, error)) (EffectResult, error) {
	receiptPath, err := s.receiptPath(req.DeploymentID, req.OperationID, req.Step)
	if err != nil {
		return EffectResult{}, err
	}
	if strings.TrimSpace(req.Verb) == "" {
		return EffectResult{}, refuse(CodeInvalidArgument, "verb is required")
	}
	inputDigest, err := CanonicalDigest(req.Input)
	if err != nil {
		return EffectResult{}, fail(CodeInvalidArgument, "digest verb input: %v", err)
	}
	fence, err := s.ReadFence(req.DeploymentID)
	if err != nil {
		return EffectResult{}, err
	}
	if req.Fence < fence.Current {
		return EffectResult{}, refuse(CodeFenceStale, "fence %d is below the accepted fence %d", req.Fence, fence.Current).
			withDetails(map[string]any{"fence": req.Fence, "current": fence.Current, "current_operation_id": fence.OperationID})
	}
	var existing Receipt
	switch err := readJSON(receiptPath, &existing); {
	case err == nil:
		if existing.InputDigest != inputDigest {
			return EffectResult{}, refuse(CodeReceiptInputMismatch, "operation %s step %s already ran with different inputs", req.OperationID, req.Step).
				withDetails(map[string]any{"recorded_input_digest": existing.InputDigest, "input_digest": inputDigest})
		}
		existing.Replayed = true
		return EffectResult{Receipt: existing, Replayed: true}, nil
	case errors.Is(err, os.ErrNotExist):
	default:
		return EffectResult{}, fail(CodeStoreIO, "read receipt: %v", err)
	}
	fencePath, _ := s.fencePath(req.DeploymentID)
	if req.Fence > fence.Current || fence.UpdatedAt == "" {
		if err := writeJSONAtomic(fencePath, Fence{SchemaVersion: SchemaVersion, Current: req.Fence, OperationID: req.OperationID, UpdatedAt: s.now().Format(time.RFC3339Nano)}); err != nil {
			return EffectResult{}, fail(CodeStoreIO, "record fence: %v", err)
		}
	}
	receipt := Receipt{
		SchemaVersion: SchemaVersion, DeploymentID: req.DeploymentID, OperationID: req.OperationID, Step: req.Step,
		Fence: req.Fence, Verb: req.Verb, InputDigest: inputDigest, StartedAt: s.now().Format(time.RFC3339Nano), Details: map[string]any{},
	}
	details, outcome, bodyErr := body(ctx)
	if details != nil {
		receipt.Details = details
	}
	receipt.CompletedAt = s.now().Format(time.RFC3339Nano)
	if bodyErr != nil {
		typed := AsError(bodyErr)
		receipt.Outcome = OutcomeFailed
		receipt.Error = &ReceiptError{Code: typed.Code, Message: typed.Message}
		if typed.Details != nil {
			for key, value := range typed.Details {
				receipt.Details[key] = value
			}
		}
	} else {
		receipt.Outcome = outcome
		if receipt.Outcome == "" {
			receipt.Outcome = OutcomeSucceeded
		}
	}
	if err := writeJSONAtomic(receiptPath, receipt); err != nil {
		return EffectResult{}, fail(CodeStoreIO, "record receipt: %v", err)
	}
	return EffectResult{Receipt: receipt}, bodyErr
}

// CanonicalDigest returns the sha256 hex of the canonical JSON encoding of
// value. The canonical form is owned by packages/cloudrelease (sorted-key
// compact JSON) so receipts, fences and release identities share one rule.
func CanonicalDigest(value any) (string, error) {
	return cloudrelease.CanonicalDigest(value)
}

// CanonicalJSON encodes value as sorted-key compact JSON.
func CanonicalJSON(value any) ([]byte, error) {
	return cloudrelease.CanonicalJSON(value)
}

func readJSON(path string, into any) error {
	raw, err := os.ReadFile(path) //nolint:gosec // paths are built from validated identifiers beneath the store root
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, into)
}

// writeJSONAtomic writes value to path through a same-directory temp file and
// a rename, so a reader never observes a torn file.
func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp-" + randomSuffix()
	if err := os.WriteFile(tmp, append(raw, '\n'), fileMode); err != nil {
		return err
	}
	if err := platform.AtomicReplace(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func randomSuffix() string {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // caller-provided artifact path
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := copyAll(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
