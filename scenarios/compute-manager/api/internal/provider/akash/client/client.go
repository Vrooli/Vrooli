// Package client contains the dependency-free HTTP/query and transaction seam
// for Akash sandbox leases. It never stores wallet material: transaction
// signing and Credential Authority resolution are delegated to Signer.
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	NetworkSandbox2 = "sandbox-2"
	marketLeaseList = "/akash/market/v1beta5/leases/list"
	chainStatusPath = "/status"
)

var (
	ErrLeaseNotFound  = errors.New("Akash lease not found")
	ErrAmbiguousLease = errors.New("Akash returned ambiguous lease readback")
)

// Config pins all non-secret network identity. The REST and CometBFT RPC URLs
// must be supplied explicitly; this client has no network or endpoint default.
type Config struct {
	Network       string
	ChainID       string
	OwnerAddress  string
	RESTEndpoint  string
	RPCEndpoint   string
	CredentialRef string
}

// LeaseID is Akash's complete lease identity. Every component is required so
// queries and mutations cannot silently broaden to an owner or deployment.
type LeaseID struct {
	Owner    string `json:"owner"`
	DSeq     string `json:"dseq"`
	GSeq     uint32 `json:"gseq"`
	OSeq     uint32 `json:"oseq"`
	Provider string `json:"provider"`
}

func (id LeaseID) Validate() error {
	if strings.TrimSpace(id.Owner) == "" || strings.TrimSpace(id.DSeq) == "" || id.GSeq == 0 || id.OSeq == 0 || strings.TrimSpace(id.Provider) == "" {
		return errors.New("Akash lease ID requires owner, dseq, nonzero gseq/oseq and provider")
	}
	if _, err := strconv.ParseUint(id.DSeq, 10, 64); err != nil {
		return fmt.Errorf("Akash dseq must be an unsigned decimal integer: %w", err)
	}
	return nil
}

// LeaseReference binds a provider lease to the sandbox chain and the durable
// Compute Manager intent that created it. IntentID is manager-owned metadata;
// it is not claimed to be an on-chain Akash field.
type LeaseReference struct {
	Network  string
	ChainID  string
	IntentID string
	ID       LeaseID
}

// Rate preserves the exact on-chain decimal amount and denomination. Amount
// is not converted to USD or assumed to be AKT/ACT.
type Rate struct {
	Amount string `json:"amount"`
	Denom  string `json:"denom"`
}

func (r Rate) Validate() error {
	if strings.TrimSpace(r.Denom) == "" {
		return errors.New("Akash lease price denomination is missing")
	}
	amount, ok := new(big.Rat).SetString(strings.TrimSpace(r.Amount))
	if !ok || amount.Sign() <= 0 {
		return errors.New("Akash lease price must be a positive exact decimal")
	}
	return nil
}

// Lease is a decoded Akash on-chain lease record. Rate is the provider's
// accepted price per block in the exact denomination returned by Akash.
type Lease struct {
	Reference LeaseReference
	State     string
	Rate      Rate
	CreatedAt time.Time
}

func (l Lease) Validate() error {
	if err := l.Reference.ID.Validate(); err != nil {
		return err
	}
	if l.Reference.Network != NetworkSandbox2 || strings.TrimSpace(l.Reference.ChainID) == "" || strings.TrimSpace(l.Reference.IntentID) == "" {
		return errors.New("Akash lease reference is missing sandbox chain or Compute Manager intent")
	}
	if l.State != "active" && l.State != "insufficient_funds" && l.State != "closed" {
		return fmt.Errorf("Akash lease has unsupported state %q", l.State)
	}
	return l.Rate.Validate()
}

// CreateRequest accepts a previously persisted deployment and the exact bid
// selected for its order. Creating a deployment and selecting an offer are
// separate Akash operations and remain outside this lease client.
type CreateRequest struct {
	IntentID     string
	LeaseID      LeaseID
	ExpectedRate Rate
}

type Operation string

const (
	OperationCreateLease Operation = "create_lease"
	OperationCloseLease  Operation = "close_lease"
)

// TransactionIntent is a public, typed action. Signer resolves CredentialRef
// from Credential Authority only when invoked; secret bytes are never passed
// into or returned by this package.
type TransactionIntent struct {
	Operation     Operation
	Network       string
	ChainID       string
	OwnerAddress  string
	CredentialRef string
	IntentID      string
	LeaseID       LeaseID
}

type TransactionReceipt struct {
	Hash string
	Code uint32
}

// Signer is the explicit authorization/effect seam. Implementations must
// resolve CredentialRef through Credential Authority and sign/broadcast only
// the provided Akash transaction intent.
type Signer interface {
	SignAndBroadcast(context.Context, TransactionIntent) (TransactionReceipt, error)
}

// Client performs read-only Akash REST queries and delegates transaction
// signing/broadcast to Signer. It refuses any chain other than sandbox-2.
type Client struct {
	config Config
	http   *http.Client
	signer Signer
}

func New(config Config, httpClient *http.Client, signer Signer) (*Client, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	copyClient := *httpClient
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{config: config, http: &copyClient, signer: signer}, nil
}

func (c *Client) CreateLease(ctx context.Context, request CreateRequest) (Lease, error) {
	if c == nil {
		return Lease{}, errors.New("Akash testnet client is unavailable")
	}
	if err := c.validateCreate(request); err != nil {
		return Lease{}, err
	}
	if existing, err := c.findExact(ctx, request.LeaseID, request.IntentID); err == nil {
		if request.ExpectedRate.Amount != "" && existing.Rate != request.ExpectedRate {
			return Lease{}, errors.New("existing Akash lease price differs from the persisted expected bid")
		}
		return existing, nil
	} else if !errors.Is(err, ErrLeaseNotFound) {
		return Lease{}, err
	}
	if c.signer == nil {
		return Lease{}, errors.New("Akash lease creation refused: Credential Authority transaction signer is unavailable")
	}
	intent := c.transactionIntent(OperationCreateLease, request.IntentID, request.LeaseID)
	receipt, broadcastErr := c.signer.SignAndBroadcast(ctx, intent)
	if broadcastErr != nil {
		// A timeout can arrive after chain inclusion. Reconcile the stable bid
		// tuple before returning an error, without issuing a second transaction.
		lease, readErr := c.findExact(ctx, request.LeaseID, request.IntentID)
		if readErr == nil {
			if request.ExpectedRate.Amount != "" && lease.Rate != request.ExpectedRate {
				return Lease{}, errors.New("Akash create response was lost and readback price differs from the persisted expected bid")
			}
			return lease, nil
		}
		return Lease{}, errors.Join(fmt.Errorf("broadcast Akash create-lease transaction: %w", broadcastErr), readErr)
	}
	if receipt.Code != 0 || strings.TrimSpace(receipt.Hash) == "" {
		return Lease{}, fmt.Errorf("Akash create-lease transaction was not accepted (code %d)", receipt.Code)
	}
	lease, err := c.findExact(ctx, request.LeaseID, request.IntentID)
	if err != nil {
		return Lease{}, fmt.Errorf("Akash create-lease transaction %s has no unique exact lease readback: %w", receipt.Hash, err)
	}
	if request.ExpectedRate.Amount != "" && lease.Rate != request.ExpectedRate {
		return Lease{}, errors.New("Akash lease readback price differs from the persisted expected bid")
	}
	return lease, nil
}

func (c *Client) GetLease(ctx context.Context, ref LeaseReference) (Lease, error) {
	if err := c.validateReference(ref); err != nil {
		return Lease{}, err
	}
	return c.findExact(ctx, ref.ID, ref.IntentID)
}

func (c *Client) ListLeases(ctx context.Context) ([]Lease, error) {
	if err := c.verifyChain(ctx); err != nil {
		return nil, err
	}
	return c.listOwnerLeases(ctx)
}

func (c *Client) ReleaseLease(ctx context.Context, ref LeaseReference) error {
	if err := c.validateReference(ref); err != nil {
		return fmt.Errorf("refusing to release non-Compute-Manager Akash lease: %w", err)
	}
	lease, err := c.findExact(ctx, ref.ID, ref.IntentID)
	if err != nil {
		return err
	}
	if lease.State == "closed" {
		return nil
	}
	if c.signer == nil {
		return errors.New("Akash lease release refused: Credential Authority transaction signer is unavailable")
	}
	receipt, broadcastErr := c.signer.SignAndBroadcast(ctx, c.transactionIntent(OperationCloseLease, ref.IntentID, ref.ID))
	if broadcastErr != nil {
		updated, readErr := c.findExact(ctx, ref.ID, ref.IntentID)
		if readErr == nil && updated.State == "closed" {
			return nil
		}
		return errors.Join(fmt.Errorf("broadcast Akash close-lease transaction: %w", broadcastErr), readErr)
	}
	if receipt.Code != 0 || strings.TrimSpace(receipt.Hash) == "" {
		return fmt.Errorf("Akash close-lease transaction was not accepted (code %d)", receipt.Code)
	}
	updated, err := c.findExact(ctx, ref.ID, ref.IntentID)
	if err != nil {
		return fmt.Errorf("Akash close-lease transaction %s has no unique exact readback: %w", receipt.Hash, err)
	}
	if updated.State != "closed" {
		return fmt.Errorf("Akash close-lease transaction %s returned but lease state is %q", receipt.Hash, updated.State)
	}
	return nil
}

func (c *Client) validateCreate(request CreateRequest) error {
	if strings.TrimSpace(request.IntentID) == "" {
		return errors.New("Akash create lease requires a persisted Compute Manager intent")
	}
	if err := request.LeaseID.Validate(); err != nil {
		return err
	}
	if request.LeaseID.Owner != c.config.OwnerAddress {
		return errors.New("Akash bid owner does not match the configured Compute Manager account")
	}
	if request.ExpectedRate.Amount != "" {
		return request.ExpectedRate.Validate()
	}
	return nil
}

func (c *Client) validateReference(ref LeaseReference) error {
	if c == nil {
		return errors.New("Akash testnet client is unavailable")
	}
	if ref.Network != c.config.Network || ref.ChainID != c.config.ChainID {
		return errors.New("Akash lease reference does not match the configured sandbox network")
	}
	if strings.TrimSpace(ref.IntentID) == "" {
		return errors.New("Akash release/read requires the persisted Compute Manager intent reference")
	}
	if err := ref.ID.Validate(); err != nil {
		return err
	}
	if ref.ID.Owner != c.config.OwnerAddress {
		return errors.New("Akash lease owner does not match the configured Compute Manager account")
	}
	return nil
}

func (c *Client) transactionIntent(operation Operation, intentID string, id LeaseID) TransactionIntent {
	return TransactionIntent{Operation: operation, Network: c.config.Network, ChainID: c.config.ChainID, OwnerAddress: c.config.OwnerAddress, CredentialRef: c.config.CredentialRef, IntentID: intentID, LeaseID: id}
}

func (c *Client) findExact(ctx context.Context, id LeaseID, intentID string) (Lease, error) {
	leases, err := c.listOwnerLeases(ctx)
	if err != nil {
		return Lease{}, err
	}
	var matches []Lease
	for _, lease := range leases {
		if lease.Reference.ID == id {
			lease.Reference.IntentID = intentID
			matches = append(matches, lease)
		}
	}
	switch len(matches) {
	case 0:
		return Lease{}, ErrLeaseNotFound
	case 1:
		return matches[0], nil
	default:
		return Lease{}, ErrAmbiguousLease
	}
}

func (c *Client) listOwnerLeases(ctx context.Context) ([]Lease, error) {
	if err := c.verifyChain(ctx); err != nil {
		return nil, err
	}
	base, err := url.Parse(c.config.RESTEndpoint)
	if err != nil {
		return nil, errors.New("Akash REST endpoint is invalid")
	}
	var all []Lease
	var nextKey string
	for page := 0; page < 100; page++ {
		endpoint := *base
		endpoint.Path = strings.TrimRight(base.Path, "/") + marketLeaseList
		query := endpoint.Query()
		query.Set("filters.owner", c.config.OwnerAddress)
		query.Set("pagination.limit", "100")
		if nextKey != "" {
			query.Set("pagination.key", nextKey)
		}
		endpoint.RawQuery = query.Encode()
		var response leasesResponse
		if err := c.getJSON(ctx, endpoint.String(), &response); err != nil {
			return nil, err
		}
		for _, item := range response.Leases {
			lease, err := item.toLease(c.config, "")
			if err != nil {
				return nil, fmt.Errorf("decode Akash lease record: %w", err)
			}
			all = append(all, lease)
		}
		nextKey = response.Pagination.NextKey
		if nextKey == "" {
			return all, nil
		}
	}
	return nil, errors.New("Akash lease query exceeded 100 pages; refusing incomplete readback")
}

func (c *Client) verifyChain(ctx context.Context) error {
	base, err := url.Parse(c.config.RPCEndpoint)
	if err != nil {
		return errors.New("Akash RPC endpoint is invalid")
	}
	base.Path = strings.TrimRight(base.Path, "/") + chainStatusPath
	var status statusResponse
	if err := c.getJSON(ctx, base.String(), &status); err != nil {
		return fmt.Errorf("verify Akash sandbox chain status: %w", err)
	}
	if strings.TrimSpace(status.Result.NodeInfo.Network) != c.config.ChainID {
		return fmt.Errorf("Akash node chain ID %q does not match configured sandbox chain %q", status.Result.NodeInfo.Network, c.config.ChainID)
	}
	return nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, destination any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build Akash query request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	response, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("Akash query request failed: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("Akash query returned HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 2<<20))
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode Akash JSON response: %w", err)
	}
	return nil
}

func validateConfig(config Config) error {
	if config.Network != NetworkSandbox2 {
		return errors.New("Akash client requires explicit sandbox-2 network selection; mainnet is refused")
	}
	chainID := strings.TrimSpace(config.ChainID)
	if chainID == "" || strings.EqualFold(chainID, "akashnet-2") || strings.Contains(strings.ToLower(chainID), "mainnet") {
		return errors.New("Akash client requires the explicit sandbox chain ID, not mainnet/default")
	}
	if strings.TrimSpace(config.OwnerAddress) == "" {
		return errors.New("Akash client requires the explicit Compute Manager owner address")
	}
	if strings.TrimSpace(config.CredentialRef) == "" || strings.ContainsAny(config.CredentialRef, " \t\r\n") || !strings.HasPrefix(config.CredentialRef, "credential-authority://") {
		return errors.New("Akash client requires a Credential Authority reference, not a raw credential")
	}
	if err := validateEndpoint(config.RESTEndpoint); err != nil {
		return fmt.Errorf("Akash REST endpoint: %w", err)
	}
	if err := validateEndpoint(config.RPCEndpoint); err != nil {
		return fmt.Errorf("Akash RPC endpoint: %w", err)
	}
	for _, endpoint := range []string{config.RESTEndpoint, config.RPCEndpoint} {
		u, _ := url.Parse(endpoint)
		host := strings.ToLower(u.Hostname())
		if host == "console-api.akash.network" || host == "api.akashnet.net" || host == "rpc.akashnet.net" {
			return errors.New("Akash client refuses managed Console/mainnet defaults; sandbox endpoints must be explicit")
		}
	}
	return nil
}

func validateEndpoint(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("must be an explicit absolute HTTPS node URL without userinfo, query or fragment")
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" {
		ip := net.ParseIP(u.Hostname())
		if u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback()) {
			return nil // Recorded-contract tests only.
		}
	}
	return errors.New("must use HTTPS (HTTP is allowed only for loopback recorded-contract tests)")
}

type statusResponse struct {
	Result struct {
		NodeInfo struct {
			Network string `json:"network"`
		} `json:"node_info"`
	} `json:"result"`
}

type leasesResponse struct {
	Leases     []leaseEnvelope `json:"leases"`
	Pagination struct {
		NextKey string `json:"next_key"`
	} `json:"pagination"`
}

type leaseEnvelope struct {
	Lease         leaseRecord `json:"lease"`
	EscrowPayment struct {
		Rate Rate `json:"rate"`
	} `json:"escrow_payment"`
}

type leaseRecord struct {
	ID        LeaseID         `json:"lease_id"`
	LegacyID  LeaseID         `json:"id"`
	State     json.RawMessage `json:"state"`
	Price     Rate            `json:"price"`
	CreatedAt string          `json:"created_at"`
}

func (item leaseEnvelope) toLease(config Config, intentID string) (Lease, error) {
	record := item.Lease
	id := record.ID
	if id.Owner == "" {
		id = record.LegacyID
	}
	if id.Owner == "" || id.DSeq == "" || id.GSeq == 0 || id.OSeq == 0 || id.Provider == "" {
		return Lease{}, errors.New("response omitted complete owner/dseq/gseq/oseq/provider identity")
	}
	rate := record.Price
	if rate.Amount == "" && item.EscrowPayment.Rate.Amount != "" {
		rate = item.EscrowPayment.Rate
	}
	state, err := decodeState(record.State)
	if err != nil {
		return Lease{}, err
	}
	lease := Lease{Reference: LeaseReference{Network: config.Network, ChainID: config.ChainID, IntentID: intentID, ID: id}, State: state, Rate: rate}
	if record.CreatedAt != "" {
		lease.CreatedAt, err = time.Parse(time.RFC3339Nano, record.CreatedAt)
		if err != nil {
			return Lease{}, fmt.Errorf("parse lease created_at: %w", err)
		}
	}
	if state != "active" && state != "insufficient_funds" && state != "closed" {
		return Lease{}, fmt.Errorf("unsupported Akash lease state %q", state)
	}
	if err := lease.Rate.Validate(); err != nil {
		return Lease{}, err
	}
	return lease, nil
}

func decodeState(raw json.RawMessage) (string, error) {
	var state string
	if err := json.Unmarshal(raw, &state); err == nil {
		state = strings.ToLower(strings.TrimSpace(state))
		state = strings.TrimPrefix(state, "lease_state_")
		state = strings.TrimPrefix(state, "lease_")
		return state, nil
	}
	var number int
	if err := json.Unmarshal(raw, &number); err != nil {
		return "", errors.New("Akash lease response omitted a recognized state")
	}
	switch number {
	case 1:
		return "active", nil
	case 2:
		return "insufficient_funds", nil
	case 3:
		return "closed", nil
	default:
		return "", fmt.Errorf("unsupported Akash lease state number %d", number)
	}
}
