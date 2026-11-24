package akash

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	akashclient "compute-manager/internal/provider/akash/client"
)

// TargetReferenceForLease encodes every immutable Akash lease coordinate and
// its record-first CM intent. The ID is opaque to callers but round-trips
// exactly; no field is reconstructed from a display name or provider URL.
func TargetReferenceForLease(ref akashclient.LeaseReference) string {
	data, err := json.Marshal(ref)
	if err != nil {
		return ""
	}
	return "akashref:" + base64.RawURLEncoding.EncodeToString(data)
}

// ParseTargetReference accepts only complete sandbox references emitted by
// TargetReferenceForLease. Legacy display references are intentionally not
// accepted by the typed sandbox client because they omit owner and sequence.
func ParseTargetReference(raw string) (akashclient.LeaseReference, error) {
	var ref akashclient.LeaseReference
	encoded := strings.TrimPrefix(strings.TrimSpace(raw), "akashref:")
	if encoded == strings.TrimSpace(raw) || encoded == "" {
		return ref, fmt.Errorf("%q is not a complete typed Akash lease reference", raw)
	}
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || json.Unmarshal(data, &ref) != nil {
		return akashclient.LeaseReference{}, fmt.Errorf("%q is not a valid typed Akash lease reference", raw)
	}
	if ref.Network != akashclient.NetworkSandbox2 || strings.TrimSpace(ref.ChainID) == "" || strings.TrimSpace(ref.IntentID) == "" {
		return akashclient.LeaseReference{}, errors.New("Akash target reference requires sandbox-2, exact chain ID and persisted Compute Manager intent")
	}
	if err := ref.ID.Validate(); err != nil {
		return akashclient.LeaseReference{}, err
	}
	return ref, nil
}
