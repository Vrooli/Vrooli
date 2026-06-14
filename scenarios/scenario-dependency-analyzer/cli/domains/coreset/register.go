// Package coreset registers the `core-set` CLI verb, which surfaces the
// reflexive core set (9-seed ∪ transitive Required closure) and the
// trusted-base subset that the Baseline Modes decision tree consumes.
package coreset

import (
	"fmt"

	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

// coreSetResponse mirrors the API's core-set payload.
type coreSetResponse struct {
	Source         string            `json:"source"`
	CoreSet        []string          `json:"core_set"`
	Seed           []string          `json:"seed"`
	AddedByClosure []string          `json:"added_by_closure"`
	TrustedBase    []string          `json:"trusted_base"`
	LoadErrors     map[string]string `json:"load_errors"`
}

// Register exposes the core-set command group.
func Register(core *cliapp.ScenarioApp) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Core Set",
		Commands: []cliapp.Command{
			{
				Name:        "core-set",
				Description: "Show the reflexive core set (9-seed ∪ required closure) and trusted-base subset",
				NeedsAPI:    true,
				Run: func(args []string) error {
					return run(core, args)
				},
			},
		},
	}
}

func run(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("core-set")
	var jsonOutput bool
	var trustedOnly bool
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.BoolVar(&trustedOnly, "trusted-base", false, "List only the trusted-base subset (never-shadowed scenarios)")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: %s core-set [--trusted-base] [--json]", support.AppName)
	}

	body, err := core.Get("/core-set", nil)
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintAPIJSON(body)
	}

	var resp coreSetResponse
	if err := support.Decode(body, &resp); err != nil {
		return err
	}

	if trustedOnly {
		report := cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Trusted base: %d scenario(s) — never shadowed (always live mode)", len(resp.TrustedBase))},
			ResultsHeading: "Trusted Base",
			Results:        resp.TrustedBase,
			RetrievalHints: []string{support.AppName + " core-set --json"},
		}
		return support.PrintList(false, report, nil)
	}

	summary := []string{
		fmt.Sprintf("Core set: %d scenario(s) (source: %s)", len(resp.CoreSet), resp.Source),
		fmt.Sprintf("Seed: %d · Added by closure: %d · Trusted base: %d", len(resp.Seed), len(resp.AddedByClosure), len(resp.TrustedBase)),
	}
	if len(resp.LoadErrors) > 0 {
		summary = append(summary, fmt.Sprintf("⚠ %d scenario(s) failed to load (seed members are still included)", len(resp.LoadErrors)))
	}

	report := cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Core Set (reflexive scenarios)",
		Results:        annotate(resp),
		RetrievalHints: []string{
			support.AppName + " core-set --json",
			support.AppName + " core-set --trusted-base",
		},
	}
	return support.PrintList(false, report, nil)
}

// annotate decorates each core-set member with its provenance ([seed]/[closure])
// and a [trusted-base] marker so the human-readable list is self-explanatory.
func annotate(resp coreSetResponse) []string {
	added := toSet(resp.AddedByClosure)
	trusted := toSet(resp.TrustedBase)
	out := make([]string, 0, len(resp.CoreSet))
	for _, name := range resp.CoreSet {
		origin := "seed"
		if added[name] {
			origin = "closure"
		}
		line := fmt.Sprintf("%s [%s]", name, origin)
		if trusted[name] {
			line += " [trusted-base]"
		}
		out = append(out, line)
	}
	return out
}

func toSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}
	return set
}
