package transitionrunner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

// SnapshotFrom is the single construction seam for immutable transition
// inputs. Subject adapters supply only their already-validated projection and
// concurrency tokens; the runner owns the transport shape.
func SnapshotFrom(input *structpb.Value, entityVersion, frontierDigest string) Snapshot {
	return Snapshot{Input: input, EntityVersion: entityVersion, FrontierDigest: frontierDigest}
}

// SnapshotFromSubject builds the immutable snapshot tokens from the current
// subject projection. Stored versions must never be passed through as the
// entity token: callers rebuild this value at both start and apply time.
func SnapshotFromSubject(input *structpb.Value, subject any, frontier any) (Snapshot, error) {
	entityVersion, err := DigestJSON("entity", subject)
	if err != nil {
		return Snapshot{}, err
	}
	frontierDigest, err := DigestJSON("frontier", frontier)
	if err != nil {
		return Snapshot{}, err
	}
	return SnapshotFrom(input, entityVersion, frontierDigest), nil
}

// DigestJSON provides a stable digest for a JSON projection. The domain
// prefix prevents unrelated projections from accidentally sharing a token.
func DigestJSON(prefix string, value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal snapshot: %w", err)
	}
	h := sha256.Sum256(append([]byte(prefix+"\x00"), raw...))
	return "sha256:" + hex.EncodeToString(h[:]), nil
}
