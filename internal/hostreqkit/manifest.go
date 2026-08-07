package hostreqkit

import "github.com/vrooli/vrooli/internal/hostreqspec"

type ToolManifest struct {
	Schema            string                             `json:"$schema,omitempty"`
	Name              string                             `json:"name"`
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
	Source            *ToolSource                        `json:"source,omitempty"`
	Requires          *hostreqspec.CapabilityRequirement `json:"requires,omitempty"`
	VerificationCheck *VerificationCheck                 `json:"verificationCheck,omitempty"`
	Version           string                             `json:"version,omitempty"`
	Notes             string                             `json:"notes,omitempty"`
}

// ToolSource declares verified fetch targets for the generic tool handler. An
// absent Source (or Type=="package") keeps the OS-package-manager path. For
// Type=="url" or "release", a matching target is preferred and installs into
// the user-local ~/.vrooli/bin with no sudo; a host without a matching target
// can still use an explicitly declared package fallback.
type ToolSource struct {
	// Type is package (default), url, or release. url and release are
	// behaviourally identical at fetch time; release documents that the URL is a
	// tagged release asset.
	Type string `json:"type"`
	// Targets maps "<os>/<arch>" (Go GOOS/GOARCH) to its fetch spec. A host with
	// no matching target falls back to its declared package, when present, and is
	// otherwise cleanly unsupported.
	Targets map[string]ToolSourceTarget `json:"targets,omitempty"`
	// Unsupported records an explicit reason for an os/arch combination that
	// has no upstream artifact or supported package-manager route.
	Unsupported map[string]string `json:"unsupported,omitempty"`
}

// ToolSourceTarget is the fetch spec for one os/arch combination.
type ToolSourceTarget struct {
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Archive string `json:"archive,omitempty"` // tar.gz | zip | none (default raw binary)
	Layout  string `json:"layout,omitempty"`  // file (default) | dir (extract whole tree into ~/.vrooli/opt/<tool>, launcher into ~/.vrooli/bin)
	BinPath string `json:"binPath,omitempty"` // path of the binary inside the archive (for dir, relative to the opt dir)
	Mode    string `json:"mode,omitempty"`    // octal file mode, e.g. 0755
	// RuntimeEnv maps an environment-variable name to a path relative to the
	// extracted opt directory. Dir-layout launchers resolve these paths at
	// install time, preventing an incompatible ambient runtime home from
	// changing the installed tool's behavior (for example Go's GOROOT).
	RuntimeEnv map[string]string `json:"runtimeEnv,omitempty"`
}

// IsDir reports whether this target installs the whole archive tree into a
// per-tool opt directory (with a launcher) rather than a single binary.
func (t ToolSourceTarget) IsDir() bool { return t.Layout == "dir" }

// SourceType returns the declared source type, defaulting to "package" when no
// Source is declared. The runtime combines this with target and host-package
// availability to choose the effective installation strategy for each host.
func (m ToolManifest) SourceType() string {
	if m.Source == nil || m.Source.Type == "" {
		return "package"
	}
	return m.Source.Type
}

// TargetFor returns the fetch target for the given os/arch ("<os>/<arch>"),
// reporting false when no target is declared for that combination.
func (s *ToolSource) TargetFor(osName, arch string) (ToolSourceTarget, bool) {
	if s == nil {
		return ToolSourceTarget{}, false
	}
	target, ok := s.Targets[osName+"/"+arch]
	return target, ok
}

// UnsupportedFor returns the explicit unsupported reason for an os/arch pair.
func (s *ToolSource) UnsupportedFor(osName, arch string) (string, bool) {
	if s == nil {
		return "", false
	}
	reason, ok := s.Unsupported[osName+"/"+arch]
	return reason, ok && reason != ""
}

type SafeguardManifest struct {
	Schema            string                `json:"$schema,omitempty"`
	Name              string                `json:"name"`
	Description       string                `json:"description"`
	Platforms         []string              `json:"platforms,omitempty"`
	Handler           string                `json:"handler"`
	Privilege         hostreqspec.Privilege `json:"privilege"`
	Bundling          hostreqspec.Bundling  `json:"bundling"`
	BundlingReason    string                `json:"bundlingReason,omitempty"`
	VerificationCheck *VerificationCheck    `json:"verificationCheck,omitempty"`
	Config            map[string]any        `json:"config,omitempty"`
	Version           string                `json:"version,omitempty"`
	Notes             string                `json:"notes,omitempty"`
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
