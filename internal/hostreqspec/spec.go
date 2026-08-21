package hostreqspec

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/vrooli/binaryfetch"
)

type Kind string

// OperatorChoice records the durable decision, if any, for an optional host
// requirement. The explicit third state matters: an absent entry means the
// operator has not answered yet, while false means the operator declined.
type OperatorChoice string

const (
	OperatorChoiceNotRecorded OperatorChoice = "not_recorded"
	OperatorChoiceOptedIn     OperatorChoice = "opted_in"
	OperatorChoiceDeclined    OperatorChoice = "declined"
)

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
	MinVersion   string    `json:"min_version,omitempty"`
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
	// InitSystem, SessionType and DisplayManager are exact host-fact gates.
	InitSystem     string `json:"initSystem,omitempty"`
	SessionType    string `json:"sessionType,omitempty"`
	DisplayManager string `json:"displayManager,omitempty"`
	// WaylandAttainable gates experiences that require a policy-compatible
	// Wayland session. A pointer preserves the distinction between false and no gate.
	WaylandAttainable *bool `json:"waylandAttainable,omitempty"`
}

// IsZero reports whether the requirement imposes no constraint at all.
func (c *CapabilityRequirement) IsZero() bool {
	if c == nil {
		return true
	}
	return c.GPU == nil && c.MinVRAMGb == 0 && len(c.Arch) == 0 && c.MinRAMGb == 0 &&
		strings.TrimSpace(c.InitSystem) == "" && strings.TrimSpace(c.SessionType) == "" &&
		strings.TrimSpace(c.DisplayManager) == "" && c.WaylandAttainable == nil
}

// CapabilityFacts are the host hardware facts a CapabilityRequirement is
// evaluated against. It is a plain value type so this low-level package needs no
// dependency on internal/hostinventory; callers translate a host snapshot into
// these facts (the seam that lets tests use a fake host).
type CapabilityFacts struct {
	HasGPU            bool
	MaxVRAMGb         float64 // largest VRAM (GiB) across detected GPUs; 0 when unknown
	Arch              string  // Go GOARCH of the host
	RAMGb             float64 // total system RAM (GiB)
	InitSystem        string
	SessionType       string
	DisplayManager    string
	WaylandAttainable bool
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
	if expected := strings.TrimSpace(c.InitSystem); expected != "" && !strings.EqualFold(expected, strings.TrimSpace(facts.InitSystem)) {
		return false, fmt.Sprintf("requires init system %q; host reports %q", expected, facts.InitSystem)
	}
	if expected := strings.TrimSpace(c.SessionType); expected != "" && !strings.EqualFold(expected, strings.TrimSpace(facts.SessionType)) {
		return false, fmt.Sprintf("requires session type %q; host reports %q", expected, facts.SessionType)
	}
	if expected := strings.TrimSpace(c.DisplayManager); expected != "" && !strings.EqualFold(expected, strings.TrimSpace(facts.DisplayManager)) {
		return false, fmt.Sprintf("requires display manager %q; host reports %q", expected, facts.DisplayManager)
	}
	if c.WaylandAttainable != nil && *c.WaylandAttainable != facts.WaylandAttainable {
		return false, fmt.Sprintf("requires Wayland attainable=%t; host reports %t", *c.WaylandAttainable, facts.WaylandAttainable)
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
	Name           string         `json:"name"`
	Kind           Kind           `json:"kind"`
	Required       bool           `json:"required"`
	MinVersion     string         `json:"min_version,omitempty"`
	OperatorChoice OperatorChoice `json:"operator_choice"`
	Config         map[string]any `json:"config,omitempty"`
	ConfigError    string         `json:"config_error,omitempty"`
	// ConfigUnconfigured is set when an optional safeguard declares required
	// parameters but the operator has not supplied any configuration.
	ConfigUnconfigured string                   `json:"config_unconfigured,omitempty"`
	ConfigNonDefault   bool                     `json:"config_non_default,omitempty"`
	Manual             bool                     `json:"manual"`
	Privilege          Privilege                `json:"privilege"`
	Bundling           Bundling                 `json:"bundling"`
	Reasons            []string                 `json:"reasons,omitempty"`
	When               []string                 `json:"when,omitempty"`
	Environments       []string                 `json:"environments,omitempty"`
	Platforms          []string                 `json:"platforms,omitempty"`
	Notes              []string                 `json:"notes,omitempty"`
	Provenance         []Provenance             `json:"provenance,omitempty"`
	Requires           *CapabilityRequirement   `json:"requires,omitempty"`
	Acquisition        *binaryfetch.Acquisition `json:"acquisition,omitempty"`
}

// ConfigString returns a declared string parameter and reports whether it was
// present with the expected type.
func (r ResolvedRequirement) ConfigString(name string) (string, bool) {
	value, ok := r.Config[strings.TrimSpace(name)]
	if !ok {
		return "", false
	}
	result, ok := value.(string)
	return result, ok
}

// ConfigInt returns a declared integer parameter. JSON numbers decode as
// float64, so fractional values are rejected instead of silently truncated.
func (r ResolvedRequirement) ConfigInt(name string) (int, bool) {
	value, ok := r.Config[strings.TrimSpace(name)]
	if !ok {
		return 0, false
	}
	switch number := value.(type) {
	case int:
		return number, true
	case float64:
		if number == float64(int(number)) {
			return int(number), true
		}
	}
	return 0, false
}

// ConfigBool returns a declared boolean parameter and reports whether it was
// present with the expected type.
func (r ResolvedRequirement) ConfigBool(name string) (bool, bool) {
	value, ok := r.Config[strings.TrimSpace(name)]
	if !ok {
		return false, false
	}
	result, ok := value.(bool)
	return result, ok
}

func CurrentPlatform() string {
	return NormalizePlatform(runtime.GOOS)
}

// ContainsPlatform compares a declaration's platform vocabulary with a host
// platform, accepting the legacy darwin spelling at the boundary.
func ContainsPlatform(values []string, target string) bool {
	target = NormalizePlatform(target)
	for _, value := range values {
		if NormalizePlatform(value) == target {
			return true
		}
	}
	return false
}

// NormalizePlatform converts operating-system identifiers at the declaration
// boundary into the vocabulary used by host manifests. The legacy darwin token
// remains readable so previously published manifests keep resolving on macOS.
func NormalizePlatform(value string) string {
	switch value = strings.ToLower(strings.TrimSpace(value)); value {
	case "darwin":
		return "macos"
	default:
		return value
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
