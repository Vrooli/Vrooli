package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

// cmdRuntime reports what agents actually did, as distinct from what the
// checked-in files declare.
//
// These two questions shared one command and one exit code. Because a runtime
// finding cannot be cleared by editing the tree, that shared exit code made
// `graph topics` permanently non-zero, so no CI gate could run it, so the
// runtime findings accumulated unchecked — actual_writer_undeclared went from 9
// to 43 between 2026-07-27 and 2026-07-31 with nothing forcing it down.
//
// This command always exits 0. It is a report, not a gate: the tree cannot be
// edited to satisfy it, and the place these findings actually change behavior
// is the responsible member's `# Contract Findings` prompt section, which is
// unaffected by this split.
func cmdRuntime(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("runtime", flag.ContinueOnError)
	team := fs.String("team", "", "Filter to one team")
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	q := url.Values{}
	if *team != "" {
		q.Set("team", *team)
	}
	var resp topicsGraphResponse
	if err := ctx.GetWithQuery("/topics/graph", q, &resp); err != nil {
		return fmt.Errorf("failed to fetch topics graph: %w", err)
	}

	runtimeFindings := findingsOfKind(resp.Validation.Findings, kindRuntime)

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(map[string]any{
			"kind":     kindRuntime,
			"team":     *team,
			"count":    len(runtimeFindings),
			"byRule":   countsByRule(runtimeFindings),
			"findings": runtimeFindings,
		})
	}

	if *team == "" {
		fmt.Println("Runtime observations (all teams)")
	} else {
		fmt.Printf("Runtime observations (team=%s)\n", *team)
	}
	fmt.Println()
	if len(runtimeFindings) == 0 {
		fmt.Println("No runtime findings.")
		return nil
	}

	counts := countsByRule(runtimeFindings)
	rules := make([]string, 0, len(counts))
	for rule := range counts {
		rules = append(rules, rule)
	}
	sort.Slice(rules, func(i, j int) bool {
		if counts[rules[i]] != counts[rules[j]] {
			return counts[rules[i]] > counts[rules[j]]
		}
		return rules[i] < rules[j]
	})
	for _, rule := range rules {
		fmt.Printf("%5d  %s\n", counts[rule], rule)
	}
	fmt.Println()
	for _, f := range runtimeFindings {
		fmt.Printf("- [%s] %s", f.Severity, f.Rule)
		if f.Team != "" || f.Member != "" {
			fmt.Printf(" %s/%s", f.Team, f.Member)
		}
		if f.Prefix != "" {
			fmt.Printf(" `%s`", f.Prefix)
		}
		fmt.Printf(": %s\n", f.Detail)
	}
	fmt.Println()
	fmt.Printf("%d runtime finding(s). This command never fails a build: a runtime finding reports what an agent did, and no edit to the tree clears it. Route adoption through the owning team's work item type.\n", len(runtimeFindings))
	return nil
}

func countsByRule(findings []topicFinding) map[string]int {
	counts := make(map[string]int, len(findings))
	for _, f := range findings {
		counts[f.Rule]++
	}
	return counts
}
