package main

import "time"

type CommitCheckKind string

const (
	CommitCheckKindPrecommit CommitCheckKind = "precommit"
)

type CommitCheckStatus string

const (
	CommitCheckStatusPassed  CommitCheckStatus = "passed"
	CommitCheckStatusFailed  CommitCheckStatus = "failed"
	CommitCheckStatusTimeout CommitCheckStatus = "timeout"
	CommitCheckStatusSkipped CommitCheckStatus = "skipped"
)

type CommitCheckRun struct {
	Kind            CommitCheckKind   `json:"kind"`
	Status          CommitCheckStatus `json:"status"`
	Command         string            `json:"command"`
	ExitCode        int               `json:"exit_code"`
	Summary         string            `json:"summary"`
	Stdout          string            `json:"stdout,omitempty"`
	Stderr          string            `json:"stderr,omitempty"`
	DurationMs      int64             `json:"duration_ms"`
	Timestamp       time.Time         `json:"timestamp"`
	CheckJobID      string            `json:"check_job_id,omitempty"`
	CandidateDigest string            `json:"candidate_digest,omitempty"`
	ConfigDigest    string            `json:"config_digest,omitempty"`
}

func commitCheckFromPrecommit(result PrecommitRunResult) CommitCheckRun {
	run := CommitCheckRun{
		Kind:         CommitCheckKindPrecommit,
		Status:       CommitCheckStatus(result.Status),
		Command:      result.Command,
		ExitCode:     result.ExitCode,
		Summary:      result.Summary,
		Stdout:       result.Stdout,
		Stderr:       result.Stderr,
		DurationMs:   result.DurationMs,
		Timestamp:    result.Timestamp,
		CheckJobID:   result.receiptID,
		ConfigDigest: result.receiptConfig,
	}
	if result.receiptTree != "" {
		run.CandidateDigest = "tree:" + result.receiptTree
	}
	return run
}
