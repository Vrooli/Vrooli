package closure

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"scenario-to-cloud/domain"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// Platform is the concrete target platform (os: linux|macos|windows, arch: amd64|arm64).
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
	in        Inputs
	ctx       context.Context
	platform  Platform
	scenarios map[string]*scenarioNode
	resources map[string]*resourceNode
	edges     []edge
	stack     []string
	// stackRequired[i] records whether the edge that put stack[i] on the
	// stack was required. A back-edge closes an orchestration cycle only when
	// every edge on the cycle is required; optional (try_start) cycles are a
	// declared degradation pattern, not an error.
	stackRequired []bool
	onStack       map[string]bool
	analyzer      struct {
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
		onStack:     map[string]bool{},
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
	if err := r.walkScenario(root, true); err != nil {
		return domain.Closure{}, err
	}

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
			if alreadyIncluded {
				continue
			}
			if err := r.walkScenario(node, true); err != nil {
				return domain.Closure{}, err
			}
		}
	}

	r.propagateRequired()
	if err := r.checkConflicts(); err != nil {
		return domain.Closure{}, err
	}
	return r.assemble()
}

func toSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = true
		}
	}
	return set
}

func (r *resolver) loadScenario(id, declaredBy string) (*scenarioNode, error) {
	if node, ok := r.scenarios[id]; ok {
		return node, nil
	}
	decl, err := r.in.Catalog.Scenario(r.ctx, id)
	if err != nil {
		details := map[string]any{"component": id, "kind": "scenario"}
		if declaredBy != "" {
			details["declared_by"] = declaredBy
		}
		if errors.Is(err, ErrNotFound) {
			return nil, unavailable(fmt.Sprintf("scenario %q is not in the catalog", id), err, details)
		}
		return nil, unavailable(fmt.Sprintf("scenario %q declaration is unreadable", id), err, details)
	}
	node := &scenarioNode{decl: decl}
	r.scenarios[id] = node
	return node, nil
}

func (r *resolver) loadResource(id, declaredBy string) (*resourceNode, error) {
	if node, ok := r.resources[id]; ok {
		return node, nil
	}
	decl, err := r.in.Catalog.Resource(r.ctx, id)
	if err != nil {
		details := map[string]any{"component": id, "kind": "resource", "declared_by": declaredBy}
		if errors.Is(err, ErrNotFound) {
			return nil, unavailable(fmt.Sprintf("resource %q declared by %s is not in the catalog", id, declaredBy), err, details)
		}
		return nil, unavailable(fmt.Sprintf("resource %q declaration is unreadable", id), err, details)
	}
	node := &resourceNode{decl: decl}
	r.resources[id] = node
	return node, nil
}

// edgePasses decides whether a declared edge brings its target into the
// closure: required edges always do; optional edges do when the manifest
// enables them (and the operator has not deselected them) or when the
// operator selected them.
func (r *resolver) edgePasses(to string, edge DependencyEdge) (passes, selected bool) {
	if edge.Required {
		return true, false
	}
	if r.selectSet[to] {
		return true, true
	}
	if edge.Enabled && !r.deselectSet[to] {
		return true, true
	}
	return false, false
}

type declaredEdge struct {
	to     string
	edge   DependencyEdge
	detail string
}

func (r *resolver) scenarioEdges(decl *ScenarioDeclaration) (scenarios, resources []declaredEdge, err error) {
	for id, edge := range decl.Scenarios {
		scenarios = append(scenarios, declaredEdge{to: id, edge: edge, detail: "declared in service.json"})
	}
	for id, edge := range decl.Resources {
		resources = append(resources, declaredEdge{to: id, edge: edge, detail: "declared in service.json"})
	}
	if r.in.Analyzer != nil {
		result, err := r.in.Analyzer.Analyze(r.ctx, decl.ID)
		if err != nil {
			return nil, nil, unavailable(fmt.Sprintf("dependency analyzer is unavailable for scenario %q", decl.ID), err, map[string]any{"component": decl.ID, "source": "analyzer"})
		}
		r.analyzer.used = true
		if result.Tool != "" {
			r.analyzer.tool = result.Tool
		} else {
			r.analyzer.tool = analyzerTool
		}
		convert := func(items []AnalyzedDependency, into *[]declaredEdge) {
			for _, item := range items {
				if !validComponentID(item.Name) {
					continue
				}
				*into = append(*into, declaredEdge{
					to:     item.Name,
					edge:   DependencyEdge{Enabled: item.Enabled, Required: item.Required, StartupPolicy: policyFor(item.Required)},
					detail: "detected by " + r.analyzer.tool,
				})
			}
		}
		convert(result.Scenarios, &scenarios)
		convert(result.Resources, &resources)
	}
	sort.Slice(scenarios, func(i, j int) bool { return lessEdge(scenarios[i], scenarios[j]) })
	sort.Slice(resources, func(i, j int) bool { return lessEdge(resources[i], resources[j]) })
	return scenarios, resources, nil
}

func lessEdge(a, b declaredEdge) bool {
	if a.to != b.to {
		return a.to < b.to
	}
	return a.detail < b.detail
}

func policyFor(required bool) string {
	if required {
		return "must_start"
	}
	return "try_start"
}

func (r *resolver) push(key string, required bool) func() {
	r.stack = append(r.stack, key)
	r.stackRequired = append(r.stackRequired, required)
	r.onStack[key] = true
	return func() {
		r.stack = r.stack[:len(r.stack)-1]
		r.stackRequired = r.stackRequired[:len(r.stackRequired)-1]
		delete(r.onStack, key)
	}
}

// backEdgeClosesRequiredCycle reports whether a back-edge to a node on the
// stack forms a cycle made only of required edges.
func (r *resolver) backEdgeClosesRequiredCycle(at string, edgeRequired bool) bool {
	if !edgeRequired {
		return false
	}
	for i := len(r.stack) - 1; i >= 0; i-- {
		if r.stack[i] == at {
			return true
		}
		if !r.stackRequired[i] {
			return false
		}
	}
	return false
}

func (r *resolver) walkScenario(node *scenarioNode, viaRequired bool) error {
	id := node.decl.ID
	defer r.push(id, viaRequired)()

	scenarioEdges, resourceEdges, err := r.scenarioEdges(node.decl)
	if err != nil {
		return err
	}
	for _, declared := range scenarioEdges {
		if declared.to == id && declared.edge.Required {
			return r.cycleError(id)
		}
		if declared.to == id {
			continue
		}
		passes, selected := r.edgePasses(declared.to, declared.edge)
		if !passes {
			continue
		}
		child, err := r.loadScenario(declared.to, id)
		if err != nil {
			return err
		}
		r.edges = append(r.edges, edge{fromKind: domain.ClosureKindScenario, from: id, toKind: domain.ClosureKindScenario, to: declared.to, required: declared.edge.Required, selected: selected, policy: declared.edge.StartupPolicy, detail: declared.detail})
		alreadyIncluded := child.included
		child.included = true
		if selected {
			child.optionalSelected = true
		}
		child.startupPolicies = append(child.startupPolicies, declared.edge.StartupPolicy)
		if declared.edge.StartupPolicy != "ignore" {
			child.member = true
		}
		child.reasons = append(child.reasons, r.reasonsForEdge(id, declared, selected)...)
		if r.onStack[declared.to] {
			if r.backEdgeClosesRequiredCycle(declared.to, declared.edge.Required) {
				return r.cycleError(declared.to)
			}
			continue
		}
		if !alreadyIncluded {
			if err := r.walkScenario(child, declared.edge.Required); err != nil {
				return err
			}
		}
	}
	for _, declared := range resourceEdges {
		passes, selected := r.edgePasses(declared.to, declared.edge)
		if !passes {
			continue
		}
		child, err := r.loadResource(declared.to, id)
		if err != nil {
			return err
		}
		r.edges = append(r.edges, edge{fromKind: domain.ClosureKindScenario, from: id, toKind: domain.ClosureKindResource, to: declared.to, required: declared.edge.Required, selected: selected, policy: declared.edge.StartupPolicy, detail: declared.detail})
		alreadyIncluded := child.included
		child.included = true
		if selected {
			child.optionalSelected = true
		}
		child.reasons = append(child.reasons, r.reasonsForEdge(id, declared, selected)...)
		if !alreadyIncluded {
			if err := r.walkResource(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *resolver) walkResource(node *resourceNode) error {
	id := node.decl.ID
	key := "resource:" + id
	defer r.push(key, true)()
	deps := append([]string(nil), node.decl.Dependencies...)
	sort.Strings(deps)
	for _, dep := range deps {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		if dep == id || r.onStack["resource:"+dep] {
			return r.cycleError("resource:" + dep)
		}
		child, err := r.loadResource(dep, "resource:"+id)
		if err != nil {
			return err
		}
		r.edges = append(r.edges, edge{fromKind: domain.ClosureKindResource, from: id, toKind: domain.ClosureKindResource, to: dep, required: true, detail: "declared in resource.json"})
		alreadyIncluded := child.included
		child.included = true
		child.reasons = append(child.reasons,
			domain.ClosureReason{Kind: domain.ClosureReasonDeclaredBy, From: "resource:" + id, Detail: "declared in resource.json"},
			domain.ClosureReason{Kind: domain.ClosureReasonTransitiveVia, From: "resource:" + id, Detail: "path " + r.pathString(dep)},
		)
		if !alreadyIncluded {
			if err := r.walkResource(child); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *resolver) reasonsForEdge(from string, declared declaredEdge, selected bool) []domain.ClosureReason {
	detail := declared.detail
	if declared.edge.Description != "" {
		detail += ": " + declared.edge.Description
	}
	reasons := []domain.ClosureReason{{Kind: domain.ClosureReasonDeclaredBy, From: from, Detail: detail}}
	if from != r.in.ScenarioID {
		reasons = append(reasons, domain.ClosureReason{Kind: domain.ClosureReasonTransitiveVia, From: from, Detail: "path " + r.pathString(declared.to)})
	}
	if selected {
		source := "manifest default (enabled optional dependency)"
		if r.selectSet[declared.to] {
			source = "operator selection"
		}
		reasons = append(reasons, domain.ClosureReason{Kind: domain.ClosureReasonSelectedBy, From: from, Detail: source})
	}
	return reasons
}

func (r *resolver) pathString(to string) string {
	parts := append(append([]string(nil), r.stack...), to)
	return strings.Join(parts, " -> ")
}

func (r *resolver) cycleError(at string) error {
	path := append([]string(nil), r.stack...)
	start := 0
	for i, id := range path {
		if id == at {
			start = i
			break
		}
	}
	cycle := append(append([]string(nil), path[start:]...), at)
	return newError(CodeCycle, "dependency cycle: "+strings.Join(cycle, " -> "), map[string]any{"path": cycle})
}

// propagateRequired computes requiredness as a fixpoint over required edges
// from required, included components. Optional selection never flips it.
func (r *resolver) propagateRequired() {
	changed := true
	for changed {
		changed = false
		for _, e := range r.edges {
			if !e.required {
				continue
			}
			var fromRequired bool
			switch e.fromKind {
			case domain.ClosureKindScenario:
				fromRequired = r.scenarios[e.from].required
			case domain.ClosureKindResource:
				fromRequired = r.resources[e.from].required
			}
			if !fromRequired {
				continue
			}
			switch e.toKind {
			case domain.ClosureKindScenario:
				if node := r.scenarios[e.to]; !node.required {
					node.required = true
					changed = true
				}
			case domain.ClosureKindResource:
				if node := r.resources[e.to]; !node.required {
					node.required = true
					changed = true
				}
			}
		}
	}
}

// checkConflicts rejects contradictory operating modes: a one-shot scenario
// with auto-restart, or a requested provider mode the resource forbids.
func (r *resolver) checkConflicts() error {
	for _, id := range sortedScenarioIDs(r.scenarios) {
		node := r.scenarios[id]
		if !node.included {
			continue
		}
		autoRestart, _ := r.autoRestart(node)
		if node.decl.RuntimeKind == "one_shot" && autoRestart {
			return newError(CodeConflict, fmt.Sprintf("scenario %q is one_shot but auto-restart is enabled", id), map[string]any{"component": id, "runtime_kind": "one_shot", "auto_restart": true})
		}
	}
	for _, id := range sortedResourceIDs(r.resources) {
		node := r.resources[id]
		if !node.included {
			continue
		}
		mode, ok := r.in.Overrides.OperatingMode[id]
		if !ok || mode == "" {
			continue
		}
		if len(node.decl.AllowedModes) > 0 && !containsFold(node.decl.AllowedModes, mode) {
			return newError(CodeConflict, fmt.Sprintf("resource %q does not allow operating mode %q", id, mode), map[string]any{"component": id, "requested_mode": mode, "allowed_modes": node.decl.AllowedModes})
		}
	}
	return nil
}

func (r *resolver) autoRestart(node *scenarioNode) (value bool, source string) {
	if override, ok := r.in.Overrides.AutoRestart[node.decl.ID]; ok {
		return override, "override"
	}
	return node.decl.AutoRestartDefault, "declared"
}

func (r *resolver) supervisionMember(node *scenarioNode) bool {
	if override, ok := r.in.Overrides.SupervisionMember[node.decl.ID]; ok {
		return override
	}
	return node.member
}

func strongestPolicy(policies []string) string {
	rank := map[string]int{"must_start": 3, "try_start": 2, "ignore": 1}
	best := ""
	for _, policy := range policies {
		if rank[policy] > rank[best] {
			best = policy
		}
	}
	if best == "" {
		return "try_start"
	}
	return best
}

func sortedScenarioIDs(nodes map[string]*scenarioNode) []string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func sortedResourceIDs(nodes map[string]*resourceNode) []string {
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (r *resolver) addUnsupported(component, code, detail string) {
	r.unsupported = append(r.unsupported, domain.ClosureUnsupported{Component: component, ReasonCode: code, Detail: detail})
}

func (r *resolver) assemble() (domain.Closure, error) {
	closure := domain.Closure{
		SchemaVersion: domain.ClosureSchemaVersion,
		ScenarioID:    r.in.ScenarioID,
		Environment:   r.in.Environment,
		Platform:      domain.ClosurePlatform{OS: r.platform.OS, Arch: r.platform.Arch},
		Sources: domain.ClosureSources{
			Catalog:          r.in.Catalog.Name(),
			Scope:            string(r.in.Scope),
			HostRequirements: r.in.HostRequirements.Name(),
			AnalyzerTool:     r.analyzer.tool,
			AnalyzerUsed:     r.analyzer.used,
		},
	}

	var includedScenarios []*ScenarioDeclaration
	var includedResources []*ResourceDeclaration
	credentials := map[string]*domain.ClosureComponent{}
	credentialOrder := []string{}

	addCredential := func(owner string, descriptor CredentialDescriptor) {
		address := descriptor.Address()
		component, ok := credentials[address]
		if !ok {
			field := descriptor.Field
			if field == "" {
				field = "value"
			}
			component = &domain.ClosureComponent{
				ID:   address,
				Kind: domain.ClosureKindCredentialDescriptor,
				Credential: &domain.ClosureCredential{
					LogicalID: descriptor.LogicalID,
					Field:     field,
					Env:       descriptor.Env,
					Required:  descriptor.Required,
					Label:     descriptor.Label,
				},
			}
			credentials[address] = component
			credentialOrder = append(credentialOrder, address)
		}
		component.Required = component.Required || descriptor.Required
		component.Credential.Required = component.Required
		component.Reasons = append(component.Reasons, domain.ClosureReason{Kind: domain.ClosureReasonCredentialOf, From: owner, Detail: "declared in credentials.descriptors"})
	}

	for _, id := range sortedScenarioIDs(r.scenarios) {
		node := r.scenarios[id]
		if !node.included {
			continue
		}
		includedScenarios = append(includedScenarios, node.decl)
		autoRestart, source := r.autoRestart(node)
		component := domain.ClosureComponent{
			ID:               id,
			Kind:             domain.ClosureKindScenario,
			Required:         node.required,
			OptionalSelected: !node.required && node.optionalSelected,
			Reasons:          normalizeReasons(node.reasons),
			Version:          node.decl.Version,
			ContentIdentity:  node.decl.ContentIdentity,
			Supervision: &domain.ClosureSupervision{
				Member:            r.supervisionMember(node),
				StartupPolicy:     strongestPolicy(node.startupPolicies),
				RuntimeKind:       node.decl.RuntimeKind,
				AutoRestart:       autoRestart,
				AutoRestartSource: source,
			},
		}
		if node.decl.Deployment.Recovery != nil {
			component.Recovery = &domain.ClosureRecovery{CodeRollback: node.decl.Deployment.Recovery.CodeRollback, SchemaStrategy: node.decl.Deployment.Recovery.SchemaStrategy}
		}
		if !node.decl.Deployment.Supports(r.platform.OS, r.platform.Arch) {
			r.addUnsupported(id, ReasonUnsupportedPlatform, fmt.Sprintf("scenario %s declares supported targets that exclude %s", id, r.platform))
		}
		closure.Components = append(closure.Components, component)
		for _, descriptor := range node.decl.Credentials {
			addCredential("scenario:"+id, descriptor)
		}
		r.collectPersistentData(&closure, "scenario:"+id, id, node.decl.Deployment)
		r.collectScenarioListeners(&closure, node.decl)
	}

	for _, id := range sortedResourceIDs(r.resources) {
		node := r.resources[id]
		if !node.included {
			continue
		}
		includedResources = append(includedResources, node.decl)
		component := domain.ClosureComponent{
			ID:               id,
			Kind:             domain.ClosureKindResource,
			Required:         node.required,
			OptionalSelected: !node.required && node.optionalSelected,
			Reasons:          normalizeReasons(node.reasons),
			Version:          node.decl.Version,
			ContentIdentity:  node.decl.ContentIdentity,
		}
		artifact, native := r.resolveResourceArtifact(node.decl)
		component.Artifact = artifact
		closure.Components = append(closure.Components, component)
		if native != nil {
			closure.Components = append(closure.Components, *native)
		}
		if node.decl.Privilege == "user" || node.decl.Privilege == "elevated" {
			closure.Privileges = append(closure.Privileges, domain.ClosurePrivilege{Effect: node.decl.Privilege, Subject: "resource:" + id, Reason: "resource.json privilege declaration"})
		}
		for _, descriptor := range node.decl.Credentials {
			addCredential("resource:"+id, descriptor)
		}
		r.collectPersistentData(&closure, "resource:"+id, id, node.decl.Deployment)
		for _, port := range node.decl.Ports {
			closure.Listeners = append(closure.Listeners, domain.ClosureListener{ID: id + "/" + port, Owner: "resource:" + id, PortName: port, Visibility: listenerVisibility(node.decl.Deployment, port)})
		}
	}

	r.addControlPlane(&closure)
	if r.in.Scope == ScopeBundle {
		for _, root := range []string{"internal", "packages"} {
			closure.Components = append(closure.Components, domain.ClosureComponent{
				ID:       root,
				Kind:     domain.ClosureKindPackage,
				Required: true,
				Reasons: []domain.ClosureReason{{
					Kind:   domain.ClosureReasonDeclaredBy,
					From:   "repo-contract:mini_vrooli_bundle",
					Detail: "the bundle profile ships the full shared " + root + " tree; it is not minimised per scenario",
				}},
			})
		}
	}

	requirements, err := r.in.HostRequirements.Resolve(r.ctx, HostRequirementsQuery{
		RepoRoot:    r.in.RepoRoot,
		Environment: r.in.Environment,
		Platform:    r.platform,
		Scenarios:   includedScenarios,
		Resources:   includedResources,
	})
	if err != nil {
		return domain.Closure{}, unavailable("host requirements could not be resolved", err, map[string]any{"source": r.in.HostRequirements.Name()})
	}
	for _, requirement := range requirements {
		kind := domain.ClosureKindTool
		if requirement.Kind == "safeguard" {
			kind = domain.ClosureKindSafeguard
		}
		component := domain.ClosureComponent{ID: requirement.Name, Kind: kind, Required: requirement.Required}
		for _, owner := range requirement.DeclaredBy {
			component.Reasons = append(component.Reasons, domain.ClosureReason{Kind: domain.ClosureReasonSafeguardOf, From: owner, Detail: strings.Join(requirement.Reasons, "; ")})
		}
		component.Reasons = normalizeReasons(component.Reasons)
		closure.Components = append(closure.Components, component)
		if requirement.Privilege == "user" || requirement.Privilege == "elevated" {
			privilege := domain.ClosurePrivilege{Effect: requirement.Privilege, Subject: requirement.Kind + ":" + requirement.Name, Reason: strings.Join(requirement.Reasons, "; ")}
			if kind == domain.ClosureKindSafeguard {
				privilege.Safeguard = requirement.Name
			}
			closure.Privileges = append(closure.Privileges, privilege)
		}
	}

	for _, address := range credentialOrder {
		component := credentials[address]
		component.Reasons = normalizeReasons(component.Reasons)
		closure.Components = append(closure.Components, *component)
	}

	closure.Capacity = r.capacity(includedScenarios, includedResources)

	sortComponents(closure.Components)
	sort.Slice(closure.PersistentData, func(i, j int) bool {
		a, b := closure.PersistentData[i], closure.PersistentData[j]
		if a.DeclaredBy != b.DeclaredBy {
			return a.DeclaredBy < b.DeclaredBy
		}
		return a.ID < b.ID
	})
	sort.Slice(closure.Listeners, func(i, j int) bool { return closure.Listeners[i].ID < closure.Listeners[j].ID })
	sort.Slice(closure.Privileges, func(i, j int) bool {
		a, b := closure.Privileges[i], closure.Privileges[j]
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		return a.Effect < b.Effect
	})
	sort.Slice(r.unsupported, func(i, j int) bool {
		a, b := r.unsupported[i], r.unsupported[j]
		if a.Component != b.Component {
			return a.Component < b.Component
		}
		return a.ReasonCode < b.ReasonCode
	})
	closure.Unsupported = r.unsupported
	if closure.Unsupported == nil {
		closure.Unsupported = []domain.ClosureUnsupported{}
	}
	if closure.PersistentData == nil {
		closure.PersistentData = []domain.ClosurePersistentData{}
	}
	if closure.Listeners == nil {
		closure.Listeners = []domain.ClosureListener{}
	}
	if closure.Privileges == nil {
		closure.Privileges = []domain.ClosurePrivilege{}
	}

	digest, err := Digest(closure)
	if err != nil {
		return domain.Closure{}, fmt.Errorf("digest closure: %w", err)
	}
	closure.Digest = digest
	return closure, nil
}

func (r *resolver) collectPersistentData(closure *domain.Closure, declaredBy, defaultOwner string, deployment DeploymentDeclaration) {
	for _, data := range deployment.PersistentData {
		owner := data.Owner
		if owner == "" {
			owner = defaultOwner
		}
		closure.PersistentData = append(closure.PersistentData, domain.ClosurePersistentData{
			ID:             data.ID,
			Owner:          owner,
			Binding:        data.Binding,
			BackupProvider: data.BackupProvider,
			MigrationOwner: data.MigrationOwner,
			DeclaredBy:     declaredBy,
		})
	}
}

func listenerVisibility(deployment DeploymentDeclaration, port string) string {
	for _, listener := range deployment.Listeners {
		if listener.Port == port && listener.Visibility != "" {
			return listener.Visibility
		}
	}
	return "private"
}

func (r *resolver) collectScenarioListeners(closure *domain.Closure, decl *ScenarioDeclaration) {
	names := make([]string, 0, len(decl.Ports))
	for name := range decl.Ports {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		visibility := listenerVisibility(decl.Deployment, name)
		if decl.ID == closure.ScenarioID && len(decl.Deployment.Listeners) == 0 && name == "ui" {
			// A deployed scenario that declares no listeners keeps the
			// launch contract it shipped under: its UI port is the one
			// public route behind the edge; every other port stays private
			// until the scenario declares otherwise.
			visibility = "public_via_edge"
		}
		listener := domain.ClosureListener{ID: decl.ID + "/" + name, Owner: "scenario:" + decl.ID, PortName: name, Visibility: visibility}
		var readiness *ReadinessDeclaration
		for _, declared := range decl.Deployment.Listeners {
			if declared.Port == name && declared.Readiness != nil {
				readiness = declared.Readiness
			}
		}
		if readiness == nil {
			if declared, ok := decl.ComponentReadiness[name]; ok {
				readiness = &declared
			}
		}
		if readiness != nil {
			listener.Readiness = &domain.ClosureReadiness{Type: readiness.Type, Path: readiness.Path, TimeoutMS: readiness.TimeoutMS}
		}
		closure.Listeners = append(closure.Listeners, listener)
	}
}

// resolveResourceArtifact binds the resource's platform artifact or records
// why the platform cannot be served. A missing artifact is never silent.
func (r *resolver) resolveResourceArtifact(decl *ResourceDeclaration) (*domain.ClosureArtifact, *domain.ClosureComponent) {
	platform := r.platform.String()
	if !decl.Deployment.Supports(r.platform.OS, r.platform.Arch) {
		r.addUnsupported(decl.ID, ReasonUnsupportedPlatform, fmt.Sprintf("resource %s declares supported targets that exclude %s", decl.ID, platform))
	}
	if _, ok := decl.Profiles.Profiles[r.in.DeploymentProfile]; !ok {
		r.addUnsupported(decl.ID, ReasonUndeclaredDeploymentProfile, fmt.Sprintf("resource %s declares no %q deployment profile", decl.ID, r.in.DeploymentProfile))
		return nil, nil
	}
	target, found := decl.Profiles.Target(r.in.DeploymentProfile, r.platform.OS, "")
	if !found {
		r.addUnsupported(decl.ID, ReasonUnsupportedPlatform, fmt.Sprintf("resource %s declares no %s target in profile %q", decl.ID, r.platform.OS, r.in.DeploymentProfile))
		return nil, nil
	}
	if target.Support == "unsupported" {
		r.addUnsupported(decl.ID, ReasonUnsupportedPlatform, fmt.Sprintf("resource %s: %s", decl.ID, firstNonEmpty(target.Reason, "declared unsupported on "+r.platform.OS)))
		return nil, nil
	}
	if _, archOK := decl.Profiles.ResolveTarget(r.in.DeploymentProfile, resourcedeployment.Platform{OS: r.platform.OS, Arch: r.platform.Arch}); !archOK {
		r.addUnsupported(decl.ID, ReasonMissingPlatformArtifact, fmt.Sprintf("resource %s declares no %s artifact for %s (declared architectures: %s)", decl.ID, r.platform.Arch, r.platform.OS, strings.Join(target.Architectures, ", ")))
		return &domain.ClosureArtifact{Platform: platform, Mode: target.Mode, Eligibility: domain.ClosureArtifactIneligible}, nil
	}
	if !strings.HasPrefix(target.Mode, "bundled-") {
		return nil, nil
	}
	if decl.Artifact == nil {
		r.addUnsupported(decl.ID, ReasonMissingPlatformArtifact, fmt.Sprintf("resource %s uses %s on %s but declares no managed_service.artifact", decl.ID, target.Mode, platform))
		return &domain.ClosureArtifact{Platform: platform, Mode: target.Mode, Eligibility: domain.ClosureArtifactIneligible}, nil
	}
	resolved, err := decl.Artifact.ForPlatform(r.platform.OS, r.platform.Arch)
	if err != nil {
		r.addUnsupported(decl.ID, ReasonMissingPlatformArtifact, fmt.Sprintf("resource %s has no digest-pinned artifact for %s: %v", decl.ID, platform, err))
		return &domain.ClosureArtifact{Platform: platform, Mode: target.Mode, Eligibility: domain.ClosureArtifactIneligible}, nil
	}
	name := resolved.Path
	if bundled, err := resolved.BundleArtifactForPlatform(r.platform.OS, r.platform.Arch); err == nil {
		name = bundled
	}
	artifact := &domain.ClosureArtifact{Platform: platform, Name: name, Digest: "sha256:" + strings.ToLower(resolved.SHA256), Mode: target.Mode, Eligibility: domain.ClosureArtifactEligible}
	native := &domain.ClosureComponent{
		ID:       decl.ID + ":server",
		Kind:     domain.ClosureKindNativeArtifact,
		Required: true,
		Version:  resolved.Version,
		Artifact: artifact,
		Reasons:  []domain.ClosureReason{{Kind: domain.ClosureReasonPlatformArtifact, From: "resource:" + decl.ID, Detail: target.Mode + " artifact for " + platform}},
	}
	return artifact, native
}

func (r *resolver) addControlPlane(closure *domain.Closure) {
	archs, ok := controlPlanePlatforms[r.platform.OS]
	eligibility := domain.ClosureArtifactBuildable
	if !ok || !containsFold(archs, r.platform.Arch) {
		eligibility = domain.ClosureArtifactIneligible
		r.addUnsupported(controlPlaneComponentID, ReasonUnsupportedControlPlanePlatform, fmt.Sprintf("the native vrooli control plane is not built for %s", r.platform))
	}
	closure.Components = append(closure.Components, domain.ClosureComponent{
		ID:       controlPlaneComponentID,
		Kind:     domain.ClosureKindNativeArtifact,
		Required: true,
		Artifact: &domain.ClosureArtifact{Platform: r.platform.String(), Name: controlPlaneComponentID, Eligibility: eligibility},
		Reasons:  []domain.ClosureReason{{Kind: domain.ClosureReasonPlatformArtifact, From: "control-plane", Detail: "native control plane binary for " + r.platform.String()}},
	})
}

func (r *resolver) capacity(scenarios []*ScenarioDeclaration, resources []*ResourceDeclaration) domain.ClosureCapacity {
	capacity := domain.ClosureCapacity{Fit: "unknown"}
	add := func(component string, requirements *Requirements) {
		if requirements == nil {
			return
		}
		contribution := domain.ClosureCapacityContribution{
			Component:   component,
			CPU:         requirements.CPUCores,
			MemoryBytes: uint64(requirements.RAMMB * mebibyte),
			DiskBytes:   uint64(requirements.DiskMB * mebibyte),
		}
		capacity.CPU += contribution.CPU
		capacity.MemoryBytes += contribution.MemoryBytes
		capacity.DiskBytes += contribution.DiskBytes
		capacity.Contributions = append(capacity.Contributions, contribution)
	}
	for _, scenario := range scenarios {
		add("scenario:"+scenario.ID, scenario.Requirements)
	}
	for _, resource := range resources {
		add("resource:"+resource.ID, resource.Requirements)
	}
	sort.Slice(capacity.Contributions, func(i, j int) bool { return capacity.Contributions[i].Component < capacity.Contributions[j].Component })

	var largest uint64
	if len(r.in.ReleaseArtifactBytes) > 0 {
		for _, size := range r.in.ReleaseArtifactBytes {
			if size > largest {
				largest = size
			}
		}
		capacity.HeadroomBasis = "release_artifacts"
	} else {
		for _, contribution := range capacity.Contributions {
			if contribution.DiskBytes > largest {
				largest = contribution.DiskBytes
			}
		}
		capacity.HeadroomBasis = "largest_declared_disk_footprint"
	}
	capacity.TransientUpdateHeadroomBytes = 2*largest + StagingHeadroomBytes

	if target := r.in.TargetCapacity; target != nil {
		var shortfalls []string
		if target.CPU < capacity.CPU {
			shortfalls = append(shortfalls, fmt.Sprintf("cpu %.2f < %.2f", target.CPU, capacity.CPU))
		}
		if target.MemoryBytes < capacity.MemoryBytes {
			shortfalls = append(shortfalls, fmt.Sprintf("memory %d < %d bytes", target.MemoryBytes, capacity.MemoryBytes))
		}
		if needed := capacity.DiskBytes + capacity.TransientUpdateHeadroomBytes; target.DiskBytes < needed {
			shortfalls = append(shortfalls, fmt.Sprintf("disk %d < %d bytes (including %d bytes transient update headroom)", target.DiskBytes, needed, capacity.TransientUpdateHeadroomBytes))
		}
		if len(shortfalls) == 0 {
			capacity.Fit = "fits"
		} else {
			capacity.Fit = "insufficient"
			r.addUnsupported("capacity", ReasonInsufficientCapacity, strings.Join(shortfalls, "; "))
		}
	}
	return capacity
}

var kindOrder = map[domain.ClosureComponentKind]int{
	domain.ClosureKindScenario:             0,
	domain.ClosureKindResource:             1,
	domain.ClosureKindPackage:              2,
	domain.ClosureKindNativeArtifact:       3,
	domain.ClosureKindTool:                 4,
	domain.ClosureKindSafeguard:            5,
	domain.ClosureKindCredentialDescriptor: 6,
}

func sortComponents(components []domain.ClosureComponent) {
	sort.SliceStable(components, func(i, j int) bool {
		a, b := components[i], components[j]
		if kindOrder[a.Kind] != kindOrder[b.Kind] {
			return kindOrder[a.Kind] < kindOrder[b.Kind]
		}
		return a.ID < b.ID
	})
}

func normalizeReasons(reasons []domain.ClosureReason) []domain.ClosureReason {
	seen := map[domain.ClosureReason]struct{}{}
	out := make([]domain.ClosureReason, 0, len(reasons))
	for _, reason := range reasons {
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		out = append(out, reason)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.From != b.From {
			return a.From < b.From
		}
		return a.Detail < b.Detail
	})
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
