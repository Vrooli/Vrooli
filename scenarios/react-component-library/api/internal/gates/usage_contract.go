package gates

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type usageContractReport struct {
	Coverage struct {
		Eligible   int `json:"eligible"`
		Contracted int `json:"contracted"`
	} `json:"coverage"`
	Errors []string `json:"errors"`
}

// ValidateUsageContracts runs the catalog's TypeScript-aware source/contract
// checker as a corpus gate. An empty corpus, missing runner or malformed report
// is reported as a gate finding so it cannot be mistaken for a clean result.
func ValidateUsageContracts(scope Scope) (Result, error) {
	root := filepath.Join(scope.Root, "scenarios", "react-component-library")
	script := filepath.Join(root, "ui", "scripts", "check-catalog-usage-contracts.mjs")
	if _, err := os.Stat(script); err != nil {
		if os.IsNotExist(err) {
			return Result{Findings: []Finding{{
				Code:        "catalog.usage_contract.runner_missing",
				File:        repoRel(scope.Root, script),
				Message:     "usage-contract checker is missing; no contract coverage was evaluated",
				Remediation: "restore the catalog usage-contract checker before treating this gate as clean",
			}}}, nil
		}
		return Result{}, err
	}
	if _, err := exec.LookPath("node"); err != nil {
		return Result{Findings: []Finding{{
			Code:        "catalog.usage_contract.runner_unavailable",
			Message:     "node is unavailable; usage-contract coverage was not evaluated",
			Remediation: "restore the scenario Node toolchain before trusting the usage-contract gate",
		}}}, nil
	}
	parent := scope.Context
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 3*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, "node", script, "--json")
	command.Dir = root
	output, commandErr := command.CombinedOutput()
	var report usageContractReport
	if err := json.Unmarshal(output, &report); err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" && commandErr != nil {
			message = commandErr.Error()
		}
		return Result{Findings: []Finding{{
			Code:        "catalog.usage_contract.report_invalid",
			Message:     fmt.Sprintf("usage-contract checker returned no valid report: %s", message),
			Remediation: "repair the checker output before treating this gate as clean",
		}}}, nil
	}
	result := Result{Inspected: report.Coverage.Eligible}
	if report.Coverage.Eligible <= 0 {
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog.usage_contract.zero_inputs",
			Message:     "usage-contract gate found no eligible assets; coverage was not evaluated",
			Remediation: "restore the catalog asset source and rerun the gate",
		})
	}
	if report.Coverage.Contracted != report.Coverage.Eligible {
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog.usage_contract.coverage_incomplete",
			Message:     fmt.Sprintf("usage-contract checker reported %d contracted assets for %d eligible assets", report.Coverage.Contracted, report.Coverage.Eligible),
			Remediation: "repair the checker report or restore every eligible latest asset's usage contract",
		})
	}
	if commandErr != nil && len(report.Errors) == 0 {
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog.usage_contract.checker_failed",
			Message:     "usage-contract checker exited unsuccessfully without reporting findings",
			Remediation: "inspect the runner failure; an unsuccessful check cannot pass",
		})
	}
	for _, problem := range report.Errors {
		result.Findings = append(result.Findings, Finding{
			Code:        "catalog.usage_contract.invalid",
			Message:     problem,
			Remediation: "align the contract's component/version and collection fields with the published asset's TypeScript props",
		})
	}
	return nonEmpty(result, "usage-contract"), nil
}
