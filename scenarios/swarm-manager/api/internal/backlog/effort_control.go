package backlog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/identity"
)

// ErrEffortPolicyUnavailable reports that the approved effort workspace policy
// could not be read at admission. It is a configuration gap, not a validation
// error in the caller's authority.
var ErrEffortPolicyUnavailable = errors.New("approved effort policy is unavailable")

// EffortControlStore is the durable owner persistence for the effort-control
// aggregate. It stores exactly one revision per effort identity and never
// derives authority from a second writable policy.
type EffortControlStore interface {
	Load(effortID string) (identity.EffortControl, error)
	Save(control identity.EffortControl) error
	List() ([]identity.EffortControl, error)
}

// FileEffortControlStore persists one JSON document per effort identity under
// a single directory. Writes are atomic (temp file + rename) so a crashed
// admission cannot leave a half-written revision in authoritative state.
type FileEffortControlStore struct {
	rootDir string
}

// NewFileEffortControlStore roots the store at the given directory.
func NewFileEffortControlStore(rootDir string) *FileEffortControlStore {
	return &FileEffortControlStore{rootDir: rootDir}
}

func (store *FileEffortControlStore) path(effortID string) (string, error) {
	effortID = strings.TrimSpace(effortID)
	if effortID == "" || effortID == "." || effortID == ".." ||
		strings.ContainsAny(effortID, `/\`) {
		return "", fmt.Errorf("effort_id must be a single path segment")
	}
	return filepath.Join(store.rootDir, effortID+".json"), nil
}

// Load returns the stored revision, or ErrNotFound when the effort has never
// been admitted.
func (store *FileEffortControlStore) Load(effortID string) (identity.EffortControl, error) {
	path, err := store.path(effortID)
	if err != nil {
		return identity.EffortControl{}, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return identity.EffortControl{}, ErrNotFound
		}
		return identity.EffortControl{}, err
	}
	var control identity.EffortControl
	if err := json.Unmarshal(raw, &control); err != nil {
		return identity.EffortControl{}, fmt.Errorf("decode effort control %q: %w", effortID, err)
	}
	return control, nil
}

// Save writes the revision atomically.
func (store *FileEffortControlStore) Save(control identity.EffortControl) error {
	path, err := store.path(control.EffortID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(control, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".effort-control-*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		os.Remove(tempName)
		return err
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempName)
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		os.Remove(tempName)
		return err
	}
	return nil
}

// List returns every admitted effort revision, sorted by effort identity so
// callers get deterministic output.
func (store *FileEffortControlStore) List() ([]identity.EffortControl, error) {
	entries, err := os.ReadDir(store.rootDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	controls := make([]identity.EffortControl, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		control, err := store.Load(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		controls = append(controls, control)
	}
	sort.Slice(controls, func(i, j int) bool { return controls[i].EffortID < controls[j].EffortID })
	return controls, nil
}

// EffortPolicySource resolves the approved policy revision and the immutable
// candidate policy for an effort. It is the seam that binds live owner policy
// at admission without creating a second writable policy authority.
type EffortPolicySource interface {
	ResolveEffortPolicy(effortID string) (identity.PolicyBinding, identity.CandidatePolicyBinding, error)
}

// FileEffortPolicySource reads the approved effort workspace `effort.json`.
// The policy binding digest covers only the reviewed policy object plus the
// effort slug, so lifecycle timestamps and handoff notes cannot masquerade as
// a policy change.
type FileEffortPolicySource struct {
	Root string
}

type effortPolicyDocument struct {
	Slug   string          `json:"slug"`
	Policy json.RawMessage `json:"policy"`
}

type effortPolicyModelSelection struct {
	ModelSelection struct {
		WorkerPreference struct {
			Runner         string `json:"runner"`
			Model          string `json:"model"`
			Effort         string `json:"effort"`
			FallbackRunner string `json:"fallback_runner"`
			FallbackModel  string `json:"fallback_model"`
		} `json:"worker_preference"`
	} `json:"model_selection"`
}

// ResolveEffortPolicy binds the policy digest and candidate policy from the
// live effort workspace revision.
func (source FileEffortPolicySource) ResolveEffortPolicy(effortID string) (identity.PolicyBinding, identity.CandidatePolicyBinding, error) {
	effortID = strings.TrimSpace(effortID)
	if effortID == "" || strings.ContainsAny(effortID, `/\`) {
		return identity.PolicyBinding{}, identity.CandidatePolicyBinding{}, fmt.Errorf("effort_id must be a single path segment")
	}
	path := filepath.Join(source.Root, effortID, "effort.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return identity.PolicyBinding{}, identity.CandidatePolicyBinding{}, fmt.Errorf("read approved effort policy: %w", err)
	}
	var document effortPolicyDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return identity.PolicyBinding{}, identity.CandidatePolicyBinding{}, fmt.Errorf("decode approved effort policy: %w", err)
	}
	if len(document.Policy) == 0 {
		return identity.PolicyBinding{}, identity.CandidatePolicyBinding{}, fmt.Errorf("approved effort policy is empty")
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, document.Policy); err != nil {
		return identity.PolicyBinding{}, identity.CandidatePolicyBinding{}, fmt.Errorf("canonicalize approved effort policy: %w", err)
	}
	sum := sha256.Sum256(append([]byte(strings.TrimSpace(document.Slug)+"\x00"), compacted.Bytes()...))
	binding := identity.PolicyBinding{Source: path, Digest: "sha256:" + hex.EncodeToString(sum[:])}

	var selection effortPolicyModelSelection
	if err := json.Unmarshal(document.Policy, &selection); err != nil {
		return identity.PolicyBinding{}, identity.CandidatePolicyBinding{}, fmt.Errorf("decode candidate policy: %w", err)
	}
	preference := selection.ModelSelection.WorkerPreference
	candidate := identity.CandidatePolicyBinding{
		EconomicalRunner: strings.TrimSpace(preference.Runner),
		EconomicalModel:  strings.TrimSpace(preference.Model),
		EconomicalEffort: strings.TrimSpace(preference.Effort),
		FallbackRunner:   strings.TrimSpace(preference.FallbackRunner),
		FallbackModel:    strings.TrimSpace(preference.FallbackModel),
	}.Bind()
	return binding, candidate, nil
}

// EffortControlService is the owner admission and amendment surface for the
// effort-control aggregate. Every mutation is version-checked against the
// durable prior revision before it is persisted.
type EffortControlService struct {
	store  EffortControlStore
	policy EffortPolicySource
}

// NewEffortControlService builds the owner service over durable persistence
// and an optional live policy source. A nil policy source preserves whatever
// binding the caller supplied, which tests and non-effort callers rely on.
func NewEffortControlService(store EffortControlStore, policy EffortPolicySource) *EffortControlService {
	return &EffortControlService{store: store, policy: policy}
}

// Admit creates the first revision of an effort. Admitting over an existing
// effort is a revision conflict: authority changes only through Amend.
func (service *EffortControlService) Admit(control identity.EffortControl) (identity.EffortControl, error) {
	if err := service.bindPolicy(&control); err != nil {
		return identity.EffortControl{}, err
	}
	control.Completion = identity.CompletionStanding{}
	current, err := service.store.Load(control.EffortID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return identity.EffortControl{}, err
	}
	var currentPtr *identity.EffortControl
	if err == nil {
		currentPtr = &current
	}
	admitted, err := control.Admit(currentPtr)
	if err != nil {
		return identity.EffortControl{}, err
	}
	if err := service.store.Save(admitted); err != nil {
		return identity.EffortControl{}, err
	}
	return admitted, nil
}

// Amend changes the reviewed authority at the exact next revision and carries
// the prior runtime completion standing forward.
func (service *EffortControlService) Amend(control identity.EffortControl) (identity.EffortControl, error) {
	if err := service.bindPolicy(&control); err != nil {
		return identity.EffortControl{}, err
	}
	current, err := service.store.Load(control.EffortID)
	if err != nil {
		return identity.EffortControl{}, err
	}
	control.Completion = current.Completion
	amended, err := control.Amend(current)
	if err != nil {
		return identity.EffortControl{}, err
	}
	if err := service.store.Save(amended); err != nil {
		return identity.EffortControl{}, err
	}
	return amended, nil
}

// Get returns the current admitted revision.
func (service *EffortControlService) Get(effortID string) (identity.EffortControl, error) {
	return service.store.Load(effortID)
}

// MarkEvidenceComplete records owner evidence completion without ever setting
// human product acceptance.
func (service *EffortControlService) MarkEvidenceComplete(effortID string, refs ...string) (identity.EffortControl, error) {
	control, err := service.store.Load(effortID)
	if err != nil {
		return identity.EffortControl{}, err
	}
	control.Completion = control.Completion.MarkEvidenceComplete(refs...)
	if err := service.store.Save(control); err != nil {
		return identity.EffortControl{}, err
	}
	return control, nil
}

// HumanAccept records an authenticated human product disposition. A blank
// actor is refused so evidence completion cannot substitute for a human.
func (service *EffortControlService) HumanAccept(effortID, actor string) (identity.EffortControl, error) {
	control, err := service.store.Load(effortID)
	if err != nil {
		return identity.EffortControl{}, err
	}
	updated, err := control.Completion.HumanAccept(actor)
	if err != nil {
		return identity.EffortControl{}, err
	}
	control.Completion = updated
	if err := service.store.Save(control); err != nil {
		return identity.EffortControl{}, err
	}
	return control, nil
}

// bindPolicy stamps the live approved policy and immutable candidate policy
// onto the revision before validation.
func (service *EffortControlService) bindPolicy(control *identity.EffortControl) error {
	if service.policy == nil {
		return nil
	}
	binding, candidate, err := service.policy.ResolveEffortPolicy(control.EffortID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrEffortPolicyUnavailable, err)
	}
	control.PolicyBinding = binding
	control.CandidatePolicy = candidate
	return nil
}
