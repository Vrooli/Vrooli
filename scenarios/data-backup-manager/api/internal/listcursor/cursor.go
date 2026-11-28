// Package listcursor encodes bounded offset cursors for deterministic owner
// list APIs. Tokens are opaque transport state, not an authorization grant.
package listcursor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

const MaxOffset = 10000

type cursor struct {
	Scope  string `json:"s"`
	Offset int    `json:"o"`
}

func Decode(scope, token string) (int, error) {
	if token == "" {
		return 0, nil
	}
	body, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("invalid page token")
	}
	var value cursor
	if err := json.Unmarshal(body, &value); err != nil || value.Scope != scope || value.Offset < 1 || value.Offset > MaxOffset {
		return 0, fmt.Errorf("invalid page token")
	}
	return value.Offset, nil
}

func Encode(scope string, offset int) string {
	body, _ := json.Marshal(cursor{Scope: scope, Offset: offset})
	return base64.RawURLEncoding.EncodeToString(body)
}
