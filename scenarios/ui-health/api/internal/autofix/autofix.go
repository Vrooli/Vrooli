// Package autofix registers ui-health's deterministic remediations against the
// shared maturity-go/autofix orchestrator. Every concrete edit is a Fixer here;
// the orchestrator owns preview/apply/idempotency.
//
// The seed fixers remediate the safe mechanical subset of manifest findings:
// `slot_dir_missing` and `slot_parent_dir_missing` are resolved by creating the
// declared-but-absent slot directory (with a `.gitkeep` so git tracks it). This
// is format-preserving and idempotent — the underlying validator re-derives what
// is missing from the same authority that detected it, so once the directory
// exists the finding (and therefore the candidate) disappears. Crucially the
// fixers drive off the *validator's* findings, so the predates-template-layout
// collapse is respected: when most slots are missing the validator emits a single
// `ui_predates_template_layout` summary instead of per-slot warnings, and the
// fixers correctly produce nothing (mass-creating directories is not a safe fix).
//
// Later phases (static interop, net-new standards) attach their own fixers to the
// same registry; this package is the single ui-health auto-fix entrypoint.
package autofix

import (
	"context"
	"fmt"
	"path/filepath"

	"ui-health/internal/services/manifestvalidation"

	autofixcore "github.com/vrooli/maturity-go/autofix"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// Candidate is the shared auto-fix candidate (re-exported for callers).
type Candidate = autofixcore.Candidate

// Finding/rule codes ui-health can auto-remediate. These mirror the codes emitted
// by internal/services/manifestvalidation and declared (fix_class=auto,
// fixer_status=implemented) in .vrooli/maturity.json. Keep the two in lockstep:
// a code listed here MUST be declared auto/implemented in maturity.json, or the
// shared ConsistencyWarnings check flags the runtime AutofixAvailable flag as
// contradicting the declaration.
const (
	RuleSlotDirMissing       = "slot_dir_missing"
	RuleSlotParentDirMissing = "slot_parent_dir_missing"
)

const gitkeepFile = ".gitkeep"

// Validator re-runs UI manifest validation for a scenario root so the fixers can
// derive remediations from the same authority that produced the findings. The
// manifestvalidation.Service satisfies it.
type Validator interface {
	ValidateScenario(ctx context.Context, scenario string) (manifestvalidation.Report, error)
}

// Fixer is ui-health's auto-fix entrypoint: a registry of per-rule fixers bound
// to a validator seam.
type Fixer struct {
	validator Validator
	registry  *autofixcore.Registry
}

// New builds the ui-health auto-fix registry over the given validator. It
// registers the manifest fixers plus the safe-subset interop fixers; this is the
// single ui-health auto-fix entrypoint across every check group.
func New(validator Validator) *Fixer {
	f := &Fixer{validator: validator}
	fixers := []autofixcore.Fixer{
		{RuleID: RuleSlotDirMissing, Preview: f.previewMissingDir(RuleSlotDirMissing), CanFix: f.canFix(RuleSlotDirMissing)},
		{RuleID: RuleSlotParentDirMissing, Preview: f.previewMissingDir(RuleSlotParentDirMissing), CanFix: f.canFix(RuleSlotParentDirMissing)},
	}
	rewriteSpecs := append(f.interopFixers(), f.standardRewriteFixers()...)
	for _, spec := range rewriteSpecs {
		spec := spec
		fixers = append(fixers, autofixcore.Fixer{
			RuleID:  spec.ruleID,
			Preview: f.previewInterop(spec),
			CanFix:  f.canFixInterop(spec),
		})
	}
	// The net-new i18n locale-parity fixer needs a sibling catalog (en.json), so
	// it cannot use the single-file rewrite machinery — register its bespoke
	// preview/canFix directly.
	fixers = append(fixers, autofixcore.Fixer{
		RuleID:  RuleStandardI18nLocaleParity,
		Preview: f.previewI18nLocaleParity,
		CanFix:  f.canFixI18nLocaleParity,
	})
	f.registry = autofixcore.NewRegistry(fixers...)
	return f
}

// FixClassFor returns the fix classification for a finding code: autofix when a
// fixer is registered for it, detection_only otherwise. This is the runtime
// counterpart of the maturity.json declaration.
func FixClassFor(code string) autofixcore.FixClass {
	switch code {
	case RuleSlotDirMissing, RuleSlotParentDirMissing,
		RuleInteropHScreen, RuleInteropProtectiveComments,
		RuleStandardTSConfigStrict, RuleStandardI18nLocaleParity:
		return autofixcore.FixClassAutofix
	default:
		return autofixcore.FixClassDetectionOnly
	}
}

// CanFix reports whether the named rule can currently remediate a finding located
// at findingPath. It re-checks live state (the directory is still missing) so the
// runtime AutofixAvailable flag never claims a fix that would no-op.
func (f *Fixer) CanFix(root, ruleID, findingPath string) bool {
	return f.registry.CanFix(root, ruleID, findingPath)
}

// PreviewFixResponse previews remediations for the resolved scenario root.
func (f *Fixer) PreviewFixResponse(scenario, root string, ruleIDs []string) (*scenariovalidationv1.FixResponse, error) {
	return f.registry.PreviewFixResponse(scenario, root, ruleIDs)
}

// ApplyFixResponse applies remediations for the resolved scenario root.
func (f *Fixer) ApplyFixResponse(scenario, root string, ruleIDs []string) (*scenariovalidationv1.FixResponse, error) {
	return f.registry.ApplyFixResponse(scenario, root, ruleIDs)
}

// previewMissingDir returns a preview func that turns each finding of the given
// code (whose Location is the absolute path of the absent directory) into a
// "create directory + .gitkeep" candidate. Driving off the validator means the
// predates-template collapse is honored automatically.
func (f *Fixer) previewMissingDir(code string) func(root string) ([]Candidate, error) {
	return func(root string) ([]Candidate, error) {
		report, err := f.validate(root)
		if err != nil {
			// A validation failure is treated as "nothing to fix" rather than a
			// hard error so a fix run never crashes on an unreadable scenario.
			return nil, nil
		}
		seen := map[string]bool{}
		var out []Candidate
		for _, finding := range report.Findings {
			if finding.Code != code {
				continue
			}
			dir := finding.Location
			if !filepath.IsAbs(dir) {
				continue
			}
			keep := filepath.Join(dir, gitkeepFile)
			if seen[keep] {
				continue
			}
			seen[keep] = true
			out = append(out, Candidate{
				RuleID:      code,
				FilePath:    keep,
				Description: fmt.Sprintf("Create the declared slot directory %s (was missing on disk).", dir),
				Before:      "",
				After:       "",
			})
		}
		return out, nil
	}
}

// canFix adapts a preview func into a CanFix predicate scoped to a single finding
// path: the rule can fix the finding when its preview would create the directory
// the finding points at.
func (f *Fixer) canFix(code string) func(root, findingPath string) bool {
	preview := f.previewMissingDir(code)
	return func(root, findingPath string) bool {
		candidates, err := preview(root)
		if err != nil || len(candidates) == 0 {
			return false
		}
		if findingPath == "" {
			return true
		}
		want := filepath.Join(findingPath, gitkeepFile)
		for _, c := range candidates {
			if c.FilePath == want {
				return true
			}
		}
		return false
	}
}

// validate runs the injected validator against the scenario rooted at root,
// threading the explicit path so scenarios outside the repo scenarios/ tree
// (e.g. test fixtures) validate correctly.
func (f *Fixer) validate(root string) (manifestvalidation.Report, error) {
	ctx := manifestvalidation.WithScenarioPath(context.Background(), root)
	return f.validator.ValidateScenario(ctx, filepath.Base(root))
}
