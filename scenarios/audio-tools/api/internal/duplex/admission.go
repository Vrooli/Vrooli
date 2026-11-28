package duplex

import (
	"context"
	"errors"

	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/providercatalog"
	"github.com/vrooli/api-core/identity"
)

var (
	ErrRemoteProcessingDenied = errors.New("audio-tools/duplex: remote processing is not allowed")
	ErrConsentRequired        = errors.New("audio-tools/duplex: explicit consent is required")
	ErrProviderUnavailable    = errors.New("audio-tools/duplex: provider or model is not available")
	ErrSessionBudgetRequired  = errors.New("audio-tools/duplex: a positive per-session budget is required")
	ErrSessionBudgetTooLarge  = errors.New("audio-tools/duplex: per-session budget exceeds the configured maximum")
	ErrConsentInvalid         = errors.New("audio-tools/duplex: consent is not valid for this source")
)

// ConsentVerifier delegates receipt authority to AI Gateway. Implementations
// must check owner, exact source, expiry and revocation; Audio Tools stores no grants.
type ConsentVerifier interface {
	Verify(context.Context, string, string, string) error
}

// ProviderEligibility resolves the exact provider/model pair against the
// active model and provider catalogs. It must reject planned, watch, or
// unknown entries. This is deliberately injected so admission policy does not
// duplicate catalog tables.
type ProviderEligibility func(providerID, modelID string) bool

// ActiveConsentAdmission allows a coordinator to recheck revocable consent
// while a remote session is active. Rechecks must use the original verified
// identity and source-bound receipt and fail closed on authority errors.
type ActiveConsentAdmission interface {
	RevalidateConsent(context.Context, Request) error
}

// ActiveConsentRequirement identifies requests whose provider declaration
// requires consent to be revalidated during a live session. It is separate
// from ActiveConsentAdmission so Coordinator can reject a missing revalidator
// before creating a hold or opening the provider.
type ActiveConsentRequirement interface {
	RequiresActiveConsentRevalidation(Request) bool
}

// AdmissionPolicy applies privacy consent and per-session/process-wide spend
// limits before any provider connection is opened. Transport handlers must
// still authenticate the caller and treat the consent field as an explicit
// user decision; this internal policy is not an identity verifier.
type AdmissionPolicy struct {
	budget             *Budget
	maxSessionBudget   uint64
	providerIsEligible ProviderEligibility
	consentVerifier    ConsentVerifier
	consentRequirement func(string) (providercatalog.ConsentRequirement, bool)
}

func (p *AdmissionPolicy) WithConsentVerifier(verifier ConsentVerifier) *AdmissionPolicy {
	if p != nil {
		p.consentVerifier = verifier
	}
	return p
}

func NewAdmissionPolicy(budget *Budget, maxSessionBudgetMicros uint64, eligible ProviderEligibility) *AdmissionPolicy {
	return &AdmissionPolicy{
		budget:             budget,
		maxSessionBudget:   maxSessionBudgetMicros,
		providerIsEligible: eligible,
		consentRequirement: providercatalog.ConsentRequirementFor,
	}
}

func (p *AdmissionPolicy) RequiresActiveConsentRevalidation(req Request) bool {
	if p == nil || p.consentRequirement == nil {
		return false
	}
	requirement, ok := p.consentRequirement(req.ProviderID)
	return ok && requirement.RevalidateActiveSessions
}

// Authorize implements the policy validation interface. Coordinators use
// Reserve for this implementation, so metered sessions hold budget before
// contacting the provider.
func (p *AdmissionPolicy) Authorize(ctx context.Context, req Request) error {
	return p.validate(ctx, req)
}

// RevalidateConsent repeats the provider, budget-bound and consent checks
// without reserving a second spend hold. It is used only after admission.
func (p *AdmissionPolicy) RevalidateConsent(ctx context.Context, req Request) error {
	return p.validate(ctx, req)
}

// Reserve performs all fail-closed checks before creating an atomic hold.
func (p *AdmissionPolicy) Reserve(ctx context.Context, req Request) (Reservation, error) {
	if err := p.validate(ctx, req); err != nil {
		return nil, err
	}
	if p.budget == nil {
		return nil, ErrNilAdmission
	}
	return p.budget.Reserve(req.BudgetMicros)
}

func (p *AdmissionPolicy) validate(ctx context.Context, req Request) error {
	if err := p.validateAdmissionRequest(ctx, req); err != nil {
		return err
	}
	return p.validateProviderConsent(ctx, req)
}

func (p *AdmissionPolicy) validateAdmissionRequest(ctx context.Context, req Request) error {
	if ctx == nil {
		return ErrNilContext
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if p == nil || p.providerIsEligible == nil {
		return ErrNilAdmission
	}
	if !req.RemoteAllowed {
		return ErrRemoteProcessingDenied
	}
	if !req.Consent {
		return ErrConsentRequired
	}
	if req.ProviderID == "" || req.Model == "" || !p.providerIsEligible(req.ProviderID, req.Model) {
		return ErrProviderUnavailable
	}
	if req.BudgetMicros == 0 || p.maxSessionBudget == 0 {
		return ErrSessionBudgetRequired
	}
	if req.BudgetMicros > p.maxSessionBudget {
		return ErrSessionBudgetTooLarge
	}
	return nil
}

func (p *AdmissionPolicy) validateProviderConsent(ctx context.Context, req Request) error {
	if p == nil || p.consentRequirement == nil {
		return ErrConsentInvalid
	}
	requirement, required := p.consentRequirement(req.ProviderID)
	if !required {
		return nil
	}
	if requirement.Source == "" || req.ConsentReceipt == "" {
		return ErrConsentInvalid
	}
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || !principal.Verified || principal.Kind != identity.ActorHuman || principal.Source != identity.SourceConsumerAccessToken || principal.Subject == "" || p.consentVerifier == nil {
		return ErrConsentInvalid
	}
	if err := p.consentVerifier.Verify(ctx, principal.Subject, req.ConsentReceipt, string(requirement.Source)); err != nil {
		return ErrConsentInvalid
	}
	return nil
}

// CatalogEligibility is the default runtime eligibility predicate for an
// exact provider/model pair. Both declarations must be active; a credential
// or a successful health check alone cannot promote a planned model.
func CatalogEligibility(providerID, modelID string) bool {
	if modelID == "" {
		return false
	}
	provider, ok := providercatalog.ByID(providerID)
	if !ok || provider.Status != providercatalog.StatusActive || provider.Kind != providercatalog.KindFullDuplex {
		return false
	}
	hasProviderCapability := false
	for _, capability := range provider.Capabilities {
		if capability == providercatalog.CapabilityDuplex {
			hasProviderCapability = true
			break
		}
	}
	if !hasProviderCapability {
		return false
	}
	model, ok := modelcatalog.Default().ByID(modelID)
	if !ok || model.Capability != modelcatalog.CapabilityDuplex || model.Readiness != modelcatalog.ReadinessAvailable {
		return false
	}
	for _, declaredProvider := range model.ProviderIDs {
		if declaredProvider == providerID {
			return true
		}
	}
	return false
}
