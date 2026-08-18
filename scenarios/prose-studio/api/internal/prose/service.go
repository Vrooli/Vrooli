package prose

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	ErrStyleResolutionConflict  = errors.New("style_resolution_conflict")
	ErrProfileDeclared          = errors.New("profile_is_declared")
	ErrProfileUnregistered      = errors.New("profile_unregistered")
	ErrDeclarationCollision     = errors.New("declaration_key_collision")
	ErrBudgetExceeded           = errors.New("session_budget_exceeded")
	ErrContextInfeasible        = errors.New("context_window_infeasible")
	ErrEmptyQuery               = errors.New("query_is_required")
	ErrUnknownLocality          = errors.New("unknown_locality")
	ErrUnknownTemperature       = errors.New("unknown_temperature_stance")
	ErrMalformedCandidateSet    = errors.New("malformed_candidate_set")
	ErrMalformedOutline         = errors.New("malformed_outline")
	ErrDeclarationRootMissing   = errors.New("declaration_root_missing")
	ErrPromptContainsIdentifier = errors.New("prompt_contains_record_identifier")
)

var recordIdentifierPattern = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b`)

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
	samplerComposite  = "composite"
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
		MeasurementTier:   "deterministic",
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

// The gateway schema subset does not enforce array cardinality. Fixed object
// slots make the three-section blog contract enforceable at generation time;
// decodeOutline also accepts the legacy array form for injected gateways.
// outlineSchema describes an ordered outline. The previous form was an object
// with exactly section_1, section_2 and section_3 as required properties, which
// pinned every document to three sections in the one place a prompt cannot
// argue with: a model asked in prose for four sections still returned three,
// because the schema required exactly three named keys.
//
// The length band is deliberately NOT expressed here. minItems and maxItems are
// outside the gateway's enforceable schema subset and are refused rather than
// ignored, so the band lives in the outline instruction and is enforced by
// decodeOutline against what actually came back.
const outlineSchema = `{"type":"array","items":{"type":"object","properties":{"intent":{"type":"string"},"summary":{"type":"string"},"target_words":{"type":"integer"}},"required":["intent","summary","target_words"]}}`

func gatewaySchema(req GatewayRequest) string {
	if strings.TrimSpace(req.SchemaJSON) != "" {
		return req.SchemaJSON
	}
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
		// Structured calls such as the typed outline deliberately return an
		// object/array as the candidate itself. Preserve that JSON verbatim so
		// the caller can decode its declared shape after the normal candidate
		// lifecycle, rather than forcing every schema into a prose string.
		if strings.TrimSpace(req.SchemaJSON) != "" {
			if !json.Valid([]byte(valueJSON)) {
				return nil, nil, fmt.Errorf("decode structured candidate: invalid JSON")
			}
			return []string{valueJSON}, []int{0}, nil
		}
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

// EmbeddingGateway is optional so lexical-only test gateways remain tiny while
// production HTTPGateway exposes the gateway-owned semantic tier.
type EmbeddingGateway interface {
	Embed(context.Context, EmbeddingRequest) (EmbeddingResponse, error)
}

type EmbeddingRequest struct {
	Role  string   `json:"role"`
	Texts []string `json:"texts"`
}

type EmbeddingResponse struct {
	Vectors    [][]float64 `json:"vectors"`
	Provider   string      `json:"provider"`
	Model      string      `json:"model"`
	Dimension  int         `json:"dimension"`
	CostMicros int64       `json:"cost_micros"`
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
		instruction += "\nDo not repeat these previously reviewed passages. Preserve the request's intent while changing the angle, structure, or assumption:\n"
		for _, text := range append(append([]string{}, req.Negative.Pinned...), req.Negative.Rejected...) {
			if strings.TrimSpace(text) != "" {
				if recordIdentifierPattern.MatchString(text) {
					return nil, fmt.Errorf("%w: negative conditioning", ErrPromptContainsIdentifier)
				}
				instruction += "- " + text + "\n"
			}
		}
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

func (g HTTPGateway) Embed(ctx context.Context, req EmbeddingRequest) (EmbeddingResponse, error) {
	if strings.TrimSpace(g.BaseURL) == "" {
		return EmbeddingResponse{}, errors.New("ai-gateway endpoint is not configured")
	}
	if len(req.Texts) == 0 {
		return EmbeddingResponse{}, errors.New("embedding texts are required")
	}
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	connectClient := inferenceconnect.NewInferenceServiceClient(client, g.BaseURL)
	resp, err := connectClient.Embed(ctx, connect.NewRequest(&inferencev1.EmbedRequest{Role: req.Role, Texts: req.Texts}))
	if err != nil {
		return EmbeddingResponse{}, fmt.Errorf("ai-gateway embedding request: %w", err)
	}
	if resp.Msg.GetError() != nil {
		return EmbeddingResponse{}, fmt.Errorf("ai-gateway embedding: %s", resp.Msg.GetError().GetMessage())
	}
	out := EmbeddingResponse{Provider: resp.Msg.GetProvider(), Model: resp.Msg.GetModel(), Dimension: int(resp.Msg.GetDimension()), CostMicros: resp.Msg.GetUsage().GetCostMicros(), Vectors: make([][]float64, 0, len(resp.Msg.GetVectors()))}
	for _, vector := range resp.Msg.GetVectors() {
		if len(vector.GetValues()) != out.Dimension {
			return EmbeddingResponse{}, errors.New("embedding_dimension_mismatch")
		}
		out.Vectors = append(out.Vectors, append([]float64(nil), vector.GetValues()...))
	}
	if len(out.Vectors) != len(req.Texts) {
		return EmbeddingResponse{}, errors.New("embedding_vector_count_mismatch")
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
		// Deliberately not the output cap. How much prior prose a section prompt
		// may carry is an input-context question; how long a section may be is an
		// output question. Seeding one from the other tied a profile's memory of
		// its own document to its section length, so a profile that wrote short
		// sections summarised every prior section through a small model exactly
		// when those sections were short enough to carry whole.
		profile.ContextPolicy.FullTextTokenBudget = defaultFullTextTokenBudget
	}
	if profile.ContextPolicy.SummarizeBeyond <= 0 {
		profile.ContextPolicy.SummarizeBeyond = profile.ContextPolicy.FullTextTokenBudget
	}
	if profile.Sampler.Kind != samplerDirect && profile.Sampler.Kind != samplerVSStandard && profile.Sampler.Kind != samplerComposite {
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
			direction := "at_least"
			if style.TargetDirections != nil && style.TargetDirections[target] != "" {
				direction = style.TargetDirections[target]
			}
			instruction.WriteString("Target ")
			instruction.WriteString(target)
			if direction == "at_most" {
				instruction.WriteString(" <= ")
			} else {
				instruction.WriteString(" >= ")
			}
			instruction.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
			instruction.WriteByte('\n')
		}
		for _, a := range style.AntiPatterns {
			instruction.WriteString("Avoid the marketing failure mode ")
			instruction.WriteString(a)
			instruction.WriteString(": ")
			instruction.WriteString(antiPatternDefinition(a))
			instruction.WriteByte('\n')
		}
	}
	if profile.Constraints.MinWords > 0 || profile.Constraints.MaxWords > 0 {
		instruction.WriteString("Length constraint: ")
		if profile.Constraints.MinWords > 0 {
			fmt.Fprintf(&instruction, "at least %d words", profile.Constraints.MinWords)
		}
		if profile.Constraints.MinWords > 0 && profile.Constraints.MaxWords > 0 {
			instruction.WriteString(" and ")
		}
		if profile.Constraints.MaxWords > 0 {
			fmt.Fprintf(&instruction, "at most %d words", profile.Constraints.MaxWords)
		}
		instruction.WriteString(".\n")
	} else {
		fmt.Fprintf(&instruction, "Length target: approximately %d words; do not answer with a short summary.\n", effectiveOutputCap(profile))
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

func antiPatternDefinition(name string) string {
	definitions := map[string]string{
		"hype drift":                            "do not promise unshipped features or unverifiable dates",
		"voice drift":                           "keep a concrete builder voice instead of corporate marketing language",
		"hallucinated engagement metrics":       "never invent numbers; label measured, estimated, aspirational, or pending telemetry claims",
		"paywall framing":                       "do not describe subscriptions as gating core capabilities",
		"OSS-as-leak framing":                   "describe self-hosting as a legitimate sovereignty path, not lost revenue",
		"coverage-gap ignorance":                "acknowledge stale or missing coverage before proposing another campaign",
		"acquisition-only hypothesis":           "pair acquisition ideas with retention reasoning or explicitly mark awareness-only",
		"capability-workaround without gap":     "tie manual workarounds to a documented capability gap",
		"narrative-flatness":                    "shape the essay as hook, introduction, body, and conclusion rather than a changelog",
		"internal-vocabulary-leakage":           "translate internal names before using them for readers",
		"missing-introduction-on-first-mention": "introduce each named system and its purpose on first mention",
		"what-without-why":                      "explain why each change mattered, not only what changed",
		"persona-disclosure-violation":          "label substantial AI-generated persona content and sponsorships as required",
		"real-person-impersonation":             "do not resemble or impersonate an identifiable real person",
		"fabricated-real-customer-testimonial":  "never invent a real customer identity or testimonial",
		"recommendation-framing-without-basis":  "do not attribute recommendations to an unidentified third party",
		"regulated-domain-advice-by-persona":    "avoid medical, financial, or legal advice in persona voice",
	}
	if definition, ok := definitions[name]; ok {
		return definition
	}
	return "keep claims grounded in supplied evidence and explain their reader value"
}

func (s *Service) Registry() Registry {
	return Registry{
		Samplers: []RegistryKind{{Kind: "direct", Description: "One gateway call per candidate.", ParameterSchema: map[string]any{"k": "integer >= 1", "temperature_stance": "string"}}, {Kind: "vs_standard", Description: "One call enumerating k candidates under tau.", ParameterSchema: map[string]any{"k": "integer >= 2", "tau": "number 0..1", "temperature_stance": "string"}}},
		Policies: []RegistryKind{{Kind: "take_first", Description: "Measurement control: first eligible.", ParameterSchema: map[string]any{}}, {Kind: "sample_uniform", Description: "Uniform among eligible candidates.", ParameterSchema: map[string]any{}}, {Kind: "threshold_then_rarest", Description: "Eligible candidate with greatest lexical rarity.", ParameterSchema: map[string]any{"threshold": "number"}}, {Kind: "coverage", Description: "Spread for human review; never a quality order.", ParameterSchema: map[string]any{"bins": "integer >= 1"}}, {Kind: "human_pick", Description: "Return the full spread for operator choice.", ParameterSchema: map[string]any{}}},
		Metrics:  []RegistryKind{{Kind: "deterministic", Description: "Reproducible lexical and readability metrics.", ParameterSchema: map[string]any{"lexicon": "string[]"}}},
		Transforms: []RegistryKind{
			{Kind: "reading_level", Description: "Rewrite a candidate toward a target reading grade.", ParameterSchema: map[string]any{"target_grade": "number"}},
			{Kind: "elaboration", Description: "Expand a candidate toward a target word count while preserving its claims.", ParameterSchema: map[string]any{"target_words": "integer >= 1"}},
			{Kind: "simplification", Description: "Shorten a candidate without dropping its declared claim.", ParameterSchema: map[string]any{"target_words": "integer >= 1"}},
		},
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
	resolved.Profile.Constraints = profile.Constraints
	resolved.Profile.MeasurementTiers = profile.MeasurementTiers
	resolved.Profile.Coherence = profile.Coherence
	// Selection policy, budget, and composition are carried too. Omitting the
	// selection policy meant a caller could compute a per-section policy, pass
	// it in, and silently get the stored profile's policy instead: the section
	// path asked for continuation selection and received the document's
	// ideation policy on every call.
	resolved.Profile.SelectionPolicy = profile.SelectionPolicy
	resolved.Profile.SelectionParams = profile.SelectionParams
	resolved.Profile.Budget = profile.Budget
	resolved.Profile.Composition = profile.Composition
	if profile.Constraints.MinWords > 0 || profile.Constraints.MaxWords > 0 || profile.Constraints.RequiredFormat != "" {
		resolved.InstructionText += fmt.Sprintf(
			"\nHARD OUTPUT CONSTRAINTS: minimum %d words, maximum %d words, required format %q. Stay within these bounds; do not explain the constraints.",
			profile.Constraints.MinWords,
			profile.Constraints.MaxWords,
			profile.Constraints.RequiredFormat,
		)
	}
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
	gwReq := GatewayRequest{Role: resolved.Profile.GatewayRole, Instruction: resolved.InstructionText, Query: req.Query, SchemaJSON: req.SchemaJSON, Strategy: resolved.Profile.Sampler.Kind, K: resolved.Profile.Sampler.K, Tau: resolved.Profile.Sampler.Tau, MaxOutputTokens: effectiveCap, TemperatureStance: resolved.Profile.Sampler.TemperatureStance, Locality: resolved.Profile.Locality, Negative: req.Negative}
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
	measurements, lexicalSet := textmetrics.AnalyzeSet(texts, mergedLexicon(resolved.Styles))
	setMeasurements := lexicalSet
	if profileRequestsSemantic(resolved.Profile) {
		if embedder, ok := s.gateway.(EmbeddingGateway); ok {
			embedded, embedErr := embedder.Embed(ctx, EmbeddingRequest{Role: "embedding.default", Texts: texts})
			if embedErr == nil {
				semanticItems, candidateSet, semanticErr := textmetrics.AnalyzeSetSemantic(texts, embedded.Vectors)
				if semanticErr == nil {
					setMeasurements = candidateSet
					measurements = semanticItems
					round.SamplingKey.MeasurementTier = "deterministic_and_semantic"
					round.MeasurementBasis = candidateSet.Basis
					round.SemanticSetMeasurements = candidateSet
				} else {
					round.MeasurementFallback = semanticErr.Error()
				}
			} else {
				round.MeasurementFallback = "embedding_unavailable: " + embedErr.Error()
			}
		} else {
			round.MeasurementFallback = "embedding_unavailable: gateway has no embedding surface"
		}
	}
	round.LexicalSetMeasurements = lexicalSet
	if round.MeasurementBasis == "" {
		round.MeasurementBasis = setMeasurements.Basis
	}
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
	selected := choose(candidates, resolved.Profile.SelectionPolicy, resolved.Profile.SelectionParams, round.SelectionSeed, req.Selection)
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

func profileRequestsSemantic(profile Profile) bool {
	for _, tier := range profile.MeasurementTiers {
		if tier == "deterministic_and_semantic" || tier == "semantic" {
			return true
		}
	}
	return false
}

func effectiveOutputCap(profile Profile) int {
	cap := profile.Sampler.MaxOutputTokens
	if cap <= 0 {
		cap = profile.Budget.MaxOutputTokens
	}
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
		if session.Status == "abandoned" || session.Status == "committed" {
			return Session{}, fmt.Errorf("cannot refine %s session", session.Status)
		}
		if candidateID == "" {
			return Session{}, errors.New("refine requires a candidate id")
		}
		operation := "reading_level"
		refined, err := s.TransformCandidate(ctx, candidateID, operation, map[string]any{"target_grade": 8})
		if err != nil {
			return Session{}, err
		}
		session.Pinned = appendUnique(session.Pinned, refined.ID)
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

// TransformCandidate applies one typed, gateway-owned rewrite and records its
// derivation on the resulting candidate. The source candidate remains intact;
// transforms are additions to the candidate graph, never silent replacement.
func (s *Service) TransformCandidate(ctx context.Context, candidateID, operation string, parameters map[string]any) (Candidate, error) {
	var source Candidate
	if err := s.loadJSON(ctx, "prose_candidates", candidateID, &source); err != nil {
		return Candidate{}, err
	}
	instruction := map[string]string{
		"reading_level":  "Rewrite this prose for the requested reading grade while preserving every claim.",
		"elaboration":    "Expand this prose to the requested word count while preserving every claim.",
		"simplification": "Shorten this prose to the requested word count without dropping any declared claim.",
	}[operation]
	if instruction == "" {
		return Candidate{}, fmt.Errorf("unknown transform operation %q", operation)
	}
	params, err := json.Marshal(parameters)
	if err != nil {
		return Candidate{}, err
	}
	responses, err := s.gateway.Generate(ctx, GatewayRequest{Role: "write.default", Instruction: instruction, Query: string(params) + "\n\nSOURCE:\n" + source.Text, Strategy: samplerDirect, K: 1, MaxOutputTokens: source.Provenance.MaxOutputTokensEffective})
	if err != nil {
		return Candidate{}, err
	}
	if len(responses) == 0 || strings.TrimSpace(responses[0].Text) == "" {
		return Candidate{}, errors.New("transform returned no candidate")
	}
	response := responses[0]
	derived := Candidate{ID: uuid.NewString(), RoundID: source.RoundID, DerivedFrom: []string{source.ID}, Text: response.Text, SetIndex: source.SetIndex, Measurements: textmetrics.Analyze(response.Text, nil), SetMeasurements: source.SetMeasurements, Provenance: source.Provenance, Eligibility: Eligibility{Eligible: true}, Transform: &Transform{Operation: operation, Parameters: parameters, SourceCandidate: source.ID, GatewayRole: "write.default", CreatedAt: s.now()}}
	derived.Provenance.Provider, derived.Provenance.ResolvedModelRef, derived.Provenance.CostMicros = response.Provider, response.Model, response.CostMicros
	if err := s.saveJSON(ctx, "prose_candidates", derived.ID, derived); err != nil {
		return Candidate{}, err
	}
	var round Round
	if err := s.loadJSON(ctx, "prose_rounds", source.RoundID, &round); err == nil {
		round.CandidateIDs = append(round.CandidateIDs, derived.ID)
		_ = s.saveJSON(ctx, "prose_rounds", round.ID, round)
	}
	return derived, nil
}

func PlanAxisCells(space AxisSpace) []AxisCell {
	if len(space.Axes) == 0 {
		return nil
	}
	cells := []AxisCell{{Variants: map[string]string{}}}
	for _, axis := range space.Axes {
		var next []AxisCell
		for _, cell := range cells {
			for _, variant := range axis.Variants {
				variants := map[string]string{}
				for key, value := range cell.Variants {
					variants[key] = value
				}
				variants[axis.Name] = variant
				next = append(next, AxisCell{Variants: variants})
			}
		}
		cells = next
	}
	for i := range cells {
		parts := make([]string, 0, len(cells[i].Variants))
		for _, axis := range space.Axes {
			parts = append(parts, axis.Name+"="+cells[i].Variants[axis.Name])
		}
		cells[i].Key = strings.Join(parts, ";")
	}
	return cells
}

// GenerateComposite executes the composite sampler as a bounded Cartesian
// plan. Each cell gets its own ordinary generation round, so provider/model,
// cost, and measurement evidence remain attached to the candidate rather than
// being flattened into an opaque aggregate.
func (s *Service) GenerateComposite(ctx context.Context, req GenerateRequest, space AxisSpace) (CompositeGeneration, error) {
	cells := PlanAxisCells(space)
	if len(cells) == 0 {
		return CompositeGeneration{}, errors.New("composite_sampler_requires_nonempty_axis_space")
	}
	profile, err := s.latestProfile(ctx, req.ProfileKey)
	if err != nil {
		return CompositeGeneration{}, err
	}
	if profile.Sampler.Kind != samplerComposite {
		return CompositeGeneration{}, fmt.Errorf("composite_sampler_requires_composite_profile: %s", profile.Sampler.Kind)
	}
	if profile.Sampler.K < 1 {
		profile.Sampler.K = 1
	}
	if profile.Sampler.K == 1 {
		profile.Sampler.Kind = samplerDirect
	} else {
		profile.Sampler.Kind = samplerVSStandard
	}
	result := CompositeGeneration{Candidates: make([]Candidate, 0, len(cells)*profile.Sampler.K), Rounds: make([]Round, 0, len(cells)), Sessions: make([]Session, 0, len(cells))}
	for _, cell := range cells {
		variants := make([]string, 0, len(cell.Variants))
		for _, axis := range space.Axes {
			variants = append(variants, axis.Name+"="+cell.Variants[axis.Name])
		}
		cellReq := req
		cellReq.SessionID = ""
		cellReq.IncludeCandidates = true
		cellReq.Query = req.Query + "\n\nAXIS CELL: " + strings.Join(variants, "; ")
		generated, generateErr := s.generateWithProfile(ctx, cellReq, profile)
		if generateErr != nil {
			return CompositeGeneration{}, fmt.Errorf("composite cell %s: %w", cell.Key, generateErr)
		}
		result.Rounds = append(result.Rounds, generated.Round)
		result.Sessions = append(result.Sessions, generated.Session)
		for _, candidate := range generated.Candidates {
			candidate.AxisCell = cell.Key
			if saveErr := s.saveJSON(ctx, "prose_candidates", candidate.ID, candidate); saveErr != nil {
				return CompositeGeneration{}, saveErr
			}
			result.Candidates = append(result.Candidates, candidate)
		}
	}
	result.Coverage = ComputeCellCoverage(space, result.Candidates)
	return result, nil
}

func ComputeCellCoverage(space AxisSpace, candidates []Candidate) CellCoverage {
	planned := PlanAxisCells(space)
	coveredSet := map[string]bool{}
	for _, candidate := range candidates {
		if candidate.AxisCell != "" {
			coveredSet[candidate.AxisCell] = true
		}
	}
	coverage := CellCoverage{Planned: planned}
	for _, cell := range planned {
		if coveredSet[cell.Key] {
			coverage.Covered = append(coverage.Covered, cell.Key)
		} else {
			coverage.Missed = append(coverage.Missed, cell.Key)
		}
	}
	return coverage
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
	negative, err := s.resolveNegativeContext(ctx, session)
	if err != nil {
		return GenerateResponse{}, err
	}
	return s.generateWithProfile(ctx, GenerateRequest{ProfileKey: session.ProfileKey, Query: session.Query, SessionID: session.ID, IncludeCandidates: includeCandidates, Negative: negative}, profile)
}

func (s *Service) resolveNegativeContext(ctx context.Context, session Session) (NegativeContext, error) {
	resolve := func(ids []string) ([]string, error) {
		out := make([]string, 0, len(ids))
		for _, id := range ids {
			var candidate Candidate
			if err := s.loadJSON(ctx, "prose_candidates", id, &candidate); err != nil {
				return nil, err
			}
			out = append(out, candidate.Text)
		}
		return out, nil
	}
	pinned, err := resolve(session.Pinned)
	if err != nil {
		return NegativeContext{}, err
	}
	rejected, err := resolve(session.Rejected)
	if err != nil {
		return NegativeContext{}, err
	}
	return NegativeContext{Pinned: pinned, Rejected: rejected}, nil
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
	distanceBasis := "lexical"
	var candidateSet textmetrics.SetMetrics
	if raw, marshalErr := json.Marshal(candidate.SetMeasurements); marshalErr == nil {
		_ = json.Unmarshal(raw, &candidateSet)
		if strings.Contains(candidateSet.Basis, "semantic") {
			distanceBasis = "semantic"
		}
	}
	event := SelectionEvent{ID: uuid.NewString(), SessionID: session.ID, CandidateID: candidateID, ConsideredCandidateIDs: roundCandidates(ctx, s.db, candidate.RoundID), Measurements: candidate.Measurements, DistanceBasis: distanceBasis, CreatedAt: s.now()}
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

// ResumeDocument continues a document from its durable outline and committed
// sections. It never regenerates a section that already has a committed
// candidate.
func (s *Service) ResumeDocument(ctx context.Context, documentID string) (Document, error) {
	var doc Document
	if err := s.loadJSON(ctx, "prose_documents", documentID, &doc); err != nil {
		return Document{}, err
	}
	return s.generateDocument(ctx, doc)
}

func (s *Service) resolveDocumentProfiles(ctx context.Context, doc Document) (Profile, Profile, error) {
	sectionKey := doc.ProfileKey
	sectionProfile, err := s.latestProfile(ctx, sectionKey)
	if err != nil {
		return Profile{}, Profile{}, err
	}
	if sectionProfile.SectionProfileKey != "" {
		sectionKey = sectionProfile.SectionProfileKey
		sectionProfile, err = s.latestProfile(ctx, sectionKey)
		if err != nil {
			return Profile{}, Profile{}, err
		}
	}
	outlineKey := doc.OutlineProfileKey
	if outlineKey == "" {
		outlineKey = sectionProfile.OutlineProfileKey
	}
	outlineProfile := sectionProfile
	if outlineKey != "" {
		outlineProfile, err = s.latestProfile(ctx, outlineKey)
		if err != nil {
			return Profile{}, Profile{}, err
		}
	}
	outlineProfile.Sampler.Kind = samplerDirect
	outlineProfile.Sampler.K = 1
	outlineProfile.SelectionPolicy = "take_first"
	return outlineProfile, sectionProfile, nil
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
	outlineProfile, sectionProfile, err := s.resolveDocumentProfiles(ctx, doc)
	if err != nil {
		return Document{}, err
	}
	if doc.OutlineID == "" || len(doc.Outline) == 0 {
		outlineProfileKey := outlineProfile.Key
		outlineProfile.Constraints = Constraints{}
		overallMinWords, overallMaxWords := sectionProfile.Constraints.MinWords, sectionProfile.Constraints.MaxWords
		if overallMaxWords <= 0 {
			overallMaxWords = 1400
		}
		if overallMinWords <= 0 {
			overallMinWords = maxInt(80, overallMaxWords/2)
		}
		// Leave a production margin above the declared floor: generated prose
		// naturally varies below outline targets, and a target sum close to the
		// floor can fail the assembled article by a material amount.
		if overallMinWords+300 < overallMaxWords {
			overallMinWords += 300
		}
		sectionCount, minSections, maxSections := resolveSectionPlan(doc, sectionProfile, overallMaxWords)
		outlineQuery := fmt.Sprintf("Create a concise ordered outline as a JSON array of %d sections for: %s. Each section must carry a distinct claim and advance the argument; no two sections may restate the same point in different words. The %d target_words values must sum to between %d and %d words for the assembled article.", sectionCount, doc.Title, sectionCount, overallMinWords, overallMaxWords)
		outline, err := s.generateWithProfile(ctx, GenerateRequest{ProfileKey: outlineProfileKey, Query: outlineQuery, SchemaJSON: outlineSchema, IncludeCandidates: true}, outlineProfile)
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
		parsedOutline, parseErr := decodeOutline(outlineCandidate.Text, minSections, maxSections)
		if parseErr != nil {
			// A weak provider may satisfy the schema while returning only one
			// section because minItems is intentionally outside the gateway's
			// enforceable subset. Retry with the named structural requirement;
			// never manufacture a fallback outline in the service.
			outline, err = s.generateWithProfile(ctx, GenerateRequest{ProfileKey: outlineProfileKey, Query: outlineQuery + fmt.Sprintf(" Return a JSON array of between %d and %d ordered section objects; do not return a single object and do not return fewer than %d.", minSections, maxSections, minSections), SchemaJSON: outlineSchema, IncludeCandidates: true}, outlineProfile)
			if err != nil {
				return Document{}, err
			}
			outlineCandidate = outline.Selected
			if outlineCandidate == nil && len(outline.Candidates) > 0 {
				outlineCandidate = &outline.Candidates[0]
			}
			if outlineCandidate == nil || !outlineCandidate.Eligibility.Eligible {
				return Document{}, fmt.Errorf("outline retry is ineligible: %s", candidateReason(outlineCandidate))
			}
			parsedOutline, parseErr = decodeOutline(outlineCandidate.Text, minSections, maxSections)
		}
		if parseErr != nil {
			return Document{}, parseErr
		}
		if err := s.commit(ctx, outline.Session, outlineCandidate.ID); err != nil {
			return Document{}, err
		}
		doc.OutlineID = outlineCandidate.ID
		doc.OutlineText = outlineCandidate.Text
		doc.OutlineProfileKey = outlineProfile.Key
		doc.OutlineProfileVersion = fmt.Sprintf("%s@%d", outlineProfile.Key, outlineProfile.Version)
		doc.Outline = parsedOutline
		doc.SectionCount = len(parsedOutline)
	}
	doc.SectionProfileVersion = fmt.Sprintf("%s@%d", sectionProfile.Key, sectionProfile.Version)
	sections := make([]Section, 0, len(doc.SectionIDs))
	for _, id := range doc.SectionIDs {
		var existing Section
		if err := s.loadJSON(ctx, "prose_sections", id, &existing); err == nil && existing.CommittedCandidateID != "" {
			sections = append(sections, existing)
		}
	}
	if err := s.saveJSON(ctx, "prose_documents", doc.ID, doc); err != nil {
		return Document{}, err
	}
	intents := make([]string, len(doc.Outline))
	for i := range doc.Outline {
		intents[i] = doc.Outline[i].Intent
	}
	for position, planned := range doc.Outline {
		if position < len(sections) && sections[position].CommittedCandidateID != "" {
			continue
		}
		section, err := s.produceSection(ctx, doc, sectionProfile, position, planned, sections, intents[position+1:], len(intents), NegativeContext{}, "")
		if err != nil {
			return Document{}, err
		}
		sections = append(sections, section)
		doc.SectionIDs = append(doc.SectionIDs, section.ID)
		doc.Sections = append(doc.Sections, section)
		if err := s.saveJSON(ctx, "prose_documents", doc.ID, doc); err != nil {
			return Document{}, err
		}
	}
	doc.Status = "sectioned"
	if err := s.saveJSON(ctx, "prose_documents", doc.ID, doc); err != nil {
		return Document{}, err
	}
	assembled, err := s.AssembleDocument(ctx, doc.ID)
	if err != nil {
		return Document{}, err
	}
	return s.repairDocument(ctx, assembled, sectionProfile, intents)
}

// produceSection draws, gates, selects, and commits the candidate set for one
// position in a document. It is shared by first assembly and by repair so a
// regenerated section is produced under exactly the same band, format, and
// selection rules as an original one; the previous inline body could only be
// reached by the first path, which is how a repair path would have drifted.
func (s *Service) produceSection(ctx context.Context, doc Document, sectionProfile Profile, position int, planned OutlineSection, prior []Section, followingIntents []string, totalSections int, negative NegativeContext, reuseID string) (Section, error) {
	id := reuseID
	if strings.TrimSpace(id) == "" {
		id = uuid.NewString()
	}
	section := Section{ID: id, DocumentID: doc.ID, Position: position, Intent: planned.Intent, Summary: planned.Summary, TargetWords: planned.TargetWords, ProfileKey: sectionProfile.Key, Context: s.buildContextSnapshot(ctx, doc, prior, followingIntents)}
	preparedContext, err := s.prepareContextSnapshot(ctx, doc, prior, section.Context)
	if err != nil {
		return Section{}, err
	}
	section.Context = preparedContext
	contextText, err := s.assembleSectionContext(ctx, doc, doc.OutlineText, section, totalSections, prior)
	if err != nil {
		return Section{}, err
	}
	// The consumer profile's document band describes the whole assembled
	// article, while each outline cell owns a smaller word budget. Carry the
	// cell budget into both the instruction and the local eligibility gate;
	// otherwise a 650-word section minimum is silently applied per section.
	sectionTargetWords := planned.TargetWords
	if sectionTargetWords <= 0 {
		sectionTargetWords = defaultTargetSectionWords
	}
	// A band, not a ceiling pinned at the target. The previous form set MaxWords
	// to the target and MinWords to zero, which gave the model one direction to
	// miss in and no floor to miss against: outline targets summing to 1100
	// words were satisfied by 490 words of eligible text.
	sectionMinWords, sectionMaxWords := sectionWordBand(sectionProfile, sectionTargetWords)
	sectionProfile.Constraints.MinWords = sectionMinWords
	sectionProfile.Constraints.MaxWords = sectionMaxWords
	sectionProfile.Constraints.RequiredFormat = sectionRequiredFormat(sectionProfile)
	sectionProfile.Sampler.MaxOutputTokens = sectionOutputCap(sectionProfile, sectionMaxWords)
	sectionProfile.Budget.MaxOutputTokens = sectionProfile.Sampler.MaxOutputTokens
	// Continuation is not variation. Selecting the section candidate most
	// distant from its own siblings is right for ideation and wrong here: it
	// picks the outlier at every position and assembles a document out of them.
	sectionProfile.SelectionPolicy = sectionSelectionPolicy(sectionProfile)
	sectionProfile.Sampler.Kind, sectionProfile.Sampler.K = sectionSampler(sectionProfile)
	sectionQuery := fmt.Sprintf("%s\n\n%s\n\n%s\n\nHARD SECTION LENGTH: write between %d and %d words; aim for approximately %d words. A section materially shorter than the floor is rejected.", doc.Title, planned.Intent, contextText, sectionMinWords, sectionMaxWords, sectionTargetWords)
	selectionContext := &SelectionContext{PriorText: s.committedSectionTexts(ctx, prior), TargetWords: sectionTargetWords}
	generated, err := s.generateWithProfile(ctx, GenerateRequest{ProfileKey: sectionProfile.Key, Query: sectionQuery, IncludeCandidates: true, Negative: negative, Selection: selectionContext}, sectionProfile)
	if err != nil {
		return Section{}, err
	}
	section.SessionID = generated.Session.ID
	selected := eligibleCandidate(generated)
	if selected == nil {
		// One bounded retry naming the measured shortfall, the same shape as the
		// outline retry. A length floor that can only fail the whole document is
		// not usable: models undershoot a stated word target routinely, and the
		// useful response is to say by how much and ask for development rather
		// than to abandon the article or to widen the band until it passes.
		shortfall := closestMiss(generated.Candidates)
		retryQuery := sectionQuery + fmt.Sprintf("\n\nThe previous attempt was rejected: it produced %d words against a floor of %d. Write a longer section by developing the point with a concrete example and its consequence, not by adding adjectives or restating the same sentence.", shortfall, sectionMinWords)
		generated, err = s.generateWithProfile(ctx, GenerateRequest{ProfileKey: sectionProfile.Key, Query: retryQuery, IncludeCandidates: true, Negative: negative, SessionID: generated.Session.ID, Selection: selectionContext}, sectionProfile)
		if err != nil {
			return Section{}, err
		}
		section.SessionID = generated.Session.ID
		selected = eligibleCandidate(generated)
	}
	if selected == nil {
		return Section{}, fmt.Errorf("section %d candidate is ineligible: %s", position, candidateReason(firstCandidate(generated)))
	}
	if err := s.commitSectionCandidate(ctx, section.ID, selected.ID, section); err != nil {
		return Section{}, err
	}
	// commitSectionCandidate takes the section by value and records the
	// committed identifier on its own copy. Recording it here too is what makes
	// this section visible to every later section: both prepareContextSnapshot
	// and assembleSectionContext skip a section whose committed candidate is
	// empty, so without this assignment the caller's slice carries uncommitted
	// copies and each section is drafted blind to the ones before it. That is
	// the mechanism behind a document whose sections restate one another.
	section.CommittedCandidateID = selected.ID
	return section, nil
}

// eligibleCandidate returns the policy's choice, and never an ineligible
// candidate. The previous form fell back to candidates[0] whenever the policy
// declined to choose, which silently committed a candidate the constraint gate
// had already rejected.
func eligibleCandidate(generated GenerateResponse) *Candidate {
	if generated.Selected != nil && generated.Selected.Eligibility.Eligible {
		return generated.Selected
	}
	for i := range generated.Candidates {
		if generated.Candidates[i].Eligibility.Eligible {
			return &generated.Candidates[i]
		}
	}
	return nil
}

func firstCandidate(generated GenerateResponse) *Candidate {
	if generated.Selected != nil {
		return generated.Selected
	}
	if len(generated.Candidates) > 0 {
		return &generated.Candidates[0]
	}
	return nil
}

// closestMiss reports the word count of the longest candidate drawn, which is
// the number worth telling the model about when every candidate fell short.
func closestMiss(candidates []Candidate) int {
	best := 0
	for _, c := range candidates {
		if words := candidateWordCount(c); words > best {
			best = words
		}
	}
	return best
}

// repairDocument regenerates the section that most repeats the rest of the
// document, using the other committed sections as negative conditioning, until
// the coherence verdict passes or the declared repair budget is spent. Without
// it the verdict is inert: assembly measured repetition, wrote the number onto
// the document, and returned that document unchanged whether it passed or not.
func (s *Service) repairDocument(ctx context.Context, doc Document, sectionProfile Profile, intents []string) (Document, error) {
	rounds := sectionProfile.Composition.MaxRepairRounds
	for attempt := 0; attempt < rounds; attempt++ {
		if coherenceVerdictPassed(doc.Coherence) {
			return doc, nil
		}
		sections, texts, err := s.loadCommittedSections(ctx, doc)
		if err != nil {
			return doc, err
		}
		if len(sections) < 2 {
			return doc, nil
		}
		target := mostRedundantSection(texts)
		if target < 0 || target >= len(sections) || target >= len(doc.Outline) {
			return doc, nil
		}
		negative := NegativeContext{Rejected: append([]string{texts[target]}, otherTexts(texts, target)...)}
		prior := sections[:target]
		following := []string{}
		if target+1 < len(intents) {
			following = intents[target+1:]
		}
		replaced, err := s.produceSection(ctx, doc, sectionProfile, target, doc.Outline[target], prior, following, len(doc.Outline), negative, sections[target].ID)
		if err != nil {
			// A repair that cannot produce an eligible section leaves the prior
			// document standing with its failed verdict intact. Reporting the
			// honest failed verdict beats replacing it with nothing.
			return doc, nil
		}
		doc.Sections[target] = replaced
		reassembled, err := s.AssembleDocument(ctx, doc.ID)
		if err != nil {
			return doc, err
		}
		doc = reassembled
	}
	return doc, nil
}

// coherenceVerdictPassed reads the assembled verdict without assuming it is
// present: a document that was never assembled has no verdict, and treating
// that as a pass would skip repair exactly when it is needed.
func coherenceVerdictPassed(coherence any) bool {
	container, ok := coherence.(map[string]any)
	if !ok {
		return false
	}
	verdict, ok := container["verdict"].(map[string]any)
	if !ok {
		return false
	}
	passed, ok := verdict["coherent"].(bool)
	return ok && passed
}

// mostRedundantSection names the section carrying the highest lexical overlap
// with any other single section, which is the one worth rewriting first.
func mostRedundantSection(texts []string) int {
	if len(texts) < 2 {
		return -1
	}
	worst, worstScore := -1, -1.0
	for i := range texts {
		var score float64
		for j := range texts {
			if i == j {
				continue
			}
			if overlap := textmetrics.CrossSectionRepetition([]string{texts[i], texts[j]}); overlap > score {
				score = overlap
			}
		}
		if score > worstScore {
			worst, worstScore = i, score
		}
	}
	return worst
}

func otherTexts(texts []string, skip int) []string {
	out := make([]string, 0, len(texts))
	for i, text := range texts {
		if i == skip {
			continue
		}
		out = append(out, text)
	}
	return out
}

// loadCommittedSections re-reads the document's sections from storage so repair
// operates on committed state rather than on an in-memory copy.
func (s *Service) loadCommittedSections(ctx context.Context, doc Document) ([]Section, []string, error) {
	sections := make([]Section, 0, len(doc.SectionIDs))
	texts := make([]string, 0, len(doc.SectionIDs))
	for _, id := range doc.SectionIDs {
		var section Section
		if err := s.loadJSON(ctx, "prose_sections", id, &section); err != nil {
			return nil, nil, err
		}
		if section.CommittedCandidateID == "" {
			continue
		}
		var candidate Candidate
		if err := s.loadJSON(ctx, "prose_candidates", section.CommittedCandidateID, &candidate); err != nil {
			return nil, nil, err
		}
		sections = append(sections, section)
		texts = append(texts, candidate.Text)
	}
	return sections, texts, nil
}

// Composition defaults. They are named constants rather than literals in the
// outline prompt because "three sections" was previously spread across the
// prompt, the decoder, and the eval, which made article shape unchangeable.
const (
	defaultTargetSectionWords   = 350
	defaultSectionWordTolerance = 0.25
	defaultMinSections          = 3
	defaultFullTextTokenBudget  = 4096
	defaultSectionCandidates    = 3
	defaultSectionOutputFloor   = 2048
	defaultSectionTokenHeadroom = 6
	minimumEmbeddingDimension   = 64
	defaultMaxSections          = 9
	policyContinuation          = "continuation_least_redundant"
)

// resolveSectionPlan returns the count to ask for and the band the decoder will
// accept. An explicitly declared count is exact; a derived count is a target
// inside the declared band, because the model chooses the shape that fits the
// subject and only the policy decides the range it may choose within.
func resolveSectionPlan(doc Document, profile Profile, maxWords int) (want, minSections, maxSections int) {
	want = resolveSectionCount(doc, profile, maxWords)
	if doc.SectionCount > 0 || profile.Composition.SectionCount > 0 {
		return want, want, want
	}
	minSections, maxSections = profile.Composition.MinSections, profile.Composition.MaxSections
	if minSections <= 0 {
		minSections = defaultMinSections
	}
	if maxSections <= 0 {
		maxSections = defaultMaxSections
	}
	if maxSections < minSections {
		maxSections = minSections
	}
	return want, minSections, maxSections
}

// resolveSectionCount decides how many sections an outline carries. Precedence
// is the document override, then the profile policy, then a count derived from
// the article's word budget and the declared words-per-section target.
func resolveSectionCount(doc Document, profile Profile, maxWords int) int {
	if doc.SectionCount > 0 {
		return doc.SectionCount
	}
	policy := profile.Composition
	if policy.SectionCount > 0 {
		return policy.SectionCount
	}
	target := policy.TargetSectionWords
	if target <= 0 {
		target = defaultTargetSectionWords
	}
	minSections, maxSections := policy.MinSections, policy.MaxSections
	if minSections <= 0 {
		minSections = defaultMinSections
	}
	if maxSections <= 0 {
		maxSections = defaultMaxSections
	}
	if maxSections < minSections {
		maxSections = minSections
	}
	count := maxWords / target
	if maxWords%target != 0 {
		count++
	}
	if count < minSections {
		count = minSections
	}
	if count > maxSections {
		count = maxSections
	}
	return count
}

// sectionWordBand converts an outline cell target into the floor and ceiling
// the section eligibility gate enforces.
func sectionWordBand(profile Profile, target int) (int, int) {
	tolerance := profile.Composition.SectionWordTolerance
	if tolerance <= 0 || tolerance >= 1 {
		tolerance = defaultSectionWordTolerance
	}
	minWords := int(float64(target) * (1 - tolerance))
	maxWords := int(float64(target) * (1 + tolerance))
	if minWords < 1 {
		minWords = 1
	}
	if maxWords <= minWords {
		maxWords = minWords + 1
	}
	return minWords, maxWords
}

// sectionRequiredFormat honours the profile instead of forcing paragraph.
// matchesRequiredFormat rejects any text carrying a list marker under
// "paragraph", so forcing that value also banned headings, lists, and code
// blocks from every long-form section.
func sectionRequiredFormat(profile Profile) string {
	if format := strings.TrimSpace(profile.Composition.SectionFormat); format != "" {
		return format
	}
	return "paragraph"
}

// sectionSelectionPolicy defaults sections to continuation rather than letting
// them inherit the profile's ideation policy.
func sectionSelectionPolicy(profile Profile) string {
	if policy := strings.TrimSpace(profile.Composition.SectionSelectionPolicy); policy != "" {
		return policy
	}
	return policyContinuation
}

// sectionOutputCap sizes the output budget for one section.
//
// The former value, twice the section's word ceiling, was wrong by roughly an
// order of magnitude against a reasoning model. max_output_tokens bounds every
// token the model emits, and a reasoning model spends most of that budget
// before it writes a visible word: at a 624-token cap the provider returned a
// truncated JSON string and the section arrived as a 12-word fragment, which
// then failed the word floor. Nothing in the failure named the cap, and the
// verbalized path masked it entirely because a k-candidate cap is k times
// larger for the same section.
//
// The multiplier is therefore headroom for invisible tokens rather than an
// estimate of prose length, and the floor matters more than the multiplier for
// the short sections where the ratio is worst.
func sectionOutputCap(profile Profile, sectionMaxWords int) int {
	if declared := profile.Composition.SectionMaxOutputTokens; declared > 0 {
		return declared
	}
	return maxInt(defaultSectionOutputFloor, sectionMaxWords*defaultSectionTokenHeadroom)
}

// sectionSampler decides how a section's candidate set is drawn. Sections
// default to direct draws rather than the profile's verbalized sampler: an
// outline wants a distribution over framings, a section wants the best
// continuation of the framing the outline already fixed, and the verbalized
// envelope adds a whole-document failure mode when one entry in the set comes
// back malformed.
func sectionSampler(profile Profile) (string, int) {
	kind := strings.TrimSpace(profile.Composition.SectionSamplerKind)
	if kind == "" {
		kind = samplerDirect
	}
	count := profile.Composition.SectionCandidates
	if count <= 0 {
		count = defaultSectionCandidates
	}
	return kind, count
}

// committedSectionTexts resolves the prose already committed to a document.
// Selection receives text, never record identifiers.
func (s *Service) committedSectionTexts(ctx context.Context, sections []Section) []string {
	out := make([]string, 0, len(sections))
	for _, section := range sections {
		if section.CommittedCandidateID == "" {
			continue
		}
		var candidate Candidate
		if err := s.loadJSON(ctx, "prose_candidates", section.CommittedCandidateID, &candidate); err != nil {
			continue
		}
		if strings.TrimSpace(candidate.Text) != "" {
			out = append(out, candidate.Text)
		}
	}
	return out
}

func decodeOutline(text string, minSections, maxSections int) ([]OutlineSection, error) {
	if minSections <= 0 {
		minSections = defaultMinSections
	}
	if maxSections < minSections {
		maxSections = minSections
	}
	var outline []OutlineSection
	if err := json.Unmarshal([]byte(text), &outline); err != nil {
		// A weak provider sometimes returns an object keyed section_1, section_2
		// rather than an array. Decode it by sorted key so the recovery is not
		// tied to a fixed section count the way the previous three-field struct
		// was, and so key order in the JSON cannot reorder the article.
		var keyed map[string]OutlineSection
		if fixedErr := json.Unmarshal([]byte(text), &keyed); fixedErr != nil || len(keyed) == 0 {
			return nil, fmt.Errorf("%w: %v", ErrMalformedOutline, err)
		}
		keys := make([]string, 0, len(keyed))
		for key := range keyed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		outline = outline[:0]
		for _, key := range keys {
			outline = append(outline, keyed[key])
		}
	}
	// A band rather than an exact count. Section count is a composition policy
	// decision, and the exact number inside the declared band is the model's to
	// make; demanding one number turned an ordinary outline into a hard failure.
	if len(outline) < minSections || len(outline) > maxSections {
		return nil, fmt.Errorf("%w: outline must contain between %d and %d sections, got %d", ErrMalformedOutline, minSections, maxSections, len(outline))
	}
	for i, section := range outline {
		if strings.TrimSpace(section.Intent) == "" || strings.TrimSpace(section.Summary) == "" || section.TargetWords <= 0 {
			return nil, fmt.Errorf("%w: section %d requires intent, summary, and positive target_words", ErrMalformedOutline, i+1)
		}
	}
	return outline, nil
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
	snapshot := ContextSnapshot{OutlineRef: doc.OutlineID, OutlineText: doc.OutlineText, FollowingIntents: append([]string(nil), following...)}
	for _, section := range prior {
		snapshot.PriorSectionRefs = append(snapshot.PriorSectionRefs, section.ID)
		snapshot.FullTextSectionRefs = append(snapshot.FullTextSectionRefs, section.ID)
	}
	return snapshot
}

func (s *Service) prepareContextSnapshot(ctx context.Context, doc Document, prior []Section, snapshot ContextSnapshot) (ContextSnapshot, error) {
	profile, err := s.latestProfile(ctx, doc.ProfileKey)
	if err != nil {
		return ContextSnapshot{}, err
	}
	snapshot.PriorSectionRefs = nil
	snapshot.FullTextSectionRefs = nil
	snapshot.SummarizedSectionRefs = nil
	snapshot.SectionSummaries = map[string]string{}
	fullBudget := profile.ContextPolicy.FullTextTokenBudget
	if fullBudget <= 0 {
		// Not the output cap. How many tokens of prior sections a prompt may
		// carry is an input-context question and has nothing to do with how long
		// a section may be; borrowing the output cap made a profile that writes
		// short sections also forget its own document, summarising prior prose
		// through a small model exactly when the sections were short enough to
		// carry whole.
		fullBudget = defaultFullTextTokenBudget
	}
	summarizeBeyond := profile.ContextPolicy.SummarizeBeyond
	if summarizeBeyond <= 0 {
		summarizeBeyond = fullBudget
	}
	fullTokens := 0
	for index := len(prior) - 1; index >= 0; index-- {
		section := prior[index]
		if section.CommittedCandidateID == "" {
			continue
		}
		var candidate Candidate
		if err := s.loadJSON(ctx, "prose_candidates", section.CommittedCandidateID, &candidate); err != nil {
			return ContextSnapshot{}, err
		}
		tokens := estimateContextTokens(candidate.Text)
		shouldSummarize := !profile.ContextPolicy.AlwaysFullPrevious && (fullTokens+tokens > fullBudget || fullTokens+tokens > summarizeBeyond)
		if shouldSummarize {
			summary, err := s.summarizeSection(ctx, candidate.Text)
			if err != nil {
				return ContextSnapshot{}, err
			}
			snapshot.SummarizedSectionRefs = append([]string{section.ID}, snapshot.SummarizedSectionRefs...)
			snapshot.SectionSummaries[section.ID] = summary
		} else {
			snapshot.FullTextSectionRefs = append([]string{section.ID}, snapshot.FullTextSectionRefs...)
			fullTokens += tokens
		}
		snapshot.PriorSectionRefs = append([]string{section.ID}, snapshot.PriorSectionRefs...)
	}
	snapshot.EstimatedTokens = fullTokens
	for _, summary := range snapshot.SectionSummaries {
		snapshot.EstimatedTokens += estimateContextTokens(summary)
	}
	return snapshot, nil
}

func (s *Service) summarizeSection(ctx context.Context, text string) (string, error) {
	responses, err := s.gateway.Generate(ctx, GatewayRequest{Role: "extract.structured", Instruction: "Summarize the supplied prose faithfully in two or three sentences. Return only the summary.", Query: text, Strategy: samplerDirect, K: 1, MaxOutputTokens: 256})
	if err != nil {
		return "", fmt.Errorf("summarize prior section: %w", err)
	}
	if len(responses) == 0 || strings.TrimSpace(responses[0].Text) == "" {
		return "", errors.New("summarize prior section: gateway returned no text")
	}
	return responses[0].Text, nil
}

// assembleSectionContext is the single seam where record references become
// prose context. Keeping this conversion centralized prevents UUIDs and other
// storage identifiers from leaking into prompts as the composition path grows.
func (s *Service) assembleSectionContext(ctx context.Context, doc Document, outlineText string, section Section, total int, prior []Section) (string, error) {
	parts := []string{
		"Document title: " + doc.Title,
		"Full outline:\n" + outlineText,
		fmt.Sprintf("Current section: %d of %d", section.Position+1, total),
		"Section intent: " + section.Intent,
	}
	if section.Summary != "" {
		parts = append(parts, "Section summary: "+section.Summary)
	}
	if section.TargetWords > 0 {
		parts = append(parts, fmt.Sprintf("Length target: approximately %d words.", section.TargetWords))
	}
	for _, previous := range prior {
		if previous.CommittedCandidateID == "" {
			continue
		}
		var candidate Candidate
		if err := s.loadJSON(ctx, "prose_candidates", previous.CommittedCandidateID, &candidate); err != nil {
			return "", err
		}
		label := previous.Intent
		if label == "" {
			label = fmt.Sprintf("section %d", previous.Position+1)
		}
		priorText := candidate.Text
		if summary, ok := section.Context.SectionSummaries[previous.ID]; ok {
			priorText = summary
		}
		parts = append(parts, fmt.Sprintf("Prior section %d (%s): %s", previous.Position+1, label, priorText))
	}
	if len(section.Context.FollowingIntents) > 0 {
		parts = append(parts, "Following section intents: "+strings.Join(section.Context.FollowingIntents, "; "))
	}
	// Prior text alone is not an instruction. Supplying it without saying what
	// to do with it produces a section that re-derives the same argument in new
	// words, which reads as repetition and measures as low lexical overlap.
	if len(prior) > 0 {
		parts = append(parts, "Continuity requirement: the passages above are already written and already published to the reader. Advance the argument from where they end. Do not restate, re-introduce, or re-summarise a point they already establish; assume the reader has read them. Introduce what only this section carries, and if this is the final section, close rather than recapitulate.")
	} else {
		parts = append(parts, "Continuity requirement: this is the opening section. Establish the subject and earn the reader's attention; do not summarise the whole article in advance.")
	}
	assembled := strings.Join(parts, "\n\n")
	if recordIdentifierPattern.MatchString(assembled) {
		return "", fmt.Errorf("%w: section context", ErrPromptContainsIdentifier)
	}
	return assembled, nil
}

func (s *Service) AssembleDocument(ctx context.Context, documentID string) (Document, error) {
	var doc Document
	if err := s.loadJSON(ctx, "prose_documents", documentID, &doc); err != nil {
		return Document{}, err
	}
	var text strings.Builder
	var sectionTexts []string
	var structure []Section
	var provenance DocumentProvenance
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
		provenance.TotalCostMicros += candidate.Provenance.CostMicros
		provenance.InputTokens += candidate.Provenance.InputTokens
		provenance.OutputTokens += candidate.Provenance.OutputTokens
		provenance.Providers = appendUnique(provenance.Providers, candidate.Provenance.Provider)
		provenance.Models = appendUnique(provenance.Models, candidate.Provenance.ResolvedModelRef)
	}
	doc.AssembledText = text.String()
	doc.Status = "assembled"
	doc.Sections = structure
	provenance.SectionCount = len(structure)
	provenance.WordCount = textmetrics.Analyze(doc.AssembledText, nil).WordCount
	doc.Provenance = provenance
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
		} else if conformanceErr != nil {
			return Document{}, conformanceErr
		}
		sectionScores = append(sectionScores, conformanceScore(result))
	}
	doc.StyleKey = styleKey
	repetition := textmetrics.CrossSectionRepetition(sectionTexts)
	drift := textmetrics.StyleDrift(sectionScores)
	// The lexical measure alone passes the failure a reader notices first.
	// Three sections that restate one argument in different words score as low
	// lexical overlap while carrying nearly the same meaning, so a document that
	// says the same thing three times can clear a lexical repetition threshold
	// comfortably. Semantic similarity over section embeddings is what catches
	// it, and it is measured rather than assumed: when the embedding tier is
	// unavailable the document says so instead of reporting a silent pass.
	semanticRepetition, semanticBasis, semanticErr := s.semanticSectionRepetition(ctx, sectionTexts)
	semanticMeasured := semanticErr == nil
	verdict := map[string]any{"cross_section_repetition": true, "style_drift": true, "semantic_section_repetition": true, "coherent": true}
	if profile.Coherence.MaxCrossSectionRepetition > 0 {
		verdict["cross_section_repetition"] = repetition <= profile.Coherence.MaxCrossSectionRepetition
	}
	if profile.Coherence.MaxStyleDrift > 0 {
		verdict["style_drift"] = drift <= profile.Coherence.MaxStyleDrift
	}
	if profile.Coherence.MaxSemanticSectionRepetition > 0 && semanticMeasured {
		verdict["semantic_section_repetition"] = semanticRepetition <= profile.Coherence.MaxSemanticSectionRepetition
	}
	verdict["coherent"] = verdict["cross_section_repetition"] == true && verdict["style_drift"] == true && verdict["semantic_section_repetition"] == true
	coherence := map[string]any{"cross_section_repetition": repetition, "style_drift": drift, "thresholds": profile.Coherence, "verdict": verdict, "basis": "textmetrics.CrossSectionRepetition and textmetrics.StyleDrift over committed section text", "semantic_measured": semanticMeasured}
	if semanticMeasured {
		coherence["semantic_section_repetition"] = semanticRepetition
		coherence["semantic_basis"] = semanticBasis
	} else if semanticErr != nil {
		coherence["semantic_unavailable"] = semanticErr.Error()
	}
	doc.Coherence = coherence
	if err := s.saveJSON(ctx, "prose_documents", doc.ID, doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}

// semanticSectionRepetition reports mean pairwise cosine similarity across
// section embeddings. It always attempts the measurement rather than gating on
// the profile's candidate-level measurement tier: whether a document repeats
// itself is a property of the document, not of how its candidate sets were
// scored, and a caller that declines the semantic tier for sampling still needs
// an honest answer here.
func (s *Service) semanticSectionRepetition(ctx context.Context, sections []string) (float64, string, error) {
	if len(sections) < 2 {
		return 0, "", errors.New("semantic_section_repetition_requires_two_sections")
	}
	embedder, ok := s.gateway.(EmbeddingGateway)
	if !ok {
		return 0, "", errors.New("embedding_unavailable: gateway exposes no embedding surface")
	}
	embedded, err := embedder.Embed(ctx, EmbeddingRequest{Role: "embedding.default", Texts: sections})
	if err != nil {
		return 0, "", fmt.Errorf("embedding_unavailable: %w", err)
	}
	// A similarity threshold is only meaningful over an embedding with enough
	// dimensions to separate meanings. A degraded embedder returning a handful
	// of components still yields a number between zero and one, and that number
	// will sit near any threshold you pick; publishing it as a measured semantic
	// verdict would be the silent pass this gate exists to remove. This is a
	// guard on the gateway's reported dimension, which is the only place the
	// fact is available before the similarity is computed.
	if embedded.Dimension < minimumEmbeddingDimension {
		return 0, "", fmt.Errorf("embedding_degenerate: dimension %d is below the %d required to support a similarity threshold", embedded.Dimension, minimumEmbeddingDimension)
	}
	_, set, err := textmetrics.AnalyzeSetSemantic(sections, embedded.Vectors)
	if err != nil {
		return 0, "", err
	}
	return set.MeanSimilarity, set.Basis, nil
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
		direction := "at_least"
		if style.TargetDirections != nil && style.TargetDirections[key] != "" {
			direction = style.TargetDirections[key]
		}
		met := known && ((direction == "at_most" && actual <= target) || (direction != "at_most" && actual >= target))
		if !met {
			missed[key] = target
		}
		verdicts[key] = map[string]any{"met": met, "actual": actual, "target": target, "direction": direction, "known": known}
	}
	antiSpans := textmetrics.LexiconSpans(text, style.AntiPatterns)
	return map[string]any{"style": style.Key, "version": style.Version, "targets_met": len(missed) == 0 && len(antiSpans) == 0, "missed": missed, "verdicts": verdicts, "anti_pattern_spans": antiSpans, "preferred_lexicon_spans": metrics.LexiconFlags}, nil
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
	case "flesch_kincaid", "flesch_kincaid_grade", "flesch_kincaid_grade_max":
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
	merged.TargetDirections = map[string]string{}
	for k, v := range parent.TargetDirections {
		merged.TargetDirections[k] = v
	}
	for k, v := range current.TargetDirections {
		if old, ok := merged.TargetDirections[k]; ok && old != v {
			return Style{}, fmt.Errorf("%w: target %s has conflicting comparators %q and %q", ErrStyleResolutionConflict, k, old, v)
		}
		merged.TargetDirections[k] = v
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
	for _, banned := range c.BannedLexicon {
		if spans := textmetrics.LexiconSpans(text, []string{banned}); len(spans) > 0 {
			return Eligibility{false, "banned_lexicon:" + spans[0].Term}
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

func choose(candidates []Candidate, policy string, params map[string]float64, seed int64, selection *SelectionContext) *Candidate {
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
	case policyContinuation:
		// Continuation inverts the rarest objective. Rarest asks which candidate
		// is least like its siblings, which is the right question for ideation
		// and the wrong one for a position in a document: it selects the outlier
		// at every position and then assembles the outliers. Here the objective
		// is the candidate that repeats the committed text least while landing
		// closest to its outline target. Both terms are deterministic
		// measurements over text, so this remains a mechanical policy and not
		// the judge-based ranker the scenario declines to ship.
		var best Candidate
		bestScore := math.Inf(1)
		for _, c := range eligible {
			score := redundancyAgainst(selection, c.Text) + 0.5*lengthMiss(selection, c)
			if score < bestScore {
				best, bestScore = c, score
			}
		}
		if best.ID == "" {
			best = eligible[0]
		}
		return &best
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

// redundancyAgainst reports the strongest lexical overlap between a candidate
// and any single already-committed passage. Max rather than mean: repeating one
// earlier section wholesale is the failure worth catching, and averaging over
// unrelated sections hides it.
func redundancyAgainst(selection *SelectionContext, text string) float64 {
	if selection == nil || len(selection.PriorText) == 0 {
		return 0
	}
	var worst float64
	for _, prior := range selection.PriorText {
		if strings.TrimSpace(prior) == "" {
			continue
		}
		if overlap := textmetrics.CrossSectionRepetition([]string{prior, text}); overlap > worst {
			worst = overlap
		}
	}
	return worst
}

// lengthMiss reports how far a candidate falls from its outline target, as a
// fraction of that target, so the term is comparable across section sizes.
func lengthMiss(selection *SelectionContext, c Candidate) float64 {
	if selection == nil || selection.TargetWords <= 0 {
		return 0
	}
	words := candidateWordCount(c)
	if words <= 0 {
		return 1
	}
	return math.Abs(float64(words-selection.TargetWords)) / float64(selection.TargetWords)
}

func candidateWordCount(c Candidate) int {
	var m textmetrics.Metrics
	raw, _ := json.Marshal(c.Measurements)
	_ = json.Unmarshal(raw, &m)
	return m.WordCount
}

func candidateRarity(c Candidate) float64 {
	var m textmetrics.SetMetrics
	raw, _ := json.Marshal(c.SetMeasurements)
	_ = json.Unmarshal(raw, &m)
	if c.SetIndex >= 0 && c.SetIndex < len(m.PairwiseSimilarity) {
		row := m.PairwiseSimilarity[c.SetIndex]
		if len(row) > 0 {
			var total float64
			for _, similarity := range row {
				total += similarity
			}
			return 1 - total/float64(len(row))
		}
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
	case "rich_prose":
		// Long-form sections that may carry headings, lists, or code while still
		// being prose. "paragraph" rejects any text containing a list marker, so
		// a section required to be a paragraph can never show a command, and a
		// dev-log post whose subject is a command cannot be written at all.
		return metrics.SentenceCount > 0 && metrics.WordCount > 0
	case "essay", "essay_shape":
		lower := strings.ToLower(string(raw))
		return metrics.WordCount >= 80 && (strings.Contains(lower, "introduction") || strings.Contains(lower, "## ") || strings.Contains(lower, "# ")) && (strings.Contains(lower, "conclusion") || strings.Contains(lower, "in conclusion") || metrics.SentenceCount >= 4)
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
