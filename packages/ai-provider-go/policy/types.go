// Package policy owns the provider-neutral model-role catalog contract used by
// AI API resources. Provider resources keep their catalogs in model-policy.json;
// this package owns the shape, validation rules, and resolution mechanics.
package policy

import (
	"encoding/json"
	"sort"
	"strings"
)

// Catalog is the on-disk model-policy.json shape. It is a deliberate superset
// of the hosted and local provider catalogs so one contract can serve both
// without forcing a local resource to invent hosted-only fields.
type Catalog struct {
	SchemaVersion  string                `json:"schema_version"`
	Provider       *ProviderProfile      `json:"provider,omitempty"`
	Roles          map[string]Role       `json:"roles"`
	Models         map[string]Model      `json:"models"`
	ExcludedModels []string              `json:"excluded_models,omitempty"`
	Constraints    Constraints           `json:"constraints"`
	Provenance     map[string]Provenance `json:"provenance"`
}

// ProviderProfile describes provider-wide capabilities that are not role or
// model data. The resource manifest remains the owner of endpoint and
// credential values; this block only declares dialect and behavior.
type ProviderProfile struct {
	ID        string   `json:"id"`
	Family    string   `json:"family"`
	Endpoints []string `json:"endpoints,omitempty"`
	// Capabilities are provider-level capabilities (streaming, tools,
	// structured_output, images, embeddings, audio, video, provider_fallbacks,
	// provider_pinning, usage_cost, spend_cap). They describe what the resource
	// can implement, not what any one role requires.
	Capabilities []string `json:"capabilities,omitempty"`
}

// Role is a logical intent plus the bounded request levers it owns. A role
// names a concrete model and fallbacks; consumers resolve the role and never
// read this file directly.
type Role struct {
	Model                 string                          `json:"model"`
	Fallbacks             []string                        `json:"fallbacks"`
	Endpoint              string                          `json:"endpoint,omitempty"`
	RequiredCapabilities  []string                        `json:"required_capabilities"`
	PreferredCapabilities []string                        `json:"preferred_capabilities,omitempty"`
	Description           string                          `json:"description"`
	Preference            int                             `json:"preference"`
	RequestDefaults       *RequestDefaults                `json:"request_defaults,omitempty"`
	SamplingSupport       map[string]SamplingSupportState `json:"sampling_support,omitempty"`
	SamplingDefaults      *SamplingDefaults               `json:"sampling_defaults,omitempty"`
	MaxTokens             *int                            `json:"max_tokens,omitempty"`
	Provenance            Provenance                      `json:"provenance"`
}

// RequestDefaults is the bounded, role-owned request lever the resource hands to
// consumers so they never invent sampling/render parameters. Values are clamped
// and enum-validated on load; omission is a valid documented state (the model's
// own defaults apply). It spans chat and image surfaces so one policy shape
// serves every endpoint family.
type RequestDefaults struct {
	Temperature    *float64 `json:"temperature,omitempty"`
	MaxTokens      *int     `json:"max_tokens,omitempty"`
	ResponseFormat string   `json:"response_format,omitempty"`
	Resolution     string   `json:"resolution,omitempty"`
	AspectRatio    string   `json:"aspect_ratio,omitempty"`
	Size           string   `json:"size,omitempty"`
	OutputFormat   string   `json:"output_format,omitempty"`
	Background     string   `json:"background,omitempty"`
	Quality        string   `json:"quality,omitempty"`
	Stream         *bool    `json:"stream,omitempty"`
}

// SamplingDefaults is the optional, bounded per-role sampling lever used by
// local providers. Values are clamped on read rather than trusted raw.
type SamplingDefaults struct {
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`
}

// Sampling is a fully resolved, clamped sampling triple.
type Sampling struct {
	Temperature    float64 `json:"temperature"`
	TopP           float64 `json:"top_p"`
	TopK           int     `json:"top_k"`
	HasTemperature bool    `json:"has_temperature"`
	HasTopP        bool    `json:"has_top_p"`
	HasTopK        bool    `json:"has_top_k"`
}

// SamplingSupportState is a declared, never-probed statement of how a role's
// upstream provider treats an explicit sampling control.
//
// Declaration rather than probing is load-bearing: there are three real states
// and only one is detectable at runtime. A rejection surfaces as an error, but
// a provider that accepts a control and silently discards it is
// indistinguishable at the call site from one that honours it. A probe would
// report success and be wrong.
type SamplingSupportState string

const (
	SamplingHonored  SamplingSupportState = "honored"
	SamplingIgnored  SamplingSupportState = "ignored"
	SamplingRejected SamplingSupportState = "rejected"
	SamplingUnknown  SamplingSupportState = "unknown"
)

// SupportFor reports the declared state for one parameter. An absent
// declaration is SamplingUnknown, never an error.
func (r Role) SupportFor(parameter string) SamplingSupportState {
	if state, ok := r.SamplingSupport[strings.TrimSpace(parameter)]; ok {
		return state
	}
	return SamplingUnknown
}

// Model is one concrete provider model and its catalog metadata.
type Model struct {
	Provider             string                `json:"provider,omitempty"`
	Family               string                `json:"family"`
	Tags                 []string              `json:"tags,omitempty"`
	Capabilities         []string              `json:"capabilities"`
	Modalities           Modalities            `json:"modalities"`
	CoordinateConvention string                `json:"coordinate_convention,omitempty"`
	Endpoints            []string              `json:"endpoints,omitempty"`
	ContextWindowTokens  int                   `json:"context_window_tokens,omitempty"`
	EmbeddingDimensions  int                   `json:"embedding_dimensions,omitempty"`
	DiskSizeGBEstimate   float64               `json:"disk_size_gb_estimate,omitempty"`
	RAMGBEstimate        float64               `json:"ram_gb_estimate,omitempty"`
	VRAMGBEstimate       float64               `json:"vram_gb_estimate,omitempty"`
	DefaultEligible      bool                  `json:"default_eligible"`
	Pricing              *Pricing              `json:"pricing,omitempty"`
	UseCaseNotes         string                `json:"use_case_notes,omitempty"`
	Caveats              []string              `json:"caveats,omitempty"`
	Provenance           map[string]Provenance `json:"provenance"`
}

// Modality is the provider-neutral inference vocabulary.
type Modality string

const (
	ModalityText   Modality = "text"
	ModalityImage  Modality = "image"
	ModalityVector Modality = "vector"
	ModalityVideo  Modality = "video"
	ModalityAudio  Modality = "audio"
)

type Modalities struct {
	Input  []Modality `json:"input"`
	Output []Modality `json:"output"`
}

// Pricing is operator-display cost metadata. Units differ across endpoint
// families, so every field is optional and Unit names the billing model. It is
// never used for runtime selection.
type Pricing struct {
	Unit               string  `json:"unit"`
	InputPerMTok       float64 `json:"input_per_mtok,omitempty"`
	OutputPerMTok      float64 `json:"output_per_mtok,omitempty"`
	PerImage           float64 `json:"per_image,omitempty"`
	ImageTokensPerMTok float64 `json:"image_tokens_per_mtok,omitempty"`
	InputMP            float64 `json:"input_mp,omitempty"`
	FirstOutputMP      float64 `json:"first_output_mp,omitempty"`
	AdditionalMP       float64 `json:"additional_mp,omitempty"`
}

// Constraints is the shared policy vocabulary and admission envelope.
type Constraints struct {
	Endpoints                          []string `json:"endpoints,omitempty"`
	CapabilityVocabulary               []string `json:"capability_vocabulary,omitempty"`
	RolePreferenceOrder                []string `json:"role_preference_order"`
	DirectModelExceptionRequiredFields []string `json:"direct_model_exception_required_fields,omitempty"`
	ProvenanceRequired                 bool     `json:"provenance_required"`
	ProvenanceSourceKinds              []string `json:"provenance_source_kinds"`
	ModalityVocabulary                 []string `json:"modality_vocabulary"`

	// Local-provider capacity policy. These are zero for hosted providers.
	ResidentModelBudgetPercent       int `json:"resident_model_budget_percent,omitempty"`
	MinimumHostMemoryHeadroomPercent int `json:"minimum_host_memory_headroom_percent,omitempty"`
	MinimumHostDiskHeadroomGB        int `json:"minimum_host_disk_headroom_gb,omitempty"`
	MaxLoadedModelsPolicy            int `json:"max_loaded_models_policy,omitempty"`
	DefaultParallelRequests          int `json:"default_parallel_requests,omitempty"`
}

// Provenance records where a policy or model claim came from.
type Provenance struct {
	SourceKind      string `json:"source_kind"`
	Confidence      string `json:"confidence"`
	Source          string `json:"source"`
	ObservedAt      string `json:"observed_at"`
	HostFingerprint string `json:"host_fingerprint,omitempty"`
	OllamaVersion   string `json:"ollama_version,omitempty"`
	SampleCount     int    `json:"sample_count"`
}

// RoleRequest is the scenario dependency shape: a role name or an object with
// reason/required.
type RoleRequest struct {
	Role     string `json:"role"`
	Reason   string `json:"reason,omitempty"`
	Required *bool  `json:"required,omitempty"`
}

func (r RoleRequest) IsRequired() bool {
	if r.Required == nil {
		return true
	}
	return *r.Required
}

func (r *RoleRequest) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*r = RoleRequest{Role: s}
		return nil
	}
	type alias RoleRequest
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = RoleRequest(aux)
	return nil
}

// DirectModelRequest is the local-provider explicit-model exception.
type DirectModelRequest struct {
	Name         string `json:"name"`
	Tag          string `json:"tag,omitempty"`
	Size         string `json:"size,omitempty"`
	Quantization string `json:"quantization,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Owner        string `json:"owner,omitempty"`
	ReviewAfter  string `json:"review_after,omitempty"`
	Required     *bool  `json:"required,omitempty"`
}

func (m DirectModelRequest) Ref() string {
	name := strings.TrimSpace(m.Name)
	tag := strings.TrimSpace(m.Tag)
	if name == "" {
		return ""
	}
	if tag == "" || strings.Contains(name, ":") {
		return name
	}
	return name + ":" + tag
}

func (m DirectModelRequest) IsRequired() bool {
	if m.Required == nil {
		return true
	}
	return *m.Required
}

func (m *DirectModelRequest) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*m = DirectModelRequest{Name: s}
		return nil
	}
	type alias DirectModelRequest
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*m = DirectModelRequest(aux)
	return nil
}

// ResolveRequest is the parsed ensure payload.
type ResolveRequest struct {
	ModelRoles      []RoleRequest        `json:"model_roles,omitempty"`
	Models          []DirectModelRequest `json:"models,omitempty"`
	DeprecatedModel string               `json:"model,omitempty"`
}

type ResolveResolution struct {
	Roles    []ResolvedRole `json:"roles"`
	Warnings []string       `json:"warnings,omitempty"`
}

type ResolvedRole struct {
	Role     string `json:"role"`
	Model    string `json:"model"`
	Endpoint string `json:"endpoint,omitempty"`
	Required bool   `json:"required"`
	Reason   string `json:"reason,omitempty"`
}

// ResolvedPolicyModel is the stable programmatic shape returned by `policy
// resolve`. Consumers build requests from it and never read the policy file
// directly.
type ResolvedPolicyModel struct {
	SchemaVersion         string                          `json:"schema_version"`
	Role                  string                          `json:"role,omitempty"`
	Source                string                          `json:"source"`
	Model                 string                          `json:"model"`
	Endpoint              string                          `json:"endpoint,omitempty"`
	Fallbacks             []string                        `json:"fallbacks,omitempty"`
	ExcludedModels        []string                        `json:"excluded_models,omitempty"`
	RequiredCapabilities  []string                        `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string                        `json:"preferred_capabilities,omitempty"`
	Capabilities          []string                        `json:"capabilities"`
	Modalities            Modalities                      `json:"modalities"`
	CoordinateConvention  string                          `json:"coordinate_convention,omitempty"`
	Endpoints             []string                        `json:"endpoints,omitempty"`
	ContextWindowTokens   int                             `json:"context_window_tokens,omitempty"`
	EmbeddingDimensions   int                             `json:"embedding_dimensions,omitempty"`
	DiskSizeGBEstimate    float64                         `json:"disk_size_gb_estimate,omitempty"`
	RAMGBEstimate         float64                         `json:"ram_gb_estimate,omitempty"`
	VRAMGBEstimate        float64                         `json:"vram_gb_estimate,omitempty"`
	DefaultEligible       bool                            `json:"default_eligible"`
	Pricing               *Pricing                        `json:"pricing,omitempty"`
	RequestDefaults       *RequestDefaults                `json:"request_defaults,omitempty"`
	SamplingDefaults      *SamplingDefaults               `json:"sampling_defaults,omitempty"`
	Sampling              *Sampling                       `json:"sampling,omitempty"`
	SamplingSupport       map[string]SamplingSupportState `json:"sampling_support,omitempty"`
	MaxTokens             *int                            `json:"max_tokens,omitempty"`
	Provider              string                          `json:"provider,omitempty"`
	Family                string                          `json:"family,omitempty"`
	Preference            int                             `json:"preference,omitempty"`
	RoleProvenance        *Provenance                     `json:"role_provenance,omitempty"`
	Provenance            map[string]Provenance           `json:"provenance,omitempty"`
}

// RoleNames returns sorted role names.
func (c Catalog) RoleNames() []string { return keys(c.Roles) }

// ModelRefs returns sorted model references.
func (c Catalog) ModelRefs() []string { return keys(c.Models) }

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// LocalResolution is the local-provider ensure resolution. It admits explicit
// direct model exceptions alongside role resolution.
type LocalResolution struct {
	Models   []ResolvedModel `json:"models"`
	Warnings []string        `json:"warnings,omitempty"`
}

// ResolvedModel is one local model reference selected by ensure.
type ResolvedModel struct {
	Ref      string `json:"ref"`
	Source   string `json:"source"`
	Role     string `json:"role,omitempty"`
	Reason   string `json:"reason,omitempty"`
	Owner    string `json:"owner,omitempty"`
	Required bool   `json:"required"`
}
