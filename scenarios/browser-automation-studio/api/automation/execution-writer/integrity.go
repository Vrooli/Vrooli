package executionwriter

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// setPayloadDigest records the canonical JSON payload bytes for an inline
// artifact. Those bytes are the published representation for console and
// network evidence, so the manifest can verify them without metadata fallbacks.
func setPayloadDigest(artifact *ArtifactData) error {
	if artifact == nil {
		return fmt.Errorf("artifact is nil")
	}
	payload, err := json.Marshal(artifact.Payload)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(payload)
	artifact.SHA256 = hex.EncodeToString(digest[:])
	size := int64(len(payload))
	artifact.SizeBytes = &size
	return nil
}
