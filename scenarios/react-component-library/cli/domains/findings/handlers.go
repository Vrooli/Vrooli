package findings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"react-component-library/cli/internal/findingledger"
)

type document struct {
	Findings []findingledger.Finding `json:"findings"`
}

func Commands() cliapp.CommandGroup {
	return cliapp.CommandGroup{Title: "Findings", Commands: []cliapp.Command{{Name: "findings", NeedsAPI: false, Description: "Read the deduplicated, adoption-prioritized validation ledger", Run: run}}}
}

func run(args []string) error {
	if len(args) == 0 || args[0] != "list" {
		return fmt.Errorf("usage: react-component-library findings list [--json] [--root <path>]")
	}
	jsonOutput := false
	root := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--root":
			if i+1 >= len(args) {
				return fmt.Errorf("--root requires a path")
			}
			i++
			root = args[i]
		default:
			return fmt.Errorf("unknown option: %s", args[i])
		}
	}
	if root == "" {
		root = filepath.Join("coverage", "findings", "ledger.json")
	}
	data, err := os.ReadFile(root)
	if os.IsNotExist(err) {
		data = []byte(`{"findings":[]}`)
	} else if err != nil {
		return err
	}
	var stored document
	if err := json.Unmarshal(data, &stored); err != nil {
		return fmt.Errorf("decode findings ledger: %w", err)
	}
	rows := findingledger.Merge(nil, stored.Findings)
	if jsonOutput {
		return cliapp.PrintReportJSON(os.Stdout, struct {
			Findings []findingledger.Finding `json:"findings"`
		}{Findings: rows})
	}
	for i, finding := range rows {
		fmt.Fprintf(os.Stdout, "%d. %s@%s %s [%s/%s] %s (%s)\n", i+1, finding.Asset, finding.Version, finding.Check, finding.Viewport, finding.Theme, strings.TrimSpace(finding.Message), finding.RankReason)
	}
	if len(rows) == 0 {
		fmt.Fprintln(os.Stdout, "No validation findings in the ledger.")
	}
	return nil
}
