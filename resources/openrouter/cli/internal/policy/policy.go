// Package policy is the OpenRouter-specific facade over the shared
// ai-provider-go policy contract. OpenRouter remains the source of truth for
// its catalog and validation profile; the shared package owns the schema,
// admission rules, and resolution mechanics.
package policy

import (
	aipolicy "github.com/vrooli/ai-provider-go/policy"
)

type (
	Policy               = policyAlias
	Role                 = aipolicy.Role
	Model                = aipolicy.Model
	Modalities           = aipolicy.Modalities
	Modality             = aipolicy.Modality
	RequestDefaults      = aipolicy.RequestDefaults
	Pricing              = aipolicy.Pricing
	Constraints          = aipolicy.Constraints
	Provenance           = aipolicy.Provenance
	SamplingSupportState = aipolicy.SamplingSupportState
	RoleRequest          = aipolicy.RoleRequest
	ResolveRequest       = aipolicy.ResolveRequest
	ResolvedRole         = aipolicy.ResolvedRole
	ResolveResolution    = aipolicy.ResolveResolution
	ResolvedPolicyModel  = aipolicy.ResolvedPolicyModel
	ValidationOptions    = aipolicy.ValidationOptions
)

const (
	SamplingHonored  = aipolicy.SamplingHonored
	SamplingIgnored  = aipolicy.SamplingIgnored
	SamplingRejected = aipolicy.SamplingRejected
	SamplingUnknown  = aipolicy.SamplingUnknown
)

// policyAlias keeps the historical openrouter `policy.Policy` name while
// embedding the shared catalog shape. Promotion preserves field access
// (p.Roles, p.ExcludedModels) used by callers and tests.
type policyAlias struct {
	aipolicy.Catalog
}

func options() ValidationOptions { return aipolicy.CloudOptions() }

// LoadFile reads, parses, and validates an OpenRouter policy file.
func LoadFile(path string) (Policy, error) {
	catalog, err := aipolicy.LoadFile(path, options())
	if err != nil {
		return Policy{}, err
	}
	return Policy{Catalog: catalog}, nil
}

// LoadDefaultFile resolves OpenRouter's policy path and loads it.
func LoadDefaultFile(getenv func(string) string) (Policy, string, error) {
	catalog, path, err := aipolicy.LoadDefaultFile(getenv, pathOptions())
	if err != nil {
		return Policy{}, "", err
	}
	return Policy{Catalog: catalog}, path, nil
}

// DefaultPolicyPath resolves OpenRouter's model-policy.json location.
func DefaultPolicyPath(getenv func(string) string) (string, error) {
	return aipolicy.DefaultPolicyPath(getenv, pathOptions())
}

func pathOptions() aipolicy.PathOptions {
	return aipolicy.PathOptions{
		ProviderName: "openrouter",
		PolicyEnv:    "OPENROUTER_MODEL_POLICY_PATH",
		SourceEnv:    "OPENROUTER_CLI_SOURCE_ROOT",
		Validation:   options(),
	}
}

// Resolve maps declared roles to concrete model refs for `ensure`.
func (p Policy) Resolve(req ResolveRequest) (ResolveResolution, error) {
	return aipolicy.Resolve(p.Catalog, req)
}

// ResolveRole resolves one role to its concrete model and role metadata.
func (p Policy) ResolveRole(roleName string) (ResolvedPolicyModel, error) {
	return aipolicy.ResolveRole(p.Catalog, roleName)
}

// ResolveModel resolves one catalog model reference.
func (p Policy) ResolveModel(modelRef string) (ResolvedPolicyModel, error) {
	return aipolicy.ResolveModel(p.Catalog, modelRef)
}

// RoleNames returns sorted role names.
func (p Policy) RoleNames() []string { return p.Catalog.RoleNames() }

// ModelRefs returns sorted model references.
func (p Policy) ModelRefs() []string { return p.Catalog.ModelRefs() }

// Validate applies the hosted-provider admission profile.
func (p Policy) Validate() error { return aipolicy.Validate(p.Catalog, options()) }

func (p Policy) isModelExcluded(model string) bool {
	for _, excluded := range p.ExcludedModels {
		if stringEqualFold(excluded, model) {
			return true
		}
	}
	return false
}

func (p Policy) roleModel(roleName string, role Role) (string, error) {
	return aipolicy.RoleModel(p.Catalog, roleName, role)
}

func stringEqualFold(a, b string) bool {
	return len(a) == len(b) && equalFoldASCII(a, b)
}

func equalFoldASCII(a, b string) bool {
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
