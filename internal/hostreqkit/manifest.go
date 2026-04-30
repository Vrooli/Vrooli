package hostreqkit

type ToolManifest struct {
	Schema            string             `json:"$schema,omitempty"`
	Name              string             `json:"name"`
	Description       string             `json:"description"`
	Commands          []string           `json:"commands"`
	VersionArgs       []string           `json:"versionArgs"`
	DefaultPackage    string             `json:"defaultPackage,omitempty"`
	Packages          map[string]string  `json:"packages,omitempty"`
	InstallHint       string             `json:"installHint,omitempty"`
	Platforms         []string           `json:"platforms,omitempty"`
	Handler           string             `json:"handler,omitempty"`
	VerificationCheck *VerificationCheck `json:"verificationCheck,omitempty"`
	Version           string             `json:"version,omitempty"`
	Notes             string             `json:"notes,omitempty"`
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
