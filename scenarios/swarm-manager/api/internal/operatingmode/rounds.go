package operatingmode

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/pathredact"
	"swarm-manager/internal/storage"
)

type RoundStatus string

const (
	RoundStatusReserved       RoundStatus = "reserved"
	RoundStatusAgentRunning   RoundStatus = "agent_running"
	RoundStatusCompleted      RoundStatus = "completed"
	RoundStatusNeedsAttention RoundStatus = "needs_attention"
	RoundStatusFailed         RoundStatus = "failed"
	RoundStatusCanceled       RoundStatus = "canceled"
)

type ArtifactUpdate struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type,omitempty"`
	Required    bool   `json:"required,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	Source      string `json:"source,omitempty"`
}

type Handoff struct {
	Summary         string   `json:"summary,omitempty"`
	CompletedPhases []string `json:"completed_phases,omitempty"`
	ChangedFiles    []string `json:"changed_files,omitempty"`
	Tests           []string `json:"tests,omitempty"`
	Blockers        []string `json:"blockers,omitempty"`
	NextStep        string   `json:"next_step,omitempty"`
	// Frontier is the elastic-slice contract's declared true frontier: the one
	// comprehensively-completable unit the next round should advance (a whole
	// phase or the remainder of a sliced one). It is the continuity signal a
	// fresh agent reads to continue from the right place.
	Frontier  string `json:"frontier,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type PromptRenderTrace struct {
	SkillID             string         `json:"skill_id"`
	SourceRevision      string         `json:"source_revision"`
	SourceVariant       string         `json:"source_variant,omitempty"`
	SourceHash          string         `json:"source_hash"`
	VariablesHash       string         `json:"variables_hash"`
	RenderedPromptHash  string         `json:"rendered_prompt_hash"`
	DefinitionDigest    string         `json:"definition_digest"`
	InputContractDigest string         `json:"input_contract_digest"`
	RedactionMetadata   map[string]any `json:"redaction_metadata,omitempty"`
}

type RoundEnvelope struct {
	ExecutionID      string             `json:"execution_id,omitempty"`
	DefinitionDigest string             `json:"definition_digest,omitempty"`
	Round            int                `json:"round"`
	Mode             string             `json:"mode"`
	ScopeKind        string             `json:"scope_kind"`
	ScopeID          string             `json:"scope_id"`
	InitiativeName   string             `json:"initiative_name,omitempty"`
	Phase            string             `json:"phase"`
	RunStrategy      string             `json:"run_strategy"`
	AgentProfileKey  string             `json:"agent_profile_key"`
	GeneratedAt      string             `json:"generated_at"`
	RunID            string             `json:"run_id,omitempty"`
	Status           RoundStatus        `json:"status"`
	Readiness        *ReadinessReport   `json:"readiness,omitempty"`
	Items            []RoundItem        `json:"items,omitempty"`
	ArtifactUpdates  []ArtifactUpdate   `json:"artifact_updates,omitempty"`
	Handoffs         []Handoff          `json:"handoffs,omitempty"`
	PromptTrace      *PromptRenderTrace `json:"prompt_trace,omitempty"`
	Payload          map[string]any     `json:"payload,omitempty"`
	Error            string             `json:"error,omitempty"`
}

type RoundItem struct {
	Ref      string `json:"ref"`
	Title    string `json:"title,omitempty"`
	Status   string `json:"status,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Effort   string `json:"effort,omitempty"`
}

var (
	ErrRoundNotFound       = errors.New("operating mode round not found")
	ErrRoundAmbiguous      = errors.New("operating mode round is ambiguous")
	ErrRoundPreflightStale = errors.New("operating mode round preflight is stale")
	roundFileRE            = regexp.MustCompile(`^round-(\d{3})\.json$`)
)

// Store persists mode rounds and artifacts under a per-scope directory. It is
// keyed by the round's scope id (the target instance id — the initiative name
// for initiative targets, the plan id/path for plan targets). InitDir maps an
// initiative-target scope id to its initiative folder; TargetDir maps a
// non-initiative target (kind + instance id) to its own root, so a plan-first
// run never creates an initiative directory. The mode's declared target kind
// selects the resolver.
type Store struct {
	InitDir   func(scopeID string) string
	TargetDir func(kind TargetKind, scopeID string) string
	Clock     func() time.Time
	// ExecutionID is injectable for deterministic execution-layout tests.
	ExecutionID func() string
	mu          sync.Mutex
}

func NewStore(initDir func(string) string) *Store {
	return &Store{InitDir: initDir}
}

// scopeDir resolves the root directory for a scope id under the given mode's
// declared target kind.
func (s *Store) scopeDir(scopeID string, def Definition) (string, error) {
	if def.Target.Kind == TargetInitiative || def.Target.Kind == "" {
		return s.initDir(scopeID), nil
	}
	if s.TargetDir == nil {
		return "", fmt.Errorf("mode %q targets %s but the round store has no target directory resolver", def.Mode, def.Target.Kind)
	}
	return s.TargetDir(def.Target.Kind, scopeID), nil
}

func (s *Store) ModeDir(initiativeName string, mode Mode) (string, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return "", err
	}
	if def.Artifact.Root == "" {
		return "", fmt.Errorf("mode %q has no artifact root", mode)
	}
	root, err := s.scopeDir(initiativeName, def)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(def.Artifact.Root)), nil
}

func (s *Store) RoundsDir(initiativeName string, mode Mode) (string, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return "", err
	}
	if def.Artifact.RoundRoot == "" {
		return "", fmt.Errorf("mode %q has no round root", mode)
	}
	root, err := s.scopeDir(initiativeName, def)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(def.Artifact.RoundRoot)), nil
}

// TargetScopeDir is the default non-initiative target directory layout:
// <dataRoot>/mode-targets/<kind>/<sanitized-instance-id>. Wired as the round
// store's TargetDir and the lock's plan-key directory resolver so a plan-first
// run stores rounds and holds its exclusive lock without touching any
// initiative folder.
func TargetScopeDir(dataRoot string, kind TargetKind, scopeID string) string {
	return filepath.Join(dataRoot, "mode-targets", string(kind), sanitizeOwnershipToken(scopeID))
}

func (s *Store) RoundPath(initiativeName string, mode Mode, number int) (string, error) {
	if number <= 0 {
		return "", fmt.Errorf("round number must be >= 1, got %d", number)
	}
	dir, err := s.RoundsDir(initiativeName, mode)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("round-%03d.json", number)), nil
}

func (s *Store) ExecutionRoundsDir(execution OperatingModeExecution) (string, error) {
	def, err := execution.DefinitionBundle.RootDefinition()
	if err != nil {
		return "", err
	}
	manifest, err := s.executionManifestPath(execution, def)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(manifest), "rounds"), nil
}

func (s *Store) ExecutionRoundPath(execution OperatingModeExecution, number int) (string, error) {
	if number <= 0 {
		return "", fmt.Errorf("round number must be >= 1, got %d", number)
	}
	dir, err := s.ExecutionRoundsDir(execution)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("round-%03d.json", number)), nil
}

func (s *Store) CreateRound(round RoundEnvelope) (RoundEnvelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.prepareRound(&round); err != nil {
		return RoundEnvelope{}, err
	}
	var dir string
	if round.ExecutionID != "" {
		execution, err := s.loadExecutionForRound(round)
		if err != nil {
			return RoundEnvelope{}, err
		}
		dir, err = s.ExecutionRoundsDir(execution)
		if err != nil {
			return RoundEnvelope{}, err
		}
	} else {
		var err error
		dir, err = s.RoundsDir(round.ScopeID, Mode(round.Mode))
		if err != nil {
			return RoundEnvelope{}, err
		}
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return RoundEnvelope{}, fmt.Errorf("create rounds dir: %w", err)
	}

	const maxAttempts = 50
	for i := 0; i < maxAttempts; i++ {
		n, err := s.nextRoundNumberForScope(round.ScopeID, Mode(round.Mode))
		if err != nil {
			return RoundEnvelope{}, err
		}
		round.Round = n
		path := filepath.Join(dir, fmt.Sprintf("round-%03d.json", n))
		if err := writeJSONExclusive(path, round); err == nil {
			return round, nil
		} else if os.IsExist(err) {
			continue
		} else {
			return RoundEnvelope{}, fmt.Errorf("create round: %w", err)
		}
	}
	return RoundEnvelope{}, fmt.Errorf("could not reserve operating-mode round after %d attempts", maxAttempts)
}

// PrepareRound is the read-only half of compare-and-reserve. It resolves the
// proposed round number and returns a digest of the immutable execution plus
// every current round. Callers may safely use the proposed envelope for
// dynamic provider resolution, prompt rendering, and spawn-request assembly.
func (s *Store) PrepareRound(round RoundEnvelope) (RoundEnvelope, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.prepareRound(&round); err != nil {
		return RoundEnvelope{}, "", err
	}
	rounds, version, err := s.roundReservationState(round)
	if err != nil {
		return RoundEnvelope{}, "", err
	}
	max := 0
	for _, existing := range rounds {
		if existing.Round > max {
			max = existing.Round
		}
	}
	round.Round = max + 1
	return round, version, nil
}

// CompareAndCreateRound is the sole mutation after phase preflight. It rejects
// any intervening execution/round change and never falls through to a later
// round number: a racing retry must recompute startability and prompt inputs.
func (s *Store) CompareAndCreateRound(round RoundEnvelope, expectedVersion string) (RoundEnvelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(expectedVersion) == "" {
		return RoundEnvelope{}, fmt.Errorf("round preflight version is required")
	}
	if err := s.prepareRound(&round); err != nil {
		return RoundEnvelope{}, err
	}
	if round.Round <= 0 {
		return RoundEnvelope{}, fmt.Errorf("preflight round number is required")
	}
	_, currentVersion, err := s.roundReservationState(round)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if currentVersion != expectedVersion {
		return RoundEnvelope{}, fmt.Errorf("%w: execution %q changed before round %d reservation", ErrRoundPreflightStale, round.ExecutionID, round.Round)
	}
	execution, err := s.loadExecutionForRound(round)
	if err != nil {
		return RoundEnvelope{}, err
	}
	dir, err := s.ExecutionRoundsDir(execution)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return RoundEnvelope{}, fmt.Errorf("create rounds dir: %w", err)
	}
	path := filepath.Join(dir, fmt.Sprintf("round-%03d.json", round.Round))
	if err := writeJSONExclusive(path, round); err != nil {
		if os.IsExist(err) {
			return RoundEnvelope{}, fmt.Errorf("%w: round %d was reserved concurrently", ErrRoundPreflightStale, round.Round)
		}
		return RoundEnvelope{}, fmt.Errorf("create round: %w", err)
	}
	return round, nil
}

func (s *Store) roundReservationState(round RoundEnvelope) ([]RoundEnvelope, string, error) {
	execution, err := s.loadExecutionForRound(round)
	if err != nil {
		return nil, "", err
	}
	rounds, err := s.ListRounds(round.ScopeID, Mode(round.Mode))
	if err != nil {
		return nil, "", err
	}
	state := struct {
		Execution OperatingModeExecution `json:"execution"`
		Rounds    []RoundEnvelope        `json:"rounds"`
	}{Execution: execution, Rounds: rounds}
	data, err := json.Marshal(state)
	if err != nil {
		return nil, "", fmt.Errorf("marshal round preflight state: %w", err)
	}
	digest, err := canonicalJSONDigest(data)
	if err != nil {
		return nil, "", fmt.Errorf("digest round preflight state: %w", err)
	}
	return rounds, digest, nil
}

func (s *Store) SaveRound(round RoundEnvelope) error {
	if err := s.prepareRound(&round); err != nil {
		return err
	}
	if round.Round <= 0 {
		return fmt.Errorf("round number must be >= 1, got %d", round.Round)
	}
	var path string
	var err error
	if round.ExecutionID != "" {
		execution, loadErr := s.loadExecutionForRound(round)
		if loadErr != nil {
			return loadErr
		}
		path, err = s.ExecutionRoundPath(execution, round.Round)
	} else {
		path, err = s.RoundPath(round.ScopeID, Mode(round.Mode), round.Round)
	}
	if err != nil {
		return err
	}
	value := any(round)
	if redacted, changed, err := pathredact.NewForArtifactPath(path).RedactJSONValue(round); err != nil {
		return fmt.Errorf("redact round: %w", err)
	} else if changed {
		value = redacted
	}
	return storage.WriteJSONAtomic(path, value)
}

func (s *Store) LoadRound(initiativeName string, mode Mode, number int) (RoundEnvelope, error) {
	paths, err := s.roundPaths(initiativeName, mode, number)
	if err != nil {
		return RoundEnvelope{}, err
	}
	if len(paths) == 0 {
		return RoundEnvelope{}, ErrRoundNotFound
	}
	if len(paths) > 1 {
		return RoundEnvelope{}, fmt.Errorf("%w: mode %q scope %q round %d has %d records", ErrRoundAmbiguous, mode, initiativeName, number, len(paths))
	}
	data, err := os.ReadFile(paths[0])
	if err != nil {
		return RoundEnvelope{}, fmt.Errorf("read round: %w", err)
	}
	return decodeRound(data)
}

func (s *Store) ListRounds(initiativeName string, mode Mode) ([]RoundEnvelope, error) {
	legacyDir, err := s.RoundsDir(initiativeName, mode)
	if err != nil {
		return nil, err
	}
	dirs := []string{legacyDir}
	executions, err := s.ListExecutions(initiativeName, mode)
	if err != nil {
		return nil, err
	}
	for _, execution := range executions {
		dir, err := s.ExecutionRoundsDir(execution)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, dir)
	}
	rounds := []RoundEnvelope{}
	for _, dir := range dirs {
		inDir, err := listRoundsInDir(dir)
		if err != nil {
			return nil, err
		}
		rounds = append(rounds, inDir...)
	}
	sort.Slice(rounds, func(i, j int) bool { return rounds[i].Round < rounds[j].Round })
	for i := 1; i < len(rounds); i++ {
		if rounds[i-1].Round == rounds[i].Round {
			return nil, fmt.Errorf("%w: mode %q scope %q round %d", ErrRoundAmbiguous, mode, initiativeName, rounds[i].Round)
		}
	}
	return rounds, nil
}

func listRoundsInDir(dir string) ([]RoundEnvelope, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read rounds dir: %w", err)
	}
	rounds := make([]RoundEnvelope, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || roundFileRE.FindStringSubmatch(entry.Name()) == nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		round, err := decodeRound(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		rounds = append(rounds, round)
	}
	return rounds, nil
}

func (s *Store) roundPaths(scopeID string, mode Mode, number int) ([]string, error) {
	paths := []string{}
	legacy, err := s.RoundPath(scopeID, mode, number)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(legacy); err == nil {
		paths = append(paths, legacy)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	executions, err := s.ListExecutions(scopeID, mode)
	if err != nil {
		return nil, err
	}
	for _, execution := range executions {
		path, err := s.ExecutionRoundPath(execution, number)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(path); err == nil {
			paths = append(paths, path)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return paths, nil
}

func (s *Store) nextRoundNumberForScope(scopeID string, mode Mode) (int, error) {
	rounds, err := s.ListRounds(scopeID, mode)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, round := range rounds {
		if round.Round > max {
			max = round.Round
		}
	}
	return max + 1, nil
}

func (s *Store) prepareRound(round *RoundEnvelope) error {
	if round == nil {
		return fmt.Errorf("round is required")
	}
	round.InitiativeName = strings.TrimSpace(round.InitiativeName)
	round.ScopeID = strings.TrimSpace(round.ScopeID)
	if round.ScopeID == "" {
		round.ScopeID = round.InitiativeName
	}
	if round.ScopeID == "" {
		return fmt.Errorf("scope_id is required")
	}
	var def Definition
	if strings.TrimSpace(round.ExecutionID) != "" {
		execution, err := s.loadExecutionForRound(*round)
		if err != nil {
			return err
		}
		def, err = execution.DefinitionBundle.RootDefinition()
		if err != nil {
			return err
		}
		if round.DefinitionDigest == "" {
			round.DefinitionDigest = execution.DefinitionDigest
		} else if round.DefinitionDigest != execution.DefinitionDigest {
			return fmt.Errorf("round definition digest %q does not match execution %q", round.DefinitionDigest, execution.DefinitionDigest)
		}
	} else {
		var err error
		def, err = DefinitionFor(Mode(round.Mode))
		if err != nil {
			return err
		}
	}
	phaseDef, err := def.PhaseDefinition(Phase(round.Phase))
	if err != nil {
		return err
	}
	round.Mode = string(def.Mode)
	round.ScopeKind = string(def.Target.Kind)
	round.Phase = string(phaseDef.Phase)
	round.RunStrategy = string(def.RunStrategy.Kind)
	if strings.TrimSpace(round.AgentProfileKey) == "" {
		round.AgentProfileKey = phaseDef.ProfileKey
	}
	if strings.TrimSpace(round.GeneratedAt) == "" {
		round.GeneratedAt = s.now().Format(time.RFC3339)
	}
	if round.Status == "" {
		round.Status = RoundStatusReserved
	}
	return nil
}

func (s *Store) loadExecutionForRound(round RoundEnvelope) (OperatingModeExecution, error) {
	return s.LoadExecution(round.ScopeID, Mode(round.Mode), round.ExecutionID)
}

func (s *Store) now() time.Time {
	if s.Clock != nil {
		return s.Clock().UTC()
	}
	return time.Now().UTC()
}

func (s *Store) initDir(initiativeName string) string {
	if s.InitDir == nil {
		return initiativeName
	}
	return s.InitDir(initiativeName)
}

func decodeRound(data []byte) (RoundEnvelope, error) {
	var round RoundEnvelope
	if err := json.Unmarshal(data, &round); err != nil {
		return RoundEnvelope{}, err
	}
	if round.Round <= 0 {
		return RoundEnvelope{}, fmt.Errorf("round number is required")
	}
	if strings.TrimSpace(round.Mode) == "" {
		return RoundEnvelope{}, fmt.Errorf("mode is required")
	}
	if strings.TrimSpace(round.Phase) == "" {
		return RoundEnvelope{}, fmt.Errorf("phase is required")
	}
	return round, nil
}

func writeJSONExclusive(path string, data any) error {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			slog.Debug("operatingmode: close round file failed", "err", closeErr)
		}
	}()
	if _, err := f.Write(bytes); err != nil {
		return err
	}
	return f.Sync()
}
