package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// scanStagedCandidate binds the Security Health request to the exact index
// blobs for changed paths. Any read, transport, or scanner coverage gap is an
// incomplete result and blocks the caller from treating the candidate as safe.
func scanStagedCandidate(ctx context.Context, git GitRunner, repoDir string, scanner CandidateSecurityScanner) PrecommitRunResult {
	result := PrecommitRunResult{
		Command:   "security-health candidate scan",
		Timestamp: time.Now().UTC(),
	}
	incomplete := func(reason string) PrecommitRunResult {
		result.Status = "incomplete"
		result.ExitCode = 1
		result.Summary = "Candidate security coverage is incomplete: " + reason
		return result
	}
	if git == nil {
		return incomplete("Git index reader is unavailable")
	}
	files, err := git.ReadStagedCandidate(ctx, repoDir)
	if err != nil {
		return incomplete("exact staged content could not be read")
	}
	if len(files) == 0 {
		result.Status = "passed"
		result.Summary = "No changed staged content required a secret scan"
		return result
	}
	if scanner == nil {
		return incomplete("security scanner is unavailable")
	}
	scan, err := scanner.Scan(ctx, files)
	if err != nil || !scan.Complete {
		if scan.FixtureOnly {
			return incomplete("fixture benign provider rejected a non-allowlisted candidate")
		}
		return incomplete("security scanner did not complete")
	}
	if len(scan.Findings) == 0 {
		result.Status = "passed"
		if scan.FixtureOnly {
			result.Summary = "Fixture-only benign provider passed; this is not staged-secret coverage or later-index revalidation"
		} else {
			result.Summary = "Candidate secret scan passed"
		}
		return result
	}
	result.Status = "failed"
	result.ExitCode = 1
	result.Summary = fmt.Sprintf("Candidate secret scan found %d likely secret(s)", len(scan.Findings))
	lines := make([]string, 0, len(scan.Findings))
	for _, finding := range scan.Findings {
		line := strings.TrimSpace(finding.FilePath + " [" + finding.RuleID + "]")
		if finding.Description != "" {
			line += ": " + strings.TrimSpace(finding.Description)
		}
		lines = append(lines, line)
	}
	result.Stdout = capOutput(strings.Join(lines, "\n"))
	return result
}
