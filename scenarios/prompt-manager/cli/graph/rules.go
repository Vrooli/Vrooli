package graph

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"prompt-manager/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliutil"
)

// ruleCatalogEntry mirrors the API's operator-facing rule identity.
type ruleCatalogEntry struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	Severity    string `json:"severity"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Actuator    string `json:"actuator"`
	Findings    int    `json:"findings"`
}

type rulesResponse struct {
	Rules  []ruleCatalogEntry `json:"rules"`
	Total  int                `json:"total"`
	Silent int                `json:"silent"`
}

// cmdRules prints the rule catalog. It is the authority the generated
// documentation tables are rendered from, so an operator and a doc build read
// the same source instead of two hand-maintained tables drifting apart.
func cmdRules(ctx appctx.Context, args []string) error {
	fs := flag.NewFlagSet("rules", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Output as JSON")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	var resp rulesResponse
	if err := ctx.Get("/topics/rules", &resp); err != nil {
		return fmt.Errorf("failed to fetch rule catalog: %w", err)
	}

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RULE\tGROUP\tSEVERITY\tKIND\tFINDINGS\tDESCRIPTION")
	for _, rule := range resp.Rules {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\n",
			rule.ID, rule.Group, rule.Severity, rule.Kind, rule.Findings, rule.Description)
	}
	_ = w.Flush()

	fmt.Println()
	fmt.Printf("%d catalogued rules. %d produced no finding in this run.\n", resp.Total, resp.Silent)
	fmt.Println("A rule that produced no finding is not necessarily dead: a clean tree is also silent. Screen on whether a test makes it fire and whether a failure names something specific to change.")
	return nil
}
