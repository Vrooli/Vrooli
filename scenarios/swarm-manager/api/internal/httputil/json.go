package httputil

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// DecodeJSONStrict decodes a single JSON object from the request body and
// rejects unknown fields or trailing data.
func DecodeJSONStrict(r *http.Request, target any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("unexpected trailing JSON content")
	}

	return nil
}
