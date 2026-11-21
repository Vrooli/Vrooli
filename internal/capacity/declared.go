package capacity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// declaredResourceManifest is intentionally smaller than the full resource
// manifest. Reconcile needs only to identify an accelerator declaration and its
// optional claim; the resource manifest package remains the owner of full
// validation.
//
// This shape used to test for a `gpu` block, which fleet_contract.go rejected on
// every managed-service resource. The two disagreed, so ollama, whisper and
// reranker were invisible to reconciliation while kokoro, kyutai-stt and
// speaker-verification raised findings permanently, including while stopped.
// One declaration ends that.
type declaredResourceManifest struct {
	Acceleration *struct {
		Backends []string           `json:"backends"`
		Require  string             `json:"require"`
		CUDA     DeclaredBackend    `json:"cuda"`
		Claim    *ResourceClaimSpec `json:"claim"`
		Capacity struct {
			Tunables []DeclaredCapacityTunable `json:"tunables"`
		} `json:"capacity"`
	} `json:"acceleration"`
}

// DeclaredBackend is the machine-compatibility subset of one accelerator
// backend declaration. Fit consumes the same manifest decode as reconciliation
// instead of introducing a second resource-tree walker.
type DeclaredBackend struct {
	MinCompute string `json:"min_compute,omitempty"`
}

// DeclaredCapacityTunable is the footprint-relevant subset of a manifest
// tunable. The manifest package owns full validation; this package only needs
// the effective value and stable footprint key.
type DeclaredCapacityTunable struct {
	Name            string `json:"name"`
	Env             string `json:"env"`
	Type            string `json:"type"`
	Default         any    `json:"default"`
	Minimum         *int64 `json:"minimum,omitempty"`
	Maximum         *int64 `json:"maximum,omitempty"`
	Enum            []any  `json:"enum,omitempty"`
	ScalesFootprint *bool  `json:"scales_footprint,omitempty"`
}

// ValidateResourceCapacityChoices rejects operator choices before they can be
// persisted. Manifests declare the permitted rungs and tunable bounds.
func ValidateResourceCapacityChoices(manifests []DeclaredResource, choices map[string]FitResourceChoice) error {
	declared := make(map[string]DeclaredResource, len(manifests))
	for _, resource := range manifests {
		declared[resource.Name] = resource
	}
	for name, choice := range choices {
		if choice.Rung == "" && len(choice.Tunables) == 0 {
			continue
		}
		resource, ok := declared[name]
		if !ok {
			return fmt.Errorf("operator state validation failed at /resources/%s/capacity: resource manifest does not exist", name)
		}
		if choice.Rung != "" {
			found := false
			for _, step := range claimSteps(resource.Claim) {
				if step.Label == choice.Rung {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("operator state validation failed at /resources/%s/capacity/rung: %q is not a declared step", name, choice.Rung)
			}
		}
		tunables := make(map[string]DeclaredCapacityTunable, len(resource.Tunables))
		for _, tunable := range resource.Tunables {
			tunables[tunable.Name] = tunable
		}
		for key, value := range choice.Tunables {
			tunable, ok := tunables[key]
			if !ok {
				return fmt.Errorf("operator state validation failed at /resources/%s/capacity/tunables/%s: tunable is not declared", name, key)
			}
			if err := validateDeclaredTunable(tunable, value); err != nil {
				return fmt.Errorf("operator state validation failed at /resources/%s/capacity/tunables/%s: %w", name, key, err)
			}
		}
	}
	return nil
}

func validateDeclaredTunable(tunable DeclaredCapacityTunable, value any) error {
	switch tunable.Type {
	case "integer":
		var number int64
		switch typed := value.(type) {
		case int:
			number = int64(typed)
		case int64:
			number = typed
		case float64:
			if typed != float64(int64(typed)) {
				return fmt.Errorf("must be an integer")
			}
			number = int64(typed)
		default:
			return fmt.Errorf("must be an integer")
		}
		if tunable.Minimum != nil && number < *tunable.Minimum {
			return fmt.Errorf("must be at least %d", *tunable.Minimum)
		}
		if tunable.Maximum != nil && number > *tunable.Maximum {
			return fmt.Errorf("must be at most %d", *tunable.Maximum)
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("must be a string")
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("must be a boolean")
		}
	}
	if len(tunable.Enum) > 0 {
		encoded, _ := json.Marshal(value)
		for _, candidate := range tunable.Enum {
			other, _ := json.Marshal(candidate)
			if string(encoded) == string(other) {
				return nil
			}
		}
		return fmt.Errorf("must be one of the declared values")
	}
	return nil
}

func (t DeclaredCapacityTunable) scalesFootprint() bool {
	return t.ScalesFootprint == nil || *t.ScalesFootprint
}

// DeclaredResource is the capacity projection of one resource manifest plus
// its repository-level enabled default.
type DeclaredResource struct {
	Name           string
	Enabled        bool
	Backends       []string
	Require        string
	CUDAMinCompute string
	Claim          *ResourceClaimSpec
	Tunables       []DeclaredCapacityTunable
}

type declaredServiceManifest struct {
	Dependencies struct {
		Resources map[string]struct {
			Enabled bool `json:"enabled"`
		} `json:"resources"`
	} `json:"dependencies"`
}

// LoadDeclaredResources is the capacity package's sole resource-manifest walk.
// Reconciliation and set-level fit both consume this projection.
func LoadDeclaredResources(root string) ([]DeclaredResource, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, nil
	}
	enabled := make(map[string]bool)
	if data, err := os.ReadFile(filepath.Join(root, ".vrooli", "service.json")); err == nil {
		var service declaredServiceManifest
		if err := json.Unmarshal(data, &service); err != nil {
			return nil, fmt.Errorf("parse root service manifest: %w", err)
		}
		for name, resource := range service.Dependencies.Resources {
			enabled[name] = resource.Enabled
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read root service manifest: %w", err)
	}

	entries, err := os.ReadDir(filepath.Join(root, "resources"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read resource declarations: %w", err)
	}
	resources := make([]DeclaredResource, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, "resources", entry.Name(), "resource.json")
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		var manifest declaredResourceManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		if manifest.Acceleration == nil {
			continue
		}
		resources = append(resources, DeclaredResource{
			Name: entry.Name(), Enabled: enabled[entry.Name()],
			Backends: append([]string(nil), manifest.Acceleration.Backends...),
			Require:  manifest.Acceleration.Require, CUDAMinCompute: manifest.Acceleration.CUDA.MinCompute,
			Claim:    manifest.Acceleration.Claim,
			Tunables: append([]DeclaredCapacityTunable(nil), manifest.Acceleration.Capacity.Tunables...),
		})
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Name < resources[j].Name })
	return resources, nil
}

// declaresAccelerator reports whether the manifest asks for any backend other
// than the CPU. It is the single answer to the question two files used to
// answer differently.
func (m declaredResourceManifest) declaresAccelerator() bool {
	if m.Acceleration == nil {
		return false
	}
	return declaresAcceleratorBackends(m.Acceleration.Backends)
}

func declaresAcceleratorBackends(backends []string) bool {
	for _, backend := range backends {
		if strings.TrimSpace(strings.ToLower(backend)) != ResourceKindCPU {
			return true
		}
	}
	return false
}

// claim returns the declared reservation, if any.
func (m declaredResourceManifest) claim() *ResourceClaimSpec {
	if m.Acceleration == nil {
		return nil
	}
	return m.Acceleration.Claim
}

// InstalledPredicate answers whether a resource is installed on this host. It
// is injected because only the resource layer knows, and the ledger must not
// depend on it.
type InstalledPredicate func(resource string) bool

// DeclaredGPUWithoutClaimFindings reports accelerator resources that have no
// active broker claim. This catches a declaration/ledger mismatch even when the
// resource is currently idle and therefore absent from the host GPU process
// snapshot. It is deliberately a finding, not an automatic claim: admission
// and resident adoption remain the only claim-creation paths.
//
// installed filters out resources that are not on this host at all. Without it
// every accelerator-declaring resource the operator has never installed raises
// a permanent warning, which is what made this finding class ignorable. A nil
// predicate reports every declaration, which is correct when the caller has no
// way to tell.
func DeclaredGPUWithoutClaimFindings(root string, ledger []CapacityClaim, installed InstalledPredicate) ([]Finding, error) {
	resources, err := LoadDeclaredResources(root)
	if err != nil {
		return nil, err
	}
	active := make(map[string]bool)
	for _, claim := range ledger {
		if claim.OwnerKind == OwnerKindResource && IsActiveClaimStatus(claim.Status) {
			active[strings.TrimSpace(claim.OwnerID)] = true
		}
	}

	var findings []Finding
	for _, resource := range resources {
		if active[resource.Name] {
			continue
		}
		if !declaresAcceleratorBackends(resource.Backends) {
			continue
		}
		if installed != nil && !installed(resource.Name) {
			// A resource that is not on this host cannot hold a claim, so the
			// absence of one is not a mismatch.
			continue
		}
		// The manifest no longer chooses a machine GPU. Reconciliation reports
		// the default device until operator state is supplied by its caller.
		gpuIndex := 0
		findings = append(findings, Finding{
			Class:        FindingDeclaredUnclaimed,
			OwnerKind:    OwnerKindResource,
			OwnerID:      resource.Name,
			ResourceKind: ResourceKindVRAM,
			GPUIndex:     &gpuIndex,
			Severity:     "warn",
			Message:      fmt.Sprintf("resource %q declares an accelerator backend but holds no active capacity claim", resource.Name),
		})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].OwnerID < findings[j].OwnerID })
	return findings, nil
}
