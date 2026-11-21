// Package claims owns the reusable factual-claim library and its evidence
// verification boundary.
package claims

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// EvidenceCheck is the re-runnable evidence attached to a claim. Command is
// intentionally local-only at this stage: hosted execution needs a distinct,
// sandboxed implementation behind Runner rather than an unsafe extension of
// this one.
type EvidenceCheck struct {
	Command        string
	ExpectedResult string
}

// CheckResult preserves the observed output for an audit trail while making
// the approval-relevant comparison explicit.
type CheckResult struct {
	ActualResult string
	Matches      bool
}

// Runner is the sole domain boundary for executing stored evidence checks.
// Services depend on this interface so tests can supply a deterministic fake
// and a future hosted runner can replace LocalRunner without rewriting claims.
type Runner interface {
	Run(context.Context, EvidenceCheck) (CheckResult, error)
}

// LocalRunner executes a check through the local POSIX shell and compares its
// complete stdout (after newline normalization) with the expected result.
// It is wired only in local deployments; it is not a sandbox.
type LocalRunner struct{}

var _ Runner = LocalRunner{}

func (LocalRunner) Run(ctx context.Context, check EvidenceCheck) (CheckResult, error) {
	command := strings.TrimSpace(check.Command)
	if command == "" {
		return CheckResult{}, fmt.Errorf("evidence check command is required")
	}

	output, err := exec.CommandContext(ctx, "sh", "-c", command).CombinedOutput()
	actual := strings.TrimSpace(string(output))
	if err != nil {
		return CheckResult{ActualResult: actual}, fmt.Errorf("execute evidence check: %w", err)
	}

	return CheckResult{ActualResult: actual, Matches: actual == strings.TrimSpace(check.ExpectedResult)}, nil
}
