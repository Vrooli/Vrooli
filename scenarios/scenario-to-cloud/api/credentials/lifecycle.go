package credentials

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"

	"github.com/google/uuid"
)

// Restarter restarts consumers on the target after a non-dual-accept
// rotation so they re-read the new version. Nil means consumers re-read on
// their own schedule and the operation records that no restart was issued.
type Restarter interface {
	Restart(ctx context.Context, target Target, consumers []string) error
}

// RecoveryClient re-provisions a replacement host from an encrypted recovery
// bundle. Verify proves the bundle opens and names what it covers; Restore
// writes it into the replacement host's authority. The passphrase is passed
// in memory and travels on standard input only.
type RecoveryClient interface {
	Verify(ctx context.Context, bundleRef, passphrase string) ([]domain.CredentialDescriptor, error)
	Restore(ctx context.Context, bundleRef, passphrase string) error
}

// Service owns the credential lifecycle for cloud deployments. It composes
// the ledger, one transport distributor and the per-class providers; it is
// the only writer of bindings and operations.
type Service struct {
	Store       Store
	Distributor Distributor
	Providers   Providers
	Restarter   Restarter
	Recovery    RecoveryClient
	// Generate mints a value for a generated class. Defaults to 32 random
	// bytes, base64url.
	Generate func(binding domain.CredentialBinding) (string, error)
	Now      func() time.Time
	NewID    func() string
	Logger   *log.Logger
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func (s *Service) newID() string {
	if s.NewID != nil {
		return s.NewID()
	}
	return uuid.NewString()
}

func (s *Service) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Printf(format, args...)
	}
}

func (s *Service) generate(binding domain.CredentialBinding) (string, error) {
	if s.Generate != nil {
		return s.Generate(binding)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *Service) provider(class domain.CredentialClass) (Provider, error) {
	if p, ok := s.Providers[class]; ok && p != nil {
		return p, nil
	}
	return nil, newError(CodeRotationRefused, "no provider is declared for credential class "+string(class)).
		WithDetail("class", string(class)).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "doc", Reference: "docs/reference/credential-lifecycle.md", Label: "Declare the provider hook for this class"})
}
