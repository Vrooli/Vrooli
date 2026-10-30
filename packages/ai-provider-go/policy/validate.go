package policy

import (
	"errors"
	"fmt"
	"strings"
)

// ValidationProfile selects the admission rules for a provider family.
type ValidationProfile string

const (
	// CloudValidation admits hosted providers: every model and role declares an
	// endpoint, a provider, and a family.
	CloudValidation ValidationProfile = "cloud"
	// LocalValidation admits local runtimes: models declare resource estimates
	// instead of hosted endpoints.
	LocalValidation ValidationProfile = "local"
)

// ValidationOptions lets a resource select the shared admission rules without
// forking the validator. Provider-specific requirements stay explicit.
type ValidationOptions struct {
	Profile                 ValidationProfile
	RequireProvider         bool
	RequireFamily           bool
	RequireModelEndpoints   bool
	RequireResourceEstimate bool
}

// CloudOptions is the hosted-provider admission profile.
func CloudOptions() ValidationOptions {
	return ValidationOptions{
		Profile:               CloudValidation,
		RequireProvider:       true,
		RequireFamily:         true,
		RequireModelEndpoints: true,
	}
}

// LocalOptions is the local-runtime admission profile.
func LocalOptions() ValidationOptions {
	return ValidationOptions{
		Profile:                 LocalValidation,
		RequireFamily:           true,
		RequireResourceEstimate: true,
	}
}

var (
	allowedResponseFormats = map[string]struct{}{"json_object": {}, "json_schema": {}, "text": {}}
	allowedResolutions     = map[string]struct{}{"512": {}, "1K": {}, "2K": {}, "4K": {}}
	// OpenRouter /api/v1/images accepts png|jpeg|webp only. SVG is not an
	// output_format value; vector models emit SVG natively.
	allowedOutputFormats = map[string]struct{}{"png": {}, "jpeg": {}, "webp": {}}
	allowedBackgrounds   = map[string]struct{}{"auto": {}, "transparent": {}, "opaque": {}}
	allowedQualities     = map[string]struct{}{"auto": {}, "low": {}, "medium": {}, "high": {}}
)

// Validate applies the selected admission profile to a catalog.
func Validate(c Catalog, opts ValidationOptions) error {
	var errs []error
	if strings.TrimSpace(c.SchemaVersion) == "" {
		errs = append(errs, errors.New("schema_version is required"))
	}
	if len(c.Roles) == 0 {
		errs = append(errs, errors.New("roles must not be empty"))
	}
	if len(c.Models) == 0 {
		errs = append(errs, errors.New("models must not be empty"))
	}

	seenExcluded := map[string]struct{}{}
	for _, model := range c.ExcludedModels {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" || trimmed != model {
			errs = append(errs, errors.New("excluded_models must contain trimmed, non-empty values"))
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seenExcluded[key]; exists {
			errs = append(errs, fmt.Errorf("excluded_models contains duplicate %q", model))
		}
		seenExcluded[key] = struct{}{}
	}

	sourceKinds := set(c.Constraints.ProvenanceSourceKinds)
	if len(sourceKinds) == 0 {
		errs = append(errs, errors.New("constraints.provenance_source_kinds must not be empty"))
	}
	capVocab := set(c.Constraints.CapabilityVocabulary)
	if opts.Profile == CloudValidation && len(capVocab) == 0 {
		errs = append(errs, errors.New("constraints.capability_vocabulary must not be empty"))
	}
	endpoints := set(c.Constraints.Endpoints)
	if opts.Profile == CloudValidation && len(endpoints) == 0 {
		errs = append(errs, errors.New("constraints.endpoints must not be empty"))
	}
	modalityVocab := set(c.Constraints.ModalityVocabulary)
	if len(modalityVocab) == 0 {
		errs = append(errs, errors.New("constraints.modality_vocabulary must not be empty"))
	}

	for name, model := range c.Models {
		path := "models." + name
		if opts.RequireProvider && strings.TrimSpace(model.Provider) == "" {
			errs = append(errs, fmt.Errorf("%s.provider is required", path))
		}
		if opts.RequireFamily && strings.TrimSpace(model.Family) == "" {
			errs = append(errs, fmt.Errorf("%s.family is required", path))
		}
		if len(model.Capabilities) == 0 {
			errs = append(errs, fmt.Errorf("%s.capabilities must not be empty", path))
		}
		for _, capability := range model.Capabilities {
			if len(capVocab) > 0 {
				if _, ok := capVocab[capability]; !ok {
					errs = append(errs, fmt.Errorf("%s capability %q is not in capability_vocabulary", path, capability))
				}
			}
		}
		if opts.RequireModelEndpoints && len(model.Endpoints) == 0 {
			errs = append(errs, fmt.Errorf("%s.endpoints must not be empty", path))
		}
		for _, endpoint := range model.Endpoints {
			if len(endpoints) > 0 {
				if _, ok := endpoints[endpoint]; !ok {
					errs = append(errs, fmt.Errorf("%s endpoint %q is not in constraints.endpoints", path, endpoint))
				}
			}
		}
		if len(model.Modalities.Input) == 0 {
			errs = append(errs, fmt.Errorf("%s.modalities.input must not be empty", path))
		}
		if len(model.Modalities.Output) == 0 {
			errs = append(errs, fmt.Errorf("%s.modalities.output must not be empty", path))
		}
		for direction, modalities := range map[string][]Modality{
			"input":  model.Modalities.Input,
			"output": model.Modalities.Output,
		} {
			for _, modality := range modalities {
				if _, ok := modalityVocab[string(modality)]; !ok {
					errs = append(errs, fmt.Errorf("%s.modalities.%s value %q is not in modality_vocabulary", path, direction, modality))
				}
			}
		}
		if opts.RequireResourceEstimate {
			if model.DiskSizeGBEstimate <= 0 {
				errs = append(errs, fmt.Errorf("%s.disk_size_gb_estimate must be positive", path))
			}
			if model.RAMGBEstimate <= 0 {
				errs = append(errs, fmt.Errorf("%s.ram_gb_estimate must be positive", path))
			}
			if model.VRAMGBEstimate <= 0 {
				errs = append(errs, fmt.Errorf("%s.vram_gb_estimate must be positive", path))
			}
		}
		if len(model.Provenance) == 0 {
			errs = append(errs, fmt.Errorf("%s.provenance must not be empty", path))
		}
		for field, prov := range model.Provenance {
			validateProvenance(&errs, path+".provenance."+field, prov, sourceKinds)
		}
	}

	for name, role := range c.Roles {
		path := "roles." + name
		if opts.Profile == CloudValidation {
			if strings.TrimSpace(role.Endpoint) == "" {
				errs = append(errs, fmt.Errorf("%s.endpoint is required", path))
			} else if _, ok := endpoints[role.Endpoint]; !ok {
				errs = append(errs, fmt.Errorf("%s.endpoint %q is not in constraints.endpoints", path, role.Endpoint))
			}
		}
		if strings.TrimSpace(role.Model) == "" {
			errs = append(errs, fmt.Errorf("%s.model is required", path))
		} else if _, ok := c.Models[role.Model]; !ok {
			errs = append(errs, fmt.Errorf("%s.model %q is not in models", path, role.Model))
		}
		for _, fallback := range role.Fallbacks {
			if _, ok := c.Models[fallback]; !ok {
				errs = append(errs, fmt.Errorf("%s fallback %q is not in models", path, fallback))
			}
		}
		if len(role.RequiredCapabilities) == 0 {
			errs = append(errs, fmt.Errorf("%s.required_capabilities must not be empty", path))
		}
		for _, capability := range role.RequiredCapabilities {
			if len(capVocab) > 0 {
				if _, ok := capVocab[capability]; !ok {
					errs = append(errs, fmt.Errorf("%s required capability %q is not in capability_vocabulary", path, capability))
				}
			}
		}
		for _, capability := range role.PreferredCapabilities {
			if len(capVocab) > 0 {
				if _, ok := capVocab[capability]; !ok {
					errs = append(errs, fmt.Errorf("%s preferred capability %q is not in capability_vocabulary", path, capability))
				}
			}
		}
		if role.Preference <= 0 {
			errs = append(errs, fmt.Errorf("%s.preference must be positive", path))
		}
		if role.MaxTokens != nil && *role.MaxTokens < MinMaxTokens {
			errs = append(errs, fmt.Errorf("%s.max_tokens %d must be >= %d", path, *role.MaxTokens, MinMaxTokens))
		}
		role.RequestDefaults.validate(path+".request_defaults", &errs)
		validateSamplingSupport(&errs, path+".sampling_support", role.SamplingSupport)
		validateProvenance(&errs, path+".provenance", role.Provenance, sourceKinds)

		if opts.Profile == CloudValidation {
			validateRoleModel(&errs, c, path+".model", role.Model, role)
			for _, fallback := range role.Fallbacks {
				validateRoleModel(&errs, c, path+" fallback "+fallback, fallback, role)
			}
		}
	}

	roleNames := set(c.RoleNames())
	for _, role := range c.Constraints.RolePreferenceOrder {
		if _, ok := roleNames[role]; !ok {
			errs = append(errs, fmt.Errorf("constraints.role_preference_order references unknown role %q", role))
		}
	}
	for name, prov := range c.Provenance {
		validateProvenance(&errs, "provenance."+name, prov, sourceKinds)
	}
	return errors.Join(errs...)
}

func (d *RequestDefaults) validate(path string, errs *[]error) {
	if d == nil {
		return
	}
	if d.Temperature != nil && (*d.Temperature < MinTemperature || *d.Temperature > MaxTemperature) {
		*errs = append(*errs, fmt.Errorf("%s.temperature %.3f out of range [%.1f,%.1f]", path, *d.Temperature, MinTemperature, MaxTemperature))
	}
	if d.MaxTokens != nil && *d.MaxTokens < MinMaxTokens {
		*errs = append(*errs, fmt.Errorf("%s.max_tokens %d must be >= %d", path, *d.MaxTokens, MinMaxTokens))
	}
	validateEnum(errs, path+".response_format", d.ResponseFormat, allowedResponseFormats)
	validateEnum(errs, path+".resolution", d.Resolution, allowedResolutions)
	validateEnum(errs, path+".output_format", d.OutputFormat, allowedOutputFormats)
	validateEnum(errs, path+".background", d.Background, allowedBackgrounds)
	validateEnum(errs, path+".quality", d.Quality, allowedQualities)
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

func validateRoleModel(errs *[]error, c Catalog, path, modelRef string, role Role) {
	model, ok := c.Models[modelRef]
	if !ok {
		*errs = append(*errs, fmt.Errorf("%s %q is not in models", path, modelRef))
		return
	}
	if !contains(model.Endpoints, role.Endpoint) {
		*errs = append(*errs, fmt.Errorf("%s %q does not serve endpoint %q", path, modelRef, role.Endpoint))
	}
	modelCaps := set(model.Capabilities)
	for _, capability := range role.RequiredCapabilities {
		if _, has := modelCaps[capability]; !has {
			*errs = append(*errs, fmt.Errorf("%s %q lacks required capability %q", path, modelRef, capability))
		}
	}
}

func validateProvenance(errs *[]error, path string, prov Provenance, sourceKinds map[string]struct{}) {
	if _, ok := sourceKinds[prov.SourceKind]; !ok {
		*errs = append(*errs, fmt.Errorf("%s.source_kind %q is not allowed", path, prov.SourceKind))
	}
	switch prov.Confidence {
	case "low", "medium", "high", "measured":
	default:
		*errs = append(*errs, fmt.Errorf("%s.confidence %q is not allowed", path, prov.Confidence))
	}
	if strings.TrimSpace(prov.Source) == "" {
		*errs = append(*errs, fmt.Errorf("%s.source is required", path))
	}
	if strings.TrimSpace(prov.ObservedAt) == "" {
		*errs = append(*errs, fmt.Errorf("%s.observed_at is required", path))
	}
	if prov.SampleCount < 0 {
		*errs = append(*errs, fmt.Errorf("%s.sample_count must be >= 0", path))
	}
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
	for _, value := range values {
		if value == target {
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
