package control

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"

	"device-control/internal/desktophelper"
	"device-control/internal/sessions"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type desktopPinWitness bool

func (w desktopPinWitness) Minted(credentialauthority.Identity, string) (bool, error) {
	return bool(w), nil
}

// The public pin is recorded by ProvisionConfig before this key can be used for
// admission. An interrupted first provision can adopt the stored key next time.
func (desktopPinWitness) RecordMint(credentialauthority.Identity, string) error { return nil }

// ProvisionDesktopAdmission stores generated signing material in the shared
// credential authority, never a config file or process environment. The returned
// owner is server-only and still requires kernel-peer authentication to acquire.
func (s *Service) ProvisionDesktopAdmission(path, deviceID string, config desktophelper.Config) (*LocalDesktopAdmission, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return nil, err
	}
	return s.provisionDesktopAdmission(authority, path, deviceID, config)
}

func (s *Service) provisionDesktopAdmission(authority *credentialauthority.Authority, path, deviceID string, config desktophelper.Config) (*LocalDesktopAdmission, error) {
	if s == nil || deviceID == "" {
		return nil, sessions.ErrDesktopAdmission
	}
	// Opaque stable scope avoids placing host/session labels in credential paths.
	encoded, err := json.Marshal(struct {
		Surface any
		Session string
	}{config.Surface, config.SessionID})
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(encoded)
	identity, err := credentialauthority.ParseIdentity("scenario/device-control/desktop/" + hex.EncodeToString(digest[:]))
	if err != nil {
		return nil, err
	}
	var key ed25519.PrivateKey
	provisioned, err := desktophelper.ProvisionConfig(path, config, func(previous string) (string, error) {
		value, err := authority.ResolveOrMint(identity, "signing-key", desktopPinWitness(previous != ""), func() (string, error) {
			_, private, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				return "", err
			}
			return base64.StdEncoding.EncodeToString(private.Seed()), nil
		})
		if err != nil {
			return "", err
		}
		seed, err := base64.StdEncoding.DecodeString(value)
		if err != nil || len(seed) != ed25519.SeedSize {
			return "", sessions.ErrDesktopAdmission
		}
		key = ed25519.NewKeyFromSeed(seed)
		return base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)), nil
	})
	if err != nil {
		return nil, err
	}
	return NewLocalDesktopAdmission(s, key, deviceID, provisioned.Surface, func(ctx context.Context) (desktophelper.Registration, error) {
		return desktophelper.ReadRegistration(ctx, path)
	})
}
