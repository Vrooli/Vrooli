// Package credentials owns the transport-free credential metadata and
// provisioning boundary. The provision value is accepted only as an argument
// to the injected authority operation and is never returned by this package.
package credentials

import (
	"context"
	"errors"
	"strings"

	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	readiness "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/readiness"
)

type Service struct {
	ListFn      func(context.Context) ([]readiness.Credential, error)
	ProvisionFn func(context.Context, string, string, string) (credentialclient.ProvisionResponse, error)
	DiagnoseFn  func(context.Context) (credentialclient.DoctorResponse, error)
}

type Credential = readiness.Credential

func (s Service) List(ctx context.Context) ([]readiness.Credential, error) {
	if s.ListFn == nil {
		return nil, errors.New("credential listing is unavailable")
	}
	return s.ListFn(ctx)
}

func (s Service) Provision(ctx context.Context, logicalID, field, value string) (credentialclient.ProvisionResponse, error) {
	logicalID = strings.TrimSpace(logicalID)
	field = strings.TrimSpace(field)
	if field == "" {
		field = "value"
	}
	if logicalID == "" || strings.TrimSpace(value) == "" {
		return credentialclient.ProvisionResponse{}, errors.New("logical_id and a non-empty credential value are required")
	}
	if s.ProvisionFn == nil {
		return credentialclient.ProvisionResponse{}, errors.New("credential provisioning is unavailable")
	}
	return s.ProvisionFn(ctx, logicalID, field, value)
}

func (s Service) Diagnose(ctx context.Context) (credentialclient.DoctorResponse, error) {
	if s.DiagnoseFn == nil {
		return credentialclient.DoctorResponse{}, errors.New("credential diagnosis is unavailable")
	}
	return s.DiagnoseFn(ctx)
}
