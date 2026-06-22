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
	Source            *ToolSource                        `json:"source,omitempty"`
	Requires          *hostreqspec.CapabilityRequirement `json:"requires,omitempty"`
	VerificationCheck *VerificationCheck                 `json:"verificationCheck,omitempty"`
	Version           string                             `json:"version,omitempty"`
	Notes             string                             `json:"notes,omitempty"`
}

// ToolSource declares how the generic tool handler installs a tool. An absent
// Source (or Type=="package") keeps the OS-package-manager path; Type=="url" or
// "release" fetches a verified binary into the user-local ~/.vrooli/bin with no
// sudo.
type ToolSource struct {
	// Type is package (default), url, or release. url and release are
	// behaviourally identical at fetch time; release documents that the URL is a
	// tagged release asset.
	Type string `json:"type"`
	// Targets maps "<os>/<arch>" (Go GOOS/GOARCH) to its fetch spec. A host with
	// no matching target is cleanly unsupported.
	Targets map[string]ToolSourceTarget `json:"targets,omitempty"`
}

// ToolSourceTarget is the fetch spec for one os/arch combination.
type ToolSourceTarget struct {
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	Archive string `json:"archive,omitempty"` // tar.gz | zip | none (default raw binary)
	Layout  string `json:"layout,omitempty"`  // file (default) | dir (extract whole tree into ~/.vrooli/opt/<tool>, launcher into ~/.vrooli/bin)
	BinPath string `json:"binPath,omitempty"` // path of the binary inside the archive (for dir, relative to the opt dir)
	Mode    string `json:"mode,omitempty"`    // octal file mode, e.g. 0755
}

// IsDir reports whether this target installs the whole archive tree into a
// per-tool opt directory (with a launcher) rather than a single binary.
func (t ToolSourceTarget) IsDir() bool { return t.Layout == "dir" }

// SourceType returns the effective source type, defaulting to "package" when no
// Source is declared. This is the single branch point distinguishing the
// package-manager path from the fetch path.
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

type SafeguardManifest struct {
	Schema            string             `json:"$schema,omitempty"`
	Name              string             `json:"name"`
	Description       string             `json:"description"`
	Platforms         []string           `json:"platforms,omitempty"`
	Handler           string             `json:"handler"`
	VerificationCheck *VerificationCheck `json:"verificationCheck,omitempty"`
	Version           string             `json:"version,omitempty"`
	Notes             string             `json:"notes,omitempty"`
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
