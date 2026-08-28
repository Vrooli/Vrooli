package main

import (
	"fmt"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	jwtSecretIdentity = "vrooli/calendar"
	jwtSecretField    = "jwt-secret"
	// 32 bytes is the HMAC-SHA256 block size, so a longer secret buys no
	// additional strength and a shorter one throws some away.
	jwtSecretBytes = 32
)

// jwtSecretResolver is the seam tests substitute. Production always uses the
// credential authority.
var jwtSecretResolver = resolveJWTSecretFromAuthority

// resolveJWTSecret returns the signing secret for Calendar API tokens.
//
// The secret is declared with provisioning "generated" because there is nowhere
// an operator could obtain it: it signs this scenario's own tokens and has no
// external issuer. Asking a person for it is asking a question with no answer,
// so the scenario mints one on first start and stores it in the credential
// authority, where recovery and diagnosis can see it like any other credential.
//
// It is resolved in process rather than injected through the environment. A
// value in the environment is readable at /proc/<pid>/environ and is inherited
// by every subprocess this scenario spawns, and Calendar spawns several.
func resolveJWTSecret() (string, error) { return jwtSecretResolver() }

func resolveJWTSecretFromAuthority() (string, error) {
	identity, err := credentialauthority.ParseIdentity(jwtSecretIdentity)
	if err != nil {
		return "", fmt.Errorf("parse calendar credential identity: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}

	return authority.ResolveOrMint(identity, jwtSecretField, nil, generateJWTSecret)
}

func generateJWTSecret() (string, error) {
	secret, err := credentialauthority.RandomBase64(jwtSecretBytes)
	if err != nil {
		return "", fmt.Errorf("generate calendar signing secret: %w", err)
	}
	return secret, nil
}
