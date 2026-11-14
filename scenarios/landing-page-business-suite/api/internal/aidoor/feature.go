package aidoor

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// Feature requests carry the scenario's instruction; LPBS never authors it.
// Gateway alone owns the published contract and validates it before dispatch.
func (d *Door) feature(w http.ResponseWriter, r *http.Request, key Key) {
	var input struct {
		Instruction string `json:"instruction"`
		Turn        struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"turn"`
		Schema json.RawMessage `json:"schema"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF {
		fail(w, 400, "invalid bounded feature request")
		return
	}
	if key.Harness != "" || key.Model != "" {
		fail(w, 403, "coding session keys cannot invoke application features")
		return
	}
	featureID := strings.TrimPrefix(r.URL.Path, "/api/v1/features/")
	if featureID == "" || strings.Contains(featureID, "/") || featureID == agentSessionFeature {
		fail(w, 404, "declared feature required")
		return
	}
	payload, _ := json.Marshal(input)
	d.funded(w, r, key, featureID, "", nil, payload)
}
