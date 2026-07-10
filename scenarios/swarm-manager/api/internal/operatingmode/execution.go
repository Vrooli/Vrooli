package operatingmode

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"swarm-manager/internal/storage"
)

const executionManifestSchemaVersion = "operating-mode-execution/v1"

type ExecutionStatus string

const (
	ExecutionStatusActive         ExecutionStatus = "active"
	ExecutionStatusCompleted      ExecutionStatus = "completed"
	ExecutionStatusNeedsAttention ExecutionStatus = "needs_attention"
	ExecutionStatusCanceled       ExecutionStatus = "canceled"
)

var (
	ErrExecutionNotFound  = errors.New("operating mode execution not found")
	ErrExecutionAmbiguous = errors.New("operating mode execution is ambiguous")
	ErrRunOwnerNotFound   = errors.New("operating mode run owner not found")
	ErrRunOwnerAmbiguous  = errors.New("operating mode run owner is ambiguous")
)

// DefinitionBundle is the immutable parent + transitive delegated-mode graph
// used to interpret one execution. Definitions are deep-copied when pinned so
// registry reloads and in-process test mutations cannot rewrite history.
type DefinitionBundle struct {
	Root        Mode                `json:"root"`
	Definitions map[Mode]Definition `json:"definitions"`
}

func (b DefinitionBundle) Definition(mode Mode) (Definition, error) {
	def, ok := b.Definitions[NormalizeMode(string(mode))]
	if !ok {
		return Definition{}, fmt.Errorf("definition bundle has no mode %q", mode)
	}
	return clonePinnedDefinition(def)
}

func (b DefinitionBundle) RootDefinition() (Definition, error) {
	return b.Definition(b.Root)
}

// PinnedPromptSource is a Phase 4 provenance slot. Phase 3 persists the stable
// shape now so later prompt pinning does not require a manifest migration.
type PinnedPromptSource struct {
	Mode        string `json:"mode"`
	Phase       string `json:"phase"`
	SkillID     string `json:"skill_id"`
	Revision    string `json:"revision,omitempty"`
	Content     string `json:"content,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
	Retention   string `json:"retention,omitempty"`
	Redacted    bool   `json:"redacted,omitempty"`
}

type ExecutionMigrationProvenance struct {
	SourceLayout string `json:"source_layout"`
	MigratedAt   string `json:"migrated_at"`
	RoundCount   int    `json:"round_count"`
}

// OperatingModeExecution pins every interpretation input owned by Phase 3 and
// reserves explicit digest/snapshot slots for the Phase 4 input contract.
type OperatingModeExecution struct {
	ExecutionID string          `json:"execution_id"`
	ScopeKind   string          `json:"scope_kind"`
	ScopeID     string          `json:"scope_id"`
	Mode        string          `json:"mode"`
	Status      ExecutionStatus `json:"status"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	CompletedAt string          `json:"completed_at,omitempty"`

	SchemaVersion          string                        `json:"schema_version"`
	DefinitionDigest       string                        `json:"definition_digest"`
	DefinitionBundle       DefinitionBundle              `json:"definition_bundle"`
	InputContractDigest    string                        `json:"input_contract_digest,omitempty"`
	CompiledInputContract  json.RawMessage               `json:"compiled_input_contract,omitempty"`
	InputSnapshotDigest    string                        `json:"input_snapshot_digest,omitempty"`
	ValidatedInputSnapshot json.RawMessage               `json:"validated_input_snapshot,omitempty"`
	InputRetentionMetadata map[string]any                `json:"input_retention_metadata,omitempty"`
	PromptPolicyMetadata   map[string]any                `json:"prompt_policy_metadata,omitempty"`
	ReachablePromptSources map[string]PinnedPromptSource `json:"reachable_prompt_sources,omitempty"`
	Migration              *ExecutionMigrationProvenance `json:"migration,omitempty"`
}

func (e OperatingModeExecution) Terminal() bool {
	return e.Status == ExecutionStatusCompleted || e.Status == ExecutionStatusCanceled
}

type RunOwner struct {
	ExecutionID string `json:"execution_id"`
	Round       int    `json:"round"`
}

type runOwnerIndex struct {
	Owners map[string]RunOwner `json:"owners"`
}

func pinDefinitionBundle(root Definition, resolve func(Mode) (Definition, error)) (DefinitionBundle, string, error) {
	if resolve == nil {
		resolve = DefinitionFor
	}
	bundle := DefinitionBundle{Root: root.Mode, Definitions: map[Mode]Definition{}}
	visiting := map[Mode]bool{}
	var visit func(Definition) error
	visit = func(def Definition) error {
		mode := NormalizeMode(string(def.Mode))
		if visiting[mode] {
			return fmt.Errorf("delegated definition cycle reaches %q", mode)
		}
		if _, ok := bundle.Definitions[mode]; ok {
			return nil
		}
		visiting[mode] = true
		pinned, err := clonePinnedDefinition(def)
		if err != nil {
			return err
		}
		pinned.ExampleRuns = nil
		bundle.Definitions[mode] = pinned
		for _, phase := range pinned.PhaseGraph.Phases {
			if !phase.Delegated() {
				continue
			}
			child, err := resolve(phase.ExecutedBy)
			if err != nil {
				return fmt.Errorf("pin delegated mode %q: %w", phase.ExecutedBy, err)
			}
			if err := visit(child); err != nil {
				return err
			}
		}
		visiting[mode] = false
		return nil
	}
	if err := visit(root); err != nil {
		return DefinitionBundle{}, "", err
	}
	data, err := json.Marshal(bundle)
	if err != nil {
		return DefinitionBundle{}, "", fmt.Errorf("marshal definition bundle: %w", err)
	}
	digest := sha256.Sum256(data)
	return bundle, fmt.Sprintf("sha256:%x", digest[:]), nil
}

func clonePinnedDefinition(def Definition) (Definition, error) {
	data, err := json.Marshal(def)
	if err != nil {
		return Definition{}, fmt.Errorf("marshal definition %q: %w", def.Mode, err)
	}
	var clone Definition
	if err := json.Unmarshal(data, &clone); err != nil {
		return Definition{}, fmt.Errorf("clone definition %q: %w", def.Mode, err)
	}
	return clone, nil
}

func (s *Store) CreateExecution(scopeID string, def Definition) (OperatingModeExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createExecutionLocked(scopeID, def, nil)
}

// ContinueOrCreateExecution returns the one resumable execution for the target,
// or creates a new manifest when every prior execution is terminal. Multiple
// resumable manifests are never resolved by precedence: they are ambiguous
// state that requires repair.
func (s *Store) ContinueOrCreateExecution(scopeID string, def Definition) (OperatingModeExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	executions, err := s.ListExecutions(scopeID, def.Mode)
	if err != nil {
		return OperatingModeExecution{}, err
	}
	var resumable []OperatingModeExecution
	for _, execution := range executions {
		if !execution.Terminal() {
			resumable = append(resumable, execution)
		}
	}
	switch len(resumable) {
	case 0:
		return s.createExecutionLocked(scopeID, def, nil)
	case 1:
		return resumable[0], nil
	default:
		return OperatingModeExecution{}, fmt.Errorf("%w: mode %q scope %q has %d resumable manifests", ErrExecutionAmbiguous, def.Mode, scopeID, len(resumable))
	}
}

func (s *Store) createExecutionLocked(scopeID string, def Definition, migration *ExecutionMigrationProvenance) (OperatingModeExecution, error) {
	scopeID = strings.TrimSpace(scopeID)
	if scopeID == "" {
		return OperatingModeExecution{}, fmt.Errorf("scope_id is required")
	}
	bundle, digest, err := pinDefinitionBundle(def, DefinitionFor)
	if err != nil {
		return OperatingModeExecution{}, err
	}
	now := s.now().Format(time.RFC3339Nano)
	executionID := uuid.NewString()
	if s.ExecutionID != nil {
		executionID = strings.TrimSpace(s.ExecutionID())
	}
	if executionID == "" {
		return OperatingModeExecution{}, fmt.Errorf("execution id generator returned an empty id")
	}
	execution := OperatingModeExecution{
		ExecutionID: executionID,
		ScopeKind:   string(def.Target.Kind), ScopeID: scopeID, Mode: string(def.Mode),
		Status: ExecutionStatusActive, CreatedAt: now, UpdatedAt: now,
		SchemaVersion:    executionManifestSchemaVersion,
		DefinitionDigest: digest, DefinitionBundle: bundle, Migration: migration,
	}
	path, err := s.executionManifestPath(execution, def)
	if err != nil {
		return OperatingModeExecution{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return OperatingModeExecution{}, fmt.Errorf("create execution directory: %w", err)
	}
	if err := writeJSONExclusive(path, execution); err != nil {
		return OperatingModeExecution{}, fmt.Errorf("create execution manifest: %w", err)
	}
	return execution, nil
}

func (s *Store) SaveExecution(execution OperatingModeExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	def, err := execution.DefinitionBundle.RootDefinition()
	if err != nil {
		return err
	}
	execution.UpdatedAt = s.now().Format(time.RFC3339Nano)
	if execution.Terminal() && execution.CompletedAt == "" {
		execution.CompletedAt = execution.UpdatedAt
	}
	path, err := s.executionManifestPath(execution, def)
	if err != nil {
		return err
	}
	return storage.WriteJSONAtomic(path, execution)
}

func (s *Store) LoadExecution(scopeID string, mode Mode, executionID string) (OperatingModeExecution, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return OperatingModeExecution{}, err
	}
	path, err := s.executionManifestPath(OperatingModeExecution{
		ExecutionID: strings.TrimSpace(executionID), ScopeID: strings.TrimSpace(scopeID), Mode: string(def.Mode), ScopeKind: string(def.Target.Kind),
	}, def)
	if err != nil {
		return OperatingModeExecution{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return OperatingModeExecution{}, ErrExecutionNotFound
		}
		return OperatingModeExecution{}, fmt.Errorf("read execution manifest: %w", err)
	}
	var execution OperatingModeExecution
	if err := json.Unmarshal(data, &execution); err != nil {
		return OperatingModeExecution{}, fmt.Errorf("parse execution manifest: %w", err)
	}
	if err := validateExecutionManifest(execution); err != nil {
		return OperatingModeExecution{}, err
	}
	return execution, nil
}

func (s *Store) ListExecutions(scopeID string, mode Mode) ([]OperatingModeExecution, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return nil, err
	}
	root, err := s.executionsDir(scopeID, def)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read executions directory: %w", err)
	}
	executions := make([]OperatingModeExecution, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		execution, err := s.LoadExecution(scopeID, mode, entry.Name())
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}
	sort.Slice(executions, func(i, j int) bool {
		if executions[i].CreatedAt == executions[j].CreatedAt {
			return executions[i].ExecutionID < executions[j].ExecutionID
		}
		return executions[i].CreatedAt < executions[j].CreatedAt
	})
	return executions, nil
}

func (s *Store) IndexRunOwner(execution OperatingModeExecution, runID string, round int) error {
	runID = strings.TrimSpace(runID)
	if runID == "" || round <= 0 {
		return fmt.Errorf("run id and positive round are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	def, err := execution.DefinitionBundle.RootDefinition()
	if err != nil {
		return err
	}
	path, err := s.runOwnerIndexPath(execution.ScopeID, def)
	if err != nil {
		return err
	}
	index, err := loadRunOwnerIndex(path)
	if err != nil {
		return err
	}
	want := RunOwner{ExecutionID: execution.ExecutionID, Round: round}
	if existing, ok := index.Owners[runID]; ok && existing != want {
		return fmt.Errorf("%w: run %q maps to both %+v and %+v", ErrRunOwnerAmbiguous, runID, existing, want)
	}
	index.Owners[runID] = want
	return storage.WriteJSONAtomic(path, index)
}

func (s *Store) ResolveRunOwner(scopeID string, mode Mode, runID string) (RunOwner, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return RunOwner{}, err
	}
	path, err := s.runOwnerIndexPath(strings.TrimSpace(scopeID), def)
	if err != nil {
		return RunOwner{}, err
	}
	index, err := loadRunOwnerIndex(path)
	if err != nil {
		return RunOwner{}, err
	}
	owner, ok := index.Owners[strings.TrimSpace(runID)]
	if !ok {
		return RunOwner{}, ErrRunOwnerNotFound
	}
	return owner, nil
}

func (s *Service) definitionBundleForRound(round RoundEnvelope) (DefinitionBundle, Definition, error) {
	if strings.TrimSpace(round.ExecutionID) == "" {
		def, err := DefinitionFor(Mode(round.Mode))
		if err != nil {
			return DefinitionBundle{}, Definition{}, err
		}
		bundle, _, err := pinDefinitionBundle(def, DefinitionFor)
		return bundle, def, err
	}
	execution, err := s.store.loadExecutionForRound(round)
	if err != nil {
		return DefinitionBundle{}, Definition{}, err
	}
	def, err := execution.DefinitionBundle.RootDefinition()
	if err != nil {
		return DefinitionBundle{}, Definition{}, err
	}
	if round.DefinitionDigest != execution.DefinitionDigest {
		return DefinitionBundle{}, Definition{}, fmt.Errorf("round definition digest %q does not match execution %q", round.DefinitionDigest, execution.DefinitionDigest)
	}
	return execution.DefinitionBundle, def, nil
}

func (s *Service) syncExecutionStatus(round RoundEnvelope) error {
	if strings.TrimSpace(round.ExecutionID) == "" {
		return nil
	}
	execution, err := s.store.loadExecutionForRound(round)
	if err != nil {
		return err
	}
	def, err := execution.DefinitionBundle.RootDefinition()
	if err != nil {
		return err
	}
	switch round.Status {
	case RoundStatusCanceled:
		execution.Status = ExecutionStatusCanceled
	case RoundStatusFailed, RoundStatusNeedsAttention:
		execution.Status = ExecutionStatusNeedsAttention
	case RoundStatusCompleted:
		if len(nextPhasesForCompletedRoundWithResolver(def, round, execution.DefinitionBundle.Definition)) == 0 {
			execution.Status = ExecutionStatusCompleted
		} else {
			execution.Status = ExecutionStatusActive
		}
	default:
		execution.Status = ExecutionStatusActive
	}
	return s.store.SaveExecution(execution)
}

func loadRunOwnerIndex(path string) (runOwnerIndex, error) {
	index := runOwnerIndex{Owners: map[string]RunOwner{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return index, nil
		}
		return runOwnerIndex{}, fmt.Errorf("read run-owner index: %w", err)
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return runOwnerIndex{}, fmt.Errorf("parse run-owner index: %w", err)
	}
	if index.Owners == nil {
		index.Owners = map[string]RunOwner{}
	}
	return index, nil
}

func validateExecutionManifest(execution OperatingModeExecution) error {
	if strings.TrimSpace(execution.ExecutionID) == "" || strings.TrimSpace(execution.ScopeID) == "" || strings.TrimSpace(execution.Mode) == "" {
		return fmt.Errorf("execution manifest requires execution_id, scope_id, and mode")
	}
	if execution.SchemaVersion != executionManifestSchemaVersion {
		return fmt.Errorf("unsupported execution manifest schema %q", execution.SchemaVersion)
	}
	if _, err := execution.DefinitionBundle.RootDefinition(); err != nil {
		return err
	}
	return nil
}

func (s *Store) executionManifestPath(execution OperatingModeExecution, def Definition) (string, error) {
	if strings.TrimSpace(execution.ExecutionID) == "" {
		return "", fmt.Errorf("execution_id is required")
	}
	root, err := s.executionsDir(execution.ScopeID, def)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, execution.ExecutionID, "manifest.json"), nil
}

func (s *Store) executionsDir(scopeID string, def Definition) (string, error) {
	root, err := s.scopeDir(scopeID, def)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(def.Artifact.Root), "executions"), nil
}

func (s *Store) runOwnerIndexPath(scopeID string, def Definition) (string, error) {
	root, err := s.scopeDir(scopeID, def)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(def.Artifact.Root), "run-owners.json"), nil
}
