package hostcapability

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/vrooli/vrooli/internal/safeguards"
)

type Verdict string

const (
	SatisfiedStructurally Verdict = "satisfied_structurally"
	Satisfied             Verdict = "satisfied"
	Undetermined          Verdict = "undetermined"
	NotApplicable         Verdict = "not_applicable"
	NotImplemented        Verdict = "not_implemented"
	Failed                Verdict = "failed"
)

type Invariant struct {
	ID            string
	Kind          string
	Statement     string
	Severity      string
	Applicability map[string]string
}

type Result struct {
	InvariantID string         `json:"invariant_id"`
	Verdict     Verdict        `json:"verdict"`
	Reason      string         `json:"reason"`
	Evidence    map[string]any `json:"evidence,omitempty"`
}

type Facts struct {
	OS                    string
	VendorID              string
	DriverPackage         string
	KernelRelease         string
	ExpectedPackage       string
	PackageNames          []string
	CandidatePackageNames []string
}

type Provider interface {
	Name() string
	Resolve(context.Context, Invariant, Facts) Result
}

type Registry struct{ providers map[string]Provider }

func NewRegistry(providers ...Provider) *Registry {
	registry := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, provider := range providers {
		registry.providers[provider.Name()] = provider
	}
	return registry
}

func (r *Registry) Resolve(ctx context.Context, invariant Invariant, facts Facts) Result {
	provider, ok := r.providers[facts.OS]
	if !ok {
		return Result{InvariantID: invariant.ID, Verdict: NotImplemented, Reason: "no provider is registered for this operating system"}
	}
	result := provider.Resolve(ctx, invariant, facts)
	result.InvariantID = invariant.ID
	return result
}

// Evaluate resolves every declaration. A result is returned for every input,
// including not_applicable and not_implemented, so callers cannot confuse a
// missing evaluation with a passing evaluation.
func Evaluate(ctx context.Context, registry *Registry, invariants []Invariant, facts Facts) []Result {
	results := make([]Result, 0, len(invariants))
	for _, invariant := range invariants {
		results = append(results, registry.Resolve(ctx, invariant, facts))
	}
	return results
}

type DarwinProvider struct{}

func (DarwinProvider) Name() string { return "darwin" }
func (DarwinProvider) Resolve(_ context.Context, invariant Invariant, _ Facts) Result {
	return Result{InvariantID: invariant.ID, Verdict: NotApplicable, Reason: "the invariant does not apply to this platform"}
}

type AptProvider struct {
	ResolveFn func(context.Context, Invariant, Facts) Result
}

func (AptProvider) Name() string { return "linux" }
func (p AptProvider) Resolve(ctx context.Context, invariant Invariant, facts Facts) Result {
	if p.ResolveFn == nil {
		return resolveLinuxInvariant(invariant, facts)
	}
	return p.ResolveFn(ctx, invariant, facts)
}

var nvidiaDriverPackageRE = regexp.MustCompile(`^nvidia-driver-([0-9]+(?:-[a-z]+)?)(?:-(open|server(?:-open)?))?$`)

func IsNvidiaDriverPackage(name string) bool {
	return nvidiaDriverPackageRE.MatchString(strings.TrimSpace(name))
}

func NvidiaPackageQueryPatterns() []string {
	return []string{"nvidia-driver-*", "linux-modules-nvidia-*"}
}

func NvidiaDriverPackageIdentity(name string) (series, flavor string) {
	match := nvidiaDriverPackageRE.FindStringSubmatch(strings.TrimSpace(name))
	if len(match) == 0 {
		return "", ""
	}
	return match[1], match[2]
}

// DeriveNvidiaModulePackage is the single platform-specific package-name
// derivation. Declarations and health checks use the invariant relationship;
// only this provider knows the apt package spelling.
func DeriveNvidiaModulePackage(driverPackage, kernelRelease string) (string, bool) {
	prefix, ok := NvidiaModulePackagePrefix(driverPackage)
	if !ok || strings.TrimSpace(kernelRelease) == "" {
		return "", false
	}
	return prefix + "-" + strings.TrimSpace(kernelRelease), true
}

// NvidiaModulePackagePrefix returns the apt package prefix for a driver
// metapackage. The caller may use it to inspect already-observed package
// names, while the spelling remains owned by this provider.
func NvidiaModulePackagePrefix(driverPackage string) (string, bool) {
	match := nvidiaDriverPackageRE.FindStringSubmatch(strings.TrimSpace(driverPackage))
	if len(match) == 0 {
		return "", false
	}
	parts := []string{"linux-modules-nvidia", match[1]}
	if match[2] != "" {
		parts = append(parts, match[2])
	}
	return strings.Join(parts, "-"), true
}

func resolveLinuxInvariant(invariant Invariant, facts Facts) Result {
	result := Result{InvariantID: invariant.ID, Verdict: NotImplemented}
	if platforms := invariant.Applicability["platforms"]; platforms != "" && !contains(strings.Split(platforms, ","), facts.OS) {
		result.Verdict = NotApplicable
		result.Reason = "the invariant does not apply to this platform"
		return result
	}
	if vendor := invariant.Applicability["vendor_id"]; vendor != "" && facts.VendorID != "" && !strings.EqualFold(vendor, facts.VendorID) {
		result.Verdict = NotApplicable
		result.Reason = "the invariant does not apply to this device vendor"
		return result
	}
	expected, ok := DeriveNvidiaModulePackage(facts.DriverPackage, facts.KernelRelease)
	if !ok && facts.ExpectedPackage != "" {
		expected, ok = facts.ExpectedPackage, true
	}
	if !ok {
		result.Reason = "the running kernel or driver package identity is unavailable"
		return result
	}
	result.Evidence = map[string]any{"expectedPackage": expected, "runningKernel": facts.KernelRelease}
	installed := contains(facts.PackageNames, expected)
	if installed {
		result.Verdict = Satisfied
		result.Reason = "the invariant is satisfied by the observed package set"
		return result
	}
	if contains(facts.CandidatePackageNames, expected) {
		result.Verdict = Failed
		result.Reason = "the required coupled package is absent and a candidate is available"
		return result
	}
	result.Verdict = Undetermined
	result.Reason = "the platform provider can derive the coupled package, but package availability is not observable"
	return result
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// EmbeddedSafeguardInvariants reads declaration data without exposing the
// safeguard manifest format to scenario checks.
func EmbeddedSafeguardInvariants(name string) ([]Invariant, error) {
	data, err := safeguards.Manifests.ReadFile(name + "/safeguard.json")
	if err != nil {
		return nil, fmt.Errorf("read safeguard %q: %w", name, err)
	}
	var manifest struct {
		Invariants []struct {
			ID            string         `json:"id"`
			Kind          string         `json:"kind"`
			Statement     string         `json:"statement"`
			Severity      string         `json:"severity"`
			Applicability map[string]any `json:"applicability"`
		} `json:"invariants"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse safeguard %q: %w", name, err)
	}
	result := make([]Invariant, 0, len(manifest.Invariants))
	for _, declaration := range manifest.Invariants {
		applicability := make(map[string]string, len(declaration.Applicability))
		for key, value := range declaration.Applicability {
			switch typed := value.(type) {
			case string:
				applicability[key] = typed
			case []any:
				var values []string
				for _, item := range typed {
					if value, ok := item.(string); ok {
						values = append(values, value)
					}
				}
				applicability[key] = strings.Join(values, ",")
			}
		}
		result = append(result, Invariant{ID: declaration.ID, Kind: declaration.Kind, Statement: declaration.Statement, Severity: declaration.Severity, Applicability: applicability})
	}
	return result, nil
}
