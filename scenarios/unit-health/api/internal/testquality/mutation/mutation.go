// Package mutation contains the small, disposable mutation pilot classifier.
// It classifies receipts; it never mutates the caller's worktree.
package mutation

import (
	"fmt"

	"unit-health/internal/testquality"
)

type Experiment struct {
	MutationID          string `json:"mutationId"`
	SourceIdentity      string `json:"sourceIdentity"`
	TestIdentity        string `json:"testIdentity"`
	InContract          bool   `json:"inContract"`
	CompileOK           bool   `json:"compileOk"`
	TestFailed          bool   `json:"testFailed"`
	BehaviorChanged     bool   `json:"behaviorChanged"`
	InfrastructureError bool   `json:"infrastructureError"`
	EvidenceAvailable   bool   `json:"evidenceAvailable"`
}

func Classify(e Experiment) (testquality.MutationDisposition, error) {
	if e.MutationID == "" || e.SourceIdentity == "" || e.TestIdentity == "" {
		return "", fmt.Errorf("mutation, source, and test identities are required")
	}
	if !e.EvidenceAvailable {
		return testquality.MutationUnknown, nil
	}
	if e.InfrastructureError {
		return testquality.MutationInfrastructureFailure, nil
	}
	if !e.CompileOK {
		return testquality.MutationInvalid, nil
	}
	if !e.InContract {
		return testquality.MutationOutOfContract, nil
	}
	if !e.BehaviorChanged {
		return testquality.MutationEquivalent, nil
	}
	if e.TestFailed {
		return testquality.MutationKilled, nil
	}
	return testquality.MutationSurvived, nil
}
