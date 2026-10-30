package inference

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"ai-gateway/internal/providers"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// ErrRoleForbidsSampling reports a caller-supplied sampling control sent to a
// role that has not declared itself overridable. It is a request defect, not
// provider incapacity, and the two get different error codes: the caller asked
// for something this role forbids, which no amount of re-routing would fix.
var ErrRoleForbidsSampling = errors.New("role does not admit caller-supplied sampling")

// ErrUnsupportedSampling reports that no remaining candidate's provider can
// honour a caller-supplied control. A caller-explicit control is a promise the
// gateway keeps or refuses; it is never silently downgraded, matching how a
// rejected schema is never degraded to unconstrained generation.
var ErrUnsupportedSampling = errors.New("no candidate can honor the requested sampling")

// ErrContextOverflow is returned before provider dispatch when provider policy
// declares a window that the assembled prompt and output cap cannot fit.
var ErrContextOverflow = errors.New("resolved model context window would be exceeded")

// resolveTemperature applies the role/caller precedence for one candidate.
//
//	caller set + role overridable   -> caller value, but only if the candidate
//	                                   declares "honored"
//	caller set + role not overridable -> ErrRoleForbidsSampling
//	role declares                   -> role value, omitted only when the
//	                                   candidate declares "rejected"
//	neither                         -> nil, the resource policy default applies
//
// The two paths deliberately draw the line in different places.
//
// A caller-supplied control is a promise, so only "honored" keeps it. "ignored"
// fails the promise exactly as thoroughly as "rejected" does — the difference is
// only whether the failure is visible — so both refuse the candidate.
//
// A role-declared default is a preference, and the question there is not "will
// this be honoured" but "will sending it break the call". Only "rejected"
// breaks it. So "ignored" and "unknown" are still sent: harmless where the
// provider discards it, and correct where an undeclared provider turns out to
// honour it after all. That is what treating "unknown" as best-effort means —
// try it, rather than silently drop the role's stated intent. It also keeps a
// deterministic role deterministic against a resource that has not yet
// published its declaration.
func resolveTemperature(request ProviderRequest, role InferenceRole, support providers.SamplingSupport) (*float64, error) {
	if request.Temperature != nil {
		if role.Sampling == nil || !role.Sampling.Overridable {
			return nil, ErrRoleForbidsSampling
		}
		if support != providers.SamplingHonored {
			return nil, ErrUnsupportedSampling
		}
		value := *request.Temperature
		return &value, nil
	}
	if role.Sampling == nil || role.Sampling.Temperature == nil {
		return nil, nil
	}
	if support == providers.SamplingRejected {
		return nil, nil
	}
	value := *role.Sampling.Temperature
	return &value, nil
}

// resolveOutputCap applies the caller/role precedence for the output budget and
// reports where the answer came from. Reporting the source matters as much as
// the number: "the gateway imposed nothing" and "the gateway imposed a cap you
// cannot see" are different facts, and only the first is true here when both
// the caller and the role are silent.
func resolveOutputCap(request ProviderRequest, resolved providers.ResolvedRole) (int32, OutputCapSource) {
	switch {
	case request.MaxOutputTokens > 0:
		return request.MaxOutputTokens, OutputCapRequest
	case resolved.MaxOutputTokens > 0:
		// The resource applies its own role cap when the gateway sends none, so
		// this is a report of what will happen rather than a value to transmit.
		return resolved.MaxOutputTokens, OutputCapRolePolicy
	default:
		return 0, OutputCapNoneImposed
	}
}

type ResourceRepository struct {
	catalog  RoleCatalog
	adapters map[string]providers.Adapter
}

func NewResourceRepository(catalog RoleCatalog, adapters []providers.Adapter) (*ResourceRepository, error) {
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	repository := &ResourceRepository{catalog: catalog, adapters: make(map[string]providers.Adapter, len(adapters))}
	for _, adapter := range adapters {
		repository.adapters[strings.ToLower(strings.TrimSpace(adapter.Provider))] = adapter
	}
	return repository, nil
}

func (r *ResourceRepository) Run(ctx context.Context, request ProviderRequest) (ProviderResult, error) {
	definition, ok := r.catalog.Roles[strings.TrimSpace(request.Role)]
	if !ok {
		return ProviderResult{}, fmt.Errorf("%w: inference role %q is not declared", ErrUnavailable, request.Role)
	}
	if definition.IsDecision() {
		return ProviderResult{}, fmt.Errorf("%w: inference role %q is a decision role; use Decide", ErrUnavailable, request.Role)
	}
	if err := validateGeometrySpendLimit(request); err != nil {
		return ProviderResult{}, err
	}
	var failures []string
	var providerCallsMade int32
	var usageIncomplete bool
	// samplingRefused records that at least one candidate was skipped purely
	// because it could not honour a caller-supplied control. It changes the
	// terminal error from "nothing worked" to the specific, actionable
	// "nothing here can sample the way you asked".
	var samplingRefused bool
	for _, candidate := range definition.Candidates {
		adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
		if ok && !profileAdmits(request.Profile, adapter.Locality) {
			continue
		}
		if !ok {
			failures = append(failures, candidate.Provider+": adapter unavailable")
			continue
		}
		resolved, err := adapter.ResolveRole(ctx, candidate.ResourceRole)
		if err != nil {
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		support := resolved.TemperatureSupport()
		temperature, err := resolveTemperature(request, definition, support)
		if err != nil {
			if errors.Is(err, ErrRoleForbidsSampling) {
				// A role that forbids overrides forbids them everywhere, so
				// walking further candidates could only produce the same answer.
				return ProviderResult{ProviderCallsMade: providerCallsMade}, fmt.Errorf("%w: inference role %q declares no overridable sampling", ErrRoleForbidsSampling, request.Role)
			}
			samplingRefused = true
			failures = append(failures, fmt.Sprintf("%s: temperature not honored by resolved role %s (%s)", candidate.Provider, candidate.ResourceRole, support))
			continue
		}
		maxOutputTokens, capSource := resolveOutputCap(request, resolved)
		prompt := providerPrompt(request, resolved.CoordinateConvention)
		if resolved.ContextWindow > 0 && estimateTokens(prompt)+int64(maxOutputTokens) > int64(resolved.ContextWindow) {
			return ProviderResult{ProviderCallsMade: providerCallsMade}, fmt.Errorf("%w: assembled input and requested output cap do not fit the resolved model policy", ErrContextOverflow)
		}
		if request.MaxProviderCalls > 0 && providerCallsMade >= request.MaxProviderCalls {
			return ProviderResult{ProviderCallsMade: providerCallsMade}, fmt.Errorf("%w: request limit of %d provider calls exhausted", ErrUnavailable, request.MaxProviderCalls)
		}
		applied := AppliedSettings{
			TemperatureSent:       temperature,
			TemperatureSupport:    support,
			MaxOutputTokens:       maxOutputTokens,
			MaxOutputTokensSource: capSource,
		}
		// Count immediately before the actual adapter dispatch, so errors are
		// included and candidate fallback can never cross the caller's ceiling.
		providerCallsMade++
		execution, err := adapter.Execute(ctx, providers.ExecutionRequest{
			Kind:      sharedv1.RequestKind_REQUEST_KIND_STRUCTURED_EXTRACTION,
			Role:      candidate.ResourceRole,
			Profile:   request.Profile,
			InputText: prompt,
			Timeout:   definition.Timeout(),
			// Only a caller-supplied cap is transmitted. A role-declared cap is
			// already the resource's own default, so re-sending it would turn a
			// resource-owned decision into a gateway-owned one.
			MaxOutputTokens: request.MaxOutputTokens,
			Temperature:     temperature,
			SchemaJSON:      providerSchema(request),
			Attachments:     requestAttachments(request),
			SpendLimit:      request.SpendLimit,
		})
		if err != nil {
			usageIncomplete = true
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		if request.SpendLimit != nil && !spendAuthorizationMatches(request, execution.SpendAuthorization) {
			usageIncomplete = true
			failures = append(failures, candidate.Provider+": provider owner did not return matching spend-cap authorization")
			continue
		}
		value, usage, err := decodeProviderResponse(candidate.Provider, execution.OutputText, prompt)
		if err != nil {
			usageIncomplete = true
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		if usageIncomplete {
			usage.inputProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
			usage.outputProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
			usage.costProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
		}
		return ProviderResult{
			ValueJSON: value, Provider: resolved.Provider, Model: resolved.Model,
			CoordinateConvention: resolved.CoordinateConvention,
			InputTokens:          usage.input, OutputTokens: usage.output, CostMicros: usage.cost,
			CreditsCharged:        usage.creditsCharged,
			AccountingStatus:      usage.accountingStatus,
			InputTokensProvenance: usage.inputProvenance, OutputTokensProvenance: usage.outputProvenance, CostProvenance: usage.costProvenance,
			Applied:            applied,
			SpendAuthorization: execution.SpendAuthorization,
			ContextWindow:      resolved.ContextWindow, ProviderCallsMade: providerCallsMade,
		}, nil
	}
	if samplingRefused {
		return ProviderResult{ProviderCallsMade: providerCallsMade}, fmt.Errorf("%w: no candidate for inference role %q honors an explicit temperature (%s)", ErrUnsupportedSampling, request.Role, strings.Join(failures, "; "))
	}
	return ProviderResult{ProviderCallsMade: providerCallsMade}, fmt.Errorf("%w: no inference candidate completed for role %q (%s)", ErrUnavailable, request.Role, strings.Join(failures, "; "))
}

func spendAuthorizationMatches(request ProviderRequest, authorization *inferencev1.SpendAuthorization) bool {
	limit := request.SpendLimit
	if limit == nil || authorization == nil || authorization.GetStatus() != "authorized" || authorization.GetAuthorizationId() == "" ||
		authorization.GetMaxCostMicros() != limit.GetMaxCostMicros() || authorization.GetOperationId() != limit.GetOperationId() ||
		authorization.GetInputFingerprint() != limit.GetInputFingerprint() || authorization.GetFundingRoute() != limit.GetFundingRoute() ||
		authorization.GetImageCount() != limit.GetImageCount() || authorization.GetOutputTokensBound() != request.MaxOutputTokens ||
		authorization.GetInputTokensBound() <= 0 || authorization.GetContextWindowTokens() < authorization.GetInputTokensBound()+authorization.GetOutputTokensBound() ||
		authorization.GetModel() == "" || authorization.GetProviderControlsSha256() == "" || authorization.GetContextSource() != "openrouter-current-endpoint-metadata" && authorization.GetContextSource() != "lpbs-current-endpoint-metadata" {
		return false
	}
	if len(authorization.GetReferenceSha256()) != len(limit.GetReferenceSha256()) {
		return false
	}
	for i, digest := range limit.GetReferenceSha256() {
		if authorization.GetReferenceSha256()[i] != digest {
			return false
		}
	}
	return true
}

func validateGeometrySpendLimit(request ProviderRequest) error {
	geometryRole := strings.HasPrefix(strings.TrimSpace(request.Role), "device.geometry.")
	remote := request.Profile == sharedv1.Profile_PROFILE_REMOTE_ONLY
	if !geometryRole {
		if request.SpendLimit != nil {
			return fmt.Errorf("%w: spend limits are supported only for device geometry roles", ErrUnavailable)
		}
		return nil
	}
	if !remote {
		if request.SpendLimit != nil {
			return fmt.Errorf("%w: local geometry requests must not carry a remote spend limit", ErrUnavailable)
		}
		return nil
	}
	limit := request.SpendLimit
	if limit == nil || limit.GetMaxCostMicros() <= 0 || limit.GetOperationId() == "" || !validHexDigest(limit.GetInputFingerprint()) || limit.GetImageCount() < 1 || limit.GetImageCount() > 4 || int(limit.GetImageCount()) != len(request.Attachments) || len(limit.GetReferenceSha256()) != len(request.Attachments) {
		return fmt.Errorf("%w: remote geometry requires a positive spend ceiling bound to operation, fingerprint, and all reference images", ErrUnavailable)
	}
	if request.MaxProviderCalls != 1 {
		return fmt.Errorf("%w: spend-limited geometry allows one provider execution", ErrUnavailable)
	}
	expectedRoute := "byok"
	if strings.TrimSpace(request.Role) == "device.geometry.vrooli_credits" {
		expectedRoute = "vrooli_credits"
	} else if strings.TrimSpace(request.Role) != "device.geometry.byok" {
		return fmt.Errorf("%w: unsupported remote geometry role", ErrUnavailable)
	}
	if limit.GetFundingRoute() != expectedRoute || !validOperationID(limit.GetOperationId()) {
		return fmt.Errorf("%w: spend authorization route or operation identity does not match the request", ErrUnavailable)
	}
	for index, attachment := range request.Attachments {
		if attachment == nil || len(attachment.GetInlineBytes()) == 0 || attachment.GetModality() != sharedv1.Modality_MODALITY_IMAGE {
			return fmt.Errorf("%w: spend-limited geometry requires pinned inline image attachments", ErrUnavailable)
		}
		digest := sha256.Sum256(attachment.GetInlineBytes())
		if hex.EncodeToString(digest[:]) != strings.ToLower(strings.TrimSpace(limit.GetReferenceSha256()[index])) {
			return fmt.Errorf("%w: reference image digest does not match the spend limit", ErrUnavailable)
		}
	}
	return nil
}

func validHexDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validOperationID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for index, char := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if char != '-' {
				return false
			}
			continue
		}
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func (r *ResourceRepository) Embed(ctx context.Context, role string, texts []string) (EmbeddingResult, error) {
	definition, ok := r.catalog.Roles[strings.TrimSpace(role)]
	if !ok {
		return EmbeddingResult{}, fmt.Errorf("%w: inference role %q is not declared", ErrUnavailable, role)
	}
	if len(texts) == 0 {
		return EmbeddingResult{}, errors.New("embedding texts are required")
	}
	var result EmbeddingResult
	for _, text := range texts {
		var vector []float64
		var providerName, model string
		var inputTokens, costMicros int64
		var failures []string
		for _, candidate := range definition.Candidates {
			adapter, available := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
			if !available {
				failures = append(failures, candidate.Provider+": adapter unavailable")
				continue
			}
			resolved, err := adapter.ResolveRole(ctx, candidate.ResourceRole)
			if err != nil {
				failures = append(failures, err.Error())
				continue
			}
			execution, err := adapter.Execute(ctx, providers.ExecutionRequest{Kind: sharedv1.RequestKind_REQUEST_KIND_TEXT_EMBEDDING, Role: candidate.ResourceRole, InputText: text, Timeout: definition.Timeout()})
			if err != nil {
				failures = append(failures, err.Error())
				continue
			}
			vector, inputTokens, costMicros, err = decodeEmbeddingResponse(candidate.Provider, execution.OutputText, text)
			if err != nil {
				failures = append(failures, err.Error())
				continue
			}
			providerName, model = resolved.Provider, resolved.Model
			break
		}
		if len(vector) == 0 {
			return EmbeddingResult{}, fmt.Errorf("%w: no embedding candidate completed for %q (%s)", ErrUnavailable, role, strings.Join(failures, "; "))
		}
		if result.Dimension == 0 {
			result.Dimension = len(vector)
		} else if result.Dimension != len(vector) {
			return EmbeddingResult{}, errors.New("embedding_dimension_mismatch")
		}
		result.Vectors = append(result.Vectors, vector)
		result.Provider, result.Model = providerName, model
		result.InputTokens += inputTokens
		result.CostMicros += costMicros
		result.InputTokensProvenance = combineProvenance(result.InputTokensProvenance, inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE)
		result.CostProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
	}
	return result, nil
}

func decodeEmbeddingResponse(provider, raw, input string) ([]float64, int64, int64, error) {
	var response struct {
		Embedding  []float64   `json:"embedding"`
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		return nil, 0, 0, fmt.Errorf("decode %s embedding: %w", provider, err)
	}
	vector := response.Embedding
	if len(vector) == 0 && len(response.Embeddings) > 0 {
		vector = response.Embeddings[0]
	}
	if len(vector) == 0 {
		return nil, 0, 0, errors.New("embedding response contained no vector")
	}
	return vector, estimateTokens(input), 0, nil
}

func requestAttachments(request ProviderRequest) []*sharedv1.Attachment {
	out := append([]*sharedv1.Attachment{}, request.Attachments...)
	for _, turn := range request.Turns {
		if turn != nil {
			out = append(out, turn.GetAttachments()...)
		}
	}
	return out
}

func providerPrompt(request ProviderRequest, coordinateConvention string) string {
	instruction := strings.TrimSpace(request.Instruction)
	if instruction == "" {
		instruction = "Produce the value described by the schema for the supplied source."
	}
	source := strings.TrimSpace(request.Source)
	if len(request.Turns) > 0 {
		var turns []string
		for _, turn := range request.Turns {
			if turn == nil {
				continue
			}
			turns = append(turns, strings.TrimSpace(turn.GetRole())+": "+strings.TrimSpace(turn.GetText()))
		}
		if len(turns) > 0 {
			source = strings.Join(turns, "\n")
		}
	}
	prompt := "You are a typed inference provider. Return only one JSON value that satisfies the supplied schema. Do not return Markdown fences or commentary.\n\n" +
		"Instruction (caller intent; separate from schema metadata):\n" + instruction + "\n\n" +
		"JSON Schema:\n" + strings.TrimSpace(providerSchema(request)) + "\n\n" +
		"Source:\n" + source
	if strings.TrimSpace(request.Role) == "locate.visual" {
		prompt += "\n\nVisual grounding contract: return an object with found, bounds, and confidence. bounds must contain exactly four numbers in the model-declared coordinate convention " + coordinateConvention + ". For normalized_1000, every x and y value is in the 0..1000 space, not the 0..1 space. For absolute_pixels, use the submitted image pixel space. If the target is not found, set found to false and bounds to [0,0,0,0]. Return confidence unchanged as a number from 0 to 1."
	}
	return prompt
}

func providerSchema(request ProviderRequest) string {
	if strings.TrimSpace(request.Role) != "locate.visual" {
		return request.SchemaJSON
	}
	// The provider sees its declared coordinate space. The gateway converts this
	// shape to the caller's canonical [0,1] contract before local validation.
	return `{"type":"object","required":["found","bounds","confidence"],"properties":{"found":{"type":"boolean"},"bounds":{"type":"array","items":{"type":"number"}},"confidence":{"type":"number","minimum":0,"maximum":1}}}`
}

type providerUsage struct {
	input, output, cost                               int64
	creditsCharged                                    *int64
	accountingStatus                                  string
	inputProvenance, outputProvenance, costProvenance inferencev1.UsageProvenance
}

func decodeProviderResponse(provider, raw, prompt string) (string, providerUsage, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case providers.ProviderOllama:
		var response struct {
			Response        string `json:"response"`
			PromptEvalCount *int64 `json:"prompt_eval_count"`
			EvalCount       *int64 `json:"eval_count"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", providerUsage{}, fmt.Errorf("decode Ollama response: %w", err)
		}
		if strings.TrimSpace(response.Response) == "" {
			return "", providerUsage{}, errors.New("Ollama returned an empty response")
		}
		usage := providerUsage{input: estimateTokens(prompt), inputProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE, costProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}
		if response.PromptEvalCount != nil && *response.PromptEvalCount >= 0 {
			usage.input, usage.inputProvenance = *response.PromptEvalCount, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
		}
		if response.EvalCount != nil && *response.EvalCount >= 0 {
			usage.output = *response.EvalCount
			usage.outputProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
		} else {
			usage.outputProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
		}
		return ExtractJSONValue(response.Response), usage, nil
	case providers.ProviderOpenRouter:
		var response struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens     *int64   `json:"prompt_tokens"`
				CompletionTokens *int64   `json:"completion_tokens"`
				Cost             *float64 `json:"cost"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", providerUsage{}, fmt.Errorf("decode OpenRouter response: %w", err)
		}
		if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
			return "", providerUsage{}, errors.New("OpenRouter returned no completion")
		}
		usage := providerUsage{}
		if response.Usage != nil && response.Usage.PromptTokens != nil && *response.Usage.PromptTokens >= 0 {
			usage.input, usage.inputProvenance = *response.Usage.PromptTokens, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
		} else {
			usage.input, usage.inputProvenance = estimateTokens(prompt), inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE
		}
		if response.Usage != nil && response.Usage.CompletionTokens != nil && *response.Usage.CompletionTokens >= 0 {
			usage.output, usage.outputProvenance = *response.Usage.CompletionTokens, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
		} else {
			usage.outputProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
		}
		if response.Usage != nil && response.Usage.Cost != nil && !math.IsNaN(*response.Usage.Cost) && !math.IsInf(*response.Usage.Cost, 0) && *response.Usage.Cost >= 0 {
			usage.cost, usage.costProvenance = int64(math.Round(*response.Usage.Cost*1_000_000)), inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
		} else {
			usage.costProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
		}
		return ExtractJSONValue(response.Choices[0].Message.Content), usage, nil
	case providers.ProviderMetered:
		var response struct {
			Content          string `json:"content"`
			PromptTokens     *int64 `json:"prompt_tokens"`
			CompletionTokens *int64 `json:"completion_tokens"`
			CreditsCharged   *int64 `json:"credits_charged"`
			AccountingStatus string `json:"accounting_status"`
			ActualCostMicros *int64 `json:"actual_cost_micros"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", providerUsage{}, fmt.Errorf("decode metered inference response: %w", err)
		}
		if strings.TrimSpace(response.Content) == "" {
			return "", providerUsage{}, errors.New("metered inference returned an empty response")
		}
		usage := providerUsage{costProvenance: inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN}
		if response.PromptTokens != nil && *response.PromptTokens >= 0 {
			usage.input, usage.inputProvenance = *response.PromptTokens, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
		} else {
			usage.input, usage.inputProvenance = estimateTokens(prompt), inferencev1.UsageProvenance_USAGE_PROVENANCE_GATEWAY_ESTIMATE
		}
		if response.CompletionTokens != nil && *response.CompletionTokens >= 0 {
			usage.output, usage.outputProvenance = *response.CompletionTokens, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
		} else {
			usage.outputProvenance = inferencev1.UsageProvenance_USAGE_PROVENANCE_UNKNOWN
		}
		// CreditsCharged is deliberately not converted to USD cost. A present
		// zero is retained as an observed value; absence stays unknown.
		if (response.AccountingStatus == "" || response.AccountingStatus == "settled") && response.CreditsCharged != nil && *response.CreditsCharged >= 0 {
			value := *response.CreditsCharged
			usage.creditsCharged = &value
		}
		if response.AccountingStatus == "settled" || response.AccountingStatus == "pending_reconciliation" {
			usage.accountingStatus = response.AccountingStatus
		}
		if response.ActualCostMicros != nil && *response.ActualCostMicros >= 0 {
			usage.cost, usage.costProvenance = *response.ActualCostMicros, inferencev1.UsageProvenance_USAGE_PROVENANCE_PROVIDER_REPORTED
		}
		return ExtractJSONValue(response.Content), usage, nil
	default:
		return "", providerUsage{}, fmt.Errorf("unsupported provider %q", provider)
	}
}

func estimateTokens(text string) int64 {
	count := int64(len([]rune(strings.TrimSpace(text))) / 4)
	if count < 1 {
		return 1
	}
	return count
}
