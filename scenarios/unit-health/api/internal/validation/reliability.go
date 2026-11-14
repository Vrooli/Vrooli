package validation

import (
	"encoding/json"
	"fmt"

	"unit-health/internal/evidence"
	"unit-health/internal/runhistory"
)

// commandComparisonIdentities describes command-scoped observations. We do not
// fabricate per-test identities from a suite exit code. The existing evidence
// owner hashes environment values so persisted selection metadata need not
// disclose secrets. Its normal source/config/toolchain key is reused unchanged.
func commandComparisonIdentities(key evidence.Key, plan ExecutionPlan) []*runhistory.ComparisonIdentity {
	out := make([]*runhistory.ComparisonIdentity, len(plan.Commands))
	for i, command := range plan.Commands {
		envKey, err := evidence.NewKey(evidence.KeyInput{Environment: command.Env})
		if err != nil {
			continue
		}
		selection := command
		selection.Env = nil
		selection.Artifacts = nil
		selection.CaptureStdout = false
		raw, err := json.Marshal(struct {
			Command           PlannedCommand
			EnvironmentDigest string
		}{selection, envKey.Digest})
		if err != nil {
			continue
		}
		identity := &runhistory.ComparisonIdentity{Evidence: key, Selection: string(raw)}
		if identity.Known() {
			out[i] = identity
		}
	}
	return out
}

func reliabilityDiagnostic(workspace string, current CommandResult, history []runhistory.CommandSample) Diagnostic {
	observation := &ReliabilityObservation{State: "unknown", Scope: "command"}
	diagnostic := Diagnostic{Kind: "reliability", WorkspaceID: workspace, Severity: "info", Reliability: observation,
		Message: "Command-scoped reliability is unknown: current comparison identity is unavailable."}
	if !current.ComparisonIdentity.Known() {
		observation.ExcludedIncompatible = len(history)
		diagnostic.Evidence = fmt.Sprintf("command=%q; %d historical outcomes retained without making a cross-run comparison", current.Command, len(history))
		return diagnostic
	}
	identity := current.ComparisonIdentity
	observation.CohortDigest, observation.Seed, observation.RetryOrdinal = identity.CohortDigest(), identity.Seed, identity.RetryOrdinal
	compatible := comparableHistory(identity, history)
	observation.ExcludedIncompatible = len(history) - len(compatible)
	eligible := pastStatuses(compatible)
	observation.ExcludedInfrastructure = len(compatible) - len(eligible)
	if reliabilityOutcome(current.Status, current.FailureClass) {
		eligible = append(eligible, current.Status)
	} else {
		observation.ExcludedInfrastructure++
	}
	observation.SampleCount = len(eligible)
	observation.Passed, observation.Failed = countStatuses(eligible)
	observation.State = "insufficient_samples"
	if len(eligible) >= minFlakeObservs {
		observation.State = "comparable_observations"
	}
	if observation.Passed > 0 && observation.Failed > 0 {
		observation.State = "suspected_instability"
	}
	diagnostic.Message = "Command-scoped observations do not prove a flaky test. Missing or incompatible history is uncomparable; original outcomes remain unchanged."
	diagnostic.Evidence = fmt.Sprintf("command=%q cohort=%s state=%s; %d eligible observations (%d passed / %d test failures), %d infrastructure or unclassified outcomes excluded, %d incompatible historical outcomes excluded", current.Command, observation.CohortDigest, observation.State, observation.SampleCount, observation.Passed, observation.Failed, observation.ExcludedInfrastructure, observation.ExcludedIncompatible)
	return diagnostic
}

func comparableHistory(identity *runhistory.ComparisonIdentity, samples []runhistory.CommandSample) []runhistory.CommandSample {
	var out []runhistory.CommandSample
	for _, sample := range samples {
		if identity.Comparable(sample.Identity) {
			out = append(out, sample)
		}
	}
	return out
}
