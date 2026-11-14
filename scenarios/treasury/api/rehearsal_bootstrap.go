//go:build treasury_rehearsal

package main

// This file is excluded from every normal Treasury build. A lifecycle-managed
// local-fork rehearsal compiles this tag into its independent artifact, keeping
// synthetic custody separate from the host authority and production startup.

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"treasury/internal/ledger"
)

const rehearsalAuthorityPassphrase = "CF2 local fork synthetic custody; never value-bearing"

func init() { openCredentialAuthority = rehearsalAuthority }

func rehearsalAuthority() (*credentialauthority.Authority, error) {
	for _, key := range []string{"TREASURY_OPERATOR_TOKEN", "TREASURY_MANDATE_SIGNING_KEY", "API_TOKEN"} {
		if os.Getenv(key) != "" {
			return nil, fmt.Errorf("rehearsal refuses ambient credential %s", key)
		}
	}
	for _, key := range []string{"TREASURY_BASE_RPC_URL", "MONEY_LEDGER_API_URL", "SCENARIO_AUTHENTICATOR_API_URL"} {
		u, err := url.Parse(os.Getenv(key))
		if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.User != nil || u.Port() == "" {
			return nil, fmt.Errorf("rehearsal requires an explicit loopback %s", key)
		}
	}
	dir := os.Getenv("TREASURY_REHEARSAL_DIR")
	if !filepath.IsAbs(dir) || strings.Contains(dir, "\x00") {
		return nil, errors.New("rehearsal requires an absolute isolated custody directory")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	authority, err := credentialauthority.OpenEncryptedFile(filepath.Join(dir, "synthetic-custody.enc"), rehearsalAuthorityPassphrase)
	if err != nil {
		return nil, err
	}
	for _, item := range []struct{ identity, field, value string }{
		{"vrooli/treasury", "operator-token", "cf2-local-operator-only"},
		{"vrooli/treasury", "mandate-signing-key", "cf2-local-mandate-signing-key-only"},
		{ledger.TreasuryServiceCredentialIdentity, "credential", "cf2-local-treasury-service-only"},
		{"vrooli/landing-page-business-suite", "service-secret", "cf2-local-lpbs-service-only"},
	} {
		identity, err := credentialauthority.ParseIdentity(item.identity)
		if err != nil {
			return nil, err
		}
		if err := authority.Put(identity, item.field, item.value); err != nil {
			return nil, err
		}
	}
	return authority, nil
}
