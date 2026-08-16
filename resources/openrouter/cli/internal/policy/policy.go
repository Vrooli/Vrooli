// Package policy loads and validates the OpenRouter model role catalog. It is
// the cloud analogue of resources/ollama/cli/internal/policy: roles describe
// intent + capability, models carry concrete OpenRouter slugs, and the resource
// CLI is the only place a runtime model slug is selected. Unlike Ollama, models
// are hosted (no disk/ram/vram), each role declares an endpoint family
// (chat/images/...), and admission is by semantic capability subset rather than
// local capacity.
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
	Model                 string           `json:"model"`
	Fallbacks             []string         `json:"fallbacks"`
	Endpoint              string           `json:"endpoint"`
	RequiredCapabilities  []string         `json:"required_capabilities"`
	PreferredCapabilities []string         `json:"preferred_capabilities,omitempty"`
	Description           string           `json:"description"`
	Preference            int              `json:"preference"`
	RequestDefaults       *RequestDefaults `json:"request_defaults,omitempty"`
	Provenance            Provenance       `json:"provenance"`
}

// RequestDefaults is the bounded, role-owned request lever the resource hands to
// consumers so they never invent sampling/render parameters. Values are clamped
// and enum-validated on load; omission is a valid documented state (the model's
// own defaults apply). It spans both chat (temperature/max_tokens/response_format)
// and image (resolution/aspect_ratio/size/output_format/background/quality)
// surfaces so a single policy shape serves every endpoint family.
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

// Sampling/render bounds — the safe envelope every declared value is checked
// against. A misconfigured policy fails load rather than pushing a runtime out
// of range.
const (
	minTemperature = 0.0
	maxTemperature = 2.0
	minMaxTokens   = 1
)

var (
	allowedResponseFormats = map[string]struct{}{"json_object": {}, "json_schema": {}, "text": {}}
	allowedResolutions     = map[string]struct{}{"512": {}, "1K": {}, "2K": {}, "4K": {}}
	// OpenRouter /api/v1/images accepts png|jpeg|webp only. SVG is NOT an
	// output_format value — vector models (e.g. Recraft) emit SVG natively and
	// the response carries media_type=image/svg+xml. A role wanting vector output
	// declares the model's svg_output capability, never output_format=svg.
	allowedOutputFormats = map[string]struct{}{"png": {}, "jpeg": {}, "webp": {}}
	allowedBackgrounds   = map[string]struct{}{"auto": {}, "transparent": {}, "opaque": {}}
	allowedQualities     = map[string]struct{}{"auto": {}, "low": {}, "medium": {}, "high": {}}
)

func (d *RequestDefaults) validate(path string, errs *[]error) {
	if d == nil {
		return
	}
	if d.Temperature != nil && (*d.Temperature < minTemperature || *d.Temperature > maxTemperature) {
		*errs = append(*errs, fmt.Errorf("%s.temperature %.3f out of range [%.1f,%.1f]", path, *d.Temperature, minTemperature, maxTemperature))
	}
	if d.MaxTokens != nil && *d.MaxTokens < minMaxTokens {
		*errs = append(*errs, fmt.Errorf("%s.max_tokens %d must be >= %d", path, *d.MaxTokens, minMaxTokens))
	}
	validateEnum(errs, path+".response_format", d.ResponseFormat, allowedResponseFormats)
	validateEnum(errs, path+".resolution", d.Resolution, allowedResolutions)
	validateEnum(errs, path+".output_format", d.OutputFormat, allowedOutputFormats)
	validateEnum(errs, path+".background", d.Background, allowedBackgrounds)
	validateEnum(errs, path+".quality", d.Quality, allowedQualities)
}

// ResolveGenerate resolves the three-way precedence for the chat-request levers
// the role owns: an explicit flag wins, an absent flag falls through to the
// role's declared request_defaults, and an absent default leaves the parameter
// off the wire so the upstream provider's own default applies. temperatureFlag
// carries the -1 "unset" sentinel and maxTokensFlag carries 0, mirroring
// resource-ollama's gateway flags. A nil receiver (the role declared no
// request_defaults) is valid and resolves to "flag or nothing".
//
// This lives beside the policy shape rather than in the command layer because
// "what a role declares" and "what a caller may override" are one decision, and
// splitting them is how request_defaults became dead configuration on the
// generate path in the first place.
func (d *RequestDefaults) ResolveGenerate(temperatureFlag float64, maxTokensFlag int) (*float64, int) {
	var temperature *float64
	switch {
	case temperatureFlag >= 0:
		value := temperatureFlag
		temperature = &value
	case d != nil && d.Temperature != nil:
		value := *d.Temperature
		temperature = &value
	}
	maxTokens := maxTokensFlag
	if maxTokens <= 0 && d != nil && d.MaxTokens != nil {
		maxTokens = *d.MaxTokens
	}
	return temperature, maxTokens
}

func validateEnum(errs *[]error, path, value string, allowed map[string]struct{}) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if _, ok := allowed[value]; !ok {
		*errs = append(*errs, fmt.Errorf("%s %q is not an allowed value", path, value))
	}
}

type Model struct {
	Provider             string                `json:"provider"`
	Family               string                `json:"family"`
	Capabilities         []string              `json:"capabilities"`
	Modalities           Modalities            `json:"modalities"`
	CoordinateConvention string                `json:"coordinate_convention,omitempty"`
	Endpoints            []string              `json:"endpoints"`
	ContextWindowTokens  int                   `json:"context_window_tokens,omitempty"`
	DefaultEligible      bool                  `json:"default_eligible"`
	Pricing              *Pricing              `json:"pricing,omitempty"`
	UseCaseNotes         string                `json:"use_case_notes,omitempty"`
	Caveats              []string              `json:"caveats,omitempty"`
	Provenance           map[string]Provenance `json:"provenance"`
}

type Modalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

// Pricing is operator-display cost metadata. Units differ across endpoint
// families (token, image, megapixel, image-token), so every field is optional
// and Unit names the billing model. It is never used for runtime selection.
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

type Constraints struct {
	Endpoints                          []string `json:"endpoints"`
	CapabilityVocabulary               []string `json:"capability_vocabulary"`
	RolePreferenceOrder                []string `json:"role_preference_order"`
	DirectModelExceptionRequiredFields []string `json:"direct_model_exception_required_fields"`
	ProvenanceRequired                 bool     `json:"provenance_required"`
	ProvenanceSourceKinds              []string `json:"provenance_source_kinds"`
	ModalityVocabulary                 []string `json:"modality_vocabulary"`
}

type Provenance struct {
	SourceKind  string `json:"source_kind"`
	Confidence  string `json:"confidence"`
	Source      string `json:"source"`
	ObservedAt  string `json:"observed_at"`
	SampleCount int    `json:"sample_count"`
}

// RoleRequest mirrors the Ollama dependency shape: a scenario may declare a role
// as a bare string or an object with reason/required.
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

// ResolveRequest is the parsed ensure payload: declared roles only. Concrete
// model fields are intentionally absent from the OpenRouter dependency schema —
// the greenfield contract forbids consumer-chosen slugs.
type ResolveRequest struct {
	ModelRoles []RoleRequest `json:"model_roles,omitempty"`
}

type ResolvedRole struct {
	Role     string `json:"role"`
	Model    string `json:"model"`
	Endpoint string `json:"endpoint"`
	Required bool   `json:"required"`
	Reason   string `json:"reason,omitempty"`
}

type ResolveResolution struct {
	Roles    []ResolvedRole `json:"roles"`
	Warnings []string       `json:"warnings,omitempty"`
}

// ResolvedPolicyModel is the stable programmatic shape returned by `policy
// resolve`. Consumers build chat/image request payloads from it and never read
// the policy file directly.
type ResolvedPolicyModel struct {
	SchemaVersion         string                `json:"schema_version"`
	Role                  string                `json:"role,omitempty"`
	Source                string                `json:"source"`
	Model                 string                `json:"model"`
	Endpoint              string                `json:"endpoint"`
	Fallbacks             []string              `json:"fallbacks,omitempty"`
	RequiredCapabilities  []string              `json:"required_capabilities,omitempty"`
	PreferredCapabilities []string              `json:"preferred_capabilities,omitempty"`
	Capabilities          []string              `json:"capabilities"`
	Modalities            Modalities            `json:"modalities"`
	CoordinateConvention  string                `json:"coordinate_convention,omitempty"`
	Endpoints             []string              `json:"endpoints"`
	ContextWindowTokens   int                   `json:"context_window_tokens,omitempty"`
	DefaultEligible       bool                  `json:"default_eligible"`
	Pricing               *Pricing              `json:"pricing,omitempty"`
	RequestDefaults       *RequestDefaults      `json:"request_defaults,omitempty"`
	Provider              string                `json:"provider,omitempty"`
	Family                string                `json:"family,omitempty"`
	RoleProvenance        *Provenance           `json:"role_provenance,omitempty"`
	Provenance            map[string]Provenance `json:"provenance,omitempty"`
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
	if value := strings.TrimSpace(getenv("OPENROUTER_MODEL_POLICY_PATH")); value != "" {
		return value, nil
	}
	for _, key := range []string{"RESOURCE_ROOT", "OPENROUTER_CLI_SOURCE_ROOT"} {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			candidate := filepath.Join(value, "model-policy.json")
			if fileExists(candidate) {
				return candidate, nil
			}
		}
	}
	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_CLI_SOURCE_ROOT"} {
		if value := strings.TrimSpace(getenv(key)); value != "" {
			candidate := filepath.Join(value, "resources", "openrouter", "model-policy.json")
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
			filepath.Join("resources", "openrouter", "model-policy.json"),
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
	return "", errors.New("locate OpenRouter model policy: set OPENROUTER_MODEL_POLICY_PATH, RESOURCE_ROOT, or VROOLI_SOURCE_ROOT")
}

// Resolve maps declared roles to concrete model refs for `ensure`. Unknown roles
// are a hard error; the resolution never falls back to a concrete model outside
// policy.
func (p Policy) Resolve(req ResolveRequest) (ResolveResolution, error) {
	var out ResolveResolution
	seen := map[string]struct{}{}
	var errs []error
	for _, roleReq := range req.ModelRoles {
		roleName := strings.TrimSpace(roleReq.Role)
		if roleName == "" {
			continue
		}
		if _, dup := seen[roleName]; dup {
			continue
		}
		seen[roleName] = struct{}{}
		role, ok := p.Roles[roleName]
		if !ok {
			errs = append(errs, fmt.Errorf("unknown model role %q", roleName))
			continue
		}
		out.Roles = append(out.Roles, ResolvedRole{
			Role:     roleName,
			Model:    role.Model,
			Endpoint: role.Endpoint,
			Required: roleReq.IsRequired(),
			Reason:   strings.TrimSpace(roleReq.Reason),
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
	resolved.Endpoint = role.Endpoint
	resolved.Fallbacks = append([]string{}, role.Fallbacks...)
	resolved.RequiredCapabilities = append([]string{}, role.RequiredCapabilities...)
	resolved.PreferredCapabilities = append([]string{}, role.PreferredCapabilities...)
	resolved.RequestDefaults = role.RequestDefaults
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
	resolved := p.resolvedPolicyModel("model", modelRef, model)
	if len(model.Endpoints) > 0 {
		resolved.Endpoint = model.Endpoints[0]
	}
	return resolved, nil
}

func (p Policy) resolvedPolicyModel(source, modelRef string, model Model) ResolvedPolicyModel {
	return ResolvedPolicyModel{
		SchemaVersion:        p.SchemaVersion,
		Source:               source,
		Model:                modelRef,
		Capabilities:         append([]string{}, model.Capabilities...),
		Modalities:           model.Modalities,
		CoordinateConvention: model.CoordinateConvention,
		Endpoints:            append([]string{}, model.Endpoints...),
		ContextWindowTokens:  model.ContextWindowTokens,
		DefaultEligible:      model.DefaultEligible,
		Pricing:              model.Pricing,
		Provider:             model.Provider,
		Family:               model.Family,
		Provenance:           copyProvenanceMap(model.Provenance),
	}
}

func (p Policy) RoleNames() []string { return keys(p.Roles) }
func (p Policy) ModelRefs() []string { return keys(p.Models) }

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
	capVocab := set(p.Constraints.CapabilityVocabulary)
	if len(capVocab) == 0 {
		errs = append(errs, errors.New("constraints.capability_vocabulary must not be empty"))
	}
	endpoints := set(p.Constraints.Endpoints)
	if len(endpoints) == 0 {
		errs = append(errs, errors.New("constraints.endpoints must not be empty"))
	}
	modalityVocab := set(p.Constraints.ModalityVocabulary)
	if len(modalityVocab) == 0 {
		errs = append(errs, errors.New("constraints.modality_vocabulary must not be empty"))
	}

	for name, model := range p.Models {
		if strings.TrimSpace(model.Provider) == "" {
			errs = append(errs, fmt.Errorf("models.%s.provider is required", name))
		}
		if strings.TrimSpace(model.Family) == "" {
			errs = append(errs, fmt.Errorf("models.%s.family is required", name))
		}
		if len(model.Capabilities) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.capabilities must not be empty", name))
		}
		for _, cap := range model.Capabilities {
			if _, ok := capVocab[cap]; !ok {
				errs = append(errs, fmt.Errorf("models.%s capability %q is not in capability_vocabulary", name, cap))
			}
		}
		if len(model.Endpoints) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.endpoints must not be empty", name))
		}
		for _, ep := range model.Endpoints {
			if _, ok := endpoints[ep]; !ok {
				errs = append(errs, fmt.Errorf("models.%s endpoint %q is not in constraints.endpoints", name, ep))
			}
		}
		if len(model.Modalities.Input) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.modalities.input must not be empty", name))
		}
		if len(model.Modalities.Output) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.modalities.output must not be empty", name))
		}
		for direction, modalities := range map[string][]string{
			"input":  model.Modalities.Input,
			"output": model.Modalities.Output,
		} {
			for _, modality := range modalities {
				if _, ok := modalityVocab[modality]; !ok {
					errs = append(errs, fmt.Errorf("models.%s.modalities.%s value %q is not in modality_vocabulary", name, direction, modality))
				}
			}
		}
		if len(model.Provenance) == 0 {
			errs = append(errs, fmt.Errorf("models.%s.provenance must not be empty", name))
		}
		for field, prov := range model.Provenance {
			validateProvenance(&errs, "models."+name+".provenance."+field, prov, sourceKinds)
		}
	}

	for name, role := range p.Roles {
		path := "roles." + name
		if strings.TrimSpace(role.Endpoint) == "" {
			errs = append(errs, fmt.Errorf("%s.endpoint is required", path))
		} else if _, ok := endpoints[role.Endpoint]; !ok {
			errs = append(errs, fmt.Errorf("%s.endpoint %q is not in constraints.endpoints", path, role.Endpoint))
		}
		if len(role.RequiredCapabilities) == 0 {
			errs = append(errs, fmt.Errorf("%s.required_capabilities must not be empty", path))
		}
		for _, cap := range role.RequiredCapabilities {
			if _, ok := capVocab[cap]; !ok {
				errs = append(errs, fmt.Errorf("%s required capability %q is not in capability_vocabulary", path, cap))
			}
		}
		for _, cap := range role.PreferredCapabilities {
			if _, ok := capVocab[cap]; !ok {
				errs = append(errs, fmt.Errorf("%s preferred capability %q is not in capability_vocabulary", path, cap))
			}
		}
		if role.Preference <= 0 {
			errs = append(errs, fmt.Errorf("%s.preference must be positive", path))
		}
		// Default and every fallback must exist, serve the role's endpoint, and
		// satisfy the role's required capabilities. This is the cloud admission
		// gate: a role can never resolve a model that cannot do the job.
		p.validateRoleModel(&errs, path+".model", role.Model, role)
		for _, fb := range role.Fallbacks {
			p.validateRoleModel(&errs, path+" fallback "+fb, fb, role)
		}
		role.RequestDefaults.validate(path+".request_defaults", &errs)
		validateProvenance(&errs, path+".provenance", role.Provenance, sourceKinds)
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

func (p Policy) validateRoleModel(errs *[]error, path, modelRef string, role Role) {
	model, ok := p.Models[modelRef]
	if !ok {
		*errs = append(*errs, fmt.Errorf("%s %q is not in models", path, modelRef))
		return
	}
	if !contains(model.Endpoints, role.Endpoint) {
		*errs = append(*errs, fmt.Errorf("%s %q does not serve endpoint %q", path, modelRef, role.Endpoint))
	}
	modelCaps := set(model.Capabilities)
	for _, cap := range role.RequiredCapabilities {
		if _, has := modelCaps[cap]; !has {
			*errs = append(*errs, fmt.Errorf("%s %q lacks required capability %q", path, modelRef, cap))
		}
	}
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

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
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
