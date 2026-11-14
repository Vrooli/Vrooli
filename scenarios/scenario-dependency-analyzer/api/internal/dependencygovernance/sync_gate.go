package dependencygovernance

import (
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/scenarios/scenario-dependency-analyzer/api/internal/installgateway"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

// SyncRequirement is one third-party Go requirement that lockfile
// synchronization would add or move in a consumer go.mod. Direct is the
// requirement's state after synchronization: a direct requirement means the
// consumer's own source imports the module, an indirect one means it reaches
// the module only through another (usually in-repo) module's requirements.
type SyncRequirement struct {
	Path    string
	Version string
	Direct  bool
}

// SyncRefusal names a requirement that synchronization may not record on its
// own, with the governance verdict and the exact next steps.
type SyncRefusal struct {
	Path      string
	Version   string
	Verdict   string
	NextSteps []string
}

func (r SyncRefusal) String() string {
	text := fmt.Sprintf("go/%s@%s (%s)", r.Path, r.Version, r.Verdict)
	if len(r.NextSteps) > 0 {
		text += ": " + strings.Join(r.NextSteps, " ")
	}
	return text
}

// GateRequirementSync decides whether go.mod synchronization may record the
// given requirements for a consumer module without a new dependency decision.
//
// A direct requirement is a dependency decision by the consumer itself, so it
// passes the same fail-closed gate as `deps install`: it must be recorded,
// approved for the scenario and surface, and in range. An indirect requirement
// was chosen upstream by the module that requires it directly, and its version
// is fixed by minimal version selection over requirements already in the repo.
// Recording it is lockfile synchronization; validation treats unrecorded
// indirect Go requirements as transitive for the same reason. A recorded
// decision still binds it: a denial refuses it, and a recorded approval range
// refuses any version outside that range, so synchronization never selects a
// version the registry has not approved.
func (r *Registry) GateRequirementSync(scenario, surface string, requirements []SyncRequirement) ([]SyncRefusal, error) {
	var refusals []SyncRefusal
	for _, req := range requirements {
		if req.Direct {
			verdict, blocked, steps, _ := r.governInstall(&governancev1.InstallDependencyRequest{
				Scenario:    scenario,
				Surface:     surface,
				Ecosystem:   "go",
				PackageName: req.Path,
				Version:     req.Version,
			}, installgateway.Resolution{})
			if verdict == "registry_error" {
				return nil, fmt.Errorf("approved dependency registry unreadable: %s", strings.Join(steps, " "))
			}
			if blocked {
				refusals = append(refusals, SyncRefusal{Path: req.Path, Version: req.Version, Verdict: verdict, NextSteps: steps})
			}
			continue
		}
		record, found, err := r.Explain("go", req.Path)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		if recordDeniesVersion(record, req.Version) {
			refusals = append(refusals, SyncRefusal{
				Path:    req.Path,
				Version: req.Version,
				Verdict: "denied",
				NextSteps: []string{firstNonEmpty(record.GetReplacement(),
					"An in-repo module this consumer depends on requires a denied package; remove it from that module or record a reviewed exception.")},
			})
			continue
		}
		if !approvedState(record) {
			continue
		}
		if decision := evaluateVersionPolicy("go", req.Version, record.GetVersionRange(), record.GetRangePolicy()); !decision.Allowed {
			verdict, steps := "out_of_range", []string{fmt.Sprintf("Version %s is outside the approved range %s. A module this consumer depends on selects it; raise it to an approved version in the consumer (or the module that requires it), or widen the range if that is intended: %s deps approved widen-range go/%s --to-major-line --apply", req.Version, record.GetVersionRange(), scenarioCLIName(), req.Path)}
			if decision.FindingClass == "VERSION_RANGE_UNPARSEABLE" {
				verdict, steps = "range_unparseable", []string{decision.Description, decision.Remediation}
			}
			refusals = append(refusals, SyncRefusal{Path: req.Path, Version: req.Version, Verdict: verdict, NextSteps: steps})
		}
	}
	return refusals, nil
}

// recordDeniesVersion mirrors ValidateObserved's DENIED_IN_USE rule: a plain
// denial covers every version, a security denial only its recorded range.
func recordDeniesVersion(record *governancev1.ApprovedDependencyRecord, version string) bool {
	switch normalize(record.GetState()) {
	case "denied", "blocked":
	default:
		return false
	}
	if normalize(record.GetRangePolicy()) != "security_denied" {
		return true
	}
	return evaluateVersionPolicy("go", version, record.GetVersionRange(), record.GetRangePolicy()).Allowed
}

// approvedState reports whether a record carries an approval whose range binds
// the version (the states ValidateObserved range-checks).
func approvedState(record *governancev1.ApprovedDependencyRecord) bool {
	switch normalize(record.GetState()) {
	case "approved", "approved_with_constraints", "needs_review", "exception", "":
		return true
	}
	return false
}
