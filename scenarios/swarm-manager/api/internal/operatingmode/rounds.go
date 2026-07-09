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
	"strconv"
	"strings"
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

type RoundEnvelope struct {
	Round           int              `json:"round"`
	Mode            string           `json:"mode"`
	ScopeKind       string           `json:"scope_kind"`
	ScopeID         string           `json:"scope_id"`
	InitiativeName  string           `json:"initiative_name,omitempty"`
	Phase           string           `json:"phase"`
	RunStrategy     string           `json:"run_strategy"`
	AgentProfileKey string           `json:"agent_profile_key"`
	GeneratedAt     string           `json:"generated_at"`
	RunID           string           `json:"run_id,omitempty"`
	Status          RoundStatus      `json:"status"`
	Readiness       *ReadinessReport `json:"readiness,omitempty"`
	Items           []RoundItem      `json:"items,omitempty"`
	ArtifactUpdates []ArtifactUpdate `json:"artifact_updates,omitempty"`
	Handoffs        []Handoff        `json:"handoffs,omitempty"`
	Payload         map[string]any   `json:"payload,omitempty"`
	Error           string           `json:"error,omitempty"`
}

type RoundItem struct {
	Ref      string `json:"ref"`
	Title    string `json:"title,omitempty"`
	Status   string `json:"status,omitempty"`
	Priority int    `json:"priority,omitempty"`
	Effort   string `json:"effort,omitempty"`
}

var (
	ErrRoundNotFound = errors.New("operating mode round not found")
	roundFileRE      = regexp.MustCompile(`^round-(\d{3})\.json$`)
)

type Store struct {
	InitDir func(initiativeName string) string
	Clock   func() time.Time
}

func NewStore(initDir func(string) string) *Store {
	return &Store{InitDir: initDir}
}

func (s *Store) ModeDir(initiativeName string, mode Mode) (string, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return "", err
	}
	if def.Artifact.Root == "" {
		return "", fmt.Errorf("mode %q has no artifact root", mode)
	}
	return filepath.Join(s.initDir(initiativeName), filepath.FromSlash(def.Artifact.Root)), nil
}

func (s *Store) RoundsDir(initiativeName string, mode Mode) (string, error) {
	def, err := DefinitionFor(mode)
	if err != nil {
		return "", err
	}
	if def.Artifact.RoundRoot == "" {
		return "", fmt.Errorf("mode %q has no round root", mode)
	}
	return filepath.Join(s.initDir(initiativeName), filepath.FromSlash(def.Artifact.RoundRoot)), nil
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

func (s *Store) CreateRound(round RoundEnvelope) (RoundEnvelope, error) {
	if err := s.prepareRound(&round); err != nil {
		return RoundEnvelope{}, err
	}
	dir, err := s.RoundsDir(round.InitiativeName, Mode(round.Mode))
	if err != nil {
		return RoundEnvelope{}, err
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return RoundEnvelope{}, fmt.Errorf("create rounds dir: %w", err)
	}

	const maxAttempts = 50
	for i := 0; i < maxAttempts; i++ {
		n, err := s.nextRoundNumber(dir)
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

func (s *Store) SaveRound(round RoundEnvelope) error {
	if err := s.prepareRound(&round); err != nil {
		return err
	}
	if round.Round <= 0 {
		return fmt.Errorf("round number must be >= 1, got %d", round.Round)
	}
	path, err := s.RoundPath(round.InitiativeName, Mode(round.Mode), round.Round)
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
	path, err := s.RoundPath(initiativeName, mode, number)
	if err != nil {
		return RoundEnvelope{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RoundEnvelope{}, ErrRoundNotFound
		}
		return RoundEnvelope{}, fmt.Errorf("read round: %w", err)
	}
	return decodeRound(data)
}

func (s *Store) ListRounds(initiativeName string, mode Mode) ([]RoundEnvelope, error) {
	dir, err := s.RoundsDir(initiativeName, mode)
	if err != nil {
		return nil, err
	}
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
	sort.Slice(rounds, func(i, j int) bool { return rounds[i].Round < rounds[j].Round })
	return rounds, nil
}

func (s *Store) nextRoundNumber(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, fmt.Errorf("read rounds dir: %w", err)
	}
	max := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		m := roundFileRE.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err == nil && n > max {
			max = n
		}
	}
	return max + 1, nil
}

func (s *Store) prepareRound(round *RoundEnvelope) error {
	if round == nil {
		return fmt.Errorf("round is required")
	}
	round.InitiativeName = strings.TrimSpace(round.InitiativeName)
	if round.InitiativeName == "" {
		return fmt.Errorf("initiative_name is required")
	}
	def, err := DefinitionFor(Mode(round.Mode))
	if err != nil {
		return err
	}
	phaseDef, err := def.PhaseDefinition(Phase(round.Phase))
	if err != nil {
		return err
	}
	round.Mode = string(def.Mode)
	round.ScopeKind = string(def.Scope.Kind)
	if strings.TrimSpace(round.ScopeID) == "" {
		round.ScopeID = round.InitiativeName
	}
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
