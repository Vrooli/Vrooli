package execplan

import (
	"strconv"
	"strings"
)

// retireActions is the terminal lifecycle in owner order: routing, runtime,
// grants, data, artifacts. Data and artifacts are retained unless the policy
// says otherwise; an irreversible data disposition requires the explicit
// policy validated in validateInputs.
func retireActions(in CompileInputs) []Action {
	lay := layoutFor(in)
	bindings := persistentBindings(in)
	legacy := legacyInventory(in)
	actions := []Action{
		{
			ID:                 OpEdgeRouteRetire,
			OwnerOperation:     OpEdgeRouteRetire,
			Effect:             EffectEdgeWrite,
			RequiredCapability: "cloud-target:edge.route-rollback",
			Inputs:             map[string]string{"domain": strings.TrimSpace(in.Manifest.Edge.Domain)},
			DependsOn:          []string{},
			Verification:       "route_absent",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		},
		{
			ID:                 OpWorkloadStop,
			OwnerOperation:     OpWorkloadStop,
			Effect:             EffectRuntimeStop,
			RequiredCapability: "privilegebroker:process.stop.scoped",
			Inputs:             map[string]string{"workdir": lay.workdir, "scenario": in.Manifest.Scenario.ID, "ports": portList(in.Manifest.Ports)},
			DependsOn:          []string{OpEdgeRouteRetire},
			Verification:       "scenario_stopped_shared_demand_kept",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		},
		{
			ID:                 OpGrantsRevoke,
			OwnerOperation:     OpGrantsRevoke,
			Effect:             EffectCredentialWrite,
			RequiredCapability: "credential-authority:revoke",
			Inputs:             map[string]string{"scenario": in.Manifest.Scenario.ID, "descriptors": strings.Join(credentialDescriptors(in.Manifest), ",")},
			DependsOn:          []string{OpWorkloadStop},
			Verification:       "grants_revoked",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		},
		{
			ID:                 OpDataRetire,
			OwnerOperation:     OpDataRetire,
			Effect:             dataRetireEffect(in),
			RequiredCapability: "cloud-target:data.retire",
			Inputs: map[string]string{
				"bindings":         strings.Join(bindings, ","),
				"legacy_preserve":  strings.Join(legacy, ","),
				"retention_policy": in.Policy.RetentionPolicy,
			},
			DependsOn:    []string{OpGrantsRevoke},
			Verification: "data_disposition_recorded",
			Recovery:     "none_required",
			Retry:        RetryObserveThenReplay,
			CancelPoint:  true,
		},
		{
			ID:                 OpArtifactsRetire,
			OwnerOperation:     OpArtifactsRetire,
			Effect:             EffectDeploymentWrite,
			RequiredCapability: "cloud-target:release.list",
			Inputs:             map[string]string{"release_id": in.ReleaseArtifacts.ID, "protected": "active,previous,backup"},
			DependsOn:          []string{OpDataRetire},
			Verification:       "protected_artifacts_retained",
			Recovery:           "none_required",
			Retry:              RetrySafeReplay,
			CancelPoint:        true,
		},
	}
	return actions
}

// stopActions is the intentional stop of the workload through the lifecycle
// owner. Nothing else changes: routes, data and artifacts stay.
func stopActions(in CompileInputs) []Action {
	lay := layoutFor(in)
	return []Action{{
		ID:                 OpWorkloadStop,
		OwnerOperation:     OpWorkloadStop,
		Effect:             EffectRuntimeStop,
		RequiredCapability: "privilegebroker:process.stop.scoped",
		Inputs:             map[string]string{"workdir": lay.workdir, "scenario": in.Manifest.Scenario.ID, "ports": portList(in.Manifest.Ports)},
		DependsOn:          []string{},
		Verification:       "scenario_stopped_shared_demand_kept",
		Recovery:           "none_required",
		Retry:              RetrySafeReplay,
		CancelPoint:        true,
	}}
}

func dataRetireEffect(in CompileInputs) string {
	if in.Policy.RetentionPolicy == RetentionDelete {
		return EffectDataWrite
	}
	return EffectNone
}

// preconditions records the material facts the plan binds to.
func preconditions(plan *Plan, in CompileInputs) []Precondition {
	pre := []Precondition{
		{Kind: PreconditionDeploymentRevision, Value: strconv.FormatUint(in.Observations.DeploymentRevision, 10)},
		{Kind: PreconditionTargetEnrollment, Value: strconv.FormatUint(in.Deployment.Target.EnrollmentGeneration, 10)},
		{Kind: PreconditionReleaseDigest, Value: plan.ReleaseDigest},
		{Kind: PreconditionClosureDigest, Value: plan.ClosureDigest},
		{Kind: PreconditionConfigurationDigest, Value: plan.ConfigurationDigest},
		{Kind: PreconditionDataSchema, Value: in.Observations.DataSchemaVersion},
		{Kind: PreconditionPrivilegeSet, Value: PrivilegeSet(plan.Actions, in.Closure)},
	}
	if backup := plan.Action(OpDataBackup); backup != nil && backup.Inputs["recovery_point_required"] != "" {
		pre = append(pre, Precondition{Kind: PreconditionRecoveryPoint, Value: backup.Inputs["recovery_point_required"]})
	}
	return pre
}
