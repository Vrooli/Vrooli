// Package akash adapts an Akash testnet lease manager to Compute Manager's
// provider contract. The client is injected by the network integration owner;
// this package owns typed requests, durable target references and effect gate
// checks without holding wallet material.
package akash

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"compute-manager/internal/provider"
	akashclient "compute-manager/internal/provider/akash/client"
	"github.com/vrooli/vrooli/packages/provider-effect-gate"
)

const Name = "akash"

// TestnetWalletCredentialRef is a reference only. The corresponding value, if
// one is ever commissioned, belongs exclusively to Credential Authority.
const TestnetWalletCredentialRef = "credential-authority://vrooli/compute-manager/akash-testnet-wallet"

var ErrClientUnavailable = errors.New("Akash testnet client is not configured")

type Config struct {
	Network             string `json:"network"`
	ChainID             string `json:"chain_id"`
	Endpoint            string `json:"endpoint"`
	ProviderAddress     string `json:"provider_address"`
	WalletCredentialRef string `json:"wallet_credential_ref"`
}

// LeaseRequest contains only non-secret request facts. IntentID is the
// record-first idempotency anchor supplied by Compute Manager.
type LeaseRequest struct {
	Network             string
	ChainID             string
	Endpoint            string
	Region              string
	Offer               string
	Image               string
	IntentID            string
	ProviderAddress     string
	WalletCredentialRef string
}

// LeaseReceipt is the provider's stable target reference. It is returned only
// after the client confirms a lease; private wallet material is never included.
type LeaseReceipt struct {
	DeploymentID string
	LeaseID      string
	Provider     string
	Endpoint     string
	CreatedAt    time.Time
	IntentID     string
}

type Client interface {
	CreateLease(context.Context, LeaseRequest) (LeaseReceipt, error)
	GetLease(context.Context, string) (LeaseReceipt, error)
	ListLeases(context.Context) ([]LeaseReceipt, error)
	ReleaseLease(context.Context, string) error
}

type Adapter struct {
	Config Config
	Gate   *effectgate.Gate
	Client Client
	// WalletConfigured reads Credential Authority presence metadata only. It
	// never resolves key material, and a missing/unavailable callback closes
	// all Akash mutations even when a test accidentally enables live mode.
	WalletConfigured func(context.Context) (bool, error)
	// SandboxClient is the typed, source-backed Akash query/lease client. The
	// legacy Client remains only for the preceding adapter contract; new runtime
	// wiring uses SandboxClient and exact, persisted lease references.
	SandboxClient SandboxClient
	Now           func() time.Time
}

// SandboxClient is the concrete Akash network contract, kept separate from
// Compute Manager's provider-neutral interface. It preserves Akash's complete
// bid tuple and on-chain denomination without converting it into a quote.
type SandboxClient interface {
	CreateLease(context.Context, akashclient.CreateRequest) (akashclient.Lease, error)
	GetLease(context.Context, akashclient.LeaseReference) (akashclient.Lease, error)
	ListLeases(context.Context) ([]akashclient.Lease, error)
	ReleaseLease(context.Context, akashclient.LeaseReference) error
}

func (*Adapter) Name() string { return Name }
func (*Adapter) Facts() provider.BillingFacts {
	return provider.BillingFacts{RoundingUnit: time.Hour, MinimumBillable: time.Hour, StoppedStillBills: true}
}

func (a *Adapter) Create(ctx context.Context, spec provider.Spec) (provider.Instance, error) {
	if err := a.requireMutation("create lease"); err != nil {
		return provider.Instance{}, err
	}
	if err := a.validateConfig(); err != nil {
		return provider.Instance{}, err
	}
	intentID := strings.TrimSpace(spec.Tags["vrooli-intent-id"])
	if intentID == "" {
		return provider.Instance{}, errors.New("Akash lease requires a persisted Compute Manager intent")
	}
	if err := a.requireConfiguredWallet(ctx); err != nil {
		return provider.Instance{}, err
	}
	if a.SandboxClient != nil {
		return a.createSandbox(ctx, spec, intentID)
	}
	if a.Client == nil {
		return provider.Instance{}, ErrClientUnavailable
	}
	receipt, err := a.Client.CreateLease(ctx, LeaseRequest{
		Network: a.Config.Network, ChainID: a.Config.ChainID, Region: spec.Region,
		Endpoint: a.Config.Endpoint,
		Offer:    spec.Size, Image: spec.Image, IntentID: intentID,
		ProviderAddress: a.Config.ProviderAddress, WalletCredentialRef: a.Config.WalletCredentialRef,
	})
	if err != nil {
		return provider.Instance{}, err
	}
	if receipt.DeploymentID == "" || receipt.LeaseID == "" || receipt.Provider != a.Config.ProviderAddress || receipt.IntentID != intentID {
		return provider.Instance{}, errors.New("Akash lease receipt is incomplete or does not bind the persisted intent")
	}
	createdAt := receipt.CreatedAt
	if createdAt.IsZero() {
		createdAt = a.now()
	}
	id := TargetReference(a.Config.ChainID, receipt.DeploymentID, receipt.LeaseID)
	tags := clone(spec.Tags)
	tags["akash-provider"] = receipt.Provider
	tags["akash-deployment-id"] = receipt.DeploymentID
	tags["akash-lease-id"] = receipt.LeaseID
	return provider.Instance{ID: id, Region: spec.Region, Size: spec.Size, Image: spec.Image, Address: receipt.Endpoint, CreatedAt: createdAt.UTC(), Tags: tags}, nil
}

func (a *Adapter) Describe(ctx context.Context, id string) (provider.Instance, error) {
	if err := a.validateConfig(); err != nil {
		return provider.Instance{}, err
	}
	if err := a.validateTargetReference(id); err != nil {
		return provider.Instance{}, err
	}
	if a.SandboxClient != nil {
		ref, err := ParseTargetReference(id)
		if err != nil {
			return provider.Instance{}, err
		}
		lease, err := a.SandboxClient.GetLease(ctx, ref)
		if err != nil {
			return provider.Instance{}, err
		}
		return sandboxInstance(lease), nil
	}
	if a.Client == nil {
		return provider.Instance{}, ErrClientUnavailable
	}
	receipt, err := a.Client.GetLease(ctx, id)
	if err != nil {
		return provider.Instance{}, err
	}
	return a.instance(receipt), nil
}

// RecoverIntent resolves a lease only from the exact bid tuple persisted with
// the still-open Compute Manager intent. Akash chain inventory does not carry
// the manager intent ID, so owner-wide List is not safe attribution evidence.
// This method is read-only and never retries CreateLease.
func (a *Adapter) RecoverIntent(ctx context.Context, request provider.IntentRecoveryRequest) (provider.Instance, bool, error) {
	if request.Provider != Name || strings.TrimSpace(request.IntentID) == "" || request.IntentID == "" {
		return provider.Instance{}, false, errors.New("Akash recovery requires the exact persisted Compute Manager intent")
	}
	if a == nil || a.SandboxClient == nil {
		return provider.Instance{}, false, ErrClientUnavailable
	}
	if a.Config.Network != akashclient.NetworkSandbox2 || strings.TrimSpace(a.Config.ChainID) == "" || strings.TrimSpace(a.Config.ProviderAddress) == "" {
		return provider.Instance{}, false, errors.New("Akash recovery requires the configured sandbox-2 chain and owner")
	}
	facts := request.Facts
	dseq := strings.TrimSpace(facts["dseq"])
	providerAddress := strings.TrimSpace(facts["provider"])
	rate := akashclient.Rate{Amount: strings.TrimSpace(facts["rate_amount"]), Denom: strings.TrimSpace(facts["rate_denom"])}
	gseq, gseqErr := strconv.ParseUint(strings.TrimSpace(facts["gseq"]), 10, 32)
	oseq, oseqErr := strconv.ParseUint(strings.TrimSpace(facts["oseq"]), 10, 32)
	leaseID := akashclient.LeaseID{Owner: a.Config.ProviderAddress, DSeq: dseq, GSeq: uint32(gseq), OSeq: uint32(oseq), Provider: providerAddress}
	if gseqErr != nil || oseqErr != nil || gseq == 0 || oseq == 0 || leaseID.Validate() != nil || rate.Validate() != nil || providerAddress == "" {
		return provider.Instance{}, false, errors.New("Akash recovery refused: persisted selected-bid tuple and exact rate are incomplete")
	}
	ref := akashclient.LeaseReference{Network: akashclient.NetworkSandbox2, ChainID: a.Config.ChainID, IntentID: request.IntentID, ID: leaseID}
	lease, err := a.SandboxClient.GetLease(ctx, ref)
	if errors.Is(err, akashclient.ErrLeaseNotFound) {
		return provider.Instance{}, false, nil
	}
	if err != nil {
		return provider.Instance{}, false, err
	}
	if lease.Reference != ref || lease.Rate != rate || lease.Validate() != nil {
		return provider.Instance{}, false, errors.New("Akash exact lease readback differs from persisted intent tuple or rate")
	}
	return sandboxInstance(lease), true, nil
}

func (a *Adapter) List(ctx context.Context) ([]provider.Instance, error) {
	if err := a.validateConfig(); err != nil {
		return nil, err
	}
	if a.SandboxClient != nil {
		leases, err := a.SandboxClient.ListLeases(ctx)
		if err != nil {
			return nil, err
		}
		instances := make([]provider.Instance, 0, len(leases))
		for _, lease := range leases {
			// The chain does not store the Compute Manager intent. A lease from
			// ListLeases therefore cannot be safely attributed to this manager.
			if strings.TrimSpace(lease.Reference.IntentID) == "" {
				return nil, errors.New("Akash lease inventory is not recoverable: no persisted Compute Manager intent-to-bid mapping exists")
			}
			instances = append(instances, sandboxInstance(lease))
		}
		return instances, nil
	}
	if a.Client == nil {
		return nil, ErrClientUnavailable
	}
	receipts, err := a.Client.ListLeases(ctx)
	if err != nil {
		return nil, err
	}
	instances := make([]provider.Instance, 0, len(receipts))
	for _, receipt := range receipts {
		instances = append(instances, a.instance(receipt))
	}
	return instances, nil
}

func (a *Adapter) Destroy(ctx context.Context, id string) error {
	if err := a.requireMutation("release lease"); err != nil {
		return err
	}
	if err := a.validateConfig(); err != nil {
		return err
	}
	if err := a.requireConfiguredWallet(ctx); err != nil {
		return err
	}
	if a.SandboxClient != nil {
		ref, err := ParseTargetReference(id)
		if err != nil {
			return fmt.Errorf("refusing to release invalid target: %w", err)
		}
		return a.SandboxClient.ReleaseLease(ctx, ref)
	}
	if a.Client == nil {
		return ErrClientUnavailable
	}
	if err := a.validateTargetReference(id); err != nil {
		return fmt.Errorf("refusing to release invalid target: %w", err)
	}
	return a.Client.ReleaseLease(ctx, id)
}

func (a *Adapter) requireConfiguredWallet(ctx context.Context) error {
	if a == nil || a.WalletConfigured == nil {
		return errors.New("Akash mutation refused: Credential Authority wallet status is unavailable")
	}
	configured, err := a.WalletConfigured(ctx)
	if err != nil {
		return fmt.Errorf("Akash mutation refused: read Credential Authority wallet status: %w", err)
	}
	if !configured {
		return errors.New("Akash mutation refused: sandbox wallet is not configured in Credential Authority")
	}
	return nil
}

func (a *Adapter) createSandbox(ctx context.Context, spec provider.Spec, intentID string) (provider.Instance, error) {
	if a.Config.Network != akashclient.NetworkSandbox2 || strings.TrimSpace(a.Config.ChainID) == "" || strings.TrimSpace(a.Config.ProviderAddress) == "" {
		return provider.Instance{}, errors.New("Akash sandbox create refused: exact sandbox-2 chain and owner configuration are required")
	}
	// Deployment creation and bid selection are separate Akash operations.
	// Until CM persists those facts, never infer a tuple from Region/Size/Image.
	tags := spec.Tags
	if tags["akash-dseq"] == "" || tags["akash-gseq"] == "" || tags["akash-oseq"] == "" || tags["akash-provider"] == "" || tags["akash-rate-amount"] == "" || tags["akash-rate-denom"] == "" {
		return provider.Instance{}, errors.New("Akash lease create refused: persisted deployment/bid tuple and exact on-chain price are unavailable")
	}
	gseqValue, err := strconv.ParseUint(tags["akash-gseq"], 10, 32)
	if err != nil || gseqValue == 0 {
		return provider.Instance{}, errors.New("Akash lease create refused: gseq must be a positive integer")
	}
	oseqValue, err := strconv.ParseUint(tags["akash-oseq"], 10, 32)
	if err != nil || oseqValue == 0 {
		return provider.Instance{}, errors.New("Akash lease create refused: oseq must be a positive integer")
	}
	request := akashclient.CreateRequest{IntentID: intentID, LeaseID: akashclient.LeaseID{Owner: a.Config.ProviderAddress, DSeq: tags["akash-dseq"], GSeq: uint32(gseqValue), OSeq: uint32(oseqValue), Provider: tags["akash-provider"]}, ExpectedRate: akashclient.Rate{Amount: tags["akash-rate-amount"], Denom: tags["akash-rate-denom"]}}
	lease, err := a.SandboxClient.CreateLease(ctx, request)
	if err != nil {
		return provider.Instance{}, err
	}
	if lease.Reference.IntentID != intentID || lease.Reference.ID != request.LeaseID || lease.Rate != request.ExpectedRate {
		return provider.Instance{}, errors.New("Akash lease readback does not match persisted Compute Manager intent, bid tuple and expected price")
	}
	return sandboxInstance(lease), nil
}

func sandboxInstance(lease akashclient.Lease) provider.Instance {
	createdAt := lease.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	return provider.Instance{ID: TargetReferenceForLease(lease.Reference), Region: lease.Reference.Network, Size: "akash-lease", CreatedAt: createdAt.UTC(), Tags: map[string]string{"vrooli-intent-id": lease.Reference.IntentID, "akash-owner": lease.Reference.ID.Owner, "akash-dseq": lease.Reference.ID.DSeq, "akash-gseq": fmt.Sprint(lease.Reference.ID.GSeq), "akash-oseq": fmt.Sprint(lease.Reference.ID.OSeq), "akash-provider": lease.Reference.ID.Provider, "akash-rate-amount": lease.Rate.Amount, "akash-rate-denom": lease.Rate.Denom, "akash-state": lease.State}}
}

func (a *Adapter) validateTargetReference(id string) error {
	if strings.HasPrefix(strings.TrimSpace(id), "akashref:") {
		ref, err := ParseTargetReference(id)
		if err != nil {
			return err
		}
		if ref.ChainID != a.Config.ChainID || ref.ID.Owner != a.Config.ProviderAddress {
			return fmt.Errorf("%q is not bound to the configured Akash chain and owner", id)
		}
		return nil
	}
	parts := strings.Split(strings.TrimSpace(id), "/")
	if len(parts) != 4 || parts[0] != "akash" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		return fmt.Errorf("%q is not a complete Akash deployment/lease reference", id)
	}
	chainID, err := url.PathUnescape(parts[1])
	if err != nil || chainID != a.Config.ChainID {
		return fmt.Errorf("%q is not bound to configured Akash chain %q", id, a.Config.ChainID)
	}
	return nil
}

func (a *Adapter) instance(receipt LeaseReceipt) provider.Instance {
	createdAt := receipt.CreatedAt
	if createdAt.IsZero() {
		createdAt = a.now()
	}
	return provider.Instance{ID: TargetReference(a.Config.ChainID, receipt.DeploymentID, receipt.LeaseID), Region: a.Config.Network, Size: "akash-lease", Address: receipt.Endpoint, CreatedAt: createdAt.UTC(), Tags: map[string]string{"vrooli-intent-id": receipt.IntentID, "akash-provider": receipt.Provider, "akash-deployment-id": receipt.DeploymentID, "akash-lease-id": receipt.LeaseID}}
}

func TargetReference(chainID, deploymentID, leaseID string) string {
	return "akash/" + url.PathEscape(chainID) + "/" + url.PathEscape(deploymentID) + "/" + url.PathEscape(leaseID)
}

func (a *Adapter) requireMutation(operation string) error {
	if a == nil || a.Gate == nil {
		return fmt.Errorf("Akash %s refused: effect gate is unavailable", operation)
	}
	return a.Gate.RequireMutation(Name, operation)
}

func (a *Adapter) validateConfig() error {
	if a == nil {
		return errors.New("Akash adapter is unavailable")
	}
	if (a.Config.Network != "testnet" && a.Config.Network != akashclient.NetworkSandbox2) || strings.TrimSpace(a.Config.ChainID) == "" || strings.Contains(strings.ToLower(a.Config.ChainID), "mainnet") {
		return errors.New("Akash adapter requires an explicitly named non-mainnet testnet chain")
	}
	endpoint, err := url.Parse(a.Config.Endpoint)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" {
		return errors.New("Akash testnet endpoint must be an HTTPS URL")
	}
	if strings.TrimSpace(a.Config.ProviderAddress) == "" || strings.TrimSpace(a.Config.WalletCredentialRef) == "" {
		return errors.New("Akash provider address and credential-authority wallet reference are required")
	}
	return nil
}

func (a *Adapter) now() time.Time {
	if a != nil && a.Now != nil {
		return a.Now().UTC()
	}
	return time.Now().UTC()
}

func clone(source map[string]string) map[string]string {
	result := make(map[string]string, len(source)+3)
	for key, value := range source {
		result[key] = value
	}
	return result
}
