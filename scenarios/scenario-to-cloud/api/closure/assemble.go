package closure

import (
	"fmt"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
	"scenario-to-cloud/domain"
	"sort"
	"strings"
)

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
			DependencyDigest: r.dependencyDigest,
			AnalyzerTool:     r.analyzer.tool,
			AnalyzerUsed:     r.analyzer.used,
		},
	}

	credentials := map[string]*domain.ClosureComponent{}
	credentialOrder := []string{}
	includedScenarios := r.appendScenarioComponents(&closure, credentials, &credentialOrder)
	includedResources := r.appendResourceComponents(&closure, credentials, &credentialOrder)

	r.addControlPlane(&closure)
	r.appendBundleRoots(&closure)
	if err := r.appendHostRequirements(&closure, includedScenarios, includedResources); err != nil {
		return domain.Closure{}, err
	}
	appendCredentialComponents(&closure, credentials, credentialOrder)

	closure.Capacity = r.capacity(includedScenarios, includedResources)
	if err := r.finalizeClosure(&closure); err != nil {
		return domain.Closure{}, err
	}
	return closure, nil
}

func (r *resolver) appendBundleRoots(closure *domain.Closure) {
	if r.in.Scope != ScopeBundle {
		return
	}
	for _, root := range []string{"internal", "packages"} {
		closure.Components = append(closure.Components, domain.ClosureComponent{
			ID: root, Kind: domain.ClosureKindPackage, Required: true,
			Reasons: []domain.ClosureReason{{Kind: domain.ClosureReasonDeclaredBy, From: "repo-contract:mini_vrooli_bundle", Detail: "the bundle profile ships the full shared " + root + " tree; it is not minimised per scenario"}},
		})
	}
}

func (r *resolver) appendHostRequirements(closure *domain.Closure, scenarios []*ScenarioDeclaration, resources []*ResourceDeclaration) error {
	requirements, err := r.in.HostRequirements.Resolve(r.ctx, HostRequirementsQuery{RepoRoot: r.in.RepoRoot, Environment: r.in.Environment, Platform: r.platform, Scenarios: scenarios, Resources: resources})
	if err != nil {
		return unavailable("host requirements could not be resolved", err, map[string]any{"source": r.in.HostRequirements.Name()})
	}
	for _, requirement := range requirements {
		appendHostRequirement(closure, requirement)
	}
	return nil
}

func appendHostRequirement(closure *domain.Closure, requirement HostRequirement) {
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

func appendCredentialComponents(closure *domain.Closure, credentials map[string]*domain.ClosureComponent, order []string) {
	for _, address := range order {
		component := credentials[address]
		component.Reasons = normalizeReasons(component.Reasons)
		closure.Components = append(closure.Components, *component)
	}
}

func (r *resolver) finalizeClosure(closure *domain.Closure) error {
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
	ensureClosureSlices(closure)
	digest, err := Digest(*closure)
	if err != nil {
		return fmt.Errorf("digest closure: %w", err)
	}
	closure.Digest = digest
	return nil
}

func ensureClosureSlices(closure *domain.Closure) {
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
}

func addCredentialDescriptor(credentials map[string]*domain.ClosureComponent, order *[]string, owner string, descriptor CredentialDescriptor) {
	address := descriptor.Address()
	component, ok := credentials[address]
	if !ok {
		field := descriptor.Field
		if field == "" {
			field = "value"
		}
		component = &domain.ClosureComponent{
			ID: address, Kind: domain.ClosureKindCredentialDescriptor,
			Credential: &domain.ClosureCredential{LogicalID: descriptor.LogicalID, Field: field, Env: descriptor.Env, Required: descriptor.Required, Label: descriptor.Label},
		}
		credentials[address] = component
		*order = append(*order, address)
	}
	component.Required = component.Required || descriptor.Required
	component.Credential.Required = component.Required
	component.Reasons = append(component.Reasons, domain.ClosureReason{Kind: domain.ClosureReasonCredentialOf, From: owner, Detail: "declared in credentials.descriptors"})
}

func (r *resolver) appendScenarioComponents(closure *domain.Closure, credentials map[string]*domain.ClosureComponent, credentialOrder *[]string) []*ScenarioDeclaration {
	var includedScenarios []*ScenarioDeclaration
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
			IncludeUI:        node.decl.ID == r.in.ScenarioID || node.includeUI,
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
			addCredentialDescriptor(credentials, credentialOrder, "scenario:"+id, descriptor)
		}
		r.collectPersistentData(closure, "scenario:"+id, id, node.decl.Deployment)
		r.collectScenarioListeners(closure, node.decl)
	}
	return includedScenarios
}

func (r *resolver) appendResourceComponents(closure *domain.Closure, credentials map[string]*domain.ClosureComponent, credentialOrder *[]string) []*ResourceDeclaration {
	var includedResources []*ResourceDeclaration
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
			addCredentialDescriptor(credentials, credentialOrder, "resource:"+id, descriptor)
		}
		r.collectPersistentData(closure, "resource:"+id, id, node.decl.Deployment)
		for _, port := range node.decl.Ports {
			closure.Listeners = append(closure.Listeners, domain.ClosureListener{ID: id + "/" + port, Owner: "resource:" + id, PortName: port, Visibility: listenerVisibility(node.decl.Deployment, port)})
		}
	}
	return includedResources
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
		closure.Listeners = append(closure.Listeners, scenarioListener(closure.ScenarioID, decl, name))
	}
}

func scenarioListener(rootScenarioID string, decl *ScenarioDeclaration, name string) domain.ClosureListener {
	visibility := listenerVisibility(decl.Deployment, name)
	if decl.ID == rootScenarioID && len(decl.Deployment.Listeners) == 0 && name == "ui" {
		// A deployed scenario that declares no listeners keeps the launch
		// contract it shipped under: its UI port is the one public route
		// behind the edge; every other port stays private until declared.
		visibility = "public_via_edge"
	}
	listener := domain.ClosureListener{ID: decl.ID + "/" + name, Owner: "scenario:" + decl.ID, PortName: name, Visibility: visibility}
	readiness, pathPrefix := scenarioListenerReadiness(decl, name)
	listener.PathPrefix = pathPrefix
	if readiness != nil {
		listener.Readiness = &domain.ClosureReadiness{Type: readiness.Type, Path: readiness.Path, TimeoutMS: readiness.TimeoutMS}
	}
	return listener
}

func scenarioListenerReadiness(decl *ScenarioDeclaration, name string) (*ReadinessDeclaration, string) {
	var readiness *ReadinessDeclaration
	var pathPrefix string
	for _, declared := range decl.Deployment.Listeners {
		if declared.Port != name {
			continue
		}
		if declared.Readiness != nil {
			readiness = declared.Readiness
		}
		pathPrefix = declared.PathPrefix
	}
	if readiness == nil {
		if declared, ok := decl.ComponentReadiness[name]; ok {
			readiness = &declared
		}
	}
	return readiness, pathPrefix
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
