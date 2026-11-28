package searcheval

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// AgentManagerReplayRequestID is the privacy-safe, deterministic bridge from a
// verified Agent Manager event to its Gateway route evidence. The role and
// consumer digest distinguishes independent task routes without persisting
// query content or other case data in the Gateway ledger.
func AgentManagerReplayRequestID(runID, eventID string, sequence int64, taskRole, consumer string) (string, error) {
	runID = strings.TrimSpace(runID)
	eventID = strings.TrimSpace(eventID)
	taskRole = strings.TrimSpace(taskRole)
	consumer = strings.TrimSpace(consumer)
	if runID == "" || eventID == "" || sequence < 0 || taskRole == "" || consumer == "" {
		return "", fmt.Errorf("Agent Manager replay request identity requires run, event, nonnegative sequence, task role, and consumer")
	}
	scope := sha256.Sum256([]byte(taskRole + "\x00" + consumer))
	return fmt.Sprintf("agent-manager-replay:v1:%s:%s:%d:%s", runID, eventID, sequence, hex.EncodeToString(scope[:])), nil
}
