package inference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"ai-gateway/internal/providers"
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
	var failures []string
	// samplingRefused records that at least one candidate was skipped purely
	// because it could not honour a caller-supplied control. It changes the
	// terminal error from "nothing worked" to the specific, actionable
	// "nothing here can sample the way you asked".
	var samplingRefused bool
	for _, candidate := range definition.Candidates {
		provider := strings.ToLower(strings.TrimSpace(candidate.Provider))
		if request.Profile == sharedv1.Profile_PROFILE_REMOTE_ONLY && provider != providers.ProviderOpenRouter && provider != providers.ProviderMetered {
			continue
		}
		if request.Profile == sharedv1.Profile_PROFILE_LOCAL_ONLY && provider != providers.ProviderOllama {
			continue
		}
		adapter, ok := r.adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
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
				return ProviderResult{}, fmt.Errorf("%w: inference role %q declares no overridable sampling", ErrRoleForbidsSampling, request.Role)
			}
			samplingRefused = true
			failures = append(failures, fmt.Sprintf("%s: temperature not honored by resolved role %s (%s)", candidate.Provider, candidate.ResourceRole, support))
			continue
		}
		maxOutputTokens, capSource := resolveOutputCap(request, resolved)
		prompt := providerPrompt(request, resolved.CoordinateConvention)
		if resolved.ContextWindow > 0 && estimateTokens(prompt)+int64(maxOutputTokens) > int64(resolved.ContextWindow) {
			return ProviderResult{}, fmt.Errorf("%w: assembled input and requested output cap do not fit the resolved model policy", ErrContextOverflow)
		}
		applied := AppliedSettings{
			TemperatureSent:       temperature,
			TemperatureSupport:    support,
			MaxOutputTokens:       maxOutputTokens,
			MaxOutputTokensSource: capSource,
		}
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
		})
		if err != nil {
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		value, inputTokens, outputTokens, costMicros, err := decodeProviderResponse(candidate.Provider, execution.OutputText, prompt)
		if err != nil {
			failures = append(failures, candidate.Provider+": "+err.Error())
			continue
		}
		return ProviderResult{
			ValueJSON: value, Provider: resolved.Provider, Model: resolved.Model,
			CoordinateConvention: resolved.CoordinateConvention,
			InputTokens:          inputTokens, OutputTokens: outputTokens, CostMicros: costMicros,
			Applied:       applied,
			ContextWindow: resolved.ContextWindow,
		}, nil
	}
	if samplingRefused {
		return ProviderResult{}, fmt.Errorf("%w: no candidate for inference role %q honors an explicit temperature (%s)", ErrUnsupportedSampling, request.Role, strings.Join(failures, "; "))
	}
	return ProviderResult{}, fmt.Errorf("%w: no inference candidate completed for role %q (%s)", ErrUnavailable, request.Role, strings.Join(failures, "; "))
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

func decodeProviderResponse(provider, raw, prompt string) (string, int64, int64, int64, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case providers.ProviderOllama:
		var response struct {
			Response  string `json:"response"`
			EvalCount int64  `json:"eval_count"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", 0, 0, 0, fmt.Errorf("decode Ollama response: %w", err)
		}
		if strings.TrimSpace(response.Response) == "" {
			return "", 0, 0, 0, errors.New("Ollama returned an empty response")
		}
		return ExtractJSONValue(response.Response), estimateTokens(prompt), response.EvalCount, 0, nil
	case providers.ProviderOpenRouter:
		var response struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     int64   `json:"prompt_tokens"`
				CompletionTokens int64   `json:"completion_tokens"`
				Cost             float64 `json:"cost"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", 0, 0, 0, fmt.Errorf("decode OpenRouter response: %w", err)
		}
		if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
			return "", 0, 0, 0, errors.New("OpenRouter returned no completion")
		}
		inputTokens := response.Usage.PromptTokens
		if inputTokens == 0 {
			inputTokens = estimateTokens(prompt)
		}
		return ExtractJSONValue(response.Choices[0].Message.Content), inputTokens, response.Usage.CompletionTokens, int64(math.Round(response.Usage.Cost * 1_000_000)), nil
	case providers.ProviderMetered:
		var response struct {
			Content          string `json:"content"`
			PromptTokens     int64  `json:"prompt_tokens"`
			CompletionTokens int64  `json:"completion_tokens"`
			CreditsCharged   int64  `json:"credits_charged"`
		}
		if err := json.Unmarshal([]byte(raw), &response); err != nil {
			return "", 0, 0, 0, fmt.Errorf("decode metered inference response: %w", err)
		}
		if strings.TrimSpace(response.Content) == "" {
			return "", 0, 0, 0, errors.New("metered inference returned an empty response")
		}
		inputTokens := response.PromptTokens
		if inputTokens == 0 {
			inputTokens = estimateTokens(prompt)
		}
		return ExtractJSONValue(response.Content), inputTokens, response.CompletionTokens, 0, nil
	default:
		return "", 0, 0, 0, fmt.Errorf("unsupported provider %q", provider)
	}
}

func estimateTokens(text string) int64 {
	count := int64(len([]rune(strings.TrimSpace(text))) / 4)
	if count < 1 {
		return 1
	}
	return count
}
