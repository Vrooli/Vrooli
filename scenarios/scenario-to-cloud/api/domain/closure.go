package domain

// ClosureSchemaVersion identifies the wire shape of Closure. Bump it when a
// field changes meaning; additive optional fields keep the version.
const ClosureSchemaVersion = "1"

// ClosurePlatform is the concrete target platform a closure was resolved for.
// Architecture is never optional: artifact eligibility is per architecture.
type ClosurePlatform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// ClosureComponentKind classifies one included component.
type ClosureComponentKind string

const (
	ClosureKindScenario             ClosureComponentKind = "scenario"
	ClosureKindResource             ClosureComponentKind = "resource"
	ClosureKindPackage              ClosureComponentKind = "package"
	ClosureKindTool                 ClosureComponentKind = "tool"
	ClosureKindSafeguard            ClosureComponentKind = "safeguard"
	ClosureKindCredentialDescriptor ClosureComponentKind = "credential_descriptor"
	ClosureKindNativeArtifact       ClosureComponentKind = "native_artifact"
)

// ClosureReasonKind says why a component is in the closure. Every component
// carries at least one reason; the reason list is the explanation surface
// shared by UI, CLI and API, so no handler adds prose of its own.
type ClosureReasonKind string

const (
	// ClosureReasonDeclaredBy: a manifest (or the analyzer's reading of one)
	// declares this component as a dependency of From.
	ClosureReasonDeclaredBy ClosureReasonKind = "declared_by"
	// ClosureReasonTransitiveVia: reached through the dependency From.
	ClosureReasonTransitiveVia ClosureReasonKind = "transitive_via"
	// ClosureReasonSystemRequired: service.system_required is true in the
	// repository catalog scope.
	ClosureReasonSystemRequired ClosureReasonKind = "system_required"
	// ClosureReasonPlatformArtifact: a release artifact the platform needs.
	ClosureReasonPlatformArtifact ClosureReasonKind = "platform_artifact"
	// ClosureReasonCredentialOf: a credential descriptor declared by From.
	ClosureReasonCredentialOf ClosureReasonKind = "credential_of"
	// ClosureReasonSafeguardOf: a host tool or safeguard declared by From.
	ClosureReasonSafeguardOf ClosureReasonKind = "safeguard_of"
	// ClosureReasonSelectedBy: an optional component the operator selected.
	ClosureReasonSelectedBy ClosureReasonKind = "selected_by"
)

// ClosureReason is one edge of the explanation graph.
type ClosureReason struct {
	Kind   ClosureReasonKind `json:"kind"`
	From   string            `json:"from"`
	Detail string            `json:"detail,omitempty"`
}

// ClosureArtifactEligibility is the per-platform verdict for an artifact.
type ClosureArtifactEligibility string

const (
	// ClosureArtifactEligible: a digest-pinned artifact exists for the platform.
	ClosureArtifactEligible ClosureArtifactEligibility = "eligible"
	// ClosureArtifactBuildable: the artifact is produced at release time for
	// this platform and its digest is bound by the release, not the closure.
	ClosureArtifactBuildable ClosureArtifactEligibility = "buildable"
	// ClosureArtifactIneligible: nothing can supply the artifact on this platform.
	ClosureArtifactIneligible ClosureArtifactEligibility = "ineligible"
)

// ClosureArtifact is the platform artifact bound to a component.
type ClosureArtifact struct {
	Platform    string                     `json:"platform"`
	Name        string                     `json:"name,omitempty"`
	Digest      string                     `json:"digest,omitempty"`
	Mode        string                     `json:"mode,omitempty"`
	Eligibility ClosureArtifactEligibility `json:"eligibility"`
}

// ClosureSupervision carries the two independent runtime controls of a
// scenario component. Member (supervision membership, from the dependency
// edge's startup_policy) and AutoRestart (from runtime.auto_restart_default or
// an operator override) are never derived from each other.
type ClosureSupervision struct {
	Member            bool   `json:"member"`
	StartupPolicy     string `json:"startup_policy"`
	RuntimeKind       string `json:"runtime_kind"`
	AutoRestart       bool   `json:"auto_restart"`
	AutoRestartSource string `json:"auto_restart_source"`
}

// ClosureCredential preserves a credential descriptor address exactly as
// declared. LogicalID and Field are never normalised or merged.
type ClosureCredential struct {
	LogicalID string `json:"logical_id"`
	Field     string `json:"field"`
	Env       string `json:"env,omitempty"`
	Required  bool   `json:"required"`
	Label     string `json:"label,omitempty"`
}

// ClosureRecovery is the declared recovery contract of a scenario component.
type ClosureRecovery struct {
	CodeRollback   string `json:"code_rollback"`
	SchemaStrategy string `json:"schema_strategy"`
}

// ClosureComponent is one included component with its inclusion reasons.
type ClosureComponent struct {
	ID               string               `json:"id"`
	Kind             ClosureComponentKind `json:"kind"`
	Required         bool                 `json:"required"`
	OptionalSelected bool                 `json:"optional_selected"`
	Reasons          []ClosureReason      `json:"reasons"`
	Version          string               `json:"version,omitempty"`
	ContentIdentity  string               `json:"content_identity,omitempty"`
	Artifact         *ClosureArtifact     `json:"artifact,omitempty"`
	Supervision      *ClosureSupervision  `json:"supervision,omitempty"`
	Credential       *ClosureCredential   `json:"credential,omitempty"`
	Recovery         *ClosureRecovery     `json:"recovery,omitempty"`
}

// ClosurePersistentData is one declared persistent data set and its owners.
type ClosurePersistentData struct {
	ID             string `json:"id"`
	Owner          string `json:"owner"`
	Binding        string `json:"binding"`
	BackupProvider string `json:"backup_provider,omitempty"`
	MigrationOwner string `json:"migration_owner"`
	DeclaredBy     string `json:"declared_by"`
}

// ClosureReadiness is the declared readiness probe of a listener.
type ClosureReadiness struct {
	Type      string `json:"type"`
	Path      string `json:"path,omitempty"`
	TimeoutMS int    `json:"timeout_ms,omitempty"`
}

// ClosureListener is one declared listener and how the edge exposes it.
type ClosureListener struct {
	ID         string            `json:"id"`
	Owner      string            `json:"owner"`
	PortName   string            `json:"port_name"`
	Visibility string            `json:"visibility"`
	Readiness  *ClosureReadiness `json:"readiness,omitempty"`
}

// ClosurePrivilege is one privilege need expressed through the existing host
// requirement effect vocabulary (none|user|elevated).
type ClosurePrivilege struct {
	Effect    string `json:"effect"`
	Subject   string `json:"subject"`
	Safeguard string `json:"safeguard,omitempty"`
	Reason    string `json:"reason"`
}

// ClosureCapacityContribution is one component's declared footprint.
type ClosureCapacityContribution struct {
	Component   string  `json:"component"`
	CPU         float64 `json:"cpu"`
	MemoryBytes uint64  `json:"memory_bytes"`
	DiskBytes   uint64  `json:"disk_bytes"`
}

// ClosureCapacity is the aggregate requirement plus the transient headroom an
// update needs (two copies of the largest release artifact plus staging).
type ClosureCapacity struct {
	CPU                          float64                       `json:"cpu"`
	MemoryBytes                  uint64                        `json:"memory_bytes"`
	DiskBytes                    uint64                        `json:"disk_bytes"`
	TransientUpdateHeadroomBytes uint64                        `json:"transient_update_headroom_bytes"`
	HeadroomBasis                string                        `json:"headroom_basis"`
	Fit                          string                        `json:"fit"`
	Contributions                []ClosureCapacityContribution `json:"contributions,omitempty"`
}

// ClosureUnsupported names one component the target cannot satisfy and why.
type ClosureUnsupported struct {
	Component  string `json:"component"`
	ReasonCode string `json:"reason_code"`
	Detail     string `json:"detail"`
}

// ClosureSources records which catalogs produced the closure. It is part of
// the digest because a closure derived without the analyzer is a different
// artifact from one derived with it.
type ClosureSources struct {
	Catalog          string `json:"catalog"`
	Scope            string `json:"scope"`
	HostRequirements string `json:"host_requirements"`
	AnalyzerTool     string `json:"analyzer_tool,omitempty"`
	AnalyzerUsed     bool   `json:"analyzer_used"`
}

// Closure is the versioned deployment closure: every component the target
// must hold for the scenario, with reasons, platform artifacts, declared data
// and listener contracts, privilege needs, capacity, and the unsupported list.
// Digest is sha256 over the canonical JSON of everything except Digest.
type Closure struct {
	SchemaVersion  string                  `json:"schema_version"`
	ScenarioID     string                  `json:"scenario_id"`
	Environment    string                  `json:"environment"`
	Platform       ClosurePlatform         `json:"platform"`
	Components     []ClosureComponent      `json:"components"`
	PersistentData []ClosurePersistentData `json:"persistent_data"`
	Listeners      []ClosureListener       `json:"listeners"`
	Privileges     []ClosurePrivilege      `json:"privileges"`
	Capacity       ClosureCapacity         `json:"capacity"`
	Unsupported    []ClosureUnsupported    `json:"unsupported"`
	Sources        ClosureSources          `json:"sources"`
	Digest         string                  `json:"digest"`
}

// Supported reports whether nothing in the closure is unsupported.
func (c Closure) Supported() bool { return len(c.Unsupported) == 0 }

// ComponentsOfKind returns the components of one kind in closure order.
func (c Closure) ComponentsOfKind(kind ClosureComponentKind) []ClosureComponent {
	var out []ClosureComponent
	for _, component := range c.Components {
		if component.Kind == kind {
			out = append(out, component)
		}
	}
	return out
}
