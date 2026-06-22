package hostreqspec

import (
	"fmt"
	"runtime"
	"strings"
)

type Kind string

const (
	KindTool      Kind = "tool"
	KindSafeguard Kind = "safeguard"
)

type Declaration struct {
	Name         string                 `json:"name"`
	Required     bool                   `json:"required"`
	Reason       string                 `json:"reason"`
	When         []string               `json:"when,omitempty"`
	Environments []string               `json:"environments,omitempty"`
	Platforms    []string               `json:"platforms,omitempty"`
	Notes        string                 `json:"notes,omitempty"`
	Manual       bool                   `json:"manual,omitempty"`
	Requires     *CapabilityRequirement `json:"requires,omitempty"`
}

// CapabilityRequirement gates a host requirement on hardware facts collected by
// internal/hostinventory. Every field is optional; an absent field imposes no
// constraint. When any present condition is unmet the requirement is cleanly
// skipped (not applicable on this host), never reported as a failure — the gate
// is advisory-skip so a CPU fallback can still be offered.
type CapabilityRequirement struct {
	// GPU, when non-nil, requires a detected GPU (true) or its absence (false).
	GPU *bool `json:"gpu,omitempty"`
	// MinVRAMGb is the minimum total VRAM (GiB) on at least one detected GPU.
	MinVRAMGb float64 `json:"minVramGb,omitempty"`
	// Arch is the set of allowed host architectures (Go GOARCH).
	Arch []string `json:"arch,omitempty"`
	// MinRAMGb is the minimum total system RAM (GiB).
	MinRAMGb float64 `json:"minRamGb,omitempty"`
}

// IsZero reports whether the requirement imposes no constraint at all.
func (c *CapabilityRequirement) IsZero() bool {
	if c == nil {
		return true
	}
	return c.GPU == nil && c.MinVRAMGb == 0 && len(c.Arch) == 0 && c.MinRAMGb == 0
}

// CapabilityFacts are the host hardware facts a CapabilityRequirement is
// evaluated against. It is a plain value type so this low-level package needs no
// dependency on internal/hostinventory; callers translate a host snapshot into
// these facts (the seam that lets tests use a fake host).
type CapabilityFacts struct {
	HasGPU    bool
	MaxVRAMGb float64 // largest VRAM (GiB) across detected GPUs; 0 when unknown
	Arch      string  // Go GOARCH of the host
	RAMGb     float64 // total system RAM (GiB)
}

// Evaluate reports whether facts satisfy the requirement. When unmet, the
// returned reason names the first failing condition (for surfacing as a
// not-applicable note). A nil/zero requirement is always satisfied.
//
// Conservative by design: an unmet condition is an advisory skip, never a hard
// failure — a CPU fallback can still be offered. Unknown VRAM (0) is treated as
// "not GPU-viable" so a VRAM floor never over-claims.
func (c *CapabilityRequirement) Evaluate(facts CapabilityFacts) (bool, string) {
	if c.IsZero() {
		return true, ""
	}
	if c.GPU != nil {
		if *c.GPU && !facts.HasGPU {
			return false, "requires a GPU; none detected on this host"
		}
		if !*c.GPU && facts.HasGPU {
			return false, "requires no GPU; one is present on this host"
		}
	}
	if c.MinVRAMGb > 0 && facts.MaxVRAMGb < c.MinVRAMGb {
		return false, fmt.Sprintf("requires >= %.0f GiB VRAM; best detected is %.1f GiB", c.MinVRAMGb, facts.MaxVRAMGb)
	}
	if len(c.Arch) > 0 && !containsFoldArch(c.Arch, facts.Arch) {
		return false, fmt.Sprintf("requires arch in %v; host arch is %q", c.Arch, facts.Arch)
	}
	if c.MinRAMGb > 0 && facts.RAMGb < c.MinRAMGb {
		return false, fmt.Sprintf("requires >= %.0f GiB RAM; host has %.1f GiB", c.MinRAMGb, facts.RAMGb)
	}
	return true, ""
}

func containsFoldArch(allowed []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range allowed {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func ValidateDeclarations(kind Kind, declarations []Declaration) error {
	seen := make(map[string]struct{}, len(declarations))
	for index, declaration := range declarations {
		name := strings.TrimSpace(declaration.Name)
		if name == "" {
			return fmt.Errorf("%s declarations[%d].name is required", kind, index)
		}
		if strings.TrimSpace(declaration.Reason) == "" {
			return fmt.Errorf("%s %q reason is required", kind, name)
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate %s declaration %q", kind, name)
		}
		seen[name] = struct{}{}
		if err := validateList(kind, name, "when", declaration.When); err != nil {
			return err
		}
		if err := validateList(kind, name, "environments", declaration.Environments); err != nil {
			return err
		}
		if err := validateList(kind, name, "platforms", declaration.Platforms); err != nil {
			return err
		}
	}
	return nil
}

func validateList(kind Kind, name, field string, values []string) error {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s %q contains an empty %s entry", kind, name, field)
		}
	}
	return nil
}

type Provenance struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"`
}

type ResolvedRequirement struct {
	Name         string                 `json:"name"`
	Kind         Kind                   `json:"kind"`
	Required     bool                   `json:"required"`
	Manual       bool                   `json:"manual"`
	Reasons      []string               `json:"reasons,omitempty"`
	When         []string               `json:"when,omitempty"`
	Environments []string               `json:"environments,omitempty"`
	Platforms    []string               `json:"platforms,omitempty"`
	Notes        []string               `json:"notes,omitempty"`
	Provenance   []Provenance           `json:"provenance,omitempty"`
	Requires     *CapabilityRequirement `json:"requires,omitempty"`
}

func CurrentPlatform() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS
	}
}

func NormalizeEnvironment(environment string) string {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "production":
		return "production"
	case "minimal":
		return "minimal"
	default:
		return "development"
	}
}
