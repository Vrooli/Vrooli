package stagedmessages

import (
	"encoding/json"
	"fmt"
)

// decodeDraftInput is the storage boundary for durable staged-message inputs.
// It validates the declared format without normalizing or rewriting the saved
// payload; callers must keep using the original job row as the durable record.
func decodeDraftInput(raw []byte) (DraftInput, error) {
	var input DraftInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return DraftInput{}, err
	}
	switch input.FormatVersion {
	case 2, 3, 4:
		return input, nil
	default:
		return DraftInput{}, fmt.Errorf("unsupported staged-message input format %d", input.FormatVersion)
	}
}
