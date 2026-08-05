package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/packages/credentialclient-go"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// CredentialRecoveryCheck periodically reports whether the host's verified
// recovery receipt still covers its configured credentials. It is deliberately
// report-only: the check never exports a bundle or mutates the authority.
type CredentialRecoveryCheck struct {
	doctor func(context.Context) (credentialclient.DoctorResponse, error)
}

type CredentialRecoveryCheckOption func(*CredentialRecoveryCheck)

func WithCredentialDoctor(doctor func(context.Context) (credentialclient.DoctorResponse, error)) CredentialRecoveryCheckOption {
	return func(check *CredentialRecoveryCheck) { check.doctor = doctor }
}

func NewCredentialRecoveryCheck(opts ...CredentialRecoveryCheckOption) *CredentialRecoveryCheck {
	check := &CredentialRecoveryCheck{
		doctor: func(ctx context.Context) (credentialclient.DoctorResponse, error) {
			client, err := autohealCredentialClient()
			if err != nil {
				return credentialclient.DoctorResponse{}, err
			}
			return client.Doctor(ctx)
		},
	}
	for _, opt := range opts {
		opt(check)
	}
	return check
}

func (c *CredentialRecoveryCheck) ID() string                 { return "infra-credential-recovery" }
func (c *CredentialRecoveryCheck) Title() string              { return "Credential Recovery Coverage" }
func (c *CredentialRecoveryCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *CredentialRecoveryCheck) IntervalSeconds() int       { return 3600 }
func (c *CredentialRecoveryCheck) Platforms() []platform.Type { return nil }
func (c *CredentialRecoveryCheck) Description() string {
	return "Reports whether a verified recovery bundle covers configured credentials"
}
func (c *CredentialRecoveryCheck) Importance() string {
	return "A missing or stale recovery bundle can make operator credentials irrecoverable"
}

func (c *CredentialRecoveryCheck) Run(ctx context.Context) checks.Result {
	started := time.Now()
	result := checks.Result{CheckID: c.ID(), Status: checks.StatusOK, Timestamp: started, Details: map[string]interface{}{}}
	diagnosis, err := c.doctor(ctx)
	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "credential recovery coverage could not be checked"
		result.Details["error"] = err.Error()
		result.Duration = time.Since(started)
		return result
	}
	result.Details["provider"] = diagnosis.Provider.Backend
	result.Details["providerCondition"] = diagnosis.Provider.Condition
	result.Details["receiptExists"] = diagnosis.Recovery.ReceiptExists
	result.Details["entryCount"] = diagnosis.Recovery.EntryCount
	result.Details["uncovered"] = append([]string(nil), diagnosis.Recovery.Uncovered...)
	switch {
	case !diagnosis.Recovery.ReceiptExists:
		result.Status = checks.StatusWarning
		result.Message = "no credential recovery bundle has been recorded"
	case len(diagnosis.Recovery.Uncovered) > 0:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("credential recovery bundle misses %d configured credential(s)", len(diagnosis.Recovery.Uncovered))
	default:
		result.Message = "credential recovery bundle covers configured credentials"
	}
	result.Duration = time.Since(started)
	return result
}
