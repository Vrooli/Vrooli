package policy

import (
	"errors"
	"fmt"
	"strings"
)

// Resolve maps declared roles to concrete model refs for hosted-provider
// `ensure`. Unknown roles are a hard error; resolution never falls back to a
// concrete model outside policy.
func Resolve(c Catalog, req ResolveRequest) (ResolveResolution, error) {
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
		role, ok := c.Roles[roleName]
		if !ok {
			errs = append(errs, fmt.Errorf("unknown model role %q", roleName))
			continue
		}
		model, modelErr := RoleModel(c, roleName, role)
		if modelErr != nil {
			errs = append(errs, modelErr)
			continue
		}
		out.Roles = append(out.Roles, ResolvedRole{
			Role:     roleName,
			Model:    model,
			Endpoint: role.Endpoint,
			Required: roleReq.IsRequired(),
			Reason:   strings.TrimSpace(roleReq.Reason),
		})
	}
	return out, errors.Join(errs...)
}

// ResolveLocal maps declared roles and explicit direct models for a local
// provider's `ensure`.
func ResolveLocal(c Catalog, req ResolveRequest) (LocalResolution, error) {
	var out LocalResolution
	seen := map[string]int{}
	add := func(model ResolvedModel) {
		model.Ref = strings.TrimSpace(model.Ref)
		if model.Ref == "" {
			return
		}
		if idx, ok := seen[model.Ref]; ok {
			if out.Models[idx].Source == "direct" && model.Source == "role" {
				out.Models[idx] = model
			}
			return
		}
		seen[model.Ref] = len(out.Models)
		out.Models = append(out.Models, model)
	}

	var errs []error
	for _, roleReq := range req.ModelRoles {
		roleName := strings.TrimSpace(roleReq.Role)
		if roleName == "" {
			continue
		}
		role, ok := c.Roles[roleName]
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

	requiredFields := set(c.Constraints.DirectModelExceptionRequiredFields)
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
		if _, known := c.Models[ref]; !known {
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

// ResolveRole resolves one role to its concrete model and role metadata.
func ResolveRole(c Catalog, roleName string) (ResolvedPolicyModel, error) {
	roleName = strings.TrimSpace(roleName)
	if roleName == "" {
		return ResolvedPolicyModel{}, errors.New("role is required")
	}
	role, ok := c.Roles[roleName]
	if !ok {
		return ResolvedPolicyModel{}, fmt.Errorf("unknown model role %q", roleName)
	}
	modelRef, err := RoleModel(c, roleName, role)
	if err != nil {
		return ResolvedPolicyModel{}, err
	}
	model, ok := c.Models[modelRef]
	if !ok {
		return ResolvedPolicyModel{}, fmt.Errorf("role %q resolved unknown model %q", roleName, modelRef)
	}
	resolved := resolvedPolicyModel(c, "role", modelRef, model)
	resolved.Role = roleName
	resolved.Endpoint = role.Endpoint
	resolved.Preference = role.Preference
	for _, fallback := range role.Fallbacks {
		if !c.isModelExcluded(fallback) {
			resolved.Fallbacks = append(resolved.Fallbacks, fallback)
		}
	}
	resolved.RequiredCapabilities = append([]string{}, role.RequiredCapabilities...)
	resolved.PreferredCapabilities = append([]string{}, role.PreferredCapabilities...)
	resolved.RequestDefaults = role.RequestDefaults
	resolved.SamplingDefaults = role.SamplingDefaults
	if role.SamplingDefaults != nil {
		sampling := role.SamplingDefaults.Resolve()
		resolved.Sampling = &sampling
	}
	resolved.SamplingSupport = copySamplingSupport(role.SamplingSupport)
	if role.MaxTokens != nil {
		maxTokens := *role.MaxTokens
		resolved.MaxTokens = &maxTokens
	}
	roleProvenance := role.Provenance
	resolved.RoleProvenance = &roleProvenance
	return resolved, nil
}

// ResolveModel resolves one catalog model reference.
func ResolveModel(c Catalog, modelRef string) (ResolvedPolicyModel, error) {
	modelRef = strings.TrimSpace(modelRef)
	if modelRef == "" {
		return ResolvedPolicyModel{}, errors.New("model is required")
	}
	if c.isModelExcluded(modelRef) {
		return ResolvedPolicyModel{}, fmt.Errorf("model %q is excluded by resource policy", modelRef)
	}
	model, ok := c.Models[modelRef]
	if !ok {
		return ResolvedPolicyModel{}, fmt.Errorf("unknown model %q", modelRef)
	}
	resolved := resolvedPolicyModel(c, "model", modelRef, model)
	if len(model.Endpoints) > 0 {
		resolved.Endpoint = model.Endpoints[0]
	}
	return resolved, nil
}

// RoleModel returns the first model allowed by the catalog's exclusion policy.
func RoleModel(c Catalog, roleName string, role Role) (string, error) {
	if !c.isModelExcluded(role.Model) {
		return role.Model, nil
	}
	for _, fallback := range role.Fallbacks {
		if !c.isModelExcluded(fallback) {
			return fallback, nil
		}
	}
	return "", fmt.Errorf("role %q has no model allowed by resource policy", roleName)
}

func resolvedPolicyModel(c Catalog, source, modelRef string, model Model) ResolvedPolicyModel {
	return ResolvedPolicyModel{
		SchemaVersion:        c.SchemaVersion,
		Source:               source,
		Model:                modelRef,
		ExcludedModels:       append([]string{}, c.ExcludedModels...),
		Capabilities:         append([]string{}, model.Capabilities...),
		Modalities:           model.Modalities,
		CoordinateConvention: model.CoordinateConvention,
		Endpoints:            append([]string{}, model.Endpoints...),
		ContextWindowTokens:  model.ContextWindowTokens,
		EmbeddingDimensions:  model.EmbeddingDimensions,
		DiskSizeGBEstimate:   model.DiskSizeGBEstimate,
		RAMGBEstimate:        model.RAMGBEstimate,
		VRAMGBEstimate:       model.VRAMGBEstimate,
		DefaultEligible:      model.DefaultEligible,
		Pricing:              model.Pricing,
		Provider:             model.Provider,
		Family:               model.Family,
		Provenance:           copyProvenanceMap(model.Provenance),
	}
}

func (c Catalog) isModelExcluded(model string) bool {
	model = strings.TrimSpace(model)
	for _, excluded := range c.ExcludedModels {
		if strings.EqualFold(model, strings.TrimSpace(excluded)) {
			return true
		}
	}
	return false
}
