//go:build !linux && !darwin

package auth

import "errors"

func currentLocalPrincipal() (string, error) {
	return "", errors.New("local principal unsupported; use the token-file fallback")
}
