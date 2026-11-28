package credentialauthority

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

const activeCredentialReferencePrefix = "vrooli-ca-active-ref-v2:"
const legacyActiveCredentialRecordPrefix = "vrooli-ca-active-v1:"
const legacyActiveCredentialRecordFormat = "vrooli.credential-authority.active/v1"

type legacyActiveCredentialRecord struct {
	Format  string `json:"format"`
	Version string `json:"version"`
	Value   string `json:"value"`
}

var ErrAmbiguousLegacyActiveRecord = errors.New("legacy credential value collides with retired active-record marker")

// v2 stores the value under an immutable, versioned data key and makes one
// metadata key the commit point. User values are never wrapped in a marker.
func encodeActiveCredentialReference(version string) (string, error) {
	if len(version) != 32 {
		return "", errors.New("active credential reference requires a version")
	}
	if _, err := hex.DecodeString(version); err != nil {
		return "", errors.New("active credential reference has an invalid version")
	}
	return activeCredentialReferencePrefix + version, nil
}

func decodeActiveCredentialReference(stored string) (string, bool, error) {
	version, ok := strings.CutPrefix(stored, activeCredentialReferencePrefix)
	if !ok {
		return "", false, nil
	}
	if len(version) != 32 {
		return "", true, errors.New("active credential reference is invalid")
	}
	if _, err := hex.DecodeString(version); err != nil {
		return "", true, errors.New("active credential reference is invalid")
	}
	return version, true, nil
}

// Old v1 inline records are ambiguous with arbitrary raw credential values.
// Runtime resolution fails closed; explicit migration tooling must ask the
// owner to identify the original value before upgrading such an item.
func decodeLegacyActiveValue(stored string) (string, error) {
	if strings.HasPrefix(stored, legacyActiveCredentialRecordPrefix) {
		return "", ErrAmbiguousLegacyActiveRecord
	}
	return stored, nil
}

// decodeLegacyActiveRecord is used only by the explicit owner-directed
// migration path. Ordinary resolution never guesses whether marker-prefixed
// bytes are an envelope or the operator's raw credential.
func decodeLegacyActiveRecord(stored string) (legacyActiveCredentialRecord, error) {
	payload, ok := strings.CutPrefix(stored, legacyActiveCredentialRecordPrefix)
	if !ok {
		return legacyActiveCredentialRecord{}, errors.New("legacy active record marker is absent")
	}
	decoded, err := base64.RawStdEncoding.DecodeString(payload)
	if err != nil {
		return legacyActiveCredentialRecord{}, errors.New("legacy active record is invalid")
	}
	var record legacyActiveCredentialRecord
	if err := json.Unmarshal(decoded, &record); err != nil {
		return legacyActiveCredentialRecord{}, errors.New("legacy active record is invalid")
	}
	if record.Format != legacyActiveCredentialRecordFormat || len(record.Version) != 32 || strings.TrimSpace(record.Value) == "" {
		return legacyActiveCredentialRecord{}, errors.New("legacy active record is invalid")
	}
	if _, err := hex.DecodeString(record.Version); err != nil {
		return legacyActiveCredentialRecord{}, errors.New("legacy active record is invalid")
	}
	return record, nil
}
