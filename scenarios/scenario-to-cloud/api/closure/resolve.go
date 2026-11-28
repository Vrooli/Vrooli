package closure

import (
	"context"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	"scenario-to-cloud/domain"
	"strings"
)

type Platform struct {
	OS   string
	Arch string
}

func (p Platform) String() string { return p.OS + "-" + p.Arch }

// Scope selects the catalog scope rules.
type Scope string

const (
	// ScopeBundle is the mini-Vrooli bundle: only components reached from the
	// selected scenario ship. System-required scenarios join only when a
	// declared edge reaches them.
	ScopeBundle Scope = "bundle"
	// ScopeRepository is a full repository install: system-required scenarios
	// and their dependencies are always included.
	ScopeRepository Scope = "repository"
)

// Overrides are the operator's selection choices. They never make a required
// dependency optional and never couple supervision membership to auto-restart.
type Overrides struct {
	// SelectOptional selects optional dependencies (scenario or resource ids)
	// that the declaring manifest leaves disabled.
	SelectOptional []string
	// DeselectOptional removes optional dependencies the declaring manifest
	// enables. A required dependency cannot be deselected.
	DeselectOptional []string
	// AutoRestart overrides runtime.auto_restart_default per scenario.
	AutoRestart map[string]bool
	// SupervisionMember overrides supervision membership per scenario.
	SupervisionMember map[string]bool
	// OperatingMode requests a provider mode per resource.
	OperatingMode map[string]string
}

// TargetCapacity is what the target machine offers.
type TargetCapacity struct {
	CPU         float64 `json:"cpu"`
	MemoryBytes uint64  `json:"memory_bytes"`
	DiskBytes   uint64  `json:"disk_bytes"`
}

// Inputs drive one closure resolution.
type Inputs struct {
	ScenarioID  string
	Environment string
	Platform    Platform
	Scope       Scope
	RepoRoot    string

	Catalog          Catalog
	Analyzer         AnalyzerClient
	HostRequirements HostRequirementsResolver

	Overrides      Overrides
	TargetCapacity *TargetCapacity
	// ReleaseArtifactBytes are the sizes of known release artifacts. The
	// transient update headroom is two copies of the largest plus staging.
	ReleaseArtifactBytes []uint64
	// DeploymentProfile names the resource deployment profile to resolve
	// artifacts from. Resources declare "desktop" today.
	DeploymentProfile string
}

const (
	// DefaultDeploymentProfile is the resource profile resources declare today.
	DefaultDeploymentProfile = "desktop"
	// StagingHeadroomBytes is the fixed staging allowance added to the
	// transient update headroom (256 MiB).
	StagingHeadroomBytes uint64 = 256 << 20

	mebibyte = 1 << 20

	// Unsupported reason codes.
	ReasonMissingPlatformArtifact         = "missing_platform_artifact"
	ReasonUnsupportedPlatform             = "unsupported_platform"
	ReasonUndeclaredDeploymentProfile     = "undeclared_deployment_profile"
	ReasonUnsupportedControlPlanePlatform = "unsupported_control_plane_platform"
	ReasonInsufficientCapacity            = "insufficient_capacity"

	controlPlaneComponentID = "vrooli"
)

// controlPlanePlatforms mirrors the platforms the native control plane is
// cross-compiled for (api/vps/native_cli.go detects the same set).
var controlPlanePlatforms = map[string][]string{"linux": {"amd64", "arm64"}}

type scenarioNode struct {
	decl             *ScenarioDeclaration
	included         bool
	required         bool
	optionalSelected bool
	reasons          []domain.ClosureReason
	startupPolicies  []string
	member           bool
	includeUI        bool
}

type resourceNode struct {
	decl             *ResourceDeclaration
	included         bool
	required         bool
	optionalSelected bool
	reasons          []domain.ClosureReason
}

type edge struct {
	fromKind domain.ClosureComponentKind
	from     string
	toKind   domain.ClosureComponentKind
	to       string
	required bool
	selected bool
	policy   string
	detail   string
}

type resolver struct {
	in               Inputs
	ctx              context.Context
	platform         Platform
	scenarios        map[string]*scenarioNode
	resources        map[string]*resourceNode
	edges            []edge
	dependencyDigest string
	analyzer         struct {
		used bool
		tool string
	}
	unsupported []domain.ClosureUnsupported
	selectSet   map[string]bool
	deselectSet map[string]bool
}

// Resolve derives the deployment closure for Inputs.
func Resolve(ctx context.Context, in Inputs) (domain.Closure, error) {
	in.ScenarioID = strings.TrimSpace(in.ScenarioID)
	if !validComponentID(in.ScenarioID) {
		return domain.Closure{}, newError(CodeInvalidRequest, "scenario id is required", map[string]any{"scenario_id": in.ScenarioID})
	}
	if in.Catalog == nil {
		return domain.Closure{}, unavailable("no component catalog is configured", nil, nil)
	}
	if in.HostRequirements == nil {
		return domain.Closure{}, unavailable("no host requirements resolver is configured", nil, nil)
	}
	canonical, err := resourcedeployment.CanonicalPlatform(in.Platform.OS, in.Platform.Arch)
	if err != nil {
		return domain.Closure{}, newError(CodeInvalidRequest, "platform is invalid", map[string]any{"os": in.Platform.OS, "arch": in.Platform.Arch, "cause": err.Error()})
	}
	if in.Environment == "" {
		in.Environment = "production"
	}
	if in.Scope == "" {
		in.Scope = ScopeBundle
	}
	if in.Scope != ScopeBundle && in.Scope != ScopeRepository {
		return domain.Closure{}, newError(CodeInvalidRequest, "scope is invalid", map[string]any{"scope": string(in.Scope)})
	}
	if in.DeploymentProfile == "" {
		in.DeploymentProfile = DefaultDeploymentProfile
	}

	r := &resolver{
		in:          in,
		ctx:         ctx,
		platform:    Platform{OS: canonical.OS, Arch: canonical.Arch},
		scenarios:   map[string]*scenarioNode{},
		resources:   map[string]*resourceNode{},
		selectSet:   toSet(in.Overrides.SelectOptional),
		deselectSet: toSet(in.Overrides.DeselectOptional),
	}

	root, err := r.loadScenario(in.ScenarioID, "")
	if err != nil {
		return domain.Closure{}, err
	}
	root.included = true
	root.required = true
	root.member = true
	root.startupPolicies = append(root.startupPolicies, "must_start")
	root.reasons = append(root.reasons, domain.ClosureReason{Kind: domain.ClosureReasonDeclaredBy, From: "selection", Detail: "selected scenario"})
	roots := []*scenarioNode{root}

	if in.Scope == ScopeRepository {
		ids, err := in.Catalog.SystemRequiredScenarios(ctx)
		if err != nil {
			return domain.Closure{}, unavailable("system-required scenario catalog is unreadable", err, nil)
		}
		for _, id := range ids {
			node, err := r.loadScenario(id, "catalog")
			if err != nil {
				return domain.Closure{}, err
			}
			alreadyIncluded := node.included
			node.included = true
			node.required = true
			node.member = true
			node.startupPolicies = append(node.startupPolicies, "must_start")
			node.reasons = append(node.reasons, domain.ClosureReason{Kind: domain.ClosureReasonSystemRequired, From: "catalog", Detail: "service.system_required is true"})
			if !alreadyIncluded {
				roots = append(roots, node)
			}
		}
	}
	if err := r.buildSharedClosure(roots); err != nil {
		return domain.Closure{}, err
	}

	r.propagateRequired()
	if err := r.checkConflicts(); err != nil {
		return domain.Closure{}, err
	}
	return r.assemble()
}
