package policy

import (
	"encoding/json"
	"errors"
	"os"
)

// LoadRightsContextFile reads the versioned operator-owned business context
// used by local model rights decisions. Request payloads must never provide
// this value.
func LoadRightsContextFile(path string) (RightsContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RightsContext{}, err
	}
	var document struct {
		SchemaVersion string        `json:"schema_version"`
		Context       RightsContext `json:"context"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		return RightsContext{}, err
	}
	c := document.Context
	if document.SchemaVersion != "1" || c.ID == "" || c.Version == "" || c.Source == "" || c.AsOf == "" || c.RevenueStatus == "" {
		return RightsContext{}, errors.New("trusted model rights context is incomplete or unsupported")
	}
	return c, nil
}
