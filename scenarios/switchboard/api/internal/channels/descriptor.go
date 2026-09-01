package channels

import "fmt"

type Descriptor struct {
	Kind          string        `json:"kind"`
	SchemaVersion int           `json:"schemaVersion"`
	ID            string        `json:"id"`
	DisplayName   string        `json:"displayName"`
	Transport     string        `json:"transport"`
	Supports      Supports      `json:"supports"`
	Limits        Limits        `json:"limits"`
	Setup         Setup         `json:"setup"`
	Cost          string        `json:"cost"`
	Requires      []Requirement `json:"requires,omitempty"`
}

type Supports struct {
	Text    bool `json:"text"`
	Images  bool `json:"images"`
	Files   bool `json:"files"`
	Groups  bool `json:"groups"`
	Threads bool `json:"threads"`
}
type Limits struct {
	MaxTextBytes  int64 `json:"maxTextBytes"`
	MaxMediaBytes int64 `json:"maxMediaBytes"`
}
type Setup struct {
	Friction int `json:"friction"`
}
type Requirement struct {
	Key         string `json:"key"`
	Description string `json:"description"`
}

func (d Descriptor) Validate() error {
	checks := []struct{ field, value string }{
		{"kind", d.Kind},
		{"schemaVersion", fmt.Sprint(d.SchemaVersion)},
		{"id", d.ID},
		{"displayName", d.DisplayName},
		{"transport", d.Transport},
		{"cost", d.Cost},
	}
	for _, c := range checks {
		if c.value == "" || (c.field == "schemaVersion" && c.value == "0") {
			return fmt.Errorf("field %s is required", c.field)
		}
	}
	if d.Limits.MaxTextBytes < 0 || d.Limits.MaxMediaBytes < 0 {
		return fmt.Errorf("field limits must be non-negative")
	}
	if d.Setup.Friction < 0 {
		return fmt.Errorf("field setup.friction must be non-negative")
	}
	return nil
}
