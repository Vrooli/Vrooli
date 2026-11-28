//go:build lpbs_rehearsal

package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/securevalue"
)

const (
	lpbsRehearsalAuthorityPassphrase = "CF2 local fork synthetic custody; never value-bearing"
	lpbsRehearsalBuild               = true
)

func init() {
	administration.SetCredentialAuthorityProvider(openLPBSRehearsalAuthority)
}

func openLPBSRehearsalAuthority() (*credentialauthority.Authority, error) {
	path := strings.TrimSpace(os.Getenv("LPBS_REHEARSAL_CREDENTIAL_PATH"))
	if !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.Contains(path, "\x00") {
		return nil, errors.New("LPBS rehearsal requires an absolute isolated credential file")
	}
	authority, err := credentialauthority.OpenEncryptedFile(path, lpbsRehearsalAuthorityPassphrase)
	if err != nil {
		return nil, err
	}
	generated := []struct {
		field string
		mint  func() (string, error)
	}{
		{"session-secret", func() (string, error) { return credentialauthority.RandomBase64(32) }},
		{"api-key-encryption-key", rehearsalSecureValueKey},
		{"remote-profile-encryption-key", rehearsalSecureValueKey},
		{"admin-mfa-encryption-key", rehearsalSecureValueKey},
		{"metrics-reader-token", administration.MintReaderToken},
		{"consumer-auth-private-key", rehearsalConsumerPrivateKey},
	}
	identity, err := credentialauthority.ParseIdentity("vrooli/landing-page-business-suite")
	if err != nil {
		return nil, err
	}
	for _, item := range generated {
		if _, err := authority.Resolve(identity, item.field); err == nil {
			continue
		} else if !errors.Is(err, credentialauthority.ErrUnconfigured) {
			return nil, err
		}
		value, err := item.mint()
		if err != nil {
			return nil, err
		}
		if err := authority.Put(identity, item.field, value); err != nil {
			return nil, err
		}
	}
	return authority, nil
}

func rehearsalSecureValueKey() (string, error) {
	ring, err := securevalue.NewRing()
	if err != nil {
		return "", err
	}
	return ring.Marshal()
}

func rehearsalConsumerPrivateKey() (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), nil
}
