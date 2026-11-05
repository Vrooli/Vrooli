package reconcile

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"regexp"
)

const EvaluatorVersion = "experience-reconcile/1"

var identityHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

// EvidenceIdentity identifies the evaluated contract and observed AX snapshot.
// Neither hash attests an application build, a candidate render, or a journey.
// Historical evidence has no identity; readers must not backfill one from the
// current contract and thereby manufacture freshness for an old observation.
type EvidenceIdentity struct {
	ContractHash     string `json:"contractHash"`
	SnapshotHash     string `json:"snapshotHash"`
	EvaluatorVersion string `json:"evaluatorVersion"`
}

func evidenceIdentityFor(contractHash string, snapshot Snapshot) EvidenceIdentity {
	if !identityHash.MatchString(contractHash) {
		return EvidenceIdentity{}
	}
	identity := EvidenceIdentity{ContractHash: contractHash, EvaluatorVersion: EvaluatorVersion}
	if snapshot.Contract == snapshotContract && len(snapshot.Flatten()) > 0 {
		encoded, err := json.Marshal(snapshot)
		if err == nil {
			identity.SnapshotHash = fmt.Sprintf("%x", sha256.Sum256(encoded))
		}
	}
	return identity
}
func withPreparedEvidenceIdentity(raw string, identity EvidenceIdentity) string {
	if identity.ContractHash == "" {
		return raw
	}
	var measurement map[string]any
	if json.Unmarshal([]byte(raw), &measurement) != nil || measurement == nil {
		return raw
	}
	measurement["evidenceIdentity"] = identity
	encoded, err := json.Marshal(measurement)
	if err != nil {
		return raw
	}
	return string(encoded)
}
func IdentityFromMeasurement(raw string) EvidenceIdentity {
	var envelope struct {
		Identity EvidenceIdentity `json:"evidenceIdentity"`
	}
	if json.Unmarshal([]byte(raw), &envelope) != nil {
		return EvidenceIdentity{}
	}
	i := envelope.Identity
	if !identityHash.MatchString(i.ContractHash) || i.EvaluatorVersion == "" {
		return EvidenceIdentity{}
	}
	if i.SnapshotHash != "" && !identityHash.MatchString(i.SnapshotHash) {
		return EvidenceIdentity{}
	}
	return i
}
