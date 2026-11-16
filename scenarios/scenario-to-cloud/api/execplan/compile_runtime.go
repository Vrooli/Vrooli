package execplan

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"sort"
	"strconv"
	"strings"

	"scenario-to-cloud/closure"
	"scenario-to-cloud/domain"
)

// selectionJSON is the one cloud-side projection of the shared setup/v1
// contract. The target action still owns application, but the executable plan
// carries the exact closure-derived selection it reviewed so onboarding and
// setup cannot silently reconstruct a different workload union.
func selectionJSON(in CompileInputs) string {
	if in.Closure == nil {
		return ""
	}
	target := in.Deployment.Target.NodeID
	if target == "" {
		target = in.Deployment.Target.MachineID
	}
	selection := closure.ToSelection(*in.Closure, target, closure.Overrides{})
	encoded, err := json.Marshal(selection)
	if err != nil {
		return ""
	}
	// Action inputs are argv-shaped metadata and must remain shell-free. The
	// setup owner decodes this URL-safe payload at the onboarding handoff.
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func runtimeActions(in CompileInputs, startOnly bool) []Action {
	lay := layoutFor(in)
	uiPort := in.Manifest.Ports["ui"]
	routeHost, routePort, routeListener := deriveEdgeRoute(in, uiPort)
	edgeSpec := deriveEdgeSpec(in, uiPort)
	ports := portList(in.Manifest.Ports)
	targetScenarios := targetScenarioIDs(in.Manifest)
	hasSecrets := in.Manifest.Secrets != nil && len(in.Manifest.Secrets.BundleSecrets) > 0
	bindings := persistentBindings(in)
	legacy := legacyInventory(in)
	update := in.Observations.ActiveReleaseDigest != "" && in.Observations.ActiveReleaseDigest != in.Release.Digest

	var actions []Action
	if hasSecrets {
		actions = append(actions, Action{
			ID:                 OpCredentialsProvision,
			OwnerOperation:     OpCredentialsProvision,
			Effect:             EffectCredentialWrite,
			RequiredCapability: "credential-authority:provision",
			Inputs: map[string]string{
				"workdir":     lay.workdir,
				"scenario":    in.Manifest.Scenario.ID,
				"descriptors": strings.Join(credentialDescriptors(in.Manifest), ","),
			},
			DependsOn:    []string{},
			Verification: "descriptors_resolvable",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		})
	}
	depScenarios := []string{}
	for _, scen := range runtimeScenarioOrder(in) {
		if scen != in.Manifest.Scenario.ID {
			depScenarios = append(depScenarios, scen)
		}
	}
	resources := runtimeResourceIDs(in)
	hasDeps := len(resources) > 0 || len(depScenarios) > 0
	if hasDeps {
		optionalScenarios := runtimeOptionalIDs(in, domain.ClosureKindScenario)
		optionalResources := runtimeOptionalIDs(in, domain.ClosureKindResource)
		actions = append(actions, Action{
			ID:                 OpRuntimeStartDeps,
			OwnerOperation:     OpRuntimeStartDeps,
			Effect:             EffectRuntimeStart,
			RequiredCapability: "vrooli:resource.start,scenario.start",
			Inputs: map[string]string{
				"workdir":            lay.workdir,
				"resources":          strings.Join(resources, ","),
				"scenarios":          strings.Join(depScenarios, ","),
				"optional_resources": strings.Join(optionalResources, ","),
				"optional_scenarios": strings.Join(optionalScenarios, ","),
			},
			DependsOn:    []string{OpCredentialsProvision},
			Verification: "dependencies_running",
			Recovery:     "restart_dependency",
			Retry:        RetryObserveThenReplay,
			CancelPoint:  true,
		})
	}
	if startOnly {
		actions = append(actions,
			Action{
				ID:                 OpWorkloadStart,
				OwnerOperation:     OpWorkloadStart,
				Effect:             EffectRuntimeStart,
				RequiredCapability: "cloud-target:release.activate",
				Inputs: map[string]string{
					"workdir":       lay.workdir,
					"scenario":      in.Manifest.Scenario.ID,
					"scenarios":     strings.Join(targetScenarios, ","),
					"ports":         ports,
					"ui_port":       strconv.Itoa(uiPort),
					"release_id":    in.ReleaseArtifacts.ID,
					"data_bindings": strings.Join(dataBindingSpecs(in), ","),
				},
				DependsOn:    []string{OpCredentialsProvision, OpRuntimeStartDeps},
				Verification: "runtime_active_release_running",
				Recovery:     "rollback_to_predecessor",
				Retry:        RetryObserveThenReplay,
				CancelPoint:  false,
			},
			readinessAction(in, []string{OpWorkloadStart}),
		)
		return actions
	}
	activateDeps := []string{OpCredentialsProvision, OpRuntimeStartDeps}
	// A first install has no prior deployment data to preserve. Requiring a
	// database-native backup client for that case makes a minimal target
	// impossible to bootstrap, even when its database is empty. Updates and
	// greenfield-with-data still take the backup before any destructive step.
	needsBackup := migrationPosture(in, update) != domain.MigrationPostureGreenfield
	if (len(bindings) > 0 || len(legacy) > 0) && needsBackup {
		verification := "recovery_point_recorded"
		precondition := ""
		if update && in.Observations.DataSchemaVersion != "" {
			precondition = in.Release.Digest + "@" + in.Observations.DataSchemaVersion
		}
		actions = append(actions, Action{
			ID:                 OpDataBackup,
			OwnerOperation:     OpDataBackup,
			Effect:             EffectDataWrite,
			RequiredCapability: "cloud-target:data.backup",
			Inputs: map[string]string{
				"bindings":                strings.Join(bindings, ","),
				"binding_specs":           strings.Join(engineBindingSpecs(in), ";"),
				"legacy_preserve":         strings.Join(legacy, ","),
				"recovery_key_ref":        in.Policy.RecoveryKeyRef,
				"schema_version":          in.Observations.DataSchemaVersion,
				"configuration_digest":    in.Release.ConfigurationDigest,
				"release_digest":          in.Release.Digest,
				"migration_posture":       migrationPosture(in, update),
				"retention_policy":        "retain_until_superseded",
				"recovery_point_required": precondition,
			},
			DependsOn:    []string{OpRuntimeStartDeps, OpCredentialsProvision},
			Verification: verification,
			Recovery:     "none_required",
			Retry:        RetryObserveThenReplay,
			CancelPoint:  true,
		})
		activateDeps = append(activateDeps, OpDataBackup)
	}
	if in.Policy.MaintenanceStrategy == StrategyMaintenance {
		actions = append(actions, Action{
			ID:                 OpWorkloadStop,
			OwnerOperation:     OpWorkloadStop,
			Effect:             EffectRuntimeStop,
			RequiredCapability: "privilegebroker:process.stop.scoped",
			Inputs: map[string]string{
				"workdir":  lay.workdir,
				"scenario": in.Manifest.Scenario.ID,
				"ports":    ports,
			},
			DependsOn:    []string{OpDataBackup, OpRuntimeStartDeps, OpCredentialsProvision},
			Verification: "scenario_stopped_shared_demand_kept",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
			Downtime:     &Downtime{ExpectedSeconds: in.Policy.MaintenanceDowntimeSeconds, Reason: "maintenance strategy stops the workload before the new release starts"},
		})
		activateDeps = append(activateDeps, OpWorkloadStop)
	}
	activate := Action{
		ID:                 OpReleaseActivate,
		OwnerOperation:     OpReleaseActivate,
		Effect:             EffectDeploymentWrite,
		RequiredCapability: "cloud-target:release.activate",
		Inputs: map[string]string{
			"release_digest":     in.Release.Digest,
			"release_id":         in.ReleaseArtifacts.ID,
			"release_dir":        lay.releaseDir,
			"scenarios":          strings.Join(targetScenarios, ","),
			"strategy":           in.Policy.MaintenanceStrategy,
			"ports":              ports,
			"data_bindings":      strings.Join(dataBindingSpecs(in), ","),
			"legacy_carry":       strings.Join(legacyCarry(in), ","),
			"legacy_root":        lay.workdir,
			"predecessor_digest": in.Observations.ActiveReleaseDigest,
		},
		DependsOn:    activateDeps,
		Verification: "runtime_ready_then_pointer_committed",
		Recovery:     "rollback_to_predecessor",
		Retry:        RetryObserveThenReplay,
		CancelPoint:  false,
	}
	actions = append(actions, activate)
	edgeDeps := []string{OpReleaseActivate}
	if profile := strings.TrimSpace(in.Manifest.Edge.ManagedDNSProfile); profile != "" {
		recordType := "CNAME"
		if net.ParseIP(strings.TrimSpace(in.Manifest.Target.VPS.Host)) != nil {
			recordType = "A"
		}
		actions = append(actions, Action{
			ID:                 OpEdgeDNSEnsure,
			OwnerOperation:     OpEdgeDNSEnsure,
			Effect:             EffectEdgeWrite,
			RequiredCapability: "tunnel-manager:dns.ensure",
			Inputs: map[string]string{
				"provider_profile": profile,
				"hostname":         strings.TrimSpace(in.Manifest.Edge.Domain),
				"record_type":      recordType,
				"content":          strings.TrimSpace(in.Manifest.Target.VPS.Host),
				"ttl":              "300",
				"proxied":          "true",
				"owner":            in.Deployment.ID,
			},
			DependsOn:    []string{OpReleaseActivate},
			Verification: "managed_dns_record_present",
			Recovery:     "reconcile_managed_dns_record",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		})
		edgeDeps = append(edgeDeps, OpEdgeDNSEnsure)
	}
	if in.Manifest.Edge.Caddy.Enabled || strings.TrimSpace(in.Manifest.Edge.Domain) != "" {
		actions = append(actions, Action{
			ID:                 OpEdgeRouteApply,
			OwnerOperation:     OpEdgeRouteApply,
			Effect:             EffectEdgeWrite,
			RequiredCapability: "cloud-target:edge.route-apply",
			Inputs: map[string]string{
				"domain":        strings.TrimSpace(in.Manifest.Edge.Domain),
				"route_host":    routeHost,
				"listener_id":   routeListener,
				"upstream_port": strconv.Itoa(routePort),
				"tls_email":     strings.TrimSpace(in.Manifest.Edge.Caddy.Email),
				"tls_enabled":   strconv.FormatBool(in.Manifest.Edge.Caddy.Enabled),
				"config_path":   "/etc/caddy/Caddyfile",
				"release_id":    in.ReleaseArtifacts.ID,
				"edge_spec":     encodeEdgeSpec(edgeSpec),
			},
			DependsOn:    edgeDeps,
			Verification: "caddy_config_validates",
			Recovery:     "restore_previous_caddyfile",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		})
	}
	actions = append(actions, readinessAction(in, []string{OpReleaseActivate, OpEdgeRouteApply}))
	if update {
		actions = append(actions, Action{
			ID:                 OpReleaseRetainPredecesor,
			OwnerOperation:     OpReleaseRetainPredecesor,
			Effect:             EffectNone,
			RequiredCapability: "cloud-target:release.list",
			Inputs: map[string]string{
				"predecessor_digest": in.Observations.ActiveReleaseDigest,
				"release_id":         in.ReleaseArtifacts.ID,
				"rollback_eligible":  strconv.FormatBool(rollbackEligible(in)),
				"rollback_reason":    rollbackReason(in),
			},
			DependsOn:    []string{OpVerifyReadiness},
			Verification: "predecessor_retained",
			Recovery:     "none_required",
			Retry:        RetrySafeReplay,
			CancelPoint:  true,
		})
	}
	return actions
}

func runtimeOptionalIDs(in CompileInputs, kind domain.ClosureComponentKind) []string {
	if in.Closure == nil {
		return nil
	}
	var ids []string
	for _, component := range in.Closure.Components {
		if component.Kind == kind && !component.Required && component.OptionalSelected {
			ids = append(ids, component.ID)
		}
	}
	return sortedUnique(ids)
}

// runtimeScenarioOrder uses the reviewed closure when available so optional
// edge selections made during closure resolution govern activation as well as
// bundle construction. Dependencies are emitted before their consumers.
func runtimeScenarioOrder(in CompileInputs) []string {
	if in.Closure == nil {
		return sortedUnique(in.Manifest.Dependencies.Scenarios)
	}
	children := map[string][]string{}
	ids := map[string]bool{}
	for _, component := range in.Closure.Components {
		if component.Kind != domain.ClosureKindScenario {
			continue
		}
		ids[component.ID] = true
		for _, reason := range component.Reasons {
			if reason.Kind == domain.ClosureReasonDeclaredBy && reason.From != "" && reason.From != component.ID {
				children[reason.From] = append(children[reason.From], component.ID)
			}
		}
	}
	ids[in.Closure.ScenarioID] = true
	ordered, visiting, visited := []string{}, map[string]bool{}, map[string]bool{}
	var visit func(string)
	visit = func(id string) {
		if visited[id] || visiting[id] {
			return
		}
		visiting[id] = true
		dependencies := append([]string(nil), children[id]...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if ids[dependency] {
				visit(dependency)
			}
		}
		visiting[id] = false
		visited[id] = true
		ordered = append(ordered, id)
	}
	visit(in.Closure.ScenarioID)
	for _, id := range sortedKeys(ids) {
		visit(id)
	}
	return ordered
}

func runtimeResourceIDs(in CompileInputs) []string {
	if in.Closure == nil {
		return sortedUnique(in.Manifest.Dependencies.Resources)
	}
	var resources []string
	for _, component := range in.Closure.Components {
		if component.Kind == domain.ClosureKindResource {
			resources = append(resources, component.ID)
		}
	}
	return sortedUnique(resources)
}
