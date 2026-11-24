package server

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// tokenIssuerHint reads only the untrusted issuer claim used to select a
// verifier. Identity is accepted only after that provider verifies the token.
func tokenIssuerHint(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Issuer string `json:"iss"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	return strings.TrimSpace(claims.Issuer)
}
