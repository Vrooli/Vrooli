package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	MaxReviewSnapshots        = 32
	MaxReviewFiles            = 10000
	MaxReviewFileBytes  int64 = 16 << 20
	MaxReviewInputBytes int64 = 64 << 20
)

// ReviewWorkspace is a derived tree owned by the retained snapshot. Root contains
// before/, after/, changes.patch and snapshot.json. Launchers must bind it read-only.
type ReviewWorkspace struct {
	Root   string `json:"root"`
	SHA256 string `json:"sha256"`
}

func ReviewSnapshotID(sandboxID, requestID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(sandboxID, []byte("review-snapshot:"+requestID.String()))
}

// ContentSHA256 excludes operation attribution, retaining the complete source
// selection, origin, body identities and patch identity in the canonical hash.
func (s ReviewSnapshot) ContentSHA256() string {
	s.ID, s.RequestID, s.SHA256, s.CreatedAt = uuid.Nil, uuid.Nil, "", time.Time{}
	raw, _ := json.Marshal(s) // this type contains only JSON-representable fields
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}
