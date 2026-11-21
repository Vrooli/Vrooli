package credentials

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/reach"
)

// BindOptions are the deployment-independent parts of a lifecycle service.
type BindOptions struct {
	Store  Store
	Logger *log.Logger
	// ExternalProbe validates an operator-supplied external credential before
	// distribution. Nil selects the shape-only probe.
	ExternalProbe func(ctx context.Context, value string) error
	// RewrapAndRestore is the backup domain's proof for encryption/recovery
	// keys. Nil leaves that class fail-closed (the predecessor is never
	// retired without the proof).
	RewrapAndRestore func(ctx context.Context, ref domain.CredentialVersionRef) error
}

// ExternalHandoffInstruction is what an operator is told when the provider
// has no revocation API.
const ExternalHandoffInstruction = "revoke the predecessor credential at the provider, then resume this rotation with operator_confirmed=true"

// ShapeProbe is the provider-less validation of an operator-supplied value:
// non-empty, bounded, no NUL bytes. Scope validation against a provider
// needs a declared probe.
func ShapeProbe(_ context.Context, value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("value is empty")
	}
	if len(value) > domain.MaxSecretValueLength {
		return fmt.Errorf("value exceeds %d bytes", domain.MaxSecretValueLength)
	}
	if strings.ContainsRune(value, 0) {
		return errors.New("value contains null bytes")
	}
	return nil
}

// BindReach builds the lifecycle over the bound reach: argv-only runner,
// credential-authority metadata client, receipted target verbs, lifecycle
// restarts and the recovery client all on the same typed transport.
func BindReach(opts BindOptions, rr reach.Reach, target identity.TargetRef) (*Service, error) {
	argv := ReachArgvRunner{Reach: rr, Target: target}
	client, err := NewReachCredentialClient(rr, target)
	if err != nil {
		return nil, err
	}
	label := TargetLabel(target)
	service := &Service{
		Store:       opts.Store,
		Logger:      opts.Logger,
		Distributor: NewSSHDistributor(argv, client, label),
		Providers:   DefaultProviders(argv, label),
		Restarter:   ArgvRestarter{Runner: argv, Label: label},
		Recovery:    ClientRecovery{Client: client, Target: label},
	}
	applyDeclaredProviders(service, opts)
	return service, nil
}

// BindBridge builds the lifecycle over Bridge grants and dispatch. The
// database owner argv has no Bridge dispatch path yet, so that class is not
// registered and rotation refuses with a typed error naming the gap.
func BindBridge(opts BindOptions, client *BridgeClient, nodeLabel string) *Service {
	service := &Service{
		Store:       opts.Store,
		Logger:      opts.Logger,
		Distributor: &BridgeDistributor{Grants: client, Dispatch: client},
		Providers:   DefaultProviders(nil, nodeLabel),
	}
	delete(service.Providers, domain.CredentialClassGeneratedDatabasePassword)
	applyDeclaredProviders(service, opts)
	return service
}

func applyDeclaredProviders(service *Service, opts BindOptions) {
	probe := opts.ExternalProbe
	if probe == nil {
		probe = ShapeProbe
	}
	service.Providers[domain.CredentialClassExternalAPICredential] = &ExternalAPIProvider{Name: "external", Probe: probe, Instruction: ExternalHandoffInstruction}
	service.Providers[domain.CredentialClassEncryptionRecoveryKey] = &EncryptionRecoveryKeyProvider{RewrapAndRestore: opts.RewrapAndRestore}
}
