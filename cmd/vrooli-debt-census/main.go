package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliout"
)

func main() {
	var jsonOutput bool
	var ciMode bool
	var root string
	var budgetPath string
	flag.BoolVar(&jsonOutput, "json", false, "emit the machine-readable census document")
	flag.BoolVar(&ciMode, "ci", os.Getenv("CI") != "", "treat unavailable analysis tools as an error")
	flag.StringVar(&root, "root", "", "repository root (defaults to the current repository)")
	flag.StringVar(&budgetPath, "budget", "", "budget file (defaults to .vrooli/internal-debt-budget.json under root)")
	flag.Parse()

	if err := run(root, budgetPath, jsonOutput, ciMode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root, budgetPath string, jsonOutput, ciMode bool) error {
	if root == "" {
		resolved, err := buildinfo.ResolveSourceRoot()
		if err != nil {
			return fmt.Errorf("resolve repository root: %w", err)
		}
		root = resolved
	}
	if budgetPath == "" {
		budgetPath = filepath.Join(root, ".vrooli", "internal-debt-budget.json")
	}
	budget, err := loadBudget(budgetPath)
	if err != nil {
		return fmt.Errorf("read debt budget: %w", err)
	}
	doc, failures := collect(root, budget)
	if jsonOutput {
		if err := cliout.WriteJSON(os.Stdout, doc); err != nil {
			return err
		}
	} else {
		rows := make([][]string, 0, len(doc.Metrics))
		for _, metric := range doc.Metrics {
			value := string(metric.Status)
			delta := "-"
			if metric.Value != nil {
				value = fmt.Sprintf("%d", *metric.Value)
				delta = fmt.Sprintf("%d", *metric.Delta)
			}
			rows = append(rows, []string{metric.Name, value, fmt.Sprintf("%d", metric.Budget), delta, metric.Phase})
		}
		if err := cliout.RenderTable(os.Stdout, []string{"Metric", "Value", "Budget", "Delta", "Phase"}, rows); err != nil {
			return err
		}
		fmt.Fprintln(os.Stdout, "Ratchet: when a metric is below budget, lower both budget and baseline in .vrooli/internal-debt-budget.json and rerun make debt.")
	}
	failures = append(failures, failedMeasurementFailures(doc, ciMode)...)
	if len(failures) > 0 {
		return fmt.Errorf("internal debt budget failed:\n- %s", joinLines(failures))
	}
	return nil
}

func joinLines(lines []string) string {
	result := ""
	for index, line := range lines {
		if index > 0 {
			result += "\n- "
		}
		result += line
	}
	return result
}
