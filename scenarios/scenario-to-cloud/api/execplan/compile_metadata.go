package execplan

import (
	"fmt"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"

	"scenario-to-cloud/domain"
)

// PrivilegeSet is the sorted, comma-joined set of privileges a plan needs:
// every capability of an effectful action plus every closure privilege.
func PrivilegeSet(actions []Action, closure *domain.Closure) string {
	set := map[string]bool{}
	for _, action := range actions {
		if action.Effect != EffectNone && action.RequiredCapability != "" {
			set[action.RequiredCapability] = true
		}
	}
	if closure != nil {
		for _, privilege := range closure.Privileges {
			set["closure:"+privilege.Effect+":"+privilege.Subject] = true
		}
	}
	return strings.Join(sortedKeys(set), ",")
}

func presentationFor(plan *Plan, in CompileInputs) Presentation {
	pres := Presentation{}
	switch plan.Outcome {
	case OutcomeNoOp:
		pres.Title = fmt.Sprintf("%s is already at the desired release", plan.ScenarioID)
		pres.Summary = "The observed release, configuration and closure digests match the desired state; no action will run."
	case OutcomeNeedsInput:
		pres.Title = fmt.Sprintf("%s needs operator input before it can be deployed", plan.ScenarioID)
		pres.Summary = fmt.Sprintf("%d required input(s) are not satisfied; resume through the onboarding handoff.", len(plan.Handoff.Missing))
	default:
		verb := "Deploy"
		switch plan.Scope {
		case ScopeRetire:
			verb = "Retire"
		case ScopeStop:
			verb = "Stop"
		}
		pres.Title = fmt.Sprintf("%s %s (%s) on %s", verb, plan.ScenarioID, plan.Scope, strings.TrimSpace(in.Manifest.Target.VPS.Host))
		pres.Summary = fmt.Sprintf("%d actions: %s", len(plan.Actions), strings.Join(plan.ActionIDs(), ", "))
		if seconds := totalDowntime(plan.Actions); seconds > 0 {
			pres.DowntimeNote = fmt.Sprintf("Strategy %s: expect up to %d seconds of unavailability while the workload restarts.", in.Policy.MaintenanceStrategy, seconds)
		} else if plan.Scope == ScopeRetire {
			pres.DowntimeNote = "The workload stops permanently; shared dependencies other deployments demand keep running."
		} else {
			pres.DowntimeNote = fmt.Sprintf("Strategy %s: no downtime is declared for this plan.", in.Policy.MaintenanceStrategy)
		}
		switch {
		case plan.Scope == ScopeRetire:
			pres.RecoveryNote = fmt.Sprintf("Persistent data disposition: %s. Active, previous and backup artifacts and recovery points are retained.", in.Policy.RetentionPolicy)
		case plan.Action(OpReleaseRetainPredecesor) != nil:
			pres.RecoveryNote = fmt.Sprintf("Predecessor %s is retained; rollback eligibility: %s (%s). Staged trees that were never activated are discarded.", in.Observations.ActiveReleaseDigest, strconv.FormatBool(rollbackEligible(in)), rollbackReason(in))
		default:
			pres.RecoveryNote = "A failed activation or readiness check leaves the prior release active; staged trees that were never activated are discarded."
		}
	}
	return pres
}

func totalDowntime(actions []Action) int {
	total := 0
	for _, action := range actions {
		if action.Downtime != nil {
			total += action.Downtime.ExpectedSeconds
		}
	}
	return total
}

func targetScenarioIDs(manifest domain.CloudManifest) []string {
	ids := sortedUnique(manifest.Bundle.Scenarios)
	// Autoheal is delivered and configured as a host safeguard, but its API is
	// an auxiliary service. A monitor startup failure must not prevent the
	// primary application release from activating on a constrained target.
	ids = slices.DeleteFunc(ids, func(id string) bool { return id == "vrooli-autoheal" })
	if len(ids) == 0 {
		ids = sortedUnique([]string{manifest.Scenario.ID})
	}
	return ids
}

// legacyInventory merges the manifest's preserve paths with the observed
// legacy inventory into one sorted list of scenarios/<id>/<relative> paths.
func legacyInventory(in CompileInputs) []string {
	set := map[string]bool{}
	for _, p := range in.Manifest.Target.VPS.PreservePaths {
		if p = path.Clean(strings.TrimSpace(p)); p != "" && p != "." {
			set[p] = true
		}
	}
	for _, p := range in.Observations.LegacyDataInventory {
		if p = path.Clean(strings.TrimSpace(p)); p != "" && p != "." {
			set[p] = true
		}
	}
	return sortedKeys(set)
}

// legacyCarry is the legacy inventory minus every path a recorded or
// declared binding covers: the heuristic applies only to unmapped paths.
func legacyCarry(in CompileInputs) []string {
	covered := map[string]bool{}
	for _, spec := range dataBindingSpecs(in) {
		if _, location, ok := strings.Cut(spec, "="); ok {
			covered["scenarios/"+location] = true
		}
	}
	var out []string
	for _, p := range legacyInventory(in) {
		skip := false
		for c := range covered {
			if p == c || strings.HasPrefix(p, c+"/") || strings.HasPrefix(c, p+"/") {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, strings.TrimPrefix(p, "scenarios/"))
		}
	}
	sort.Strings(out)
	return out
}

func persistentBindings(in CompileInputs) []string {
	if in.Closure == nil {
		return nil
	}
	set := map[string]bool{}
	for _, data := range in.Closure.PersistentData {
		set[data.ID+"@"+data.Owner] = true
	}
	return sortedKeys(set)
}

// dataBindingSpecs lists filesystem bindings as <id>=<scenario>/<relative>:
// the closure's dir bindings and the mappings recorded by the legacy
// conversion (recorded mappings win for the same id).
func dataBindingSpecs(in CompileInputs) []string {
	set := map[string]string{}
	if in.Closure != nil {
		for _, data := range in.Closure.PersistentData {
			kind, locator, ok := strings.Cut(strings.TrimSpace(data.Binding), ":")
			if !ok || kind != "dir" || strings.TrimSpace(data.Owner) == "" {
				continue
			}
			rel := strings.Trim(path.Clean("/"+locator), "/")
			if rel == "" || strings.Contains(locator, "..") {
				continue
			}
			set[data.ID] = data.Owner + "/" + rel
		}
	}
	for _, spec := range in.Observations.PersistentDataBindings {
		id, location, ok := strings.Cut(strings.TrimSpace(spec), "=")
		if !ok || strings.TrimSpace(id) == "" {
			continue
		}
		set[strings.TrimSpace(id)] = strings.TrimPrefix(strings.TrimSpace(location), "scenarios/")
	}
	out := make([]string, 0, len(set))
	for id, location := range set {
		out = append(out, id+"="+location)
	}
	sort.Strings(out)
	return out
}

// engineBindingSpecs renders the persistent data the recovery point must
// cover as <id>:<kind>:<locator>:<owner>:<migration_owner>: every declared
// binding plus every legacy path the heuristic still carries, so undeclared
// data is captured until it is declared or retired.
func engineBindingSpecs(in CompileInputs) []string {
	var out []string
	if in.Closure != nil {
		for _, data := range in.Closure.PersistentData {
			out = append(out, data.ID+":"+strings.TrimSpace(data.Binding)+":"+strings.TrimSpace(data.Owner)+":"+strings.TrimSpace(data.MigrationOwner))
		}
	}
	for _, p := range legacyInventory(in) {
		scenario, rel, ok := strings.Cut(strings.TrimPrefix(p, "scenarios/"), "/")
		if !ok {
			continue
		}
		id := "legacy-" + scenario + "-" + strings.ReplaceAll(rel, "/", "-")
		out = append(out, id+":dir:"+rel+":"+scenario+":scenario")
	}
	sort.Strings(out)
	return out
}

func migrationPosture(in CompileInputs, update bool) string {
	switch {
	case !update:
		if len(legacyInventory(in)) > 0 {
			return domain.MigrationPostureGreenfieldWithData
		}
		return domain.MigrationPostureGreenfield
	case in.Observations.DataSchemaVersion != "":
		return domain.MigrationPostureProductionEvolution
	default:
		return domain.MigrationPostureGreenfieldWithData
	}
}

// rollbackEligible mirrors backup.EvaluateRollback for the preview: a
// predecessor is eligible when the scenario declares code rollback and the
// schema strategy keeps the current data readable by the predecessor.
func rollbackEligible(in CompileInputs) bool {
	return rollbackReason(in) == "predecessor_retained" || rollbackReason(in) == "no_schema"
}

func rollbackReason(in CompileInputs) string {
	recovery := scenarioRecovery(in)
	switch {
	case in.Observations.ActiveReleaseDigest == "":
		return "no_predecessor"
	case recovery != nil && recovery.CodeRollback == "none":
		return "code_rollback_not_declared"
	case len(persistentBindings(in)) == 0:
		return "no_schema"
	case recovery == nil || recovery.SchemaStrategy == "":
		return "schema_strategy_undeclared"
	case recovery.SchemaStrategy == "explicit_restore":
		return "explicit_restore_required"
	case recovery.SchemaStrategy == "none":
		return "no_schema"
	default:
		return "predecessor_retained"
	}
}

func scenarioRecovery(in CompileInputs) *domain.ClosureRecovery {
	if in.Closure == nil {
		return nil
	}
	for _, component := range in.Closure.ComponentsOfKind(domain.ClosureKindScenario) {
		if component.ID == in.Deployment.ScenarioID || component.ID == "scenario:"+in.Deployment.ScenarioID {
			return component.Recovery
		}
	}
	for _, component := range in.Closure.ComponentsOfKind(domain.ClosureKindScenario) {
		if component.Recovery != nil {
			return component.Recovery
		}
	}
	return nil
}

func credentialDescriptors(manifest domain.CloudManifest) []string {
	if manifest.Secrets == nil {
		return nil
	}
	set := map[string]bool{}
	for _, secret := range manifest.Secrets.BundleSecrets {
		ref := strings.TrimSpace(secret.ID) + ":" + strings.TrimSpace(secret.Class)
		if secret.Descriptor != nil {
			ref += ":" + strings.TrimSpace(secret.Descriptor.LogicalID) + ":" + strings.TrimSpace(secret.Descriptor.Field)
		}
		set[ref] = true
	}
	return sortedKeys(set)
}

func bundleSHA(in CompileInputs) string {
	return strings.TrimPrefix(in.Release.Digest, "sha256:")
}

func releaseDirName(digest string) string {
	name := strings.ReplaceAll(digest, ":", "-")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}

func portList(ports domain.ManifestPorts) string {
	keys := make([]string, 0, len(ports))
	for key := range ports {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, ports[key]))
	}
	return strings.Join(parts, ",")
}

func remoteJoin(elem ...string) string {
	cleaned := make([]string, 0, len(elem))
	for _, e := range elem {
		if e = strings.TrimSpace(e); e != "" {
			cleaned = append(cleaned, e)
		}
	}
	return path.Clean(path.Join(cleaned...))
}

func sortedUnique(in []string) []string {
	set := map[string]bool{}
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			set[v] = true
		}
	}
	return sortedKeys(set)
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
