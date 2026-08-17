package prose

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	ErrEmptyQuery              = errors.New("query_is_required")
	ErrUnknownLocality         = errors.New("unknown_locality")
	ErrUnknownTemperature      = errors.New("unknown_temperature_stance")
	ErrMalformedCandidateSet   = errors.New("malformed_candidate_set")
	ErrDeclarationRootMissing  = errors.New("declaration_root_missing")
)

// underRoot reports whether path sits inside base, comparing cleaned absolute
// paths so a sibling directory sharing a name prefix cannot match.
func underRoot(base, path string) bool {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// localityNames maps the profile-facing locality vocabulary onto the gateway's
// Profile enum. The names are the enum's own, minus the PROFILE_ prefix, so a
// declaration file never has to know the proto spelling.
var localityNames = map[string]sharedv1.Profile{
	"local_only":        sharedv1.Profile_PROFILE_LOCAL_ONLY,
	"local_first":       sharedv1.Profile_PROFILE_LOCAL_FIRST,
	"remote_only":       sharedv1.Profile_PROFILE_REMOTE_ONLY,
	"quality_first":     sharedv1.Profile_PROFILE_QUALITY_FIRST,
	"cheap_first":       sharedv1.Profile_PROFILE_CHEAP_FIRST,
	"privacy_sensitive": sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE,
}

// localityProfile resolves a declared locality. An empty stance keeps the
// historical local-first default; an unrecognised one is a request defect
// rather than a silent fallback, because a profile that asks for remote-only
// and quietly runs locally is the failure this field exists to prevent.
func localityProfile(name string) (sharedv1.Profile, error) {
	trimmed := strings.TrimSpace(strings.ToLower(name))
	if trimmed == "" {
		return sharedv1.Profile_PROFILE_LOCAL_FIRST, nil
	}
	profile, ok := localityNames[trimmed]
	if !ok {
		return sharedv1.Profile_PROFILE_UNSPECIFIED, fmt.Errorf("%w: %q", ErrUnknownLocality, name)
	}
	return profile, nil
}

// samplingControls turns a profile's temperature stance into an explicit
// gateway control. "ignored" and "role_default" send nothing, which the gateway
// reads as "use the role's declared sampling". Any other value must parse as a
// temperature; an unparseable stance is a defect rather than a silent no-op,
// which is what the stance previously was.
func samplingControls(stance string) (*sharedv1.SamplingControls, error) {
	trimmed := strings.TrimSpace(strings.ToLower(stance))
	switch trimmed {
	case "", "ignored", "role_default":
		return nil, nil
	}
	temperature, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: %q is neither a role-default stance nor a temperature", ErrUnknownTemperature, stance)
	}
	return &sharedv1.SamplingControls{Temperature: &temperature}, nil
}

const (
	samplerDirect     = "direct"
	samplerVSStandard = "vs_standard"
)

// samplingKeyOf builds a round's effective generation identity. The output cap
// comes from what the gateway reported imposing rather than what the profile
// requested, because a set generated under a role-policy cap is not comparable
// to one generated under a caller cap even when the numbers coincide — which is
// exactly why SamplingKey carries the source alongside the value.
func samplingKeyOf(sampler Sampler, responses []GatewayCandidate) textmetrics.SamplingKey {
	// Tau is not carried: it is a parameter of the verbalized strategy, and the
	// strategy is the variable an experiment varies. It stays recorded in full on
	// Round.Strategy, which is provenance rather than a comparability condition.
	key := textmetrics.SamplingKey{
		K:                 sampler.K,
		TemperatureStance: sampler.TemperatureStance,
	}
	if len(responses) > 0 {
		key.MaxOutputTokens = responses[0].MaxOutputTokensEffective
		key.MaxOutputTokenSource = responses[0].MaxOutputTokensSource
	}
	return key
}

// CompareRounds refuses two rounds whose generation conditions differ, so a
// caller cannot place two set-diversity numbers side by side unless they were
// produced under the same effective sampling. This is OT-P0-024's enforcement
// point: comparability is decided from recorded keys, not from the caller's
// assurance that the two runs were "the same except for the strategy".
func (s *Service) CompareRounds(ctx context.Context, leftID, rightID string) error {
	left, err := s.loadRound(ctx, leftID)
	if err != nil {
		return err
	}
	right, err := s.loadRound(ctx, rightID)
	if err != nil {
		return err
	}
	return textmetrics.Comparable(left.SamplingKey, right.SamplingKey)
}

func (s *Service) loadRound(ctx context.Context, id string) (Round, error) {
	var round Round
	if err := s.loadJSON(ctx, "prose_rounds", id, &round); err != nil {
		return Round{}, fmt.Errorf("load round %s: %w", id, err)
	}
	return round, nil
}

// vsCandidateSchema elicits the verbalized distribution: each entry pairs the
// prose with the model's own probability for it. minItems is deliberately absent
// because the gateway's enforceable schema subset does not carry it, so the
// candidate count rides in the instruction and is checked after decode. The
// probability bounds are in-schema precisely because minimum/maximum are in that
// subset, which makes the gateway's local validator the one enforcing them.
const vsCandidateSchema = `{"type":"array","items":{"type":"object","properties":{"text":{"type":"string"},"probability":{"type":"number","minimum":0,"maximum":1}},"required":["text","probability"]}}`

func gatewaySchema(req GatewayRequest) string {
	if req.Strategy == samplerVSStandard {
		return vsCandidateSchema
	}
	if req.K > 1 {
		return `{"type":"array","items":{"type":"string"}}`
	}
	return `{"type":"string"}`
}

// verbalizedCandidate is the wire shape of one entry in a verbalized
// distribution. It exists only inside the decode: the probability is read to
// derive a rank and then dropped, and never reaches a stored record.
type verbalizedCandidate struct {
	Text        string  `json:"text"`
	Probability float64 `json:"probability"`
}

// decodeCandidates turns a gateway value into prose plus, for a verbalized
// strategy, the rank each candidate holds under the model's own probabilities.
// Order is the model's emission order throughout; ranks are carried alongside
// rather than applied as a sort, because reordering the set by the model's
// probability would make a quality proxy the presentation order.
func decodeCandidates(req GatewayRequest, valueJSON string) ([]string, []int, error) {
	if req.Strategy == samplerVSStandard {
		var entries []verbalizedCandidate
		if err := json.Unmarshal([]byte(valueJSON), &entries); err != nil {
			return nil, nil, fmt.Errorf("decode verbalized candidate set: %w", err)
		}
		if len(entries) == 0 {
			return nil, nil, errors.New("ai-gateway returned no prose candidates")
		}
		texts := make([]string, len(entries))
		for i, entry := range entries {
			if strings.TrimSpace(entry.Text) == "" {
				return nil, nil, fmt.Errorf("%w: candidate %d carries no text", ErrMalformedCandidateSet, i+1)
			}
			// The gateway validates these bounds too. Re-checking here keeps the
			// rank honest even if this package is ever pointed at a seam that does
			// not, which is how the ordinal became a fabrication the first time.
			if entry.Probability < 0 || entry.Probability > 1 {
				return nil, nil, fmt.Errorf("%w: candidate %d reports probability %v outside [0,1]", ErrMalformedCandidateSet, i+1, entry.Probability)
			}
			texts[i] = entry.Text
		}
		return texts, verbalizedOrdinals(entries), nil
	}
	var texts []string
	if req.K == 1 {
		if err := json.Unmarshal([]byte(valueJSON), &texts); err != nil {
			var text string
			if err := json.Unmarshal([]byte(valueJSON), &text); err != nil {
				return nil, nil, fmt.Errorf("decode ai-gateway text: %w", err)
			}
			texts = []string{text}
		}
	} else if err := json.Unmarshal([]byte(valueJSON), &texts); err != nil {
		return nil, nil, fmt.Errorf("decode ai-gateway candidate set: %w", err)
	}
	if len(texts) == 0 {
		return nil, nil, errors.New("ai-gateway returned no prose candidates")
	}
	// A strategy that elicits no probability carries no ordering signal. Zero
	// says that, where a positional index would have claimed a signal that the
	// model never gave.
	return texts, make([]int, len(texts)), nil
}

// verbalizedOrdinals ranks entries by descending verbalized probability, 1 being
// the highest. Ties keep emission order so the rank is deterministic.
func verbalizedOrdinals(entries []verbalizedCandidate) []int {
	order := make([]int, len(entries))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return entries[order[a]].Probability > entries[order[b]].Probability
	})
	ordinals := make([]int, len(entries))
	for rank, index := range order {
		ordinals[index] = rank + 1
	}
	return ordinals
}

// verbalizedInstruction asks the model to enumerate a distribution rather than
// answer once. The tail clause is what does the work: an instance-level request
// lands on the mode, so the threshold pushes the set off it. The final clause is
// not decoration either — without it a model reads "unlikely" as "strange" and
// spends the set on novelty instead of on genuinely different readings.
func verbalizedInstruction(k int, tau float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Return %d substantially different responses to the request, as a JSON array.\n", k)
	b.WriteString("Give each entry the response text and an estimated probability, between 0 and 1, that the response carries relative to the full distribution of plausible responses.\n")
	if tau > 0 {
		fmt.Fprintf(&b, "Sample from the tail of that distribution: prefer valid responses that ordinary prompting would leave with little probability mass, and aim for each candidate's probability to fall below %.2f.\n", tau)
	}
	b.WriteString("Vary the assumption, the framing, the structure, and the angle between candidates, not merely the wording.\n")
	b.WriteString("Do not make a candidate strange for the sake of novelty. Every candidate must stay logically sound, relevant to the request, and useful on its own, and must obey the voice above.")
	return b.String()
}

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
	if strings.TrimSpace(req.Query) == "" {
		return nil, ErrEmptyQuery
	}
	// The direct strategy is one draft per call by definition, so a k-slot direct
	// set is k governed requests. vs_standard is one request that enumerates the
	// whole set. This keys on the strategy, not the role: the strategy decides
	// how many calls a set costs, and a role is free to serve either.
	if req.Strategy == samplerDirect && req.K > 1 {
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
	schema := gatewaySchema(req)
	instruction := req.Instruction
	if len(req.Negative.Pinned) > 0 || len(req.Negative.Rejected) > 0 {
		negative, _ := json.Marshal(req.Negative)
		instruction += "\nDo not repeat candidates represented by this prior-round context: " + string(negative)
	}
	locality, err := localityProfile(req.Locality)
	if err != nil {
		return nil, err
	}
	sampling, err := samplingControls(req.TemperatureStance)
	if err != nil {
		return nil, err
	}
	// Source carries what the caller asked for; Instruction carries how to write
	// it. Sending the query as Source is what makes the request about anything.
	resp, err := connectClient.Run(ctx, connect.NewRequest(&inferencev1.RunRequest{
		Source:          req.Query,
		SchemaJson:      schema,
		Instruction:     instruction,
		Role:            req.Role,
		Profile:         locality,
		Sampling:        sampling,
		MaxOutputTokens: int32(req.MaxOutputTokens),
	}))
	if err != nil {
		return nil, fmt.Errorf("ai-gateway request: %w", err)
	}
	if resp.Msg.GetError() != nil {
		if resp.Msg.GetError().GetCode() == inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_CONTEXT_OVERFLOW {
			return nil, fmt.Errorf("%w: gateway refused the assembled context before provider dispatch: %s", ErrContextInfeasible, resp.Msg.GetError().GetMessage())
		}
		return nil, fmt.Errorf("ai-gateway inference: %s", resp.Msg.GetError().GetMessage())
	}
	texts, ordinals, err := decodeCandidates(req, resp.Msg.GetValueJson())
	if err != nil {
		return nil, err
	}
	provider, model := resp.Msg.GetProvider(), resp.Msg.GetModel()
	usage := resp.Msg.GetUsage()
	settings := resp.Msg.GetApplied()
	out := make([]GatewayCandidate, len(texts))
	for i, text := range texts {
		// The gateway reports no model context window today: AppliedSettings
		// carries sampling and output-cap facts only, and no resource policy
		// declares a window. Undeclared is therefore the truth, and a constant
		// invented here would be indistinguishable from a measured one.
		candidate := GatewayCandidate{Text: text, Provider: provider, Model: model, HintOrdinal: ordinals[i]}
		if usage != nil {
			candidate.InputTokens = int(usage.GetInputTokens())
			candidate.OutputTokens = int(usage.GetOutputTokens())
			candidate.CostMicros = usage.GetCostMicros()
		}
		if settings != nil {
			candidate.TemperatureSupport = settings.GetTemperatureSupport().String()
			candidate.Temperature = settings.GetTemperatureSent()
			// The cap the gateway imposed, from the gateway. Echoing the profile's
			// requested cap back as provenance would report the request rather
			// than what happened, which is the one thing provenance must not do.
			candidate.MaxOutputTokensEffective = int(settings.GetMaxOutputTokensEffective())
			candidate.MaxOutputTokensSource = settings.GetMaxOutputTokensSource().String()
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
	// declarationsRoot is the scenario's own declaration root, used when a
	// reindex caller names none. Empty means no default is configured.
	declarationsRoot string
}

// SetDeclarationsRoot names the root a rootless Reindex should rescan.
func (s *Service) SetDeclarationsRoot(root string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.declarationsRoot = strings.TrimSpace(root)
}

func (s *Service) defaultDeclarationsRoot() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.declarationsRoot
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
		profile.Sampler.Kind = samplerDirect
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
	if profile.Sampler.Kind != samplerDirect && profile.Sampler.Kind != samplerVSStandard {
		return Profile{}, fmt.Errorf("unknown sampler kind %q", profile.Sampler.Kind)
	}
	// A one-candidate distribution is the mode with extra steps: the technique
	// only means anything across a set, so refuse the configuration rather than
	// let a profile claim a strategy it cannot exercise.
	if profile.Sampler.Kind == samplerVSStandard && profile.Sampler.K < 2 {
		return Profile{}, fmt.Errorf("sampler kind %q requires k >= 2, got %d", samplerVSStandard, profile.Sampler.K)
	}
	if profile.Sampler.Kind == samplerVSStandard && (profile.Sampler.Tau < 0 || profile.Sampler.Tau > 1) {
		return Profile{}, fmt.Errorf("sampler tau must fall in [0,1], got %v", profile.Sampler.Tau)
	}
	if profile.Locality == "" {
		profile.Locality = "local_first"
	}
	// Reject an unusable locality or temperature stance at write time, so a
	// profile cannot sit in the registry looking valid and fail every generate.
	if _, err := localityProfile(profile.Locality); err != nil {
		return Profile{}, err
	}
	if _, err := samplingControls(profile.Sampler.TemperatureStance); err != nil {
		return Profile{}, err
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
	// Machine-generation provenance is carried on the candidate record as a
	// constant (see Provenance.Disclosure). It must never be asked of the model,
	// which answers such an instruction inside the prose itself.
	instruction.WriteString("You are a prose writer. Write the prose the request asks for, and nothing else.\n")
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
		for _, exemplar := range style.Exemplars {
			instruction.WriteString("Example voice:\n")
			instruction.WriteString(exemplar)
			instruction.WriteByte('\n')
		}
		if len(style.Lexicon) > 0 {
			instruction.WriteString("Preferred lexicon: ")
			instruction.WriteString(strings.Join(style.Lexicon, ", "))
			instruction.WriteByte('\n')
		}
		for target, value := range style.Targets {
			instruction.WriteString("Target ")
			instruction.WriteString(target)
			instruction.WriteString(" >= ")
			instruction.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
			instruction.WriteByte('\n')
		}
		for _, a := range style.AntiPatterns {
			instruction.WriteString("Avoid: ")
			instruction.WriteString(a)
			instruction.WriteByte('\n')
		}
	}
	// The temperature stance is a transport concern and becomes a SamplingControls
	// message, so it is not named here. The candidate count and threshold are the
	// opposite: the enforceable schema subset carries neither, so a verbalized
	// strategy has to ask for them in words. Built here rather than at the gateway
	// seam so the text ResolveProfile reports is the text that will be sent.
	if profile.Sampler.Kind == samplerVSStandard {
		instruction.WriteString(verbalizedInstruction(profile.Sampler.K, profile.Sampler.Tau))
	} else {
		instruction.WriteString("Return prose only. Respect the declared constraints and preserve the requested intent.")
	}
	return ResolvedProfile{Profile: profile, Styles: styles, InstructionText: instruction.String()}, nil
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
	if strings.TrimSpace(req.Query) == "" {
		return GenerateResponse{}, ErrEmptyQuery
	}
	effectiveCap := effectiveOutputCap(resolved.Profile)
	gwReq := GatewayRequest{Role: resolved.Profile.GatewayRole, Instruction: resolved.InstructionText, Query: req.Query, Strategy: resolved.Profile.Sampler.Kind, K: resolved.Profile.Sampler.K, Tau: resolved.Profile.Sampler.Tau, MaxOutputTokens: effectiveCap, TemperatureStance: resolved.Profile.Sampler.TemperatureStance, Locality: resolved.Profile.Locality, Negative: req.Negative}
	responses, err := s.gateway.Generate(ctx, gwReq)
	if err != nil {
		return GenerateResponse{}, err
	}
	shortSet := len(responses) < resolved.Profile.Sampler.K
	if len(responses) > resolved.Profile.Sampler.K {
		responses = responses[:resolved.Profile.Sampler.K]
	}
	selectionSeed := time.Now().UnixNano()
	round := Round{ID: uuid.NewString(), SessionID: session.ID, Strategy: resolved.Profile.Sampler, SelectionSeed: selectionSeed, SamplingKey: samplingKeyOf(resolved.Profile.Sampler, responses), NegativeContext: req.Negative}
	if shortSet && len(responses) == 0 {
		return GenerateResponse{}, fmt.Errorf("sampler returned no candidates")
	}
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
		capValue, capSource := response.MaxOutputTokensEffective, response.MaxOutputTokensSource
		if capValue == 0 && capSource == "" {
			capValue, capSource = effectiveCap, "profile"
		}
		candidate := Candidate{ID: id, RoundID: round.ID, Text: response.Text, SetIndex: i, Measurements: measurements[i], SetMeasurements: setMeasurements, Eligibility: gate(measurements[i], resolved.Profile.Constraints, response.Text), Provenance: Provenance{ProfileVersion: fmt.Sprintf("%s@%d", resolved.Profile.Key, resolved.Profile.Version), StyleVersions: styleVersions(resolved.Styles), Strategy: resolved.Profile.Sampler.Kind, StrategyParameters: resolved.Profile.Sampler, Provider: response.Provider, ResolvedModelRef: response.Model, GatewayRole: resolved.Profile.GatewayRole, TemperatureSent: response.Temperature, TemperatureSupport: response.TemperatureSupport, MaxOutputTokensEffective: capValue, MaxOutputTokensSource: capSource, InputTokens: response.InputTokens, OutputTokens: response.OutputTokens, CostMicros: cost, MachineGenerated: true, Disclosure: "machine_generated", ContextSnapshot: &ContextSnapshot{EstimatedTokens: estimateContextTokens(req.Query)}}}
		if resolved.Profile.Sampler.Kind == samplerVSStandard {
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
	selected := choose(candidates, resolved.Profile.SelectionPolicy, resolved.Profile.SelectionParams, round.SelectionSeed)
	response := GenerateResponse{Session: session, Round: round, Candidates: candidates}
	if shortSet {
		response.Degraded = &DegradedOutcome{Kind: "short_candidate_set", Reason: fmt.Sprintf("received %d of %d candidates", len(responses), resolved.Profile.Sampler.K), RequestedCandidates: resolved.Profile.Sampler.K, ReceivedCandidates: len(responses), MaxOutputTokensEffective: effectiveCap, MaxOutputTokensSource: "profile_derived"}
	}
	if selected != nil {
		response.Selected = selected
	}
	response.SelectedCandidates = coverageCandidates(candidates, resolved.Profile.SelectionPolicy, resolved.Profile.SelectionParams)
	if !req.IncludeCandidates {
		response.Candidates = nil
	}
	return response, nil
}

func effectiveOutputCap(profile Profile) int {
	cap := profile.Budget.MaxOutputTokens
	if cap <= 0 {
		cap = 8192
	}
	if profile.Sampler.Kind == samplerVSStandard && profile.Sampler.K > 1 {
		return cap * profile.Sampler.K
	}
	return cap
}

func estimateContextTokens(text string) int {
	count := len([]rune(strings.TrimSpace(text))) / 4
	if count < 1 && strings.TrimSpace(text) != "" {
		return 1
	}
	return count
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

// Reindex rescans one consumer's declaration directory. A scan is authoritative
// only over the subtree it walked: it may unregister a declaration it can prove
// is gone from that subtree, and it must leave every other consumer's records
// alone. Both halves of that were wrong before — an absent root scanned nothing
// and then unregistered every declaration in the database.
func (s *Service) Reindex(ctx context.Context, root string) ([]Declaration, error) {
	if strings.TrimSpace(root) == "" {
		// An omitted root means "this scenario's own declarations", which is what
		// a caller asking for a plain rescan wants. It never means "everything",
		// because a scan with no subtree can prove nothing missing.
		root = s.defaultDeclarationsRoot()
	}
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("%w: no root was given and no default is configured", ErrDeclarationRootMissing)
	}
	base := filepath.Join(root, ".vrooli", "prose-studio")
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		// A root that does not resolve is an operator or caller error. Reporting
		// it beats scanning zero files and calling every absent file deleted.
		return nil, fmt.Errorf("%w: %s", ErrDeclarationRootMissing, base)
	}
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
			// Only this scan's own subtree is in scope. Another consumer's
			// declaration is not missing merely because this scan did not look
			// at it, and OT-P0-025 has many consumers each owning their own.
			if !underRoot(base, path) {
				continue
			}
			// Confirm the absence rather than infer it. A file the walk skipped
			// for any reason is still on disk, and unregistering it would retire
			// a live record on the strength of a walk that never read it.
			if _, statErr := os.Stat(path); statErr == nil {
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
	if len(sections) == 0 {
		return s.generateDocument(ctx, doc)
	}
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

// generateDocument is the service-owned long-form path. The outline is a
// normal candidate round, followed by one passage-level session per section.
// Callers do not manufacture committed ids or context snapshots.
func (s *Service) generateDocument(ctx context.Context, doc Document) (Document, error) {
	if strings.TrimSpace(doc.ProfileKey) == "" {
		return Document{}, errors.New("document profile_key is required")
	}
	if err := s.saveJSON(ctx, "prose_documents", doc.ID, doc); err != nil {
		return Document{}, err
	}
	outline, err := s.Generate(ctx, GenerateRequest{ProfileKey: doc.ProfileKey, Query: "Create a concise ordered outline for: " + doc.Title, IncludeCandidates: true})
	if err != nil {
		return Document{}, err
	}
	outlineCandidate := outline.Selected
	if outlineCandidate == nil && len(outline.Candidates) > 0 {
		outlineCandidate = &outline.Candidates[0]
	}
	if outlineCandidate == nil || !outlineCandidate.Eligibility.Eligible {
		return Document{}, fmt.Errorf("outline candidate is ineligible: %s", candidateReason(outlineCandidate))
	}
	if err := s.commit(ctx, outline.Session, outlineCandidate.ID); err != nil {
		return Document{}, err
	}
	doc.OutlineID = outlineCandidate.ID
	intents := outlineIntents(outlineCandidate.Text)
	// A one-line provider answer is not an outline. Keep the service-owned
	// section contract stable so the document path still exercises independent
	// passage sessions and context snapshots.
	if len(intents) < 2 {
		intents = []string{"opening", "context", "evidence", "implications", "conclusion"}
	}
	sections := make([]Section, 0, len(intents))
	for position, intent := range intents {
		section := Section{ID: uuid.NewString(), DocumentID: doc.ID, Position: position, Intent: intent, ProfileKey: doc.ProfileKey, Context: s.buildContextSnapshot(ctx, doc, sections, intents[position+1:])}
		contextText, err := s.contextText(ctx, doc, sections, section.Context)
		if err != nil {
			return Document{}, err
		}
		if err := validateDynamicContext(contextText, doc.ProfileKey, s, ctx); err != nil {
			return Document{}, err
		}
		generated, err := s.Generate(ctx, GenerateRequest{ProfileKey: doc.ProfileKey, Query: intent + "\n\n" + contextText, IncludeCandidates: true})
		if err != nil {
			return Document{}, err
		}
		section.SessionID = generated.Session.ID
		selected := generated.Selected
		if selected == nil && len(generated.Candidates) > 0 {
			selected = &generated.Candidates[0]
		}
		if selected == nil || !selected.Eligibility.Eligible {
			return Document{}, fmt.Errorf("section %d candidate is ineligible: %s", position, candidateReason(selected))
		}
		if err := s.commitSectionCandidate(ctx, section.ID, selected.ID, section); err != nil {
			return Document{}, err
		}
		sections = append(sections, section)
	}
	doc.SectionIDs = make([]string, 0, len(sections))
	for _, section := range sections {
		doc.SectionIDs = append(doc.SectionIDs, section.ID)
	}
	doc.Status = "sectioned"
	if err := s.saveJSON(ctx, "prose_documents", doc.ID, doc); err != nil {
		return Document{}, err
	}
	return s.AssembleDocument(ctx, doc.ID)
}

func outlineIntents(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "-•*0123456789. "))
		if line != "" && len([]rune(line)) <= 180 {
			out = append(out, line)
		}
	}
	return out
}

func candidateReason(candidate *Candidate) string {
	if candidate == nil {
		return "no candidate returned"
	}
	if candidate.Eligibility.Reason != "" {
		return candidate.Eligibility.Reason
	}
	return "candidate not returned"
}

func (s *Service) commitSectionCandidate(ctx context.Context, sectionID, candidateID string, section Section) error {
	var candidate Candidate
	if err := s.loadJSON(ctx, "prose_candidates", candidateID, &candidate); err != nil {
		return err
	}
	if !candidate.Eligibility.Eligible {
		return fmt.Errorf("candidate %s is ineligible: %s", candidateID, candidate.Eligibility.Reason)
	}
	section.ID = sectionID
	section.CommittedCandidateID = candidateID
	if err := s.saveSection(ctx, section); err != nil {
		return err
	}
	return s.commit(ctx, Session{ID: section.SessionID, ProfileKey: section.ProfileKey}, candidateID)
}

func (s *Service) buildContextSnapshot(ctx context.Context, doc Document, prior []Section, following []string) ContextSnapshot {
	snapshot := ContextSnapshot{OutlineRef: doc.OutlineID, FollowingIntents: append([]string(nil), following...)}
	for _, section := range prior {
		snapshot.PriorSectionRefs = append(snapshot.PriorSectionRefs, section.ID)
	}
	return snapshot
}

func (s *Service) contextText(ctx context.Context, doc Document, prior []Section, snapshot ContextSnapshot) (string, error) {
	var parts []string
	parts = append(parts, "Selected outline: "+snapshot.OutlineRef)
	for _, section := range prior {
		var candidate Candidate
		if section.CommittedCandidateID == "" {
			continue
		}
		if err := s.loadJSON(ctx, "prose_candidates", section.CommittedCandidateID, &candidate); err != nil {
			return "", err
		}
		parts = append(parts, "Prior section "+section.ID+": "+candidate.Text)
	}
	if len(snapshot.FollowingIntents) > 0 {
		parts = append(parts, "Following section intents: "+strings.Join(snapshot.FollowingIntents, "; "))
	}
	return strings.Join(parts, "\n"), nil
}

func validateDynamicContext(contextText, profileKey string, service *Service, ctx context.Context) error {
	profile, err := service.latestProfile(ctx, profileKey)
	if err != nil {
		return err
	}
	if profile.ContextPolicy.DeclaredContextCeiling > 0 && estimateContextTokens(contextText)+profile.Budget.MaxOutputTokens > profile.ContextPolicy.DeclaredContextCeiling {
		return fmt.Errorf("%w: assembled section context exceeds profile ceiling", ErrContextInfeasible)
	}
	return nil
}

func (s *Service) AssembleDocument(ctx context.Context, documentID string) (Document, error) {
	var doc Document
	if err := s.loadJSON(ctx, "prose_documents", documentID, &doc); err != nil {
		return Document{}, err
	}
	var text strings.Builder
	var sectionTexts []string
	var structure []Section
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
		sectionTexts = append(sectionTexts, candidate.Text)
		structure = append(structure, section)
	}
	doc.AssembledText = text.String()
	doc.Status = "assembled"
	doc.Sections = structure
	profile, err := s.latestProfile(ctx, doc.ProfileKey)
	if err != nil {
		return Document{}, err
	}
	styleKey := doc.StyleKey
	if styleKey == "" && len(profile.StyleRefs) > 0 {
		styleKey = profile.StyleRefs[0]
	}
	var sectionScores []float64
	for _, sectionText := range sectionTexts {
		result, conformanceErr := s.Conformance(ctx, styleKey, sectionText)
		if conformanceErr != nil && styleKey == "" {
			result = map[string]any{"targets_met": true}
		}
		sectionScores = append(sectionScores, conformanceScore(result))
	}
	doc.StyleKey = styleKey
	doc.Coherence = map[string]any{"cross_section_repetition": textmetrics.CrossSectionRepetition(sectionTexts), "style_drift": textmetrics.StyleDrift(sectionScores), "basis": "textmetrics.CrossSectionRepetition and textmetrics.StyleDrift over committed section text"}
	if err := s.saveJSON(ctx, "prose_documents", doc.ID, doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

func conformanceScore(result map[string]any) float64 {
	if result == nil {
		return 0
	}
	verdicts, ok := result["verdicts"].(map[string]map[string]any)
	if !ok || len(verdicts) == 0 {
		if met, ok := result["targets_met"].(bool); ok && met {
			return 1
		}
		return 0
	}
	var met float64
	for _, verdict := range verdicts {
		if value, ok := verdict["met"].(bool); ok && value {
			met++
		}
	}
	return met / float64(len(verdicts))
}

func (s *Service) Conformance(ctx context.Context, styleKey, text string) (map[string]any, error) {
	style, err := s.resolveStyle(ctx, styleKey, map[string]bool{})
	if err != nil {
		return nil, err
	}
	metrics := textmetrics.Analyze(text, style.Lexicon)
	missed := map[string]float64{}
	verdicts := map[string]map[string]any{}
	for key, target := range style.Targets {
		actual, known := targetValue(metrics, key)
		met := known && actual >= target
		if !met {
			missed[key] = target
		}
		verdicts[key] = map[string]any{"met": met, "actual": actual, "target": target, "known": known}
	}
	return map[string]any{"style": style.Key, "version": style.Version, "targets_met": len(missed) == 0, "missed": missed, "verdicts": verdicts, "anti_pattern_spans": metrics.LexiconFlags}, nil
}

func targetValue(metrics textmetrics.Metrics, key string) (float64, bool) {
	switch strings.ToLower(key) {
	case "mattr":
		return metrics.MATTR, true
	case "type_token_ratio":
		return metrics.TypeTokenRatio, true
	case "compression_ratio":
		return metrics.CompressionRatio, true
	case "self_repetition":
		return metrics.SelfRepetition, true
	case "burstiness":
		return metrics.Burstiness, true
	case "flesch_kincaid":
		return metrics.Readability.FleschKincaid, true
	case "dale_chall":
		return metrics.Readability.DaleChall, true
	case "gunning_fog":
		return metrics.Readability.GunningFog, true
	default:
		return 0, false
	}
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

// validateStaticFeasibility refuses a profile whose worst-case section cannot
// fit the resolved model's declared context window.
//
// The window is undeclared today, and an undeclared window can refuse nothing:
// there is no ceiling to exceed. This previously compared against a literal
// 32768, which made the check report a verdict it had not earned — the worst
// outcome available, because a profile that "passed" was never measured. The
// refusal path stays live and takes effect the moment a real window arrives.
func (s *Service) validateStaticFeasibility(p Profile) error {
	if p.ContextPolicy.DeclaredContextCeiling <= 0 {
		return nil
	}
	worst := p.ContextPolicy.FullTextTokenBudget + p.ContextPolicy.SummarizeBeyond + p.Budget.MaxOutputTokens
	if worst > p.ContextPolicy.DeclaredContextCeiling {
		return fmt.Errorf("%w: profile worst-case requires %d tokens, profile ceiling is %d", ErrContextInfeasible, worst, p.ContextPolicy.DeclaredContextCeiling)
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

func gate(measurement any, c Constraints, text string) Eligibility {
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
	grade := m.Readability.FleschKincaid
	if c.MinGrade > 0 && grade < c.MinGrade {
		return Eligibility{false, fmt.Sprintf("min_grade:%.2f", c.MinGrade)}
	}
	if c.MaxGrade > 0 && grade > c.MaxGrade {
		return Eligibility{false, fmt.Sprintf("max_grade:%.2f", c.MaxGrade)}
	}
	if c.RequiredFormat != "" && !matchesRequiredFormat(c.RequiredFormat, m, text) {
		return Eligibility{false, "required_format:" + c.RequiredFormat}
	}
	return Eligibility{Eligible: true}
}

func choose(candidates []Candidate, policy string, params map[string]float64, seed int64) *Candidate {
	eligible := make([]Candidate, 0, len(candidates))
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
		if seed == 0 {
			seed = 1
		}
		// This randomness is deliberately deterministic and non-secret: the seed
		// is part of selection reproducibility, never an authentication value.
		picked := rand.New(rand.NewSource(seed)).Intn(len(eligible)) // #nosec G404 -- seeded selection policy, not security randomness
		return &eligible[picked]
	case "threshold_then_rarest":
		var best Candidate
		bestScore := -1.0
		threshold := 0.0
		if params != nil {
			threshold = params["threshold"]
		}
		for _, c := range eligible {
			score := candidateRarity(c)
			if score < threshold {
				continue
			}
			if score > bestScore {
				best, bestScore = c, score
			}
		}
		if best.ID == "" && len(eligible) > 0 {
			best = eligible[0]
		}
		return &best
	default:
		return &eligible[0]
	}
}

func candidateRarity(c Candidate) float64 {
	var m textmetrics.SetMetrics
	raw, _ := json.Marshal(c.SetMeasurements)
	_ = json.Unmarshal(raw, &m)
	for i, row := range m.PairwiseSimilarity {
		if i >= len(m.PairwiseSimilarity) || len(row) == 0 {
			continue
		}
		if c.SetIndex < len(m.PairwiseSimilarity) {
			row = m.PairwiseSimilarity[c.SetIndex]
			var total float64
			for _, similarity := range row {
				total += similarity
			}
			return 1 - total/float64(len(row))
		}
		_ = i
	}
	return 1 - m.MeanSimilarity
}

func coverageCandidates(candidates []Candidate, policy string, params map[string]float64) []Candidate {
	if policy != "coverage" || len(candidates) == 0 {
		return nil
	}
	bins := 3
	if params != nil && params["bins"] > 0 {
		bins = int(params["bins"])
	}
	if bins > len(candidates) {
		bins = len(candidates)
	}
	out := make([]Candidate, 0, bins)
	for i := 0; i < bins; i++ {
		index := i * (len(candidates) - 1) / maxInt(1, bins-1)
		out = append(out, candidates[index])
	}
	return out
}

func matchesRequiredFormat(format string, metrics textmetrics.Metrics, text string) bool {
	raw := []byte(text)
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "paragraph", "prose":
		return metrics.SentenceCount > 0 && !strings.Contains(string(raw), "\n-")
	case "bullet_list", "bullets":
		return strings.Contains(string(raw), "\n-") || strings.HasPrefix(strings.TrimSpace(string(raw)), "-")
	case "markdown":
		return strings.Contains(string(raw), "#") || strings.Contains(string(raw), "**")
	case "json":
		var value any
		return json.Unmarshal(raw, &value) == nil
	default:
		return true
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
