package credentialclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/resources/securestore"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type InProcessOptions struct {
	Authority   *credentialauthority.Authority
	StateDir    string
	Descriptors func() ([]CredentialRef, error)
}

type inProcessClient struct {
	authority   *credentialauthority.Authority
	stateDir    string
	descriptors func() ([]CredentialRef, error)
}

func NewInProcess(options InProcessOptions) (Client, error) {
	if options.Authority == nil {
		return nil, fmt.Errorf("in-process credential authority is required")
	}
	return &inProcessClient{authority: options.Authority, stateDir: options.StateDir, descriptors: options.Descriptors}, nil
}

func (c *inProcessClient) Provision(_ context.Context, request ProvisionRequest) (ProvisionResponse, error) {
	identity, err := credentialauthority.ParseIdentity(request.Identity)
	if err != nil {
		return ProvisionResponse{}, err
	}
	if err := c.authority.Put(identity, request.Field, request.Value); err != nil {
		return ProvisionResponse{}, err
	}
	return ProvisionResponse{Identity: string(identity), Field: request.Field, Provider: c.authority.Provider(), Status: "provisioned"}, nil
}

func (c *inProcessClient) Status(_ context.Context, identity, field string) (CredentialStatus, error) {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return CredentialStatus{}, err
	}
	status := c.authority.Status(parsed, field)
	return CredentialStatus{Identity: string(status.Identity), Field: status.Field, Configured: status.Configured, Provider: status.Provider, ProviderState: string(status.ProviderState), ProviderDetail: status.ProviderDetail}, nil
}

func (c *inProcessClient) Resolve(_ context.Context, identity, field string) (string, error) {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return "", err
	}
	return c.authority.Resolve(parsed, field)
}

func (c *inProcessClient) Delete(_ context.Context, identity, field string) error {
	parsed, err := credentialauthority.ParseIdentity(identity)
	if err != nil {
		return err
	}
	return c.authority.Delete(parsed, field)
}

func (c *inProcessClient) List(_ context.Context) ([]CredentialRef, error) {
	if c.descriptors == nil {
		return []CredentialRef{}, nil
	}
	return c.descriptors()
}

func (c *inProcessClient) Doctor(ctx context.Context) (DoctorResponse, error) {
	diagnosis := securestore.Diagnose()
	descriptors, err := c.List(ctx)
	if err != nil {
		return DoctorResponse{}, err
	}
	response := DoctorResponse{Credentials: descriptors, Recovery: RecoveryStatus{Uncovered: []string{}}}
	response.Provider = ProviderDiagnosis{Platform: diagnosis.Platform, Adapter: diagnosis.Adapter, Backend: diagnosis.Backend, Condition: diagnosis.Condition, Available: diagnosis.Available, Writable: diagnosis.Writable, Explanation: diagnosis.Explanation, Fix: diagnosis.Fix}
	if c.stateDir != "" {
		if receipt, found, receiptErr := credentialauthority.ReadRecoveryReceipt(c.stateDir); receiptErr == nil && found {
			response.Recovery.ReceiptExists = true
			response.Recovery.ExportedAt = receipt.ExportedAt.Format("2006-01-02T15:04:05Z07:00")
			response.Recovery.EntryCount = len(receipt.Entries)
			for _, descriptor := range descriptors {
				if !descriptor.Required || !credentialConfigured(c.authority, descriptor) {
					continue
				}
				identity, parseErr := credentialauthority.ParseIdentity(descriptor.LogicalID)
				if parseErr != nil || !receipt.Covers(identity, descriptor.Field) {
					response.Recovery.Uncovered = append(response.Recovery.Uncovered, descriptor.LogicalID+":"+descriptor.Field)
				}
			}
		} else {
			for _, descriptor := range descriptors {
				if descriptor.Required && credentialConfigured(c.authority, descriptor) {
					response.Recovery.Uncovered = append(response.Recovery.Uncovered, descriptor.LogicalID+":"+descriptor.Field)
				}
			}
		}
	}
	return response, nil
}

func credentialConfigured(authority *credentialauthority.Authority, descriptor CredentialRef) bool {
	identity, err := credentialauthority.ParseIdentity(descriptor.LogicalID)
	return err == nil && authority.Status(identity, descriptor.Field).Configured
}

func (c *inProcessClient) KeyringInspect(_ context.Context, path string) (KeyringReport, error) {
	report, err := c.authority.KeyringInspect(path)
	return KeyringReport{Path: report.Path, Loadable: report.Loadable, Repaired: report.Repaired}, err
}

func (c *inProcessClient) KeyringRepair(_ context.Context, path string) (KeyringReport, error) {
	report, err := c.authority.KeyringRepair(path)
	return KeyringReport{Path: report.Path, Loadable: report.Loadable, Repaired: report.Repaired}, err
}

func (c *inProcessClient) RecoveryExport(_ context.Context, request RecoveryExportRequest) (RecoveryExportResponse, error) {
	if len(request.Entries) == 0 {
		return RecoveryExportResponse{}, fmt.Errorf("recovery export requires at least one credential")
	}
	entries := make([]credentialauthority.RecoveryEntry, 0, len(request.Entries))
	for _, ref := range request.Entries {
		identity, err := credentialauthority.ParseIdentity(ref.LogicalID)
		if err != nil {
			return RecoveryExportResponse{}, err
		}
		entries = append(entries, credentialauthority.RecoveryEntry{Identity: identity, Field: ref.Field})
	}
	bundle, err := c.authority.ExportRecovery(entries, request.Passphrase)
	if err != nil {
		return RecoveryExportResponse{}, err
	}
	if strings.TrimSpace(request.OutputPath) == "" {
		return RecoveryExportResponse{}, fmt.Errorf("recovery output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(request.OutputPath), 0o700); err != nil {
		return RecoveryExportResponse{}, err
	}
	if err := os.WriteFile(request.OutputPath, bundle, 0o600); err != nil {
		return RecoveryExportResponse{}, err
	}
	if c.stateDir != "" {
		_ = credentialauthority.WriteRecoveryReceipt(c.stateDir, request.OutputPath, entries, time.Now())
	}
	return RecoveryExportResponse{Path: request.OutputPath, EntryCount: len(entries)}, nil
}

func (c *inProcessClient) RecoveryVerify(_ context.Context, request RecoveryVerifyRequest) (RecoveryVerifyResponse, error) {
	bundle, err := os.ReadFile(request.InputPath)
	if err != nil {
		return RecoveryVerifyResponse{}, err
	}
	manifest, err := credentialauthority.InspectRecovery(bundle, request.Passphrase)
	if err != nil {
		return RecoveryVerifyResponse{}, err
	}
	entries := make([]string, 0, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		entries = append(entries, string(entry.Identity)+":"+entry.Field)
	}
	return RecoveryVerifyResponse{Version: manifest.Version, Entries: entries}, nil
}

func (c *inProcessClient) RecoveryRestore(_ context.Context, request RecoveryRestoreRequest) error {
	bundle, err := os.ReadFile(request.InputPath)
	if err != nil {
		return err
	}
	return c.authority.RestoreRecovery(bundle, request.Passphrase)
}

func (c *inProcessClient) StoreStatus(context.Context) (StoreStatus, error) {
	status, err := securestore.DescribeStore()
	if err != nil {
		return StoreStatus{}, err
	}
	return StoreStatus{Path: status.Path, Initialized: status.Initialized, Unlocked: status.Unlocked, Entries: status.Entries, Active: status.Active}, nil
}

var _ Client = (*inProcessClient)(nil)
