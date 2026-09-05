package execution

import (
	"context"
	"testing"

	"swarm-manager/internal/agentmanager"
)

// commitExecutionRoundForTest preserves historical-record coverage without
// retaining the retired operation runner in production code.
func commitExecutionRoundForTest(t *testing.T, service *Service, operationID, outcome string) {
	t.Helper()
	next, terminal := executionRecordStatusForOutcome(outcome)
	if !terminal {
		return
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	records, err := service.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	for i := range records {
		if records[i].OpExecutionID != operationID {
			continue
		}
		previous := records[i].Status
		records[i].Status = next
		records[i].UpdatedAt = nowRFC3339()
		if next == StatusFailed {
			records[i].FailureReason = executionCommitAbstainReason(outcome)
		}
		var candidates []string
		service.applyTerminalTransition(context.Background(), &records[i], agentmanager.RunState{}, next, &candidates)
		if err := service.store.Save(records); err != nil {
			t.Fatal(err)
		}
		service.dispatchStatusAndLog(records[i], previous)
		return
	}
}

func executionRecordStatusForOutcome(outcome string) (Status, bool) {
	switch outcome {
	case "completed":
		return StatusCompleted, true
	case "blocked", "needs-attention":
		return StatusFailed, true
	default:
		return "", false
	}
}

func executionCommitAbstainReason(outcome string) string {
	if outcome == "blocked" {
		return "execution run reported blocked; parked for operator review"
	}
	return "execution run could not derive an honest outcome; parked for operator review"
}
