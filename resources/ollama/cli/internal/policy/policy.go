// Package policy loads and validates the Ollama model role catalog.
package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
)

func isSHA256(value string) bool {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	return err == nil && len(decoded) == 32
}

type Policy struct {
	SchemaVersion string                `json:"schema_version"`
	Roles         map[string]Role       `json:"roles"`
	Models        map[string]Model      `json:"models"`
	Constraints   Constraints           `json:"constraints"`
	Provenance    map[string]Provenance `json:"provenance"`
	Retention     Retention             `json:"retention"`
	Residency     Residency             `json:"residency"`
	loadedSHA256  string
}

// LoadedSHA256 reports the digest of the exact policy bytes read by the
// loader. It is empty for Policy values assembled without LoadFile.
func (p Policy) LoadedSHA256() string { return p.loadedSHA256 }

type Role struct {
	Model                string            `json:"model"`
	Fallbacks            []string          `json:"fallbacks"`
	RequiredCapabilities []string          `json:"required_capabilities"`
	Description          string            `json:"description"`
	Preference           int               `json:"preference"`
	SamplingDefaults     *SamplingDefaults `json:"sampling_defaults,omitempty"`
	// SamplingSupport declares how this role's models treat an explicit
	// sampling control, per parameter. See SamplingSupportState.
	SamplingSupport map[string]SamplingSupportState `json:"sampling_support,omitempty"`
	// MaxTokens is the role-owned output cap applied when a caller passes no
	// --max-tokens. Without it a local caller who sends nothing is uncapped:
	// num_predict is omitted from the request and generation is bounded only by
	// the context window. Omission remains valid and means exactly that.
	MaxTokens  *int       `json:"max_tokens,omitempty"`
	Provenance Provenance `json:"provenance"`
}

// SamplingSupportState is a declared, never-probed statement of how a role's
// provider treats an explicit sampling control.
//
// Declaration rather than probing is load-bearing, not a convenience: a
// provider that accepts a control and silently discards it is indistinguishable
// at the call site from one that honours it, so a successful call is not
// evidence of support. Declare conservatively — SamplingIgnored is the safe
// default and SamplingHonored requires first-party evidence.
type SamplingSupportState string

const (
	// SamplingHonored means the provider applies the value.
	SamplingHonored SamplingSupportState = "honored"
	// SamplingIgnored means the provider accepts the field and discards it.
	SamplingIgnored SamplingSupportState = "ignored"
	// SamplingRejected means the provider fails the request when the field is
	// present. A consumer must omit the control or route elsewhere.
	SamplingRejected SamplingSupportState = "rejected"
	// SamplingUnknown means the support state is not established. Consumers
	// treat it as SamplingIgnored: best effort, no promise.
	SamplingUnknown SamplingSupportState = "unknown"
)

// samplingSupportStates is the closed value vocabulary. samplingParameters is
// the closed key vocabulary: an unrecognised key would otherwise degrade
// silently to "undeclared", which is the exact failure this declaration exists
// to prevent.
var (
	samplingSupportStates = map[SamplingSupportState]struct{}{
		SamplingHonored: {}, SamplingIgnored: {}, SamplingRejected: {}, SamplingUnknown: {},
	}
	samplingParameters = map[string]struct{}{
		"temperature": {}, "top_p": {}, "top_k": {}, "seed": {},
	}
)

// SupportFor reports the declared state for one parameter. An absent
// declaration is SamplingUnknown, never an error: a role that says nothing has
// made no promise.
func (r Role) SupportFor(parameter string) SamplingSupportState {
	if state, ok := r.SamplingSupport[strings.TrimSpace(parameter)]; ok {
		return state
	}
	return SamplingUnknown
}

func validateSamplingSupport(errs *[]error, path string, support map[string]SamplingSupportState) {
	for parameter, state := range support {
		if _, ok := samplingParameters[strings.TrimSpace(parameter)]; !ok {
			*errs = append(*errs, fmt.Errorf("%s key %q is not a known sampling parameter", path, parameter))
		}
		if _, ok := samplingSupportStates[state]; !ok {
			*errs = append(*errs, fmt.Errorf("%s.%s %q is not an allowed support state", path, parameter, state))
		}
	}
}

// SamplingDefaults is an optional, bounded per-role sampling lever. It is the
// control surface for "how deterministic should this role's generations be":
// a tool-grounding role (code.local) pins a low temperature so weak local
// models stop fabricating; a chat role keeps moderate variety. Values are
// clamped on read (see Resolve) — never trusted raw — and omission is a valid,
// documented state: no sampling keys are pinned and the model's own defaults
// apply.
type SamplingDefaults struct {
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`
}

// Sampling is a fully-resolved, clamped sampling triple ready to write into a
// runtime config (e.g. opencode.json provider options). Only the fields the
// role actually declared are marked present.
type Sampling struct {
	Temperature    float64 `json:"temperature"`
	TopP           float64 `json:"top_p"`
	TopK           int     `json:"top_k"`
	HasTemperature bool    `json:"has_temperature"`
	HasTopP        bool    `json:"has_top_p"`
	HasTopK        bool    `json:"has_top_k"`
}

// Sampling bound constants — the safe envelope every declared value is clamped
// into on read. A misconfigured policy can never push a runtime out of range.
const (
	minTemperature = 0.0
	maxTemperature = 2.0
	minTopP        = 0.0
	maxTopP        = 1.0
	minTopK        = 0
	maxTopK        = 1000
	// minMaxTokens mirrors the OpenRouter policy envelope so the two resources
	// reject the same misconfiguration. A cap below 1 is not a cap.
	minMaxTokens = 1
)

// Resolve clamps the declared sampling into the safe envelope. A nil receiver
// (role declared no sampling_defaults) resolves to an all-absent Sampling so
// callers omit the keys and fall back to the model's own defaults.
func (s *SamplingDefaults) Resolve() Sampling {
	var out Sampling
	if s == nil {
		return out
	}
	if s.Temperature != nil {
		out.Temperature = clampFloat(*s.Temperature, minTemperature, maxTemperature)
		out.HasTemperature = true
	}
	if s.TopP != nil {
		out.TopP = clampFloat(*s.TopP, minTopP, maxTopP)
		out.HasTopP = true
	}
	if s.TopK != nil {
		out.TopK = clampInt(*s.TopK, minTopK, maxTopK)
		out.HasTopK = true
	}
	return out
}

func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

type Model struct {
	Rights               *sharedpolicy.ArtifactRights          `json:"rights,omitempty"`
	RuntimeIdentity      *sharedpolicy.RuntimeArtifactIdentity `json:"runtime_identity,omitempty"`
	Family               string                                `json:"family"`
	Tags                 []string                              `json:"tags"`
	Capabilities         []string                              `json:"capabilities"`
	Modalities           Modalities                            `json:"modalities"`
	CoordinateConvention string                                `json:"coordinate_convention,omitempty"`
	ContextWindowTokens  int                                   `json:"context_window_tokens,omitempty"`
	EmbeddingDimensions  int                                   `json:"embedding_dimensions,omitempty"`
	DiskSizeGBEstimate   float64                               `json:"disk_size_gb_estimate"`
	RAMGBEstimate        float64                               `json:"ram_gb_estimate"`
	VRAMGBEstimate       float64                               `json:"vram_gb_estimate"`
	DefaultEligible      bool                                  `json:"default_eligible"`
	UseCaseNotes         string                                `json:"use_case_notes"`
	Caveats              []string                              `json:"caveats"`
	Provenance           map[string]Provenance                 `json:"provenance"`
}

// Modality is the provider-neutral inference vocabulary shared by local and
// hosted model catalogs. Video and audio are deliberately admitted here even
// though no Ollama role currently consumes them; adding a vocabulary value is
// not permission to route a role through it.
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

type Constraints struct {
	ResidentModelBudgetPercent         int      `json:"resident_model_budget_percent"`
	MinimumHostMemoryHeadroomPercent   int      `json:"minimum_host_memory_headroom_percent"`
	MinimumHostDiskHeadroomGB          int      `json:"minimum_host_disk_headroom_gb"`
	DefaultParallelRequests            int      `json:"default_parallel_requests"`
	RolePreferenceOrder                []string `json:"role_preference_order"`
	DirectModelExceptionRequiredFields []string `json:"direct_model_exception_required_fields"`
	ProvenanceRequired                 bool     `json:"provenance_required"`
	ProvenanceSourceKinds              []string `json:"provenance_source_kinds"`
	ModalityVocabulary                 []string `json:"modality_vocabulary"`
}

type Provenance struct {
	SourceKind      string `json:"source_kind"`
	Confidence      string `json:"confidence"`
	Source          string `json:"source"`
	ObservedAt      string `json:"observed_at"`
	HostFingerprint string `json:"host_fingerprint,omitempty"`
	OllamaVersion   string `json:"ollama_version,omitempty"`
	SampleCount     int    `json:"sample_count"`
}

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

type ResolveRequest struct {
	ModelRoles      []RoleRequest        `json:"model_roles,omitempty"`
	Models          []DirectModelRequest `json:"models,omitempty"`
	DeprecatedModel string               `json:"model,omitempty"`
}

type Resolution struct {
	Models   []ResolvedModel `json:"models"`
	Warnings []string        `json:"warnings,omitempty"`
}

type ResolvedModel struct {
	Ref      string `json:"ref"`
	Source   string `json:"source"`
	Role     string `json:"role,omitempty"`
	Reason   string `json:"reason,omitempty"`
	Owner    string `json:"owner,omitempty"`
	Required bool   `json:"required"`
}

type ResolvedPolicyModel struct {
	SchemaVersion        string            `json:"schema_version"`
	Role                 string            `json:"role,omitempty"`
	Source               string            `json:"source"`
	Model                string            `json:"model"`
	Fallbacks            []string          `json:"fallbacks,omitempty"`
	RequiredCapabilities []string          `json:"required_capabilities,omitempty"`
	Capabilities         []string          `json:"capabilities"`
	Modalities           Modalities        `json:"modalities"`
	CoordinateConvention string            `json:"coordinate_convention,omitempty"`
	ContextWindowTokens  int               `json:"context_window_tokens,omitempty"`
	EmbeddingDimensions  int               `json:"embedding_dimensions,omitempty"`
	DiskSizeGBEstimate   float64           `json:"disk_size_gb_estimate"`
	RAMGBEstimate        float64           `json:"ram_gb_estimate"`
	VRAMGBEstimate       float64           `json:"vram_gb_estimate"`
	DefaultEligible      bool              `json:"default_eligible"`
	SamplingDefaults     *SamplingDefaults `json:"sampling_defaults,omitempty"`
	Sampling             *Sampling         `json:"sampling,omitempty"`
	// SamplingSupport and MaxTokens are role-owned, so they are populated only
	// by ResolveRole. A direct model reference has no role to declare them.
	SamplingSupport map[string]SamplingSupportState       `json:"sampling_support,omitempty"`
	MaxTokens       *int                                  `json:"max_tokens,omitempty"`
	RoleProvenance  *Provenance                           `json:"role_provenance,omitempty"`
	Provenance      map[string]Provenance                 `json:"provenance,omitempty"`
	Rights          *sharedpolicy.ArtifactRights          `json:"rights,omitempty"`
	RightsDecision  *sharedpolicy.RightsDecision          `json:"rights_decision,omitempty"`
	RuntimeIdentity *sharedpolicy.RuntimeArtifactIdentity `json:"runtime_identity,omitempty"`
	// Preference is the role's policy ranking. It is zero for a direct model
	// reference, which has no role to rank.
	Preference int `json:"preference,omitempty"`
}

func LoadFile(path string) (Policy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, err
	}
	// Reject the retired experimental context-tier field explicitly. Silently
	// ignoring it would make a stale configuration look active to operators.
	var shape struct {
		Roles map[string]map[string]json.RawMessage `json:"roles"`
	}
	if err := json.Unmarshal(raw, &shape); err != nil {
		return Policy{}, fmt.Errorf("parse model policy: %w", err)
	}
	for name, role := range shape.Roles {
		if _, exists := role["runtime_policy"]; exists {
			return Policy{}, fmt.Errorf("roles.%s.runtime_policy is unsupported; LM9 must reintroduce context tiers only after measured suites are admitted", name)
		}
	}
	var p Policy
	if err := json.Unmarshal(raw, &p); err != nil {
		return Policy{}, fmt.Errorf("parse model policy: %w", err)
	}
	policyDigest := sha256.Sum256(raw)
	p.loadedSHA256 = hex.EncodeToString(policyDigest[:])
	if err := p.Validate(); err != nil {
		return Policy{}, err
	}
	return p, nil
}

func LoadDefaultFile(getenv func(string) string) (Policy, string, error) {
	path, err := DefaultPolicyPath(getenv)
	if err != nil {
		return Policy{}, "", err
	}
	p, err := LoadFile(path)
	if err != nil {
		return Policy{}, "", err
	}
	return p, path, nil
}

func DefaultPolicyPath(getenv func(string) string) (string, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	for _, key := range []string{"OLLAMA_MODEL_POLICY_PATH"} {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			return value, nil
		}
	}
	for _, key := range []string{"RESOURCE_ROOT", "OLLAMA_CLI_SOURCE_ROOT"} {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			candidate := filepath.Join(value, "model-policy.json")
			if fileExists(candidate) {
				return candidate, nil
			}
		}
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT"} {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			candidate := filepath.Join(value, "resources", "ollama", "model-policy.json")
			if fileExists(candidate) {
				return candidate, nil
			}
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := wd; ; dir = filepath.Dir(dir) {
		for _, rel := range []string{
			filepath.Join("resources", "ollama", "model-policy.json"),
			filepath.Join("..", "..", "..", "model-policy.json"),
		} {
			candidate := filepath.Clean(filepath.Join(dir, rel))
			if fileExists(candidate) {
				return candidate, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "", errors.New("locate Ollama model policy: set OLLAMA_MODEL_POLICY_PATH, RESOURCE_ROOT, or VROOLI_SOURCE_ROOT")
}

// LoadDefaultRightsContext reads the same versioned operator-owned context as
// AI Gateway. The file path comes from deployment configuration or the source
// tree root; request flags and input data never choose it.
func LoadDefaultRightsContext(getenv func(string) string) (sharedpolicy.RightsContext, error) {
	context, err := sharedpolicy.LoadDefaultRightsContext(getenv)
	if err != nil {
		return sharedpolicy.RightsContext{}, fmt.Errorf("load trusted model rights context: %w", err)
	}
	return context, nil
}

func (p Policy) Resolve(req ResolveRequest) (Resolution, error) {
	var out Resolution
	seen := map[string]int{}
	add := func(m ResolvedModel) {
		m.Ref = strings.TrimSpace(m.Ref)
		if m.Ref == "" {
			return
		}
		if idx, ok := seen[m.Ref]; ok {
			if out.Models[idx].Source == "direct" && m.Source == "role" {
				out.Models[idx] = m
			}
			return
		}
		seen[m.Ref] = len(out.Models)
		out.Models = append(out.Models, m)
	}

	var errs []error
	for _, roleReq := range req.ModelRoles {
		roleName := strings.TrimSpace(roleReq.Role)
		if roleName == "" {
			continue
		}
		role, ok := p.Roles[roleName]
		if !ok {
			errs = append(errs, fmt.Errorf("unknown model role %q", roleName))
			continue
		}
		add(ResolvedModel{
			Ref:      role.Model,
			Source:   "role",
			Role:     roleName,
			Reason:   strings.TrimSpace(roleReq.Reason),
			Required: roleReq.IsRequired(),
		})
	}

	if model := strings.TrimSpace(req.DeprecatedModel); model != "" {
		out.Warnings = append(out.Warnings, fmt.Sprintf("dependency field model is deprecated; use model_roles or a documented models exception for %q", model))
		add(ResolvedModel{Ref: model, Source: "direct", Required: true})
	}

	requiredFields := set(p.Constraints.DirectModelExceptionRequiredFields)
	for _, direct := range req.Models {
		ref := direct.Ref()
		if ref == "" {
			continue
		}
		var missing []string
		if _, required := requiredFields["reason"]; required && strings.TrimSpace(direct.Reason) == "" {
			missing = append(missing, "reason")
		}
		if _, required := requiredFields["owner"]; required && strings.TrimSpace(direct.Owner) == "" {
			missing = append(missing, "owner")
		}
		if _, required := requiredFields["review_after"]; required && strings.TrimSpace(direct.ReviewAfter) == "" {
			missing = append(missing, "review_after")
		}
		if len(missing) > 0 {
			out.Warnings = append(out.Warnings, fmt.Sprintf("direct model %q is missing exception metadata: %s", ref, strings.Join(missing, ", ")))
		} else {
			out.Warnings = append(out.Warnings, fmt.Sprintf("direct model %q bypasses model_roles; keep reason/owner/review metadata current", ref))
		}
		if _, known := p.Models[ref]; !known {
			out.Warnings = append(out.Warnings, fmt.Sprintf("direct model %q is not in model-policy catalog; capacity planner estimates may be unavailable", ref))
		}
		add(ResolvedModel{
			Ref:      ref,
			Source:   "direct",
			Reason:   strings.TrimSpace(direct.Reason),
			Owner:    strings.TrimSpace(direct.Owner),
			Required: direct.IsRequired(),
		})
	}
	return out, errors.Join(errs...)
}

func (p Policy) ResolveRole(roleName string) (ResolvedPolicyModel, error) {
	roleName = strings.TrimSpace(roleName)
	if roleName == "" {
		return ResolvedPolicyModel{}, errors.New("role is required")
	}
	role, ok := p.Roles[roleName]
	if !ok {
		return ResolvedPolicyModel{}, fmt.Errorf("unknown model role %q", roleName)
	}
	model, ok := p.Models[role.Model]
	if !ok {
		return ResolvedPolicyModel{}, fmt.Errorf("role %q resolved unknown model %q", roleName, role.Model)
	}
	resolved := p.resolvedPolicyModel("role", role.Model, model)
	resolved.Role = roleName
	resolved.Preference = role.Preference
	resolved.Fallbacks = append([]string{}, role.Fallbacks...)
	resolved.RequiredCapabilities = append([]string{}, role.RequiredCapabilities...)
	resolved.SamplingDefaults = role.SamplingDefaults
	sampling := role.SamplingDefaults.Resolve()
	resolved.Sampling = &sampling
	resolved.SamplingSupport = copySamplingSupport(role.SamplingSupport)
	if role.MaxTokens != nil {
		maxTokens := *role.MaxTokens
		resolved.MaxTokens = &maxTokens
	}
	roleProvenance := role.Provenance
	resolved.RoleProvenance = &roleProvenance
	return resolved, nil
}

// ResolveRoleWithRights returns the same resolved model plus a fresh decision
// for the exact model artifact, use and trusted organization context.
func (p Policy) ResolveRoleWithRights(roleName string, use sharedpolicy.RightsUse, context sharedpolicy.RightsContext, now time.Time) (ResolvedPolicyModel, error) {
	resolved, err := p.ResolveRole(roleName)
	if err != nil {
		return ResolvedPolicyModel{}, err
	}
	decision, err := p.EvaluateModelRights(resolved.Model, use, context, now)
	if err != nil {
		return ResolvedPolicyModel{}, err
	}
	resolved.RightsDecision = &decision
	return resolved, nil
}

// ResolveRoleWithRightsFallback selects the first role candidate whose declared
// capabilities and current rights both permit the requested use. Rights are
// evaluated against the caller-supplied trusted context; unknown and denied
// candidates are skipped, and the complete chain fails closed if none qualify.
func (p Policy) ResolveRoleWithRightsFallback(roleName string, use sharedpolicy.RightsUse, context sharedpolicy.RightsContext, now time.Time) (ResolvedPolicyModel, error) {
	roleName = strings.TrimSpace(roleName)
	role, ok := p.Roles[roleName]
	if !ok {
		return ResolvedPolicyModel{}, fmt.Errorf("unknown model role %q", roleName)
	}
	candidates := append([]string{role.Model}, role.Fallbacks...)
	var rejected []string
	for _, ref := range candidates {
		model, exists := p.Models[ref]
		if !exists || !hasAllCapabilities(model.Capabilities, role.RequiredCapabilities) {
			rejected = append(rejected, fmt.Sprintf("%s: required capabilities unavailable", ref))
			continue
		}
		decision := sharedpolicy.EvaluateRights(model.Rights, use, context, now)
		if decision.Status != sharedpolicy.RightsAllowed {
			rejected = append(rejected, fmt.Sprintf("%s: rights decision %s (%s)", ref, decision.Status, decision.Reason))
			continue
		}
		resolved := p.resolvedPolicyModel("role", ref, model)
		resolved.Role = roleName
		resolved.Preference = role.Preference
		resolved.Fallbacks = append([]string{}, role.Fallbacks...)
		resolved.RequiredCapabilities = append([]string{}, role.RequiredCapabilities...)
		resolved.SamplingDefaults = role.SamplingDefaults
		sampling := role.SamplingDefaults.Resolve()
		resolved.Sampling = &sampling
		resolved.SamplingSupport = copySamplingSupport(role.SamplingSupport)
		if role.MaxTokens != nil {
			maxTokens := *role.MaxTokens
			resolved.MaxTokens = &maxTokens
		}
		roleProvenance := role.Provenance
		resolved.RoleProvenance = &roleProvenance
		resolved.RightsDecision = &decision
		return resolved, nil
	}
	return ResolvedPolicyModel{}, fmt.Errorf("role %q has no rights-eligible model for %s: %s", roleName, use, strings.Join(rejected, "; "))
}

// RoleAllowsModel verifies that modelRef is the exact primary or fallback
// identity declared for roleName and that it retains all role capabilities.
func (p Policy) RoleAllowsModel(roleName, modelRef string) bool {
	role, ok := p.Roles[strings.TrimSpace(roleName)]
	if !ok {
		return false
	}
	modelRef = strings.TrimSpace(modelRef)
	if role.Model != modelRef {
		found := false
		for _, fallback := range role.Fallbacks {
			if fallback == modelRef {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	model, ok := p.Models[modelRef]
	return ok && hasAllCapabilities(model.Capabilities, role.RequiredCapabilities)
}

func hasAllCapabilities(model, required []string) bool {
	declared := set(model)
	for _, capability := range required {
		if _, ok := declared[capability]; !ok {
			return false
		}
	}
	return true
}

func (p Policy) ResolveModel(modelRef string) (ResolvedPolicyModel, error) {
	modelRef = strings.TrimSpace(modelRef)
	if modelRef == "" {
		return ResolvedPolicyModel{}, errors.New("model is required")
	}
	model, ok := p.Models[modelRef]
	if !ok {
		return ResolvedPolicyModel{}, fmt.Errorf("unknown model %q", modelRef)
	}
	return p.resolvedPolicyModel("model", modelRef, model), nil
}

// ResolveModelReference resolves one exact catalog identity, accepting the
// standard omitted :latest tag only when that full identity exists.
func (p Policy) ResolveModelReference(modelRef string) (ResolvedPolicyModel, error) {
	modelRef = strings.TrimSpace(modelRef)
	resolved, err := p.ResolveModel(modelRef)
	if err == nil || strings.Contains(modelRef, ":") {
		return resolved, err
	}
	return p.ResolveModel(modelRef + ":latest")
}

// EvaluateModelRights evaluates resource-owned facts against trusted context.
// Request payloads must never supply the organization context.
func (p Policy) EvaluateModelRights(modelRef string, use sharedpolicy.RightsUse, context sharedpolicy.RightsContext, now time.Time) (sharedpolicy.RightsDecision, error) {
	modelRef = strings.TrimSpace(modelRef)
	model, ok := p.Models[modelRef]
	if !ok {
		return sharedpolicy.RightsDecision{}, fmt.Errorf("unknown model %q", modelRef)
	}
	return sharedpolicy.EvaluateRights(model.Rights, use, context, now), nil
}

// CheckLocalInferenceRights is an early fail-closed preflight. An allowed
// result still must be bound to the currently installed Ollama model digest by
// AuthorizeLocalInferenceArtifact immediately before the inference call.
func CheckLocalInferenceRights(getenv func(string) string, role, expectedModel string) error {
	_, err := currentLocalInferenceModel(getenv, role, expectedModel)
	return err
}

// CheckExpectedLocalInferenceRights verifies that the current resource-owned
// policy/context reproduce a gateway decision receipt. The paired timestamp
// keeps the receipt digest stable while still binding the exact current facts.
func CheckExpectedLocalInferenceRights(getenv func(string) string, role, expectedModel, expectedDecisionSHA256, expectedEvaluatedAt string) error {
	expectedDecisionSHA256 = strings.TrimSpace(expectedDecisionSHA256)
	expectedEvaluatedAt = strings.TrimSpace(expectedEvaluatedAt)
	if expectedDecisionSHA256 == "" && expectedEvaluatedAt == "" {
		return CheckLocalInferenceRights(getenv, role, expectedModel)
	}
	if expectedDecisionSHA256 == "" || expectedEvaluatedAt == "" || !isSHA256(expectedDecisionSHA256) {
		return errors.New("expected local rights decision requires both a SHA-256 and evaluation timestamp")
	}
	evaluatedAt, err := time.Parse(time.RFC3339Nano, expectedEvaluatedAt)
	if err != nil || evaluatedAt.UTC().Format(time.RFC3339Nano) != expectedEvaluatedAt {
		return errors.New("expected local rights decision timestamp is invalid or noncanonical")
	}
	_, decision, err := currentLocalInferenceDecision(getenv, role, expectedModel, evaluatedAt)
	if err != nil {
		return err
	}
	if decision.DecisionSHA256 != expectedDecisionSHA256 {
		return errors.New("current local rights policy or trusted context differs from the expected decision; refusing inference")
	}
	return nil
}

// CheckExpectedLocalInferenceEvidence checks a plan-time decision against one
// freshly loaded resource policy/context snapshot. When receipt and file hashes
// are present, all four values are required and the canonical receipt is
// recomputed from current owner facts at the plan's timestamp.
func CheckExpectedLocalInferenceEvidence(getenv func(string) string, role, expectedModel, expectedDecisionSHA256, expectedEvaluatedAt, expectedReceiptSHA256, expectedPolicyFileSHA256 string) error {
	if strings.TrimSpace(expectedReceiptSHA256) == "" && strings.TrimSpace(expectedPolicyFileSHA256) == "" {
		return CheckExpectedLocalInferenceRights(getenv, role, expectedModel, expectedDecisionSHA256, expectedEvaluatedAt)
	}
	if strings.TrimSpace(expectedReceiptSHA256) == "" || strings.TrimSpace(expectedPolicyFileSHA256) == "" || strings.TrimSpace(expectedDecisionSHA256) == "" || strings.TrimSpace(expectedEvaluatedAt) == "" {
		return errors.New("expected local rights evidence requires receipt SHA, policy-file SHA, decision SHA, and evaluation timestamp")
	}
	evidence, err := readCurrentLocalInferenceEvidence(getenv, role, expectedModel, expectedEvaluatedAt)
	if err != nil {
		return err
	}
	if evidence.PolicySHA256 != strings.TrimSpace(expectedPolicyFileSHA256) {
		return errors.New("current Ollama policy-file SHA-256 differs from the planned policy; refusing inference")
	}
	if evidence.Receipt.ReceiptSHA256 != strings.TrimSpace(expectedReceiptSHA256) || evidence.Receipt.Decision.DecisionSHA256 != strings.TrimSpace(expectedDecisionSHA256) || evidence.Receipt.Decision.EvaluatedAt != strings.TrimSpace(expectedEvaluatedAt) {
		return errors.New("current Ollama rights receipt differs from the planned rights decision; refusing inference")
	}
	return nil
}

// AuthorizeLocalInferenceArtifact reloads the current policy and operator
// context, resolves the exact role/model again, and binds its rights to the
// digest returned by Ollama's current model inventory. Ollama reports this as
// the SHA-256 digest of the model contents; an absent or mismatched digest is
// never an authorization.
func AuthorizeLocalInferenceArtifact(getenv func(string) string, role, expectedModel, observedDigest string) error {
	return AuthorizeLocalInferenceArtifactDecision(getenv, role, expectedModel, observedDigest, "", "")
}

// AuthorizeLocalInferenceArtifactDecision repeats the exact decision check
// immediately before dispatch and binds it to the live Ollama contents digest.
func AuthorizeLocalInferenceArtifactDecision(getenv func(string) string, role, expectedModel, observedDigest, expectedDecisionSHA256, expectedEvaluatedAt string) error {
	var model Model
	var err error
	if strings.TrimSpace(expectedDecisionSHA256) == "" && strings.TrimSpace(expectedEvaluatedAt) == "" {
		model, err = currentLocalInferenceModel(getenv, role, expectedModel)
	} else {
		expectedDecisionSHA256 = strings.TrimSpace(expectedDecisionSHA256)
		expectedEvaluatedAt = strings.TrimSpace(expectedEvaluatedAt)
		if expectedDecisionSHA256 == "" || expectedEvaluatedAt == "" || !isSHA256(expectedDecisionSHA256) {
			return errors.New("expected local rights decision requires both a SHA-256 and evaluation timestamp")
		}
		evaluatedAt, parseErr := time.Parse(time.RFC3339Nano, expectedEvaluatedAt)
		if parseErr != nil || evaluatedAt.UTC().Format(time.RFC3339Nano) != expectedEvaluatedAt {
			return errors.New("expected local rights decision timestamp is invalid or noncanonical")
		}
		var decision sharedpolicy.RightsDecision
		model, decision, err = currentLocalInferenceDecision(getenv, role, expectedModel, evaluatedAt)
		if err == nil && decision.DecisionSHA256 != expectedDecisionSHA256 {
			err = errors.New("current local rights policy or trusted context differs from the expected decision; refusing inference")
		}
	}
	if err != nil {
		return err
	}
	binding := model.RuntimeIdentity
	if binding == nil || binding.Provider != sharedpolicy.RuntimeProviderOllama || binding.Model != expectedModel ||
		binding.ArtifactID != model.Rights.ArtifactID || binding.ArtifactRevision != model.Rights.Revision ||
		binding.DigestKind != sharedpolicy.RuntimeDigestOllamaModelContents || binding.Source != sharedpolicy.RuntimeSourceOllamaTags ||
		strings.TrimSpace(binding.ObservedAt) == "" || strings.TrimSpace(observedDigest) == "" || binding.Digest != observedDigest {
		return fmt.Errorf("local model %q runtime artifact identity binding is missing or does not match current Ollama inventory; refusing inference", expectedModel)
	}
	return nil
}

// AuthorizeLocalInferenceArtifactEvidence verifies exact plan-time rights,
// policy-file and model identity from one fresh resource snapshot, then binds
// the result to the current Ollama inventory immediately before inference.
func AuthorizeLocalInferenceArtifactEvidence(getenv func(string) string, role, expectedModel, observedDigest, expectedDecisionSHA256, expectedEvaluatedAt, expectedReceiptSHA256, expectedPolicyFileSHA256 string) error {
	evidence, err := readCurrentLocalInferenceEvidence(getenv, role, expectedModel, expectedEvaluatedAt)
	if err != nil {
		return err
	}
	if evidence.PolicySHA256 != strings.TrimSpace(expectedPolicyFileSHA256) || evidence.Receipt.ReceiptSHA256 != strings.TrimSpace(expectedReceiptSHA256) || evidence.Receipt.Decision.DecisionSHA256 != strings.TrimSpace(expectedDecisionSHA256) || evidence.Receipt.Decision.EvaluatedAt != strings.TrimSpace(expectedEvaluatedAt) {
		return errors.New("current Ollama rights receipt, policy-file SHA-256 or decision differs from the planned evidence; refusing inference")
	}
	return requireObservedRuntimeIdentity(evidence.Model, expectedModel, observedDigest)
}

func currentLocalInferenceModel(getenv func(string) string, role, expectedModel string) (Model, error) {
	model, _, err := currentLocalInferenceDecision(getenv, role, expectedModel, time.Now())
	return model, err
}

func currentLocalInferenceDecision(getenv func(string) string, role, expectedModel string, evaluatedAt time.Time) (Model, sharedpolicy.RightsDecision, error) {
	p, modelFacts, context, err := loadCurrentLocalInferenceFacts(getenv, role, expectedModel)
	if err != nil {
		return Model{}, sharedpolicy.RightsDecision{}, err
	}
	decision, err := p.EvaluateModelRights(expectedModel, sharedpolicy.RightsUseLocalInference, context, evaluatedAt)
	if err != nil {
		return Model{}, sharedpolicy.RightsDecision{}, fmt.Errorf("evaluate local model rights for %q: %w", expectedModel, err)
	}
	if decision.Status != sharedpolicy.RightsAllowed {
		return Model{}, decision, fmt.Errorf("local model %q rights decision %s (%s); refusing inference", expectedModel, decision.Status, decision.Reason)
	}
	return modelFacts, decision, nil
}

type localInferenceEvidenceSnapshot struct {
	PolicySHA256 string
	Model        Model
	Receipt      sharedpolicy.RightsDecisionReceipt
}

func readCurrentLocalInferenceEvidence(getenv func(string) string, role, expectedModel, expectedEvaluatedAt string) (localInferenceEvidenceSnapshot, error) {
	evaluatedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(expectedEvaluatedAt))
	if err != nil || evaluatedAt.UTC().Format(time.RFC3339Nano) != strings.TrimSpace(expectedEvaluatedAt) {
		return localInferenceEvidenceSnapshot{}, errors.New("expected local rights decision timestamp is invalid or noncanonical")
	}
	p, model, context, err := loadCurrentLocalInferenceFacts(getenv, role, expectedModel)
	if err != nil {
		return localInferenceEvidenceSnapshot{}, err
	}
	receipt := sharedpolicy.NewRightsDecisionReceipt(sharedpolicy.RightsDecisionReceiptExpectation{
		PolicySchemaVersion: p.SchemaVersion, Role: strings.TrimSpace(role), Provider: sharedpolicy.RuntimeProviderOllama,
		Model: strings.TrimSpace(expectedModel), Use: sharedpolicy.RightsUseLocalInference,
		ArtifactRights: model.Rights, RuntimeIdentity: model.RuntimeIdentity, Context: context,
	}, evaluatedAt)
	if err := sharedpolicy.RequireAllowedRightsDecisionReceipt(receipt); err != nil {
		return localInferenceEvidenceSnapshot{}, fmt.Errorf("local model %q rights decision %s (%s); refusing inference", expectedModel, receipt.Decision.Status, receipt.Decision.Reason)
	}
	return localInferenceEvidenceSnapshot{PolicySHA256: p.LoadedSHA256(), Model: model, Receipt: receipt}, nil
}

func loadCurrentLocalInferenceFacts(getenv func(string) string, role, expectedModel string) (Policy, Model, sharedpolicy.RightsContext, error) {
	p, _, err := LoadDefaultFile(getenv)
	if err != nil {
		return Policy{}, Model{}, sharedpolicy.RightsContext{}, fmt.Errorf("load model rights policy: %w", err)
	}
	modelRef := strings.TrimSpace(expectedModel)
	if modelRef == "" {
		return Policy{}, Model{}, sharedpolicy.RightsContext{}, errors.New("authorized model is required")
	}
	if strings.TrimSpace(role) != "" {
		if !p.RoleAllowsModel(role, modelRef) {
			return Policy{}, Model{}, sharedpolicy.RightsContext{}, fmt.Errorf("model %q is not an eligible candidate for role %q; refusing inference", modelRef, role)
		}
	} else {
		resolved, resolveErr := p.ResolveModelReference(modelRef)
		if resolveErr != nil {
			return Policy{}, Model{}, sharedpolicy.RightsContext{}, fmt.Errorf("resolve model rights identity %q: %w", modelRef, resolveErr)
		}
		if resolved.Model != modelRef {
			return Policy{}, Model{}, sharedpolicy.RightsContext{}, fmt.Errorf("model identity %q is not canonical; current catalog identity is %q", modelRef, resolved.Model)
		}
	}
	context, err := LoadDefaultRightsContext(getenv)
	if err != nil {
		return Policy{}, Model{}, sharedpolicy.RightsContext{}, fmt.Errorf("authorize local model %q: %w", modelRef, err)
	}
	model, ok := p.Models[modelRef]
	if !ok {
		return Policy{}, Model{}, sharedpolicy.RightsContext{}, fmt.Errorf("unknown model %q", modelRef)
	}
	return p, model, context, nil
}

func requireObservedRuntimeIdentity(model Model, expectedModel, observedDigest string) error {
	binding := model.RuntimeIdentity
	if binding == nil || binding.Provider != sharedpolicy.RuntimeProviderOllama || binding.Model != expectedModel ||
		model.Rights == nil || binding.ArtifactID != model.Rights.ArtifactID || binding.ArtifactRevision != model.Rights.Revision ||
		binding.DigestKind != sharedpolicy.RuntimeDigestOllamaModelContents || binding.Source != sharedpolicy.RuntimeSourceOllamaTags ||
		strings.TrimSpace(binding.ObservedAt) == "" || strings.TrimSpace(observedDigest) == "" || binding.Digest != observedDigest {
		return fmt.Errorf("local model %q runtime artifact identity binding is missing or does not match current Ollama inventory; refusing inference", expectedModel)
	}
	return nil
}

func (p Policy) resolvedPolicyModel(source, modelRef string, model Model) ResolvedPolicyModel {
	return ResolvedPolicyModel{
		SchemaVersion: p.SchemaVersion,
		Source:        source,
		Model:         modelRef,
		Capabilities:  append([]string{}, model.Capabilities...),
		Modalities: Modalities{
			Input:  append([]Modality{}, model.Modalities.Input...),
			Output: append([]Modality{}, model.Modalities.Output...),
		},
		CoordinateConvention: model.CoordinateConvention,
		ContextWindowTokens:  model.ContextWindowTokens,
		EmbeddingDimensions:  model.EmbeddingDimensions,
		DiskSizeGBEstimate:   model.DiskSizeGBEstimate,
		RAMGBEstimate:        model.RAMGBEstimate,
		VRAMGBEstimate:       model.VRAMGBEstimate,
		DefaultEligible:      model.DefaultEligible,
		Provenance:           copyProvenanceMap(model.Provenance),
		Rights:               model.Rights,
		RuntimeIdentity:      model.RuntimeIdentity,
	}
}

func (p Policy) RoleNames() []string {
	return keys(p.Roles)
}

func (p Policy) ModelRefs() []string {
	return keys(p.Models)
}

func (p Policy) Validate() error {
	var errs []error
	if strings.TrimSpace(p.SchemaVersion) == "" {
		errs = append(errs, errors.New("schema_version is required"))
	}
	if len(p.Roles) == 0 {
		errs = append(errs, errors.New("roles must not be empty"))
	}
	if len(p.Models) == 0 {
		errs = append(errs, errors.New("models must not be empty"))
	}
	sourceKinds := set(p.Constraints.ProvenanceSourceKinds)
	if len(sourceKinds) == 0 {
		errs = append(errs, errors.New("constraints.provenance_source_kinds must not be empty"))
	}
	modalityVocab := set(p.Constraints.ModalityVocabulary)
	if len(modalityVocab) == 0 {
		errs = append(errs, errors.New("constraints.modality_vocabulary must not be empty"))
	}
	errs = append(errs, p.Retention.validate(p)...)
	errs = append(errs, p.Residency.validate(p)...)
	for name, role := range p.Roles {
		if strings.TrimSpace(role.Model) == "" {
			errs = append(errs, fmt.Errorf("roles.%s.model is required", name))
		} else if _, ok := p.Models[role.Model]; !ok {
			errs = append(errs, fmt.Errorf("roles.%s.model %q is not in models", name, role.Model))
		}
		for _, fallback := range role.Fallbacks {
			if _, ok := p.Models[fallback]; !ok {
				errs = append(errs, fmt.Errorf("roles.%s fallback %q is not in models", name, fallback))
			}
		}
		if len(role.RequiredCapabilities) == 0 {
			errs = append(errs, fmt.Errorf("roles.%s.required_capabilities must not be empty", name))
		}
		if role.Preference <= 0 {
			errs = append(errs, fmt.Errorf("roles.%s.preference must be positive", name))
		}
		if role.MaxTokens != nil && *role.MaxTokens < minMaxTokens {
			errs = append(errs, fmt.Errorf("roles.%s.max_tokens %d must be >= %d", name, *role.MaxTokens, minMaxTokens))
		}
		validateSamplingSupport(&errs, "roles."+name+".sampling_support", role.SamplingSupport)
		validateProvenance(&errs, "roles."+name+".provenance", role.Provenance, sourceKinds)
	}
	for name, model := range p.Models {
		if binding := model.RuntimeIdentity; binding != nil {
			if model.Rights == nil || binding.Provider != sharedpolicy.RuntimeProviderOllama || binding.Model != name ||
				binding.ArtifactID != model.Rights.ArtifactID || binding.ArtifactRevision != model.Rights.Revision ||
				binding.DigestKind != sharedpolicy.RuntimeDigestOllamaModelContents || !isSHA256(binding.Digest) ||
				binding.Source != sharedpolicy.RuntimeSourceOllamaTags {
				errs = append(errs, fmt.Errorf("models.%s.runtime_identity must bind matching Ollama model, artifact rights, digest kind, digest and source", name))
			}
			if _, err := time.Parse("2006-01-02", binding.ObservedAt); err != nil {
				errs = append(errs, fmt.Errorf("models.%s.runtime_identity.observed_at must be YYYY-MM-DD", name))
			}
		}
		if strings.TrimSpace(model.Family) == "" {
			errs = append(errs, fmt.Errorf("models.%s.family is required", name))
		}
		if len(model.Capabilities) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.capabilities must not be empty", name))
		}
		if len(model.Modalities.Input) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.modalities.input must not be empty", name))
		}
		if len(model.Modalities.Output) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.modalities.output must not be empty", name))
		}
		for direction, modalities := range map[string][]Modality{
			"input":  model.Modalities.Input,
			"output": model.Modalities.Output,
		} {
			for _, modality := range modalities {
				if _, ok := modalityVocab[string(modality)]; !ok {
					errs = append(errs, fmt.Errorf("models.%s.modalities.%s value %q is not in modality_vocabulary", name, direction, modality))
				}
			}
		}
		if model.DiskSizeGBEstimate <= 0 {
			errs = append(errs, fmt.Errorf("models.%s.disk_size_gb_estimate must be positive", name))
		}
		if model.RAMGBEstimate <= 0 {
			errs = append(errs, fmt.Errorf("models.%s.ram_gb_estimate must be positive", name))
		}
		if model.VRAMGBEstimate <= 0 {
			errs = append(errs, fmt.Errorf("models.%s.vram_gb_estimate must be positive", name))
		}
		if len(model.Provenance) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.provenance must not be empty", name))
		}
		for field, prov := range model.Provenance {
			validateProvenance(&errs, "models."+name+".provenance."+field, prov, sourceKinds)
		}
	}
	roleNames := set(keys(p.Roles))
	for _, role := range p.Constraints.RolePreferenceOrder {
		if _, ok := roleNames[role]; !ok {
			errs = append(errs, fmt.Errorf("constraints.role_preference_order references unknown role %q", role))
		}
	}
	for name, prov := range p.Provenance {
		validateProvenance(&errs, "provenance."+name, prov, sourceKinds)
	}
	return errors.Join(errs...)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func validateProvenance(errs *[]error, path string, p Provenance, sourceKinds map[string]struct{}) {
	if _, ok := sourceKinds[p.SourceKind]; !ok {
		*errs = append(*errs, fmt.Errorf("%s.source_kind %q is not allowed", path, p.SourceKind))
	}
	switch p.Confidence {
	case "low", "medium", "high", "measured":
	default:
		*errs = append(*errs, fmt.Errorf("%s.confidence %q is not allowed", path, p.Confidence))
	}
	if strings.TrimSpace(p.Source) == "" {
		*errs = append(*errs, fmt.Errorf("%s.source is required", path))
	}
	if strings.TrimSpace(p.ObservedAt) == "" {
		*errs = append(*errs, fmt.Errorf("%s.observed_at is required", path))
	}
	if p.SampleCount < 0 {
		*errs = append(*errs, fmt.Errorf("%s.sample_count must be >= 0", path))
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func set(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func copySamplingSupport(in map[string]SamplingSupportState) map[string]SamplingSupportState {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]SamplingSupportState, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyProvenanceMap(in map[string]Provenance) map[string]Provenance {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]Provenance, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
