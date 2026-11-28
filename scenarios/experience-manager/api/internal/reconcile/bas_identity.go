package reconcile

import (
	"context"
	"errors"
	"time"

	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	basCapturePrincipal          = "experience-manager-bas-capture"
	basCaptureAudience           = "scenario-authenticator:browser-automation-studio"
	basCaptureScope              = "browser-automation-studio:capture"
	basCaptureCredentialIdentity = "scenario-authenticator/service-principal/experience-manager-bas-capture/browser-automation-studio"
)

// NewBASCaptureTokens creates a lazily exchanged, exact-identity token source.
// Its credential is reread from Credential Authority on each exchange, so
// operator rotation and revocation take effect without an EM restart.
func NewBASCaptureTokens(options ...servicetoken.Option) *servicetoken.Tokens {
	return servicetoken.New(servicetoken.Config{
		Name: "Experience Manager BAS capture",
		Credential: func(context.Context) (string, error) {
			authority, err := credentialauthority.Default()
			if err != nil {
				return "", err
			}
			identity, err := credentialauthority.ParseIdentity(basCaptureCredentialIdentity)
			if err != nil {
				return "", err
			}
			return authority.Require(identity, "credential")
		},
		Expect:        servicetoken.ExpectIdentity(basCapturePrincipal, basCaptureAudience, basCaptureScope),
		RefreshMargin: 30 * time.Second,
		MinRemaining:  30 * time.Second,
		Describe:      describeBASCaptureIdentityFailure,
	}, options...)
}

func describeBASCaptureIdentityFailure(kind, _ error) error {
	switch kind {
	case servicetoken.ErrNotIssued:
		return servicetoken.Kind(kind, errors.New("BAS capture service credential is unavailable"))
	case servicetoken.ErrExchangeFailed:
		return servicetoken.Kind(kind, errors.New("BAS capture service credential exchange failed"))
	case servicetoken.ErrNoUsableToken, servicetoken.ErrIdentityMismatch:
		return servicetoken.Kind(kind, errors.New("BAS capture service identity is invalid"))
	default:
		return servicetoken.Kind(kind, errors.New("BAS capture service identity is unavailable"))
	}
}
