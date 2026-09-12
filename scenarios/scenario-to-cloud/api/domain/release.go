package domain

// ReleaseInputsSchemaVersion is stamped on every inputs.json.
const ReleaseInputsSchemaVersion = 1

// Reproducibility verdicts for one artifact of a release.
const (
	ReproducibilityReproducible    = "reproducible"
	ReproducibilityNotReproducible = "not_reproducible"
	ReproducibilityUnverified      = "unverified"
	ReproducibilityDeterministic   = "deterministic"
)

// Source snapshot kinds. A build from a working tree never claims isolation.
const (
	ReleaseSnapshotWorkingTree = "working-tree"
)

// ReleaseInputs is the content manifest of every material build input of one
// release (inputs.json). It records references and digests only: credential
// entries are descriptor references, never values.
type ReleaseInputs struct {
	SchemaVersion     int                       `json:"schema_version"`
	ReleaseDigest     string                    `json:"release_digest"`
	BuiltAt           string                    `json:"built_at"`
	Builder           ReleaseBuilder            `json:"builder"`
	Source            ReleaseSource             `json:"source"`
	Bundle            ReleaseBundle             `json:"bundle"`
	Dependencies      ReleaseDependencies       `json:"dependencies"`
	NativeCLI         ReleaseNativeCLI          `json:"native_cli"`
	ResourceArtifacts []ReleaseResourceArtifact `json:"resource_artifacts"`
	Toolchain         ReleaseToolchain          `json:"toolchain"`
	Configuration     ReleaseConfiguration      `json:"configuration"`
	Credentials       []ReleaseCredentialRef    `json:"credentials"`
	Provenance        ReleaseProvenance         `json:"provenance"`
	Reproducibility   ReleaseReproducibility    `json:"reproducibility"`
	Limitations       []string                  `json:"limitations"`
}

// ReleaseBuilder identifies who built the release and under which policy.
type ReleaseBuilder struct {
	Identity string `json:"identity"`
	Node     string `json:"node,omitempty"`
	User     string `json:"user,omitempty"`
}

// ReleaseSource records the scenario source as it was read.
type ReleaseSource struct {
	// Commit is the VCS head where available; empty when the tree is not a
	// repository or the VCS tool was unavailable.
	Commit string `json:"commit,omitempty"`
	// Dirty is true when the included roots differed from Commit. A dirty
	// build is recorded, not refused.
	Dirty bool `json:"dirty"`
	// DirtyPaths counts the modified or untracked included paths (0 when the
	// state could not be observed; see Limitations).
	DirtyPaths int `json:"dirty_paths"`
	// Snapshot is always "working-tree": the builder reads files in place and
	// never claims an isolated snapshot.
	Snapshot string `json:"snapshot"`
	// ContentManifestSHA256 is sha256 over the sorted "path\tsha256\n" lines of
	// every regular file the bundle carries (generated files included).
	ContentManifestSHA256 string   `json:"content_manifest_sha256"`
	FileCount             int      `json:"file_count"`
	IncludeRoots          []string `json:"include_roots"`
	Excludes              []string `json:"excludes"`
}

// ReleaseBundle describes the produced bundle artifact.
type ReleaseBundle struct {
	FileName  string `json:"file_name"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// ReleaseDependencies records the closure the release was built against.
type ReleaseDependencies struct {
	ClosureDigest       string   `json:"closure_digest"`
	AnalyzerTool        string   `json:"analyzer_tool,omitempty"`
	AnalyzerFingerprint string   `json:"analyzer_fingerprint,omitempty"`
	AnalyzerGeneratedAt string   `json:"analyzer_generated_at,omitempty"`
	Scenarios           []string `json:"scenarios"`
	Resources           []string `json:"resources"`
}

// ReleaseNativeCLI is the native control-plane binary and its build provenance.
type ReleaseNativeCLI struct {
	FileName  string   `json:"file_name"`
	SHA256    string   `json:"sha256"`
	GOOS      string   `json:"goos"`
	GOARCH    string   `json:"goarch"`
	SizeBytes int64    `json:"size_bytes"`
	Package   string   `json:"package"`
	ModuleDir string   `json:"module_dir"`
	GoVersion string   `json:"go_version"`
	Args      []string `json:"args"`
	// Env lists only the pinned build variables (never the inherited
	// environment).
	Env []string `json:"env"`
}

// ReleaseResourceArtifact is one platform artifact a resource contributes.
type ReleaseResourceArtifact struct {
	Component   string   `json:"component"`
	Platform    string   `json:"platform"`
	Name        string   `json:"name,omitempty"`
	Digest      string   `json:"digest,omitempty"`
	Mode        string   `json:"mode,omitempty"`
	Eligibility string   `json:"eligibility"`
	LicenseRefs []string `json:"license_refs"`
}

// ReleaseToolchain records the build toolchain.
type ReleaseToolchain struct {
	GoVersion  string   `json:"go_version"`
	HostGOOS   string   `json:"host_goos"`
	HostGOARCH string   `json:"host_goarch"`
	GoFlags    []string `json:"go_flags"`
}

// ReleaseConfiguration records the nonsecret configuration digest and its rule.
type ReleaseConfiguration struct {
	Digest string `json:"digest"`
	Schema string `json:"schema"`
	Rule   string `json:"rule"`
}

// ReleaseCredentialRef is a descriptor reference. It never carries a value.
type ReleaseCredentialRef struct {
	ID         string             `json:"id"`
	Class      string             `json:"class"`
	Required   bool               `json:"required"`
	Descriptor *DescriptorAddress `json:"descriptor,omitempty"`
	TargetType string             `json:"target_type"`
	TargetName string             `json:"target_name"`
}

// ReleaseProvenance names the trust policy and the signer reference.
type ReleaseProvenance struct {
	Policy string `json:"policy"`
	// TrustMode is the ArtifactTrustMode the build resolved from configuration.
	TrustMode string `json:"trust_mode"`
	// SignerKeyID is the release-authority key id when the manifest was signed.
	SignerKeyID string `json:"signer_key_id,omitempty"`
	// SignatureFile is the relative path of the detached signature when signed.
	SignatureFile string `json:"signature_file,omitempty"`
}

// ReleaseReproducibility records what was proven, not what was hoped.
type ReleaseReproducibility struct {
	Bundle    string `json:"bundle"`
	NativeCLI string `json:"native_cli"`
	Reason    string `json:"reason,omitempty"`
}

// ReleaseSummary is the API projection of one stored release.
type ReleaseSummary struct {
	Digest              string            `json:"release_digest"`
	Dir                 string            `json:"dir"`
	Complete            bool              `json:"complete"`
	BundleSHA256        string            `json:"bundle_sha256"`
	NativeCLI           ReleaseNativeCLI  `json:"native_cli"`
	ClosureDigest       string            `json:"closure_digest"`
	ConfigurationDigest string            `json:"configuration_digest"`
	Provenance          ReleaseProvenance `json:"provenance"`
	BuiltAt             string            `json:"built_at,omitempty"`
}
