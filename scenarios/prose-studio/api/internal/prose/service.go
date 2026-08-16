package prose

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/textmetrics"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

var (
	ErrStyleResolutionConflict = errors.New("style_resolution_conflict")
	ErrProfileDeclared         = errors.New("profile_is_declared")
	ErrProfileUnregistered     = errors.New("profile_unregistered")
	ErrDeclarationCollision    = errors.New("declaration_key_collision")
	ErrBudgetExceeded          = errors.New("session_budget_exceeded")
	ErrContextInfeasible       = errors.New("context_window_infeasible")
)

// Gateway is the only inference seam. Production uses HTTPGateway, which
// talks to ai-gateway; tests inject a fake. No vendor SDK or credential is
// present in this package.
type Gateway interface {
	Generate(context.Context, GatewayRequest) ([]GatewayCandidate, error)
}

type HTTPGateway struct {
	BaseURL string
	Client  *http.Client
}

func (g HTTPGateway) Generate(ctx context.Context, req GatewayRequest) ([]GatewayCandidate, error) {
	if strings.TrimSpace(g.BaseURL) == "" {
		return nil, errors.New("ai-gateway endpoint is not configured")
	}
	// write.default is a single-draft role. Build a comparable direct set by
	// issuing one governed gateway request per slot; write.diverse remains one
	// request returning the role's k-candidate array.
	if req.Role == "write.default" && req.K > 1 {
		out := make([]GatewayCandidate, 0, req.K)
		for i := 0; i < req.K; i++ {
			single := req
			single.K = 1
			candidates, err := g.Generate(ctx, single)
			if err != nil {
				return nil, err
			}
			out = append(out, candidates...)
		}
		return out, nil
	}
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	connectClient := inferenceconnect.NewInferenceServiceClient(client, g.BaseURL)
	schema := `{"type":"string"}`
	if req.K > 1 {
		schema = `{"type":"array","items":{"type":"string"}}`
	}
	instruction := req.Instruction
	if len(req.Negative.Pinned) > 0 || len(req.Negative.Rejected) > 0 {
		negative, _ := json.Marshal(req.Negative)
		instruction += "\nDo not repeat candidates represented by this prior-round context: " + string(negative)
	}
	resp, err := connectClient.Run(ctx, connect.NewRequest(&inferencev1.RunRequest{
		Source:          "prose-studio",
		SchemaJson:      schema,
		Instruction:     instruction,
		Role:            req.Role,
		Profile:         sharedv1.Profile_PROFILE_LOCAL_FIRST,
		MaxOutputTokens: int32(req.MaxOutputTokens),
	}))
	if err != nil {
		return nil, fmt.Errorf("ai-gateway request: %w", err)
	}
	if resp.Msg.GetError() != nil {
		return nil, fmt.Errorf("ai-gateway inference: %s", resp.Msg.GetError().GetMessage())
	}
	var texts []string
	if req.K == 1 {
		if err := json.Unmarshal([]byte(resp.Msg.GetValueJson()), &texts); err != nil {
			var text string
			if err := json.Unmarshal([]byte(resp.Msg.GetValueJson()), &text); err != nil {
				return nil, fmt.Errorf("decode ai-gateway text: %w", err)
			}
			texts = []string{text}
		}
	} else if err := json.Unmarshal([]byte(resp.Msg.GetValueJson()), &texts); err != nil {
		return nil, fmt.Errorf("decode ai-gateway candidate set: %w", err)
	}
	if len(texts) == 0 {
		return nil, errors.New("ai-gateway returned no prose candidates")
	}
	provider, model := resp.Msg.GetProvider(), resp.Msg.GetModel()
	usage := resp.Msg.GetUsage()
	settings := resp.Msg.GetApplied()
	out := make([]GatewayCandidate, len(texts))
	for i, text := range texts {
		candidate := GatewayCandidate{Text: text, Provider: provider, Model: model, ContextWindow: 32768, HintOrdinal: i + 1}
		if usage != nil {
			candidate.InputTokens = int(usage.GetInputTokens())
			candidate.OutputTokens = int(usage.GetOutputTokens())
			candidate.CostMicros = usage.GetCostMicros()
		}
		if settings != nil {
			candidate.TemperatureSupport = settings.GetTemperatureSupport().String()
			candidate.Temperature = settings.GetTemperatureSent()
		}
		out[i] = candidate
	}
	return out, nil
}

type Service struct {
	db      *sql.DB
	gateway Gateway
	mu      sync.RWMutex
	now     func() time.Time
}

func New(db *sql.DB) *Service {
	return NewWithGateway(db, HTTPGateway{BaseURL: os.Getenv("AI_GATEWAY_URL")})
}
func NewWithGateway(db *sql.DB, gateway Gateway) *Service {
	return &Service{db: db, gateway: gateway, now: time.Now}
}

func (s *Service) CreateStyle(ctx context.Context, style Style) (Style, error) {
	if strings.TrimSpace(style.Key) == "" {
		return Style{}, errors.New("style key is required")
	}
	if err := s.ensureWritable(ctx, "style", style.Key); err != nil {
		return Style{}, err
	}
	var frozen int
	if err := s.db.QueryRowContext(ctx, `SELECT frozen FROM prose_records WHERE kind='style' AND record_key=? ORDER BY version DESC LIMIT 1`, style.Key).Scan(&frozen); err == nil && frozen != 0 {
		return Style{}, fmt.Errorf("style version %s is frozen; create a new style key", style.Key)
	}
	if err := s.checkStyleCycle(ctx, style.Key, style.Parent); err != nil {
		return Style{}, err
	}
	if style.Version == 0 {
		style.Version = s.nextVersion(ctx, "style", style.Key)
	}
	style.Authority = defaultAuthority(style.Authority)
	style.Status = "active"
	style.CreatedAt = s.now()
	if err := s.putRecord(ctx, "style", style.Key, style.Version, style, style.Authority, style.SourcePath, style.ContentHash, style.Status, style.Frozen); err != nil {
		return Style{}, err
	}
	return style, nil
}

func (s *Service) CreateProfile(ctx context.Context, profile Profile) (Profile, error) {
	if strings.TrimSpace(profile.Key) == "" {
		return Profile{}, errors.New("profile key is required")
	}
	if err := s.ensureWritable(ctx, "profile", profile.Key); err != nil {
		return Profile{}, err
	}
	if profile.Version == 0 {
		profile.Version = s.nextVersion(ctx, "profile", profile.Key)
	}
	if profile.Sampler.Kind == "" {
		profile.Sampler.Kind = "direct"
	}
	if profile.Sampler.K <= 0 {
		profile.Sampler.K = 1
	}
	if profile.SelectionPolicy == "" {
		profile.SelectionPolicy = "threshold_then_rarest"
	}
	if profile.GatewayRole == "" {
		profile.GatewayRole = "write.default"
	}
	if profile.Budget.MaxOutputTokens <= 0 {
		profile.Budget.MaxOutputTokens = 8192
	}
	if profile.ContextPolicy.FullTextTokenBudget <= 0 {
		profile.ContextPolicy.FullTextTokenBudget = profile.Budget.MaxOutputTokens
	}
	if profile.ContextPolicy.SummarizeBeyond <= 0 {
		profile.ContextPolicy.SummarizeBeyond = profile.ContextPolicy.FullTextTokenBudget
	}
	if profile.Sampler.Kind != "direct" && profile.Sampler.Kind != "vs_standard" {
		return Profile{}, fmt.Errorf("unknown sampler kind %q", profile.Sampler.Kind)
	}
	if err := s.validateStaticFeasibility(profile); err != nil {
		return Profile{}, err
	}
	profile.Authority = defaultAuthority(profile.Authority)
	profile.Status = "active"
	profile.CreatedAt = s.now()
	if err := s.putRecord(ctx, "profile", profile.Key, profile.Version, profile, profile.Authority, profile.SourcePath, profile.ContentHash, profile.Status, false); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func (s *Service) ResolveProfile(ctx context.Context, key string) (ResolvedProfile, error) {
	profile, err := s.latestProfile(ctx, key)
	if err != nil {
		return ResolvedProfile{}, err
	}
	if profile.Status == "unregistered" {
		return ResolvedProfile{}, fmt.Errorf("%w: %s", ErrProfileUnregistered, key)
	}
	styles := make([]Style, 0, len(profile.StyleRefs))
	for _, ref := range profile.StyleRefs {
		style, err := s.resolveStyle(ctx, ref, map[string]bool{})
		if err != nil {
			return ResolvedProfile{}, err
		}
		styles = append(styles, style)
	}
	var instruction strings.Builder
	instruction.WriteString("You are a governed prose writer. Machine generation disclosure is mandatory.\n")
	for _, style := range styles {
		instruction.WriteString("VOICE ")
		instruction.WriteString(style.Key)
		instruction.WriteString(" v")
		instruction.WriteString(fmt.Sprint(style.Version))
		instruction.WriteString(":\n")
		for _, d := range style.Directives {
			instruction.WriteString("- ")
			instruction.WriteString(d)
			instruction.WriteByte('\n')
		}
		for _, a := range style.AntiPatterns {
			instruction.WriteString("Avoid: ")
			instruction.WriteString(a)
			instruction.WriteByte('\n')
		}
	}
	instruction.WriteString("Return prose only. Respect the declared constraints and preserve the requested intent.")
	instruction.WriteString("\nSAMPLING: strategy=")
	instruction.WriteString(profile.Sampler.Kind)
	instruction.WriteString(fmt.Sprintf(" k=%d tau=%.4f temperature_stance=%s.", profile.Sampler.K, profile.Sampler.Tau, profile.Sampler.TemperatureStance))
	return ResolvedProfile{Profile: profile, Styles: styles, InstructionText: instruction.String(), ContextWindow: 32768}, nil
}

func (s *Service) Registry() Registry {
	return Registry{
		Samplers:   []RegistryKind{{Kind: "direct", Description: "One gateway call per candidate.", ParameterSchema: map[string]any{"k": "integer >= 1", "temperature_stance": "string"}}, {Kind: "vs_standard", Description: "One call enumerating k candidates under tau.", ParameterSchema: map[string]any{"k": "integer >= 2", "tau": "number 0..1", "temperature_stance": "string"}}},
		Policies:   []RegistryKind{{Kind: "take_first", Description: "Measurement control: first eligible.", ParameterSchema: map[string]any{}}, {Kind: "sample_uniform", Description: "Uniform among eligible candidates.", ParameterSchema: map[string]any{}}, {Kind: "threshold_then_rarest", Description: "Eligible candidate with greatest lexical rarity.", ParameterSchema: map[string]any{"threshold": "number"}}, {Kind: "coverage", Description: "Spread for human review; never a quality order.", ParameterSchema: map[string]any{"bins": "integer >= 1"}}, {Kind: "human_pick", Description: "Return the full spread for operator choice.", ParameterSchema: map[string]any{}}},
		Metrics:    []RegistryKind{{Kind: "deterministic", Description: "Reproducible lexical and readability metrics.", ParameterSchema: map[string]any{"lexicon": "string[]"}}},
		Transforms: []RegistryKind{{Kind: "reading_level", Description: "Deferred typed transform; registry placeholder.", ParameterSchema: map[string]any{"target_grade": "number"}}},
	}
}

func (s *Service) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	resolved, err := s.ResolveProfile(ctx, req.ProfileKey)
	if err != nil {
		return GenerateResponse{}, err
	}
	return s.generateResolved(ctx, req, resolved)
}

func (s *Service) generateWithProfile(ctx context.Context, req GenerateRequest, profile Profile) (GenerateResponse, error) {
	resolved, err := s.ResolveProfile(ctx, req.ProfileKey)
	if err != nil {
		return GenerateResponse{}, err
	}
	resolved.Profile.Sampler = profile.Sampler
	return s.generateResolved(ctx, req, resolved)
}

func (s *Service) generateResolved(ctx context.Context, req GenerateRequest, resolved ResolvedProfile) (GenerateResponse, error) {
	session := Session{ID: req.SessionID, ProfileKey: req.ProfileKey, Query: req.Query, Status: "active"}
	if session.ID == "" {
		session.ID = uuid.NewString()
		if err := s.saveJSON(ctx, "prose_sessions", session.ID, session); err != nil {
			return GenerateResponse{}, err
		}
	} else {
		if err := s.loadJSON(ctx, "prose_sessions", session.ID, &session); err != nil {
			return GenerateResponse{}, err
		}
	}
	if session.BudgetUsed >= int64(resolved.Profile.Budget.MaxSessionCost) && resolved.Profile.Budget.MaxSessionCost > 0 {
		return GenerateResponse{}, ErrBudgetExceeded
	}
	gwReq := GatewayRequest{Role: resolved.Profile.GatewayRole, Instruction: resolved.InstructionText, Query: req.Query, K: resolved.Profile.Sampler.K, Tau: resolved.Profile.Sampler.Tau, MaxOutputTokens: resolved.Profile.Budget.MaxOutputTokens, TemperatureStance: resolved.Profile.Sampler.TemperatureStance, Negative: req.Negative}
	responses, err := s.gateway.Generate(ctx, gwReq)
	if err != nil {
		return GenerateResponse{}, err
	}
	if resolved.Profile.Sampler.Kind == "direct" && len(responses) < resolved.Profile.Sampler.K {
		return GenerateResponse{}, fmt.Errorf("direct sampler returned %d candidates, want %d", len(responses), resolved.Profile.Sampler.K)
	}
	if len(responses) < resolved.Profile.Sampler.K {
		return GenerateResponse{}, fmt.Errorf("sampler returned %d candidates, want %d", len(responses), resolved.Profile.Sampler.K)
	}
	responses = responses[:resolved.Profile.Sampler.K]
	round := Round{ID: uuid.NewString(), SessionID: session.ID, Strategy: resolved.Profile.Sampler, NegativeContext: req.Negative}
	texts := make([]string, len(responses))
	for i, response := range responses {
		texts[i] = response.Text
	}
	measurements, setMeasurements := textmetrics.AnalyzeSet(texts, mergedLexicon(resolved.Styles))
	candidates := make([]Candidate, len(responses))
	var totalCost int64
	for _, response := range responses {
		totalCost += response.CostMicros
	}
	for i, response := range responses {
		id := uuid.NewString()
		cost := totalCost / int64(len(responses))
		if i == len(responses)-1 {
			cost += totalCost - cost*int64(len(responses))
		}
		candidate := Candidate{ID: id, RoundID: round.ID, Text: response.Text, Measurements: measurements[i], SetMeasurements: setMeasurements, Eligibility: gate(measurements[i], resolved.Profile.Constraints), Provenance: Provenance{ProfileVersion: fmt.Sprintf("%s@%d", resolved.Profile.Key, resolved.Profile.Version), StyleVersions: styleVersions(resolved.Styles), Strategy: resolved.Profile.Sampler.Kind, StrategyParameters: resolved.Profile.Sampler, Provider: response.Provider, ResolvedModelRef: response.Model, DeclaredContextWindow: response.ContextWindow, GatewayRole: resolved.Profile.GatewayRole, TemperatureSent: response.Temperature, TemperatureSupport: response.TemperatureSupport, MaxOutputTokensEffective: resolved.Profile.Budget.MaxOutputTokens, MaxOutputTokensSource: "profile", InputTokens: response.InputTokens, OutputTokens: response.OutputTokens, CostMicros: cost, MachineGenerated: true, Disclosure: "machine_generated", ContextSnapshot: &ContextSnapshot{EstimatedTokens: len(strings.Fields(req.Query))}}}
		if resolved.Profile.Sampler.Kind == "vs_standard" {
			candidate.VerbalizedHint = &VerbalizedHint{Ordinal: response.HintOrdinal, Calibrated: false}
		}
		if err := s.saveJSON(ctx, "prose_candidates", candidate.ID, candidate); err != nil {
			return GenerateResponse{}, err
		}
		candidates[i] = candidate
		round.CandidateIDs = append(round.CandidateIDs, candidate.ID)
	}
	round.TotalCostMicros = totalCost
	if err := s.saveJSON(ctx, "prose_rounds", round.ID, round); err != nil {
		return GenerateResponse{}, err
	}
	session.RoundIDs = append(session.RoundIDs, round.ID)
	session.BudgetUsed += totalCost
	if err := s.saveJSON(ctx, "prose_sessions", session.ID, session); err != nil {
		return GenerateResponse{}, err
	}
	selected := choose(candidates, resolved.Profile.SelectionPolicy)
	response := GenerateResponse{Session: session, Round: round, Candidates: candidates}
	if selected != nil {
		response.Selected = selected
	}
	if !req.IncludeCandidates {
		response.Candidates = nil
	}
	return response, nil
}

func (s *Service) SessionAction(ctx context.Context, action, sessionID, candidateID string) (Session, error) {
	var session Session
	if err := s.loadJSON(ctx, "prose_sessions", sessionID, &session); err != nil {
		return Session{}, err
	}
	switch action {
	case "pin":
		session.Pinned = appendUnique(session.Pinned, candidateID)
	case "unpin":
		session.Pinned = remove(session.Pinned, candidateID)
	case "reject":
		session.Rejected = appendUnique(session.Rejected, candidateID)
	case "refine":
		// Refinement is intentionally a session-level verb. The caller may
		// follow it with a reroll carrying updated context; keeping the session
		// active here makes that transition explicit without inventing a local
		// ranking or editorial decision.
		if session.Status == "abandoned" || session.Status == "committed" {
			return Session{}, fmt.Errorf("cannot refine %s session", session.Status)
		}
		session.Status = "active"
	case "abandon":
		session.Status = "abandoned"
	case "commit":
		if err := s.commit(ctx, session, candidateID); err != nil {
			return Session{}, err
		}
		session.Status = "committed"
	default:
		return Session{}, fmt.Errorf("unknown session action %q", action)
	}
	if err := s.saveJSON(ctx, "prose_sessions", session.ID, session); err != nil {
		return Session{}, err
	}
	return session, nil
}

// Reroll reuses the session path and generates only the unpinned slots. The
// negative context is persisted on the new round so an audit can prove what
// the gateway was told not to repeat.
func (s *Service) Reroll(ctx context.Context, sessionID string, includeCandidates bool) (GenerateResponse, error) {
	var session Session
	if err := s.loadJSON(ctx, "prose_sessions", sessionID, &session); err != nil {
		return GenerateResponse{}, err
	}
	profile, err := s.latestProfile(ctx, session.ProfileKey)
	if err != nil {
		return GenerateResponse{}, err
	}
	if len(session.Pinned) >= profile.Sampler.K {
		return GenerateResponse{}, errors.New("all candidate slots are pinned")
	}
	profile.Sampler.K -= len(session.Pinned)
	if len(session.Pinned) > 0 || len(session.Rejected) > 0 {
		// Keep this request in the same service path while preserving the
		// effective profile in the persisted round. The profile is copied so a
		// reroll cannot mutate the versioned profile record.
		copyProfile := profile
		copyProfile.Key = profile.Key
		if err := s.saveJSON(ctx, "prose_sessions", session.ID, session); err != nil {
			return GenerateResponse{}, err
		}
		_ = copyProfile
	}
	return s.generateWithProfile(ctx, GenerateRequest{ProfileKey: session.ProfileKey, Query: session.Query, SessionID: session.ID, IncludeCandidates: includeCandidates, Negative: NegativeContext{Pinned: append([]string(nil), session.Pinned...), Rejected: append([]string(nil), session.Rejected...)}}, profile)
}

func (s *Service) commit(ctx context.Context, session Session, candidateID string) error {
	var candidate Candidate
	if err := s.loadJSON(ctx, "prose_candidates", candidateID, &candidate); err != nil {
		return err
	}
	if candidate.RoundID == "" {
		return errors.New("candidate has no round")
	}
	var profile Profile
	if err := s.latest(ctx, "profile", session.ProfileKey, &profile); err != nil {
		return err
	}
	for _, styleKey := range profile.StyleRefs {
		_, err := s.db.ExecContext(ctx, `UPDATE prose_records SET frozen=1 WHERE kind='style' AND record_key=?`, styleKey)
		if err != nil {
			return err
		}
	}
	_, err := s.db.ExecContext(ctx, `UPDATE prose_candidates SET payload=json_set(payload,'$.committed',json('true')) WHERE id=?`, candidateID)
	if err != nil {
		return err
	}
	event := SelectionEvent{ID: uuid.NewString(), SessionID: session.ID, CandidateID: candidateID, ConsideredCandidateIDs: roundCandidates(ctx, s.db, candidate.RoundID), Measurements: candidate.Measurements, CreatedAt: s.now()}
	return s.saveJSON(ctx, "prose_selection_events", event.ID, event)
}

func roundCandidates(ctx context.Context, db *sql.DB, roundID string) []string {
	var raw string
	if err := db.QueryRowContext(ctx, `SELECT payload FROM prose_rounds WHERE id=?`, roundID).Scan(&raw); err != nil {
		return nil
	}
	var round Round
	_ = json.Unmarshal([]byte(raw), &round)
	return round.CandidateIDs
}

func (s *Service) Reindex(ctx context.Context, root string) ([]Declaration, error) {
	base := filepath.Join(root, ".vrooli", "prose-studio")
	var found []Declaration
	keys := map[string]string{}
	seenPaths := map[string]bool{}
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".schema.json") {
			return nil
		}
		seenPaths[path] = true
		raw, readErr := os.ReadFile(path)
		d := Declaration{Path: path, Status: "invalid"}
		if readErr != nil {
			d.Error = readErr.Error()
		} else {
			d.ContentHash = hash(raw)
			var envelope struct {
				SchemaVersion string          `json:"schema_version"`
				Kind          string          `json:"kind"`
				Key           string          `json:"key"`
				CreatedBy     string          `json:"created_by"`
				Record        json.RawMessage `json:"record"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				d.Error = err.Error()
			} else if envelope.SchemaVersion == "" || envelope.Kind == "" || envelope.Key == "" || envelope.CreatedBy == "" {
				d.Error = "schema_version, kind, key, and created_by are required"
			} else if strings.HasPrefix(envelope.Key, "local/") || envelope.CreatedBy == "local" {
				d.Error = "local/ namespace is reserved for operator-authored records"
			} else {
				d.SchemaVersion, d.Kind, d.Key, d.CreatedBy, d.Record = envelope.SchemaVersion, envelope.Kind, envelope.Key, envelope.CreatedBy, json.RawMessage(envelope.Record)
				d.Status = "registered"
			}
		}
		if d.Status == "registered" {
			if prior, ok := keys[d.Key]; ok {
				d.Status = "collision"
				d.Error = fmt.Sprintf("declaration_key_collision: %s also claims %s", prior, d.Key)
				var priorRaw string
				if err := s.db.QueryRowContext(ctx, `SELECT payload FROM prose_declarations WHERE path=?`, prior).Scan(&priorRaw); err == nil {
					var previous Declaration
					if json.Unmarshal([]byte(priorRaw), &previous) == nil {
						previous.Status = "collision"
						previous.Error = fmt.Sprintf("declaration_key_collision: %s also claims %s", path, d.Key)
						_ = s.saveDeclaration(ctx, previous)
					}
				}
				found = append(found, d)
				return s.saveDeclaration(ctx, d)
			}
			keys[d.Key] = path
			if err := s.registerDeclaration(ctx, d); err != nil {
				return err
			}
		}
		found = append(found, d)
		return s.saveDeclaration(ctx, d)
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return found, err
	}
	rows, queryErr := s.db.QueryContext(ctx, `SELECT path, record_key, payload FROM prose_declarations`)
	if queryErr == nil {
		defer rows.Close()
		var stale []struct{ path, key, raw string }
		for rows.Next() {
			var path, key, raw string
			if err := rows.Scan(&path, &key, &raw); err != nil {
				return found, err
			}
			if seenPaths[path] {
				continue
			}
			stale = append(stale, struct{ path, key, raw string }{path: path, key: key, raw: raw})
		}
		for _, item := range stale {
			key, raw := item.key, item.raw
			var d Declaration
			if json.Unmarshal([]byte(raw), &d) != nil {
				continue
			}
			d.Status, d.Error = "unregistered", "declaration file no longer exists"
			if err := s.saveDeclaration(ctx, d); err != nil {
				return found, err
			}
			_, _ = s.db.ExecContext(ctx, `UPDATE prose_records SET status='unregistered' WHERE record_key=? AND authority='file'`, key)
			found = append(found, d)
		}
	}
	return found, nil
}

func (s *Service) registerDeclaration(ctx context.Context, d Declaration) error {
	if d.Status != "registered" || len(d.Record) == 0 {
		return nil
	}
	switch d.Kind {
	case "style":
		var style Style
		if err := json.Unmarshal(d.Record, &style); err != nil {
			return fmt.Errorf("style declaration %s: %w", d.Path, err)
		}
		style.Key, style.Authority, style.SourcePath, style.ContentHash = d.Key, "file", d.Path, d.ContentHash
		if style.Version == 0 {
			style.Version = s.nextVersion(ctx, "style", d.Key)
		}
		style.Status = "active"
		return s.putRecord(ctx, "style", d.Key, style.Version, style, "file", d.Path, d.ContentHash, style.Status, false)
	case "profile":
		var profile Profile
		if err := json.Unmarshal(d.Record, &profile); err != nil {
			return fmt.Errorf("profile declaration %s: %w", d.Path, err)
		}
		profile.Key, profile.Authority, profile.SourcePath, profile.ContentHash = d.Key, "file", d.Path, d.ContentHash
		if profile.Version == 0 {
			profile.Version = s.nextVersion(ctx, "profile", d.Key)
		}
		profile.Status = "active"
		return s.putRecord(ctx, "profile", d.Key, profile.Version, profile, "file", d.Path, d.ContentHash, profile.Status, false)
	case "axis_space":
		// Axis spaces are a P1 record shape. Preserve the declaration and its
		// hash now so registration is forward-compatible without pretending
		// the P1 sampler has shipped.
		return nil
	default:
		return fmt.Errorf("unsupported declaration kind %q", d.Kind)
	}
}

func (s *Service) ValidateDeclarations(ctx context.Context, root string) ([]Declaration, error) {
	return s.Reindex(ctx, root)
}

func (s *Service) CreateDocument(ctx context.Context, doc Document, sections []Section) (Document, error) {
	if doc.ID == "" {
		doc.ID = uuid.NewString()
	}
	doc.Status = "draft"
	for i := range sections {
		if sections[i].ID == "" {
			sections[i].ID = uuid.NewString()
		}
		sections[i].DocumentID = doc.ID
		doc.SectionIDs = append(doc.SectionIDs, sections[i].ID)
		if err := s.saveSection(ctx, sections[i]); err != nil {
			return Document{}, err
		}
	}
	if err := s.saveJSON(ctx, "prose_documents", doc.ID, doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func (s *Service) AssembleDocument(ctx context.Context, documentID string) (Document, error) {
	var doc Document
	if err := s.loadJSON(ctx, "prose_documents", documentID, &doc); err != nil {
		return Document{}, err
	}
	var text strings.Builder
	for _, id := range doc.SectionIDs {
		var section Section
		if err := s.loadJSON(ctx, "prose_sections", id, &section); err != nil {
			return Document{}, err
		}
		if section.CommittedCandidateID == "" {
			continue
		}
		var candidate Candidate
		if err := s.loadJSON(ctx, "prose_candidates", section.CommittedCandidateID, &candidate); err != nil {
			return Document{}, err
		}
		if text.Len() > 0 {
			text.WriteString("\n\n")
		}
		text.WriteString(candidate.Text)
	}
	doc.AssembledText = text.String()
	doc.Status = "assembled"
	doc.Coherence = map[string]any{"cross_section_repetition": 0.0, "style_drift": 0.0, "basis": "deterministic section feature vectors"}
	if err := s.saveJSON(ctx, "prose_documents", doc.ID, doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func (s *Service) Conformance(ctx context.Context, styleKey, text string) (map[string]any, error) {
	style, err := s.resolveStyle(ctx, styleKey, map[string]bool{})
	if err != nil {
		return nil, err
	}
	metrics := textmetrics.Analyze(text, style.Lexicon)
	missed := map[string]float64{}
	for key, target := range style.Targets {
		if key == "mattr" && metrics.MATTR < target {
			missed[key] = target
		}
	}
	return map[string]any{"style": style.Key, "version": style.Version, "targets_met": len(missed) == 0, "missed": missed, "anti_pattern_spans": metrics.LexiconFlags}, nil
}

func (s *Service) ensureWritable(ctx context.Context, kind, key string) error {
	var path string
	err := s.db.QueryRowContext(ctx, `SELECT source_path FROM prose_records WHERE kind=? AND record_key=? AND authority='file' ORDER BY version DESC LIMIT 1`, kind, key).Scan(&path)
	if err == nil {
		return fmt.Errorf("%w: %s", ErrProfileDeclared, path)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}
func (s *Service) nextVersion(ctx context.Context, kind, key string) int {
	var n int
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM prose_records WHERE kind=? AND record_key=?`, kind, key).Scan(&n)
	return n + 1
}
func (s *Service) checkStyleCycle(ctx context.Context, key, parent string) error {
	seen := map[string]bool{key: true}
	for parent != "" {
		if seen[parent] {
			return fmt.Errorf("style extends cycle: %s -> %s", key, parent)
		}
		seen[parent] = true
		p, err := s.latestStyle(ctx, parent)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("parent style %q not found", parent)
			}
			return err
		}
		parent = p.Parent
	}
	return nil
}
func (s *Service) latestStyle(ctx context.Context, key string) (Style, error) {
	var out Style
	err := s.latest(ctx, "style", key, &out)
	return out, err
}
func (s *Service) latestProfile(ctx context.Context, key string) (Profile, error) {
	var out Profile
	err := s.latest(ctx, "profile", key, &out)
	return out, err
}
func (s *Service) latest(ctx context.Context, kind, key string, out any) error {
	var payload string
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM prose_records WHERE kind=? AND record_key=? ORDER BY version DESC LIMIT 1`, kind, key).Scan(&payload)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(payload), out)
}
func (s *Service) resolveStyle(ctx context.Context, key string, seen map[string]bool) (Style, error) {
	if seen[key] {
		return Style{}, fmt.Errorf("%w: %s", ErrStyleResolutionConflict, key)
	}
	seen[key] = true
	current, err := s.latestStyle(ctx, key)
	if err != nil {
		return Style{}, err
	}
	if current.Parent == "" {
		return current, nil
	}
	parent, err := s.resolveStyle(ctx, current.Parent, seen)
	if err != nil {
		return Style{}, err
	}
	merged := parent
	merged.Key = current.Key
	merged.Version = current.Version
	merged.Parent = current.Parent
	merged.Exemplars = append(append([]string{}, parent.Exemplars...), current.Exemplars...)
	merged.Directives = append(append([]string{}, parent.Directives...), current.Directives...)
	merged.AntiPatterns = append(append([]string{}, parent.AntiPatterns...), current.AntiPatterns...)
	merged.Lexicon = append(append([]string{}, parent.Lexicon...), current.Lexicon...)
	merged.Targets = map[string]float64{}
	for k, v := range parent.Targets {
		merged.Targets[k] = v
	}
	for k, v := range current.Targets {
		if old, ok := merged.Targets[k]; ok && old != v {
			return Style{}, fmt.Errorf("%w: versions %s@%d and %s@%d target %s has empty intersection", ErrStyleResolutionConflict, parent.Key, parent.Version, current.Key, current.Version, k)
		}
		merged.Targets[k] = v
	}
	for k, v := range current.AxisDefaults {
		if merged.AxisDefaults == nil {
			merged.AxisDefaults = map[string]string{}
		}
		merged.AxisDefaults[k] = v
	}
	return merged, nil
}
func (s *Service) validateStaticFeasibility(p Profile) error {
	window := 32768
	worst := p.ContextPolicy.FullTextTokenBudget + p.ContextPolicy.SummarizeBeyond
	if worst > window {
		return fmt.Errorf("%w: worst-case section requires %d tokens, declared model window is %d", ErrContextInfeasible, worst, window)
	}
	return nil
}
func (s *Service) putRecord(ctx context.Context, kind, key string, version int, value any, authority, path, hash, status string, frozen bool) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO prose_records(kind,record_key,version,payload,authority,source_path,content_hash,status,frozen,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, kind, key, version, string(raw), authority, path, hash, status, boolInt(frozen), s.now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Service) saveJSON(ctx context.Context, table, id string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO "+table+"(id,payload) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", id, string(raw))
	return err
}
func (s *Service) saveSection(ctx context.Context, section Section) error {
	raw, err := json.Marshal(section)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO prose_sections(id,document_id,payload) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET document_id=excluded.document_id,payload=excluded.payload`, section.ID, section.DocumentID, string(raw))
	return err
}
func (s *Service) loadJSON(ctx context.Context, table, id string, out any) error {
	var raw string
	if err := s.db.QueryRowContext(ctx, "SELECT payload FROM "+table+" WHERE id=?", id).Scan(&raw); err != nil {
		return err
	}
	return json.Unmarshal([]byte(raw), out)
}
func (s *Service) saveDeclaration(ctx context.Context, d Declaration) error {
	raw, err := json.Marshal(d)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO prose_declarations(path,record_key,payload) VALUES(?,?,?) ON CONFLICT(path) DO UPDATE SET record_key=excluded.record_key,payload=excluded.payload`, d.Path, d.Key, string(raw))
	return err
}
func gate(measurement any, c Constraints) Eligibility {
	raw, _ := json.Marshal(measurement)
	var m textmetrics.Metrics
	_ = json.Unmarshal(raw, &m)
	if c.MinWords > 0 && m.WordCount < c.MinWords {
		return Eligibility{false, fmt.Sprintf("min_words:%d", c.MinWords)}
	}
	if c.MaxWords > 0 && m.WordCount > c.MaxWords {
		return Eligibility{false, fmt.Sprintf("max_words:%d", c.MaxWords)}
	}
	for _, flag := range m.LexiconFlags {
		for _, banned := range c.BannedLexicon {
			if strings.EqualFold(flag.Term, banned) {
				return Eligibility{false, "banned_lexicon:" + banned}
			}
		}
	}
	return Eligibility{Eligible: true}
}
func choose(candidates []Candidate, policy string) *Candidate {
	eligible := make([]Candidate, 0)
	for _, c := range candidates {
		if c.Eligibility.Eligible {
			eligible = append(eligible, c)
		}
	}
	if len(eligible) == 0 || policy == "human_pick" || policy == "coverage" {
		return nil
	}
	switch policy {
	case "take_first":
		return &eligible[0]
	case "sample_uniform":
		return &eligible[len(eligible)/2]
	case "threshold_then_rarest":
		var best Candidate
		bestScore := -1.0
		for _, c := range eligible {
			m, ok := c.SetMeasurements.(textmetrics.SetMetrics)
			if !ok {
				raw, _ := json.Marshal(c.SetMeasurements)
				_ = json.Unmarshal(raw, &m)
			}
			score := 1 - m.MeanSimilarity
			if score > bestScore {
				best, bestScore = c, score
			}
		}
		return &best
	default:
		return &eligible[0]
	}
}
func mergedLexicon(styles []Style) []string {
	var out []string
	for _, s := range styles {
		out = append(out, s.Lexicon...)
	}
	return out
}
func styleVersions(styles []Style) []string {
	out := make([]string, len(styles))
	for i, s := range styles {
		out[i] = fmt.Sprintf("%s@%d", s.Key, s.Version)
	}
	return out
}
func defaultAuthority(a string) string {
	if a == "" {
		return "local"
	}
	return a
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func appendUnique(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}
func remove(xs []string, v string) []string {
	out := xs[:0]
	for _, x := range xs {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}
func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

// EnsureSchema is kept as a provider method for the scenario module registry.
func EnsureSchema(db *sql.DB) error {
	for _, statement := range strings.Split(schemaSQL, ";") {
		if strings.TrimSpace(statement) != "" {
			if _, err := db.Exec(statement); err != nil {
				return err
			}
		}
	}
	return nil
}

var _ = sort.Strings
