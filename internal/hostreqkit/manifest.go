package hostreqkit

import (
	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type ToolManifest struct {
	Schema            string                             `json:"$schema,omitempty"`
	Name              string                             `json:"name"`
	Capability        string                             `json:"capability,omitempty"`
	CapabilityRole    string                             `json:"capability_role,omitempty"`
	Description       string                             `json:"description"`
	Commands          []string                           `json:"commands"`
	VersionArgs       []string                           `json:"versionArgs"`
	DefaultPackage    string                             `json:"defaultPackage,omitempty"`
	Packages          map[string]string                  `json:"packages,omitempty"`
	InstallHint       string                             `json:"installHint,omitempty"`
	Platforms         []string                           `json:"platforms,omitempty"`
	Handler           string                             `json:"handler,omitempty"`
	Manual            bool                               `json:"manual,omitempty"`
	Privilege         hostreqspec.Privilege              `json:"privilege,omitempty"`
	PrivilegeReason   string                             `json:"privilegeReason,omitempty"`
	Bundling          hostreqspec.Bundling               `json:"bundling"`
	Acquisition       *ToolSource                        `json:"acquisition,omitempty"`
	Requires          *hostreqspec.CapabilityRequirement `json:"requires,omitempty"`
	VerificationCheck *VerificationCheck                 `json:"verificationCheck,omitempty"`
	Version           string                             `json:"version,omitempty"`
	Notes             string                             `json:"notes,omitempty"`
}

// ToolSource and ToolSourceTarget are aliases, not independent schema models.
// Acquisition is shared by tools, resources, release staging, and desktop
// bundling.
type (
	ToolSource       = binaryfetch.Acquisition
	ToolSourceTarget = binaryfetch.AcquisitionTarget
)

// SourceType returns the declared acquisition kind, defaulting to "package"
// when no acquisition is declared. The runtime combines this with target and host-package
// availability to choose the effective installation strategy for each host.
func (m ToolManifest) SourceType() string {
	if m.Acquisition == nil || m.Acquisition.Kind == "" {
		return "package"
	}
	return m.Acquisition.Kind
}

// TargetFor returns the fetch target for the given os/arch, reporting false
// when no target is declared for that combination.
func TargetFor(s *ToolSource, osName, arch string) (ToolSourceTarget, bool) {
	if s == nil {
		return ToolSourceTarget{}, false
	}
	target, err := s.Resolve(binaryfetch.Facts{"os": osName, "arch": arch})
	if err != nil {
		return ToolSourceTarget{}, false
	}
	return target, true
}

// UnsupportedFor returns the explicit unsupported reason for an os/arch pair.
func UnsupportedFor(s *ToolSource, osName, arch string) (string, bool) {
	if s == nil {
		return "", false
	}
	for _, target := range s.Targets {
		matches, err := targetMatches(target, osName, arch)
		if err == nil && matches && target.Unsupported != "" {
			return target.Unsupported, true
		}
	}
	return "", false
}

func targetMatches(target ToolSourceTarget, osName, arch string) (bool, error) {
	for key, value := range target.When {
		if key != "os" && key != "arch" {
			return false, nil
		}
		actual := osName
		if key == "arch" {
			actual = arch
		}
		if actual != value {
			return false, nil
		}
	}
	return true, nil
}

type SafeguardManifest struct {
	Schema            string                    `json:"$schema,omitempty"`
	Name              string                    `json:"name"`
	Capability        string                    `json:"capability,omitempty"`
	CapabilityRole    string                    `json:"capability_role,omitempty"`
	Description       string                    `json:"description"`
	Platforms         []string                  `json:"platforms,omitempty"`
	RiskClass         string                    `json:"risk_class,omitempty"`
	PlatformStatus    map[string]PlatformStatus `json:"platform_status,omitempty"`
	Invariants        []InvariantDeclaration    `json:"invariants,omitempty"`
	Storage           map[string]any            `json:"storage,omitempty"`
	Handler           string                    `json:"handler"`
	Privilege         hostreqspec.Privilege     `json:"privilege"`
	Bundling          hostreqspec.Bundling      `json:"bundling"`
	BundlingReason    string                    `json:"bundlingReason,omitempty"`
	VerificationCheck *VerificationCheck        `json:"verificationCheck,omitempty"`
	Config            map[string]any            `json:"config,omitempty"`
	Version           string                    `json:"version,omitempty"`
	Notes             string                    `json:"notes,omitempty"`
}

type PlatformStatus struct {
	Status    string `json:"status"`
	Mechanism string `json:"mechanism,omitempty"`
	Evidence  any    `json:"evidence,omitempty"`
}

type InvariantDeclaration struct {
	ID            string         `json:"id"`
	Kind          string         `json:"kind"`
	Statement     string         `json:"statement"`
	Severity      string         `json:"severity"`
	Applicability map[string]any `json:"applicability,omitempty"`
	Notes         string         `json:"notes,omitempty"`
}

type VerificationCheck struct {
	Command        string   `json:"command,omitempty"`
	Args           []string `json:"args,omitempty"`
	ExpectExitCode *int     `json:"expectExitCode,omitempty"`
	Files          []string `json:"files,omitempty"`
}

func (m ToolManifest) PackageNameForHost(host Host) string {
	if value, ok := m.Packages[host.PackageManager]; ok {
		return value
	}
	return m.DefaultPackage
}
