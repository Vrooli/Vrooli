package support

import "encoding/json"

// TypeDefinition mirrors one entry in the `definitions` map returned by
// `GET /api/types`.
type TypeDefinition struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Fields      []string `json:"fields,omitempty"`
}

// TypesResponse is the response shape from `GET /api/types`.
type TypesResponse struct {
	Types       []string                  `json:"types"`
	Definitions map[string]TypeDefinition `json:"definitions"`
}

// GenerateResponse is the response envelope from `POST /api/generate/:type`
// and `POST /api/generate/custom`. The `data` field is polymorphic: a JSON
// array for json/csv formats and a string for xml/sql formats.
type GenerateResponse struct {
	Success   bool            `json:"success"`
	Type      string          `json:"type,omitempty"`
	Count     int             `json:"count,omitempty"`
	Format    string          `json:"format,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
	Schema    json.RawMessage `json:"schema,omitempty"`
	Note      string          `json:"note,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}
