package consumeridentity

import (
	"errors"
	"strings"
	"time"
)

const (
	CreditAssertionVersion                = 1
	CreditAdmissionScope                  = "credits:admit"
	CreditGatewayAudience                 = "scenario:ai-gateway"
	CreditAssertionSource                 = CreditAssertionSourceAIDoor
	CreditAssertionSourceAIDoor           = "lpbs-aidoor"
	CreditAssertionSourceMeteredInference = "lpbs-metered-inference"
)

var ErrCreditAssertionMismatch = errors.New("credit assertion scope mismatch")

type CreditAssertionExpectation struct {
	BusinessAccountID string
	ActorID           string
	AppKey            string
	FeatureID         string
	MaxCredits        int64
	Audience          string
	OperationID       string
	Source            string
}

type CreditAssertionOwnerBinding struct {
	AppKey      string
	FeatureID   string
	OperationID string
	Source      string
}

func (s *Signer) SignCreditAssertion(expect CreditAssertionExpectation, ttl time.Duration) (string, time.Time, error) {
	if err := validateCreditAssertion(expect); err != nil {
		return "", time.Time{}, err
	}
	return s.SignWithTTL(Claims{
		Subject: expect.ActorID,
		UserID:  expect.ActorID,
		Credit: &CreditAssertion{
			Version: CreditAssertionVersion, BusinessAccountID: expect.BusinessAccountID,
			ActorID: expect.ActorID, AppKey: expect.AppKey, FeatureID: expect.FeatureID, MaxCredits: expect.MaxCredits, Audience: expect.Audience,
			OperationID: expect.OperationID, Source: expect.Source, Scopes: []string{CreditAdmissionScope},
		},
	}, ttl)
}

func VerifyCreditAssertion(verifier Verifier, token string, expect CreditAssertionExpectation) (Claims, error) {
	if err := validateCreditAssertion(expect); err != nil {
		return Claims{}, err
	}
	claims, err := verifier.Verify(token)
	if err != nil {
		return Claims{}, err
	}
	if err := ValidateCreditAssertionClaims(claims, expect); err != nil {
		return Claims{}, err
	}
	return claims, nil
}

func ValidateCreditAssertionClaims(claims Claims, expect CreditAssertionExpectation) error {
	if err := validateCreditAssertion(expect); err != nil {
		return err
	}
	a := claims.Credit
	if a == nil || a.Version != CreditAssertionVersion || a.BusinessAccountID != expect.BusinessAccountID ||
		a.ActorID != expect.ActorID || claims.Subject != expect.ActorID || a.AppKey != expect.AppKey || a.FeatureID != expect.FeatureID || a.MaxCredits != expect.MaxCredits ||
		a.Audience != expect.Audience || a.OperationID != expect.OperationID || a.Source != expect.Source ||
		!hasExactCreditAdmissionScope(a.Scopes) {
		return ErrCreditAssertionMismatch
	}
	return nil
}

// ValidateCreditAssertionOwnerClaims verifies a signed token against its
// scenario-declared app/feature owner and the request's exact operation. The
// account, actor, and cap are taken from the already signature-verified claim.
func ValidateCreditAssertionOwnerClaims(claims Claims, binding CreditAssertionOwnerBinding) error {
	a := claims.Credit
	if a == nil || a.AppKey != binding.AppKey || a.FeatureID != binding.FeatureID || a.OperationID != binding.OperationID || a.Source != binding.Source {
		return ErrCreditAssertionMismatch
	}
	return ValidateCreditAssertionClaims(claims, CreditAssertionExpectation{
		BusinessAccountID: a.BusinessAccountID, ActorID: a.ActorID, AppKey: binding.AppKey,
		FeatureID: binding.FeatureID, MaxCredits: a.MaxCredits, Audience: CreditGatewayAudience,
		OperationID: binding.OperationID, Source: binding.Source,
	})
}

func validateCreditAssertion(expect CreditAssertionExpectation) error {
	if strings.TrimSpace(expect.BusinessAccountID) == "" || strings.TrimSpace(expect.ActorID) == "" ||
		strings.TrimSpace(expect.AppKey) == "" || strings.TrimSpace(expect.FeatureID) == "" || expect.MaxCredits <= 0 || expect.Audience != CreditGatewayAudience ||
		strings.TrimSpace(expect.OperationID) == "" || !knownCreditAssertionSource(expect.Source) {
		return errors.New("complete LPBS credit assertion binding is required")
	}
	return nil
}

func knownCreditAssertionSource(source string) bool {
	switch source {
	case CreditAssertionSourceAIDoor, CreditAssertionSourceMeteredInference:
		return true
	default:
		return false
	}
}

func hasExactCreditAdmissionScope(scopes []string) bool {
	return len(scopes) == 1 && scopes[0] == CreditAdmissionScope
}
