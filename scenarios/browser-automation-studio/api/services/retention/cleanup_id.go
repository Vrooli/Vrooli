package retention

import (
	"strings"

	"github.com/google/uuid"
)

const recordingCleanupIDPrefix = "recording:"

// parseRecordingCleanupID parses the UUID carried by a prefixed cleanup item.
func parseRecordingCleanupID(raw string) (uuid.UUID, error) {
	if !strings.HasPrefix(raw, recordingCleanupIDPrefix) {
		return uuid.Nil, ErrInvalidCleanupPreview
	}
	return uuid.Parse(strings.TrimPrefix(raw, recordingCleanupIDPrefix))
}
