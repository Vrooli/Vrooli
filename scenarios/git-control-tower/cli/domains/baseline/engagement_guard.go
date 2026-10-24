package baseline

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// openEngagementsFor returns every floor manifest recorded for scenario, in any
// mode and whether or not its idle TTL has lapsed. An expired manifest is still
// open: the lifecycle resolver keeps routing live to its restore point until
// promote, abandon or gc removes it.
func openEngagementsFor(ctx context.Context, scenario string) ([]engagementView, error) {
	all, err := listEngagements(ctx)
	if err != nil {
		return nil, err
	}
	var out []engagementView
	for _, e := range all {
		if e.Scenario == scenario {
			out = append(out, e)
		}
	}
	return out, nil
}

// guardOpenEngagement is the O15 guard `baseline start` runs before it captures
// or copies anything. The restore point of an open engagement is the frozen
// Baseline live serves from, so a start never re-captures over it:
//   - no open engagement for the scenario: (nil, nil), start proceeds;
//   - the same slug without --replace: refused with check/promote/cycle/abandon
//     guidance;
//   - the same slug with --replace: the engagement is returned so start takes
//     it over and keeps its restore point;
//   - another slug: refused. v1 runs one engagement per scenario, and a second
//     shadow manifest makes live restarts fail ("multiple open shadow
//     engagements"). The usual cause is a missing --name, so the error names
//     the open slug.
//
// A floor read failure refuses too: start must not capture blind.
func guardOpenEngagement(ctx context.Context, scenario, slug string, replace bool) (*engagementView, error) {
	open, err := openEngagementsFor(ctx, scenario)
	if err != nil {
		return nil, fmt.Errorf("baseline start refused: could not check for an open engagement before capturing a restore point: %w", err)
	}
	var takeover *engagementView
	for i := range open {
		e := open[i]
		if e.Slug != slug {
			return nil, otherEngagementError(e, slug)
		}
		if !replace {
			return nil, openEngagementError(e)
		}
		takeover = &e
	}
	return takeover, nil
}

// openEngagementError explains why start refused an open engagement of the
// same slug and lists the verbs that end or take it over.
func openEngagementError(e engagementView) error {
	ref := fmt.Sprintf("--scenario %s --name %s", e.Scenario, e.Slug)
	var b strings.Builder
	fmt.Fprintf(&b, "baseline start refused: %s/%s is already an open %s engagement%s; its restore point is the frozen baseline, so start did not capture or copy anything.\n",
		e.Scenario, e.Slug, e.Mode, engagementAge(e))
	fmt.Fprintf(&b, "  validate it:        git-control-tower baseline check %s\n", ref)
	fmt.Fprintf(&b, "  keep the work:      git-control-tower baseline promote %s (then start again with the same --name)\n", ref)
	if e.Mode == modeShadow {
		fmt.Fprintf(&b, "  keep + new shadow:  git-control-tower baseline cycle %s\n", ref)
	}
	fmt.Fprintf(&b, "  discard the work:   git-control-tower baseline abandon %s\n", ref)
	if e.Expired {
		fmt.Fprintf(&b, "  reap (expired):     git-control-tower baseline gc\n")
	}
	fmt.Fprintf(&b, "  take it over:       git-control-tower baseline start %s --replace (keeps its restore point)", ref)
	return fmt.Errorf("%s", b.String())
}

// otherEngagementError explains why start refused a second engagement for a
// scenario that already has one under another slug.
func otherEngagementError(e engagementView, requested string) error {
	ref := fmt.Sprintf("--scenario %s --name %s", e.Scenario, e.Slug)
	var b strings.Builder
	fmt.Fprintf(&b, "baseline start refused: %s already has an open %s engagement %q%s; a scenario has one engagement at a time, so start did not create %q (did you mean --name %s?).\n",
		e.Scenario, e.Mode, e.Slug, engagementAge(e), requested, e.Slug)
	fmt.Fprintf(&b, "  inspect it:         git-control-tower baseline status\n")
	fmt.Fprintf(&b, "  keep the work:      git-control-tower baseline promote %s\n", ref)
	if e.Mode == modeShadow {
		fmt.Fprintf(&b, "  keep + new shadow:  git-control-tower baseline cycle %s\n", ref)
	}
	fmt.Fprintf(&b, "  discard the work:   git-control-tower baseline abandon %s", ref)
	return fmt.Errorf("%s", b.String())
}

// engagementAge renders " (opened …, touched …)" from the manifest timestamps,
// or "" when the floor did not report them.
func engagementAge(e engagementView) string {
	var parts []string
	if e.CreatedAt != nil {
		parts = append(parts, "opened "+e.CreatedAt.UTC().Format(time.RFC3339))
	}
	if e.LastTouchedAt != nil {
		parts = append(parts, "touched "+e.LastTouchedAt.UTC().Format(time.RFC3339))
	}
	if e.Expired {
		parts = append(parts, "idle TTL expired")
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
