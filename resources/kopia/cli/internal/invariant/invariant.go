// Package invariant declares cross-cutting safety rules the resource must
// uphold and provides scanners tests use to enforce them. The headline rule:
// encryption is non-negotiable and on by default, so the resource must NEVER
// emit any kopia flag that touches encryption (it relies entirely on kopia's
// secure default, AES256-GCM-HMAC-SHA256).
package invariant

import "strings"

// FindEncryptionFlag scans the argv of every produced kopia call and returns
// the first token that references encryption (which the resource must never
// emit), plus true when one is found.
func FindEncryptionFlag(calls [][]string) (string, bool) {
	for _, args := range calls {
		for _, tok := range args {
			if strings.Contains(strings.ToLower(tok), "encrypt") {
				return tok, true
			}
		}
	}
	return "", false
}

// FindCredentialInArgs scans argv for a secret value that should only ever
// travel through the environment, returning the offending token when found.
func FindCredentialInArgs(calls [][]string, secrets ...string) (string, bool) {
	for _, args := range calls {
		for _, tok := range args {
			for _, secret := range secrets {
				if secret != "" && strings.Contains(tok, secret) {
					return tok, true
				}
			}
		}
	}
	return "", false
}
