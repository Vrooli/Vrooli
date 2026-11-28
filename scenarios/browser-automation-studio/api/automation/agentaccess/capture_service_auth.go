package agentaccess

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/principalrole"
)

var (
	ErrCaptureLaneFull    = errors.New("BAS system capture lane is full")
	ErrCaptureRateLimited = errors.New("BAS system capture request rate exceeded")
)

const (
	CaptureServicePrincipal = "experience-manager-bas-capture"
	CaptureServiceAudience  = "scenario-authenticator:browser-automation-studio"
	CaptureServiceScope     = "browser-automation-studio:capture"
	CaptureServiceResource  = "browser-automation-studio"
	CaptureServiceSuffix    = "-bas-capture"
)

var CaptureServiceRole = principalrole.NewRole(
	CaptureServiceAudience,
	CaptureServiceScope,
	CaptureServiceResource,
	CaptureServiceSuffix,
	"BAS capture",
)

// CaptureServiceVerifier builds a verifier pinned to the BAS resource's
// exclusive audience and scenario-authenticator issuer.
func CaptureServiceVerifier() authn.TokenVerifier {
	return CaptureServiceVerifierWithConfig(authn.JWTConfig{})
}

// CaptureServiceVerifierWithConfig permits a trusted JWKS source in tests;
// the role still owns issuer, actor kind and audience constraints.
func CaptureServiceVerifierWithConfig(cfg authn.JWTConfig) authn.TokenVerifier {
	cfg.Audience = CaptureServiceAudience
	cfg.ExclusiveAudience = true
	cfg.Kind = apiidentity.ActorService
	cfg.Issuer = "scenario-authenticator"
	return authn.NewScenarioAuthenticatorProvider(cfg)
}

// CaptureServiceAdmission verifies the exact Experience Manager principal and
// reserves one bounded slot for accountless system captures.
type CaptureServiceAdmission struct {
	verifier authn.TokenVerifier
	slots    chan struct{}
	rateMu   sync.Mutex
	rateMax  int
	rateSpan time.Duration
	rateAt   time.Time
	rateUsed int
	now      func() time.Time
}

func NewCaptureServiceAdmission(verifier authn.TokenVerifier, maxConcurrent int) (*CaptureServiceAdmission, error) {
	return NewCaptureServiceAdmissionWithLimits(verifier, maxConcurrent, 12, time.Minute)
}

func NewCaptureServiceAdmissionWithLimits(verifier authn.TokenVerifier, maxConcurrent, maxRequests int, window time.Duration) (*CaptureServiceAdmission, error) {
	if verifier == nil {
		return nil, errors.New("BAS capture service verifier is required")
	}
	if maxConcurrent < 1 {
		return nil, errors.New("BAS capture service lane limit must be positive")
	}
	if maxRequests < 1 || window <= 0 {
		return nil, errors.New("BAS capture service request-rate limit must be positive")
	}
	return &CaptureServiceAdmission{verifier: verifier, slots: make(chan struct{}, maxConcurrent), rateMax: maxRequests, rateSpan: window, now: time.Now}, nil
}

// Authenticate checks the exact service role without consuming rate or
// concurrency capacity. CaptureService uses this to refuse service-role reads.
func (a *CaptureServiceAdmission) Authenticate(ctx context.Context, headers http.Header) (apiidentity.Principal, error) {
	if a == nil || a.verifier == nil {
		return apiidentity.Principal{}, errors.New("BAS capture service admission unavailable")
	}
	token, found := strings.CutPrefix(strings.TrimSpace(headers.Get("Authorization")), "Bearer ")
	token = strings.TrimSpace(token)
	if !found || token == "" || strings.ContainsAny(token, " \t\r\n") {
		return apiidentity.Principal{}, errors.New("BAS capture service bearer token required")
	}
	principal, err := a.verifier.Verify(ctx, token)
	if err != nil {
		return apiidentity.Principal{}, err
	}
	source, err := CaptureServiceRole.Source(principal)
	if err != nil {
		return apiidentity.Principal{}, err
	}
	if source != "experience-manager" || principal.Subject != CaptureServicePrincipal || principal.Issuer != "scenario-authenticator" {
		return apiidentity.Principal{}, errors.New("BAS capture service principal is not authorized")
	}
	return principal, nil
}

// RejectServiceToken prevents this capture-only role from being treated as a
// general BAS artifact reader. Other callers continue through the existing
// artifact owner route and its current authentication context.
func (a *CaptureServiceAdmission) RejectServiceToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := a.Authenticate(r.Context(), r.Header)
		if err == nil && principal.Subject == CaptureServicePrincipal &&
			principal.Audience == CaptureServiceAudience && len(principal.Scopes) == 1 && principal.Scopes[0] == CaptureServiceScope {
			http.Error(w, "BAS capture-only service identity cannot read artifacts", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// IsLimitError classifies finite service-lane exhaustion for transport status
// mapping without exposing implementation details to the handler.
func (a *CaptureServiceAdmission) IsLimitError(err error) bool {
	return errors.Is(err, ErrCaptureLaneFull) || errors.Is(err, ErrCaptureRateLimited)
}

// Admit verifies the bearer and returns a release function after reserving the
// bounded lane. Authorization errors and lane exhaustion fail closed.
func (a *CaptureServiceAdmission) Admit(ctx context.Context, headers http.Header) (apiidentity.Principal, func(), error) {
	if a == nil || a.verifier == nil || a.slots == nil {
		return apiidentity.Principal{}, nil, errors.New("BAS capture service admission unavailable")
	}
	principal, err := a.Authenticate(ctx, headers)
	if err != nil {
		return apiidentity.Principal{}, nil, err
	}
	select {
	case a.slots <- struct{}{}:
		if !a.allowRate(a.now()) {
			<-a.slots
			return apiidentity.Principal{}, nil, ErrCaptureRateLimited
		}
		return principal, func() { <-a.slots }, nil
	default:
		return apiidentity.Principal{}, nil, ErrCaptureLaneFull
	}
}

func (a *CaptureServiceAdmission) allowRate(now time.Time) bool {
	a.rateMu.Lock()
	defer a.rateMu.Unlock()
	if a.rateAt.IsZero() || now.Sub(a.rateAt) >= a.rateSpan || now.Before(a.rateAt) {
		a.rateAt = now
		a.rateUsed = 0
	}
	if a.rateUsed >= a.rateMax {
		return false
	}
	a.rateUsed++
	return true
}
