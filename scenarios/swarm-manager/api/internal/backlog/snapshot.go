package backlog

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

// immutableBacklogSnapshotVersion identifies the item state that a bounded
// plan-author or plan-repair workflow was authorized to inspect. It is not a
// workshop-round version and remains stable across the retirement of
// file-backed workshop history.
func immutableBacklogSnapshotVersion(item BacklogItem) string {
	payload, _ := json.Marshal(item)
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("sha256:%x", digest[:])
}
