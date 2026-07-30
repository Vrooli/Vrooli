package hostreqspec

import (
	"fmt"
	"runtime"
	"strings"
)

type Kind string

// Privilege describes the highest privilege an object needs on a target host.
// It is deliberately independent of Bundling: a vendorable binary may need
// elevated installation on one OS while still being safe to ship in a bundle.
type Privilege string

const (
	PrivilegeNone     Privilege = "none"
	PrivilegeUser     Privilege = "user"
	PrivilegeElevated Privilege = "elevated"
)

// Bundling describes whether a host requirement may be part of a Tier 2
// desktop application.
type Bundling string

const (
	BundlingVendorable   Bundling = "vendorable"
	BundlingHostRequired Bundling = "host-required"
	BundlingProhibited   Bundling = "prohibited"
)

const (
	KindTool      Kind = "tool"
	KindSafeguard Kind = "safeguard"
)

type Declaration struct {
	Name         string    `json:"name"`
	Required     bool      `json:"required"`
	Reason       string    `json:"reason"`
	When         []string  `json:"when,omitempty"`
	Environments []string  `json:"environments,omitempty"`
	Platforms    []string  `json:"platforms,omitempty"`
	Notes        string    `json:"notes,omitempty"`
	Manual       bool      `json:"manual,omitempty"`
	Privilege    Privilege `json:"privilege,omitempty"`
	// PrivilegeReason is mandatory for an explicit tool privilege because tools
	// otherwise derive their value from their source mechanism per platform.
	PrivilegeReason string                 `json:"privilegeReason,omitempty"`
	Bundling        Bundling               `json:"bundling,omitempty"`
	Requires        *CapabilityRequirement `json:"requires,omitempty"`
}

// DerivePrivilege returns an explicit value when the manifest explains why the
// normal source-mechanism rule is wrong. Release artifacts install into a user
// directory. Package-manager installs need elevation on Linux and Windows but
// not on macOS, where Homebrew is user-owned.
func (d Declaration) DerivePrivilege(platform string) Privilege {
	if d.Privilege != "" {
		return d.Privilege
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "darwin" {
		platform = "macos"
	}
	// Host declarations do not carry a source type. Their default is user: the
	// object is consumed from an already configured host. Tool manifests supply
	// an explicit or derived value during registry resolution.
	if platform == "linux" || platform == "windows" {
		return PrivilegeUser
	}
	return PrivilegeUser
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
		if declaration.Privilege != "" && !isPrivilege(declaration.Privilege) {
			return fmt.Errorf("%s %q has unknown privilege %q", kind, name, declaration.Privilege)
		}
		if declaration.Privilege != "" && strings.TrimSpace(declaration.PrivilegeReason) == "" {
			return fmt.Errorf("%s %q explicit privilege requires privilegeReason", kind, name)
		}
		if declaration.Bundling != "" && !isBundling(declaration.Bundling) {
			return fmt.Errorf("%s %q has unknown bundling %q", kind, name, declaration.Bundling)
		}
	}
	return nil
}

func isPrivilege(value Privilege) bool {
	return value == PrivilegeNone || value == PrivilegeUser || value == PrivilegeElevated
}

func isBundling(value Bundling) bool {
	return value == BundlingVendorable || value == BundlingHostRequired || value == BundlingProhibited
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
	Privilege    Privilege              `json:"privilege"`
	Bundling     Bundling               `json:"bundling"`
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
