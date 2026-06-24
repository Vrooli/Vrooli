// Package policy loads and validates the Ollama model role catalog.
package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Policy struct {
	SchemaVersion string                `json:"schema_version"`
	Roles         map[string]Role       `json:"roles"`
	Models        map[string]Model      `json:"models"`
	Constraints   Constraints           `json:"constraints"`
	Provenance    map[string]Provenance `json:"provenance"`
}

type Role struct {
	Model                string            `json:"model"`
	Fallbacks            []string          `json:"fallbacks"`
	RequiredCapabilities []string          `json:"required_capabilities"`
	Description          string            `json:"description"`
	Preference           int               `json:"preference"`
	SamplingDefaults     *SamplingDefaults `json:"sampling_defaults,omitempty"`
	Provenance           Provenance        `json:"provenance"`
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
	Family              string                `json:"family"`
	Tags                []string              `json:"tags"`
	Capabilities        []string              `json:"capabilities"`
	ContextWindowTokens int                   `json:"context_window_tokens,omitempty"`
	EmbeddingDimensions int                   `json:"embedding_dimensions,omitempty"`
	DiskSizeGBEstimate  float64               `json:"disk_size_gb_estimate"`
	RAMGBEstimate       float64               `json:"ram_gb_estimate"`
	VRAMGBEstimate      float64               `json:"vram_gb_estimate"`
	DefaultEligible     bool                  `json:"default_eligible"`
	UseCaseNotes        string                `json:"use_case_notes"`
	Caveats             []string              `json:"caveats"`
	Provenance          map[string]Provenance `json:"provenance"`
}

type Constraints struct {
	ResidentModelBudgetPercent         int      `json:"resident_model_budget_percent"`
	MinimumHostMemoryHeadroomPercent   int      `json:"minimum_host_memory_headroom_percent"`
	MinimumHostDiskHeadroomGB          int      `json:"minimum_host_disk_headroom_gb"`
	MaxLoadedModelsPolicy              int      `json:"max_loaded_models_policy"`
	DefaultParallelRequests            int      `json:"default_parallel_requests"`
	RolePreferenceOrder                []string `json:"role_preference_order"`
	DirectModelExceptionRequiredFields []string `json:"direct_model_exception_required_fields"`
	ProvenanceRequired                 bool     `json:"provenance_required"`
	ProvenanceSourceKinds              []string `json:"provenance_source_kinds"`
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
	SchemaVersion        string                `json:"schema_version"`
	Role                 string                `json:"role,omitempty"`
	Source               string                `json:"source"`
	Model                string                `json:"model"`
	Fallbacks            []string              `json:"fallbacks,omitempty"`
	RequiredCapabilities []string              `json:"required_capabilities,omitempty"`
	Capabilities         []string              `json:"capabilities"`
	ContextWindowTokens  int                   `json:"context_window_tokens,omitempty"`
	EmbeddingDimensions  int                   `json:"embedding_dimensions,omitempty"`
	DiskSizeGBEstimate   float64               `json:"disk_size_gb_estimate"`
	RAMGBEstimate        float64               `json:"ram_gb_estimate"`
	VRAMGBEstimate       float64               `json:"vram_gb_estimate"`
	DefaultEligible      bool                  `json:"default_eligible"`
	SamplingDefaults     *SamplingDefaults     `json:"sampling_defaults,omitempty"`
	Sampling             *Sampling             `json:"sampling,omitempty"`
	RoleProvenance       *Provenance           `json:"role_provenance,omitempty"`
	Provenance           map[string]Provenance `json:"provenance,omitempty"`
}

func LoadFile(path string) (Policy, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, err
	}
	var p Policy
	if err := json.Unmarshal(raw, &p); err != nil {
		return Policy{}, fmt.Errorf("parse model policy: %w", err)
	}
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
	resolved.Fallbacks = append([]string{}, role.Fallbacks...)
	resolved.RequiredCapabilities = append([]string{}, role.RequiredCapabilities...)
	resolved.SamplingDefaults = role.SamplingDefaults
	sampling := role.SamplingDefaults.Resolve()
	resolved.Sampling = &sampling
	roleProvenance := role.Provenance
	resolved.RoleProvenance = &roleProvenance
	return resolved, nil
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

func (p Policy) resolvedPolicyModel(source, modelRef string, model Model) ResolvedPolicyModel {
	return ResolvedPolicyModel{
		SchemaVersion:       p.SchemaVersion,
		Source:              source,
		Model:               modelRef,
		Capabilities:        append([]string{}, model.Capabilities...),
		ContextWindowTokens: model.ContextWindowTokens,
		EmbeddingDimensions: model.EmbeddingDimensions,
		DiskSizeGBEstimate:  model.DiskSizeGBEstimate,
		RAMGBEstimate:       model.RAMGBEstimate,
		VRAMGBEstimate:      model.VRAMGBEstimate,
		DefaultEligible:     model.DefaultEligible,
		Provenance:          copyProvenanceMap(model.Provenance),
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
		validateProvenance(&errs, "roles."+name+".provenance", role.Provenance, sourceKinds)
	}
	for name, model := range p.Models {
		if strings.TrimSpace(model.Family) == "" {
			errs = append(errs, fmt.Errorf("models.%s.family is required", name))
		}
		if len(model.Capabilities) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.capabilities must not be empty", name))
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
