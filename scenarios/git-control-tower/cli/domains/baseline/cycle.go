// cycle.go implements `git-control-tower baseline cycle` — the small-promotion
// cadence (DL-7): promote a shadow engagement, then at once open a fresh shadow
// engagement under the same name, so a long goal keeps exactly one engagement
// and live never drifts far behind the working tree.
//
//	promote (gate → drain → data snapshot → migrate → re-point → restart →
//	         status probe → --probe-cmd → shadow teardown → clean;
//	         any restart/probe failure auto-rolls back and stops the cycle)
//	  → start --mode shadow --name <same> --no-anchor (same idle TTL)
//
// It composes promoteEngagement and startEngagement and owns no state, so a
// retry reads the floor and continues from where the last attempt stopped:
//   - shadow engagement open             → promote it, then re-create;
//   - promote interrupted after re-point → promote resumes it, then re-create;
//   - no engagement under the name        → a previous cycle promoted; re-create;
//   - re-create failed at shadow stand-up → that half-created engagement is
//     removed, so the next retry re-creates instead of promoting it again.
package baseline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
)

// sharedSourcesUnfrozenMarker is the text of the control plane's fail-closed
// capture refusal (recovery.ErrSharedSourcesUnresolved).
const sharedSourcesUnfrozenMarker = "shared sources could not be frozen"

// shadowStandUpError marks a start that wrote its engagement manifest but could
// not start the shadow instance, so cycle knows the manifest is its own.
type shadowStandUpError struct{ err error }

func (e shadowStandUpError) Error() string { return e.err.Error() }
func (e shadowStandUpError) Unwrap() error { return e.err }

// cycleResult is the one summary a cycle prints (also the --json shape).
type cycleResult struct {
	Scenario   string         `json:"scenario"`
	Slug       string         `json:"slug"`
	Promoted   bool           `json:"promoted"`
	RolledBack bool           `json:"rolledBack"`
	Recreated  bool           `json:"recreated"`
	Message    string         `json:"message"`
	Steps      []string       `json:"steps"`
	Next       string         `json:"next,omitempty"`
	Promote    *promoteResult `json:"promote,omitempty"`
	Start      *startResult   `json:"start,omitempty"`
}

func runCycleCmd(core *cliapp.ScenarioApp, args []string) error {
	var f promoteFlags
	fs := newFlagSet("baseline cycle")
	f.bind(fs, "", "Engagement slug to promote and re-create (required — the same name keeps one engagement)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, err := f.params()
	if err != nil {
		return err
	}
	if strings.TrimSpace(p.scenario) == "" {
		return fmt.Errorf("--scenario is required")
	}
	if strings.TrimSpace(p.slug) == "" {
		return fmt.Errorf("--name is required: cycle promotes and re-creates one named engagement (pass the open engagement's name; `baseline status` lists it)")
	}

	res, cycleErr := cycleEngagement(core, p)
	if f.jsonOut {
		if err := printJSON(res); err != nil {
			return err
		}
	} else {
		writeCycleSummary(os.Stdout, res)
	}
	if cycleErr != nil {
		os.Exit(1)
	}
	return nil
}

// cycleEngagement promotes the named shadow engagement and re-creates it. It
// returns the summary in every case; a non-nil error means the cycle stopped
// before a fresh shadow was running, and the summary says where and what to do.
func cycleEngagement(core *cliapp.ScenarioApp, p promoteParams) (cycleResult, error) {
	scenario := strings.TrimSpace(p.scenario)
	slug := strings.TrimSpace(p.slug)
	p.scenario, p.slug = scenario, slug
	res := cycleResult{Scenario: scenario, Slug: slug}
	retry := fmt.Sprintf("re-run `git-control-tower baseline cycle --scenario %s --name %s` (same flags)", scenario, slug)
	ctx := context.Background()

	open, err := openEngagementsFor(ctx, scenario)
	if err != nil {
		res.Message = "cycle not started: could not read open engagements"
		return res, fmt.Errorf("%s: %w", res.Message, err)
	}
	var eng *engagementView
	for i := range open {
		if open[i].Slug != slug {
			res.Message = fmt.Sprintf("cycle refused: %s's open engagement is %q, not %q (did you mean --name %s?)", scenario, open[i].Slug, slug, open[i].Slug)
			return res, errors.New(res.Message)
		}
		eng = &open[i]
	}

	ttl := ""
	switch {
	case eng == nil:
		res.Steps = append(res.Steps, fmt.Sprintf("no open engagement %s/%s — nothing to promote (an earlier cycle promoted it); re-creating the shadow", scenario, slug))
	case eng.Mode == modeShadow || interruptedPromote(*eng):
		pr, perr := promoteEngagement(core, p)
		res.Promote = &pr
		res.Promoted, res.RolledBack = pr.Promoted, pr.RolledBack
		for _, s := range pr.Steps {
			res.Steps = append(res.Steps, "promote: "+s)
		}
		if perr != nil {
			if pr.RolledBack {
				// The rollback steps already carry the cause.
				res.Message = "promote rolled back — live serves the baseline copy again; the engagement and its shadow stay open"
			} else {
				res.Message = "promote did not complete — the engagement stays open"
				res.Steps = append(res.Steps, "promote: ✗ "+perr.Error())
			}
			res.Next = "fix the cause, then " + retry
			return res, perr
		}
		if d, err := time.ParseDuration(strings.TrimSpace(eng.TTL)); err == nil && d > 0 {
			ttl = d.String()
		}
	default:
		res.Message = fmt.Sprintf("cycle refused: %s/%s is a live engagement; cycle re-creates shadows — accept it with `git-control-tower baseline promote --scenario %s --name %s`", scenario, slug, scenario, slug)
		return res, errors.New(res.Message)
	}

	// Re-create: same name, shadow only, no anchor run (journeys are the gate;
	// the anchor is a busy-exclusive comprehensive run).
	sr, serr := startEngagement(core, startParams{
		scenario: scenario, mode: modeShadow, slug: slug, ttl: ttl,
		noAnchor: true, requireMode: modeShadow,
	})
	if serr != nil {
		res.Steps = append(res.Steps, "re-create: ✗ "+serr.Error())
		res.Message = promotedPrefix(res) + "the fresh shadow was not created"
		res.Next = "fix the cause, then " + retry + " — it re-creates without promoting again"
		if strings.Contains(serr.Error(), sharedSourcesUnfrozenMarker) {
			res.Next = fmt.Sprintf("fix the shared-package freeze, then %s; or accept a live build against the repository's current shared packages with `git-control-tower baseline start --scenario %s --name %s --mode shadow --no-anchor --allow-unfrozen`", retry, scenario, slug)
		}
		var standUp shadowStandUpError
		if errors.As(serr, &standUp) {
			note, removed := removeHalfCreatedEngagement(ctx, scenario, slug)
			res.Steps = append(res.Steps, note)
			if !removed {
				res.Next = fmt.Sprintf("run `git-control-tower baseline abandon --scenario %s --name %s` (the half-created engagement would otherwise be promoted again), fix the cause, then %s", scenario, slug, retry)
			}
		}
		return res, serr
	}
	res.Start = &sr
	res.Recreated = true
	res.Steps = append(res.Steps, fmt.Sprintf("re-create: shadow engagement %s/%s opened (restore point %s, instance %s@%s, no diff anchor — `baseline check` needs one; validate the shadow with journeys)",
		scenario, slug, sr.RestorePoint, scenario, sr.Variant))
	for _, note := range sr.DataPopulation {
		res.Steps = append(res.Steps, "re-create: shadow data: "+note)
	}
	res.Message = promotedPrefix(res) + "fresh shadow engagement " + slug + " is running"
	return res, nil
}

// removeHalfCreatedEngagement drops the engagement a failed re-create just
// wrote (its restore point is a copy of the already-promoted working tree), so
// the next cycle re-creates it instead of promoting it again. Best-effort; it
// reports whether the engagement is gone.
func removeHalfCreatedEngagement(ctx context.Context, scenario, slug string) (string, bool) {
	_, _ = runCommand(ctx, "vrooli", "scenario", "stop", scenario, "--instance", modeShadow)
	if _, err := runCommand(ctx, "vrooli", "recovery", "clean", "--scenario", scenario, "--slug", slug); err != nil {
		return fmt.Sprintf("re-create: ⚠ could not remove the half-created engagement: %v", err), false
	}
	return "re-create: half-created engagement removed (shadow stopped, manifest and restore point dropped)", true
}

func promotedPrefix(res cycleResult) string {
	if res.Promoted {
		return "promoted shadow → live; "
	}
	return ""
}

// writeCycleSummary renders the one cycle summary: a headline, the ordered
// steps, and the next action when the cycle stopped early.
func writeCycleSummary(w io.Writer, res cycleResult) {
	mark := "✓"
	if !res.Recreated {
		mark = "✗"
	}
	fmt.Fprintf(w, "%s cycle %s/%s: %s\n", mark, res.Scenario, res.Slug, res.Message)
	for _, s := range res.Steps {
		fmt.Fprintf(w, "  · %s\n", s)
	}
	if res.Promote != nil && res.Promote.RolledBack && res.Promote.DataSnapshot != "" {
		fmt.Fprintf(w, "  data snapshot for manual restore: %s\n", res.Promote.DataSnapshot)
	}
	if res.Next != "" {
		fmt.Fprintf(w, "  next: %s\n", res.Next)
	}
}
