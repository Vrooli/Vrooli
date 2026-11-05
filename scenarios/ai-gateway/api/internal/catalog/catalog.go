// Package catalog carries resource-owned provider declarations into a deployed
// gateway. The generated payload contains declarations, never credentials.
package catalog

import (
	_ "embed"
	"encoding/json"
)

//go:embed providers.generated.json
var embedded []byte

type Provider struct {
	ID                 string          `json:"id"`
	Family             string          `json:"family"`
	Locality           string          `json:"locality"`
	Endpoint           string          `json:"endpoint"`
	CredentialIdentity string          `json:"credential_identity,omitempty"`
	CredentialField    string          `json:"credential_field,omitempty"`
	Policy             json.RawMessage `json:"policy"`
}

type Bundle struct {
	Roles     json.RawMessage            `json:"roles"`
	Features  json.RawMessage            `json:"features"`
	Providers map[string]Provider        `json:"providers"`
	Harnesses map[string]json.RawMessage `json:"harnesses"`
}

func Embedded() (Bundle, error) {
	var bundle Bundle
	err := json.Unmarshal(embedded, &bundle)
	return bundle, err
}
