package x402

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"
)

var (
	ErrNotFound            = errors.New("x402 object not found")
	ErrInProgress          = errors.New("x402 settlement is already in progress")
	ErrUnknown             = errors.New("x402 settlement outcome is unknown")
	ErrRecoveryUnavailable = errors.New("x402 protected payment recovery is unavailable")
	ErrBindingConflict     = errors.New("x402 payment resource binding conflicts")
	ErrStateConflict       = errors.New("x402 admission state does not allow this operation")
)

type Price struct {
	ID                string    `json:"id"`
	ResourceURL       string    `json:"resource_url"`
	Description       string    `json:"description"`
	MIMEType          string    `json:"mime_type"`
	Network           string    `json:"network"`
	Scheme            string    `json:"scheme"`
	Amount            string    `json:"amount"`
	AmountMinor       int64     `json:"amount_minor"`
	Currency          string    `json:"currency"`
	PayTo             string    `json:"pay_to"`
	Asset             string    `json:"asset"`
	AssetDecimals     int       `json:"asset_decimals"`
	MaxTimeoutSeconds uint64    `json:"max_timeout_seconds"`
	ExtraJSON         string    `json:"extra_json"`
	CreatedAt         time.Time `json:"created_at"`
}

type Admission struct {
	ID                string    `json:"id"`
	PriceID           string    `json:"price_id"`
	PayloadDigest     string    `json:"payload_digest"`
	Status            string    `json:"status"`
	Payer             string    `json:"payer"`
	TransactionID     string    `json:"transaction_id"`
	Network           string    `json:"network"`
	Mode              string    `json:"mode,omitempty"`
	RecoveryReference string    `json:"-"`
	JobID             string    `json:"job_id,omitempty"`
	ResourceID        string    `json:"resource_id,omitempty"`
	OrderID           string    `json:"order_id,omitempty"`
	Detail            string    `json:"detail"`
	Recovery          bool      `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type VerifyResult struct {
	Valid  bool
	Payer  string
	Reason string
}

type SettleResult struct {
	Success     bool
	Payer       string
	Transaction string
	Network     string
	Reason      string
}

type Facilitator interface {
	Verify(context.Context, json.RawMessage, json.RawMessage) (VerifyResult, error)
	Settle(context.Context, json.RawMessage, json.RawMessage) (SettleResult, error)
}

type InboundRepository interface {
	CreatePrice(context.Context, Price) (Price, error)
	GetPrice(context.Context, string) (Price, error)
	GetAdmissionByDigest(context.Context, string) (Admission, error)
	ClaimAdmission(context.Context, Admission) (Admission, bool, error)
	CompleteAdmission(context.Context, string, string, SettleResult, time.Time, Price) (Admission, error)
}

// ResourceAdmissionRepository durably binds a verified payment to the exact
// LPBS job, resource, and order before any facilitator settlement is attempted.
type ResourceAdmissionRepository interface {
	GetResourceAdmission(context.Context, string) (Admission, error)
	ClaimResourceAdmission(context.Context, Admission) (Admission, bool, error)
	BeginResourceSettlement(context.Context, string, time.Time) (Admission, bool, error)
	CompleteResourceSettlement(context.Context, string, string, SettleResult, time.Time, Price) (Admission, error)
	CancelResourceAdmission(context.Context, string, string, string, string, time.Time) (Admission, error)
	ClearResourceRecoveryReference(context.Context, string) error
}

type RecoveryVault interface {
	StoreSettlementRecoveryInput(context.Context, string, string) (string, error)
	ResolveSettlementRecoveryInput(context.Context, string) (string, error)
	DeleteSettlementRecoveryInput(context.Context, string) error
}

type Gate struct {
	repository    InboundRepository
	facilitator   Facilitator
	now           func() time.Time
	mode          string
	recoveryVault RecoveryVault
}

func (g *Gate) SetRecoveryVault(vault RecoveryVault) { g.recoveryVault = vault }

func NewGate(repository InboundRepository, facilitator Facilitator) (*Gate, error) {
	if repository == nil || facilitator == nil {
		return nil, fmt.Errorf("%w: repository and facilitator are required", ErrInvalid)
	}
	return &Gate{repository: repository, facilitator: facilitator, now: time.Now, mode: "production"}, nil
}

// SetRuntimeMode tags resource admissions with the explicitly configured
// runtime. Local-fork mode requires a loopback facilitator endpoint.
func (g *Gate) SetRuntimeMode(mode string) error {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" || mode == "production" {
		g.mode = "production"
		return nil
	}
	if mode != "local-fork" {
		return fmt.Errorf("unsupported x402 runtime mode %q", mode)
	}
	local, ok := g.facilitator.(interface{ LocalOnly() bool })
	if !ok || !local.LocalOnly() {
		return fmt.Errorf("local-fork mode requires a loopback x402 facilitator")
	}
	g.mode = mode
	return nil
}

func (g *Gate) Declare(ctx context.Context, price Price) (Price, error) {
	price.ID = strings.TrimSpace(price.ID)
	price.ResourceURL = strings.TrimSpace(price.ResourceURL)
	price.Description = strings.TrimSpace(price.Description)
	price.MIMEType = strings.TrimSpace(price.MIMEType)
	price.Network = strings.ToLower(strings.TrimSpace(price.Network))
	price.Scheme = strings.ToLower(strings.TrimSpace(price.Scheme))
	price.Amount = strings.TrimSpace(price.Amount)
	price.Currency = strings.ToUpper(strings.TrimSpace(price.Currency))
	price.PayTo = strings.TrimSpace(price.PayTo)
	price.Asset = strings.ToLower(strings.TrimSpace(price.Asset))
	price.ExtraJSON = strings.TrimSpace(price.ExtraJSON)
	if price.CreatedAt.IsZero() {
		price.CreatedAt = g.now().UTC()
	}
	if err := validatePrice(price); err != nil {
		return Price{}, err
	}
	return g.repository.CreatePrice(ctx, price)
}

// PaymentRequired returns the exact base64 value for the x402 v2
// Payment-Required response header.
func (g *Gate) PaymentRequired(ctx context.Context, priceID string) (string, error) {
	price, err := g.repository.GetPrice(ctx, strings.TrimSpace(priceID))
	if err != nil {
		return "", err
	}
	requirement, err := priceRequirement(price)
	if err != nil {
		return "", err
	}
	var accepted any
	if err := json.Unmarshal(requirement, &accepted); err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]any{
		"x402Version": 2,
		"resource":    map[string]any{"url": price.ResourceURL, "description": price.Description, "mimeType": price.MIMEType},
		"accepts":     []any{accepted}, "extensions": map[string]any{},
	})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(body), nil
}

// Admit verifies and settles one signed payment. A durable payload claim is
// committed after verification but before settlement, so concurrent replays
// cannot call the facilitator twice. Settled replays return the same receipt.
func (g *Gate) Admit(ctx context.Context, priceID, paymentHeader string) (Admission, error) {
	price, err := g.repository.GetPrice(ctx, strings.TrimSpace(priceID))
	if err != nil {
		return Admission{}, err
	}
	payload, accepted, digest, err := decodeInboundPayment(paymentHeader)
	if err != nil {
		return Admission{}, err
	}
	expected, err := priceRequirement(price)
	if err != nil {
		return Admission{}, err
	}
	if !jsonEqual(accepted, expected) {
		return Admission{}, fmt.Errorf("%w: signed accepted requirements do not match the declared price", ErrInvalid)
	}
	if existing, getErr := g.repository.GetAdmissionByDigest(ctx, digest); getErr == nil {
		return replayAdmission(existing, price.ID)
	} else if !errors.Is(getErr, ErrNotFound) {
		return Admission{}, getErr
	}
	verified, err := g.facilitator.Verify(ctx, payload, expected)
	if err != nil {
		return Admission{}, fmt.Errorf("verify payment: %w", err)
	}
	if !verified.Valid || strings.TrimSpace(verified.Payer) == "" {
		return Admission{}, fmt.Errorf("%w: facilitator verification refused payment: %s", ErrInvalid, verified.Reason)
	}
	now := g.now().UTC()
	claim, claimed, err := g.repository.ClaimAdmission(ctx, Admission{
		ID: "x402-inbound:" + strings.TrimPrefix(digest, "sha256:"), PriceID: price.ID,
		PayloadDigest: digest, Status: "calling", Payer: verified.Payer, Network: price.Network,
		CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return Admission{}, err
	}
	if !claimed {
		return replayAdmission(claim, price.ID)
	}
	settled, settleErr := g.facilitator.Settle(ctx, payload, expected)
	status := "settled"
	if settleErr != nil {
		status = "unknown"
		settled = SettleResult{Payer: verified.Payer, Network: price.Network, Reason: "facilitator settlement response unavailable: " + settleErr.Error()}
	} else if !settled.Success || settled.Transaction == "" || settled.Payer == "" || settled.Network != price.Network {
		status = "failed"
		if settled.Reason == "" {
			settled.Reason = "facilitator returned an incomplete or mismatched settlement"
		}
	}
	completed, err := g.repository.CompleteAdmission(context.WithoutCancel(ctx), claim.ID, status, settled, g.now().UTC(), price)
	if err != nil {
		return Admission{}, err
	}
	if status == "unknown" {
		return completed, ErrUnknown
	}
	if status == "failed" {
		return completed, fmt.Errorf("%w: %s", ErrInvalid, completed.Detail)
	}
	return completed, nil
}

// VerifyForResource verifies and durably authorizes an x402 payment for one
// previously-created LPBS job/order. It never calls facilitator settlement.
func (g *Gate) VerifyForResource(ctx context.Context, priceID, jobID, resourceID, orderID, paymentHeader string) (Admission, error) {
	repository, ok := g.repository.(ResourceAdmissionRepository)
	if !ok {
		return Admission{}, fmt.Errorf("resource-scoped x402 admission repository is unavailable")
	}
	price, err := g.repository.GetPrice(ctx, strings.TrimSpace(priceID))
	if err != nil {
		return Admission{}, err
	}
	jobID, resourceID, orderID = strings.TrimSpace(jobID), strings.TrimSpace(resourceID), strings.TrimSpace(orderID)
	if jobID == "" || resourceID == "" || orderID == "" {
		return Admission{}, fmt.Errorf("%w: job_id, resource_id and order_id are required", ErrInvalid)
	}
	if g.recoveryVault == nil {
		return Admission{}, ErrRecoveryUnavailable
	}
	payload, accepted, digest, err := decodeInboundPayment(paymentHeader)
	if err != nil {
		return Admission{}, err
	}
	expected, err := priceRequirement(price)
	if err != nil {
		return Admission{}, err
	}
	if !jsonEqual(accepted, expected) {
		return Admission{}, fmt.Errorf("%w: signed accepted requirements do not match the declared price", ErrInvalid)
	}
	if existing, getErr := repository.GetResourceAdmission(ctx, digest); getErr == nil {
		return replayResourceAdmission(existing, price.ID, jobID, resourceID, orderID)
	} else if !errors.Is(getErr, ErrNotFound) {
		return Admission{}, getErr
	}
	verified, err := g.facilitator.Verify(ctx, payload, expected)
	if err != nil {
		return Admission{}, fmt.Errorf("verify payment: %w", err)
	}
	if !verified.Valid || strings.TrimSpace(verified.Payer) == "" {
		return Admission{}, fmt.Errorf("%w: facilitator verification refused payment: %s", ErrInvalid, verified.Reason)
	}
	now := g.now().UTC()
	id := "x402-resource:" + strings.TrimPrefix(digest, "sha256:")
	recoveryReference, err := g.recoveryVault.StoreSettlementRecoveryInput(ctx, id, strings.TrimSpace(paymentHeader))
	if err != nil {
		return Admission{}, fmt.Errorf("%w: store protected x402 recovery payload: %v", ErrRecoveryUnavailable, err)
	}
	claim, claimed, err := repository.ClaimResourceAdmission(ctx, Admission{
		ID: id, PriceID: price.ID,
		PayloadDigest: digest, Status: "verified", Payer: verified.Payer, Network: price.Network, Mode: g.mode,
		RecoveryReference: recoveryReference, JobID: jobID, ResourceID: resourceID, OrderID: orderID, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return Admission{}, err
	}
	if !claimed {
		if claim.ID != id {
			_ = g.recoveryVault.DeleteSettlementRecoveryInput(context.WithoutCancel(ctx), recoveryReference)
		}
		return replayResourceAdmission(claim, price.ID, jobID, resourceID, orderID)
	}
	return claim, nil
}

// SettleAuthorization settles a previously verified resource admission. The
// same signed payload is required so retries can be bound to the original
// authorization digest. An unknown facilitator outcome remains durable and
// can be reconciled by replaying the same authorization under its admission ID.
func (g *Gate) SettleAuthorization(ctx context.Context, admissionID, jobID, resourceID, orderID string) (Admission, error) {
	repository, ok := g.repository.(ResourceAdmissionRepository)
	if !ok {
		return Admission{}, fmt.Errorf("resource-scoped x402 admission repository is unavailable")
	}
	stored, err := repository.GetResourceAdmission(ctx, strings.TrimSpace(admissionID))
	if err != nil {
		return Admission{}, err
	}
	if stored.JobID != strings.TrimSpace(jobID) || stored.ResourceID != strings.TrimSpace(resourceID) || stored.OrderID != strings.TrimSpace(orderID) {
		return Admission{}, fmt.Errorf("%w: admission resource binding does not match", ErrBindingConflict)
	}
	if stored.Status == "settled" || stored.Status == "failed" {
		_ = g.cleanupRecoveryPayload(ctx, repository, stored)
		return replayAdmission(stored, stored.PriceID)
	}
	if stored.Status == "cancelled" {
		return stored, ErrStateConflict
	}
	if g.recoveryVault == nil || stored.RecoveryReference == "" {
		return Admission{}, ErrRecoveryUnavailable
	}
	paymentHeader, err := g.recoveryVault.ResolveSettlementRecoveryInput(ctx, stored.RecoveryReference)
	if err != nil {
		return Admission{}, fmt.Errorf("%w: %v", ErrRecoveryUnavailable, err)
	}
	price, err := g.repository.GetPrice(ctx, stored.PriceID)
	if err != nil {
		return Admission{}, err
	}
	payload, accepted, digest, err := decodeInboundPayment(paymentHeader)
	if err != nil {
		return Admission{}, err
	}
	if digest != stored.PayloadDigest {
		return Admission{}, fmt.Errorf("%w: protected payment payload does not match the verified admission", ErrInvalid)
	}
	expected, err := priceRequirement(price)
	if err != nil {
		return Admission{}, err
	}
	if !jsonEqual(accepted, expected) {
		return Admission{}, fmt.Errorf("%w: signed accepted requirements do not match the declared price", ErrInvalid)
	}
	claim, claimed, err := repository.BeginResourceSettlement(ctx, stored.ID, g.now().UTC())
	if err != nil {
		return Admission{}, err
	}
	if !claimed {
		return replayResourceAdmission(claim, price.ID, jobID, resourceID, orderID)
	}
	settled, settleErr := g.facilitator.Settle(ctx, payload, expected)
	status := "settled"
	if settleErr != nil {
		status = "unknown"
		settled = SettleResult{Payer: stored.Payer, Network: price.Network, Reason: "facilitator settlement response unavailable: " + settleErr.Error()}
	} else if !settled.Success || settled.Transaction == "" || settled.Payer != stored.Payer || settled.Network != price.Network {
		status = "failed"
		if claim.Recovery {
			// A failed replay does not prove that the earlier dispatched settle
			// failed. Keep the admission recoverable until a conclusive receipt
			// is obtained for the same signed nonce.
			status = "unknown"
		}
		if settled.Reason == "" {
			settled.Reason = "facilitator returned an incomplete or mismatched settlement"
		}
	}
	completed, err := repository.CompleteResourceSettlement(context.WithoutCancel(ctx), stored.ID, status, settled, g.now().UTC(), price)
	if err != nil {
		return Admission{}, err
	}
	if status == "settled" || status == "failed" {
		_ = g.cleanupRecoveryPayload(context.WithoutCancel(ctx), repository, completed)
	}
	if status == "unknown" {
		return completed, ErrUnknown
	}
	if status == "failed" {
		return completed, fmt.Errorf("%w: %s", ErrInvalid, completed.Detail)
	}
	return completed, nil
}

func (g *Gate) cleanupRecoveryPayload(ctx context.Context, repository ResourceAdmissionRepository, admission Admission) error {
	if admission.RecoveryReference == "" {
		return nil
	}
	if g.recoveryVault == nil {
		return ErrRecoveryUnavailable
	}
	if err := g.recoveryVault.DeleteSettlementRecoveryInput(ctx, admission.RecoveryReference); err != nil {
		return fmt.Errorf("%w: delete protected payload: %v", ErrRecoveryUnavailable, err)
	}
	if err := repository.ClearResourceRecoveryReference(ctx, admission.ID); err != nil {
		return fmt.Errorf("clear protected payload reference: %w", err)
	}
	return nil
}

// CancelAuthorization terminally releases an authorization that has not begun
// settlement, which LPBS uses after delivery verification fails.
func (g *Gate) CancelAuthorization(ctx context.Context, admissionID, jobID, resourceID, orderID string) (Admission, error) {
	repository, ok := g.repository.(ResourceAdmissionRepository)
	if !ok {
		return Admission{}, fmt.Errorf("resource-scoped x402 admission repository is unavailable")
	}
	stored, err := repository.GetResourceAdmission(ctx, strings.TrimSpace(admissionID))
	if err != nil {
		return Admission{}, err
	}
	if stored.JobID != strings.TrimSpace(jobID) || stored.ResourceID != strings.TrimSpace(resourceID) || stored.OrderID != strings.TrimSpace(orderID) {
		return Admission{}, ErrBindingConflict
	}
	if stored.Status == "cancelled" {
		if err := g.cleanupRecoveryPayload(ctx, repository, stored); err != nil {
			return stored, err
		}
		return stored, nil
	}
	cancelled, err := repository.CancelResourceAdmission(ctx, stored.ID, jobID, resourceID, orderID, g.now().UTC())
	if err != nil {
		return Admission{}, err
	}
	if err := g.cleanupRecoveryPayload(context.WithoutCancel(ctx), repository, cancelled); err != nil {
		return cancelled, err
	}
	return cancelled, nil
}

func replayResourceAdmission(value Admission, priceID, jobID, resourceID, orderID string) (Admission, error) {
	if value.PriceID != priceID || value.JobID != jobID || value.ResourceID != resourceID || value.OrderID != orderID {
		return value, fmt.Errorf("%w: payment authorization is already bound to another job, resource, order, or price", ErrBindingConflict)
	}
	switch value.Status {
	case "verified":
		return value, nil
	case "settled":
		return value, nil
	case "unknown":
		// A verify retry after a lost response must recover the durable
		// admission ID so the caller can continue settlement recovery.
		return value, nil
	case "cancelled":
		return value, nil
	case "failed":
		return value, fmt.Errorf("%w: %s", ErrInvalid, value.Detail)
	default:
		return value, ErrInProgress
	}
}

func replayAdmission(value Admission, priceID string) (Admission, error) {
	if value.PriceID != priceID {
		return Admission{}, fmt.Errorf("%w: payment payload was already bound to another price", ErrInvalid)
	}
	switch value.Status {
	case "settled":
		return value, nil
	case "failed":
		return value, fmt.Errorf("%w: %s", ErrInvalid, value.Detail)
	case "unknown":
		return value, ErrUnknown
	default:
		return value, ErrInProgress
	}
}

func validatePrice(price Price) error {
	resource, resourceErr := url.Parse(price.ResourceURL)
	if resourceErr != nil || resource.Hostname() == "" || resource.User != nil || resource.Fragment != "" || resource.RawQuery != "" || resource.Scheme != "https" && !(resource.Scheme == "http" && isLoopback(resource.Hostname())) {
		return fmt.Errorf("%w: resource_url must be an absolute HTTPS URL without userinfo, query, or fragment (loopback HTTP is allowed)", ErrInvalid)
	}
	amount, ok := new(big.Int).SetString(price.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return fmt.Errorf("%w: amount must be a positive base-10 integer", ErrInvalid)
	}
	if price.ID == "" || price.ResourceURL == "" || price.MIMEType == "" || !strings.HasPrefix(price.Network, "eip155:") || price.Scheme != "exact" || price.AmountMinor <= 0 || price.Currency == "" || !isEVMAddress(price.PayTo) || !isEVMAddress(price.Asset) || price.AssetDecimals < 2 || price.AssetDecimals > 30 || price.MaxTimeoutSeconds == 0 || price.ExtraJSON == "" {
		return fmt.Errorf("%w: complete v2 exact EIP-155 price fields are required", ErrInvalid)
	}
	multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(price.AssetDecimals-2)), nil)
	want := new(big.Int).Mul(big.NewInt(price.AmountMinor), multiplier)
	if amount.Cmp(want) != 0 {
		return fmt.Errorf("%w: amount does not exactly equal amount_minor at the declared asset decimals", ErrInvalid)
	}
	var method transferMethod
	if json.Unmarshal([]byte(price.ExtraJSON), &method) != nil || method.AssetTransferMethod != "eip3009" || method.Name == "" || method.Version == "" {
		return fmt.Errorf("%w: extra_json must declare a complete eip3009 transfer method", ErrInvalid)
	}
	return nil
}

func priceRequirement(price Price) (json.RawMessage, error) {
	var extra any
	if err := json.Unmarshal([]byte(price.ExtraJSON), &extra); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"scheme": price.Scheme, "network": price.Network, "amount": price.Amount,
		"payTo": price.PayTo, "maxTimeoutSeconds": price.MaxTimeoutSeconds,
		"asset": price.Asset, "extra": extra,
	})
}

func decodeInboundPayment(header string) (json.RawMessage, json.RawMessage, string, error) {
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(header))
	if err != nil || len(decoded) == 0 || len(decoded) > maxResponseBytes {
		return nil, nil, "", fmt.Errorf("%w: Payment-Signature is not bounded base64 JSON", ErrInvalid)
	}
	var envelope struct {
		X402Version int             `json:"x402Version"`
		Accepted    json.RawMessage `json:"accepted"`
	}
	if err := json.Unmarshal(decoded, &envelope); err != nil || envelope.X402Version != 2 || len(envelope.Accepted) == 0 {
		return nil, nil, "", fmt.Errorf("%w: Payment-Signature must be a complete x402 v2 payload", ErrInvalid)
	}
	digest := sha256.Sum256(decoded)
	return decoded, envelope.Accepted, "sha256:" + hex.EncodeToString(digest[:]), nil
}

func jsonEqual(left, right []byte) bool {
	var l, r any
	return json.Unmarshal(left, &l) == nil && json.Unmarshal(right, &r) == nil && bytes.Equal(mustJSON(l), mustJSON(r))
}

func mustJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}

// SQLiteInboundRepository owns the atomic claim, terminal receipt, and inflow
// outbox commit used by the gate.
type SQLiteInboundRepository struct {
	db interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	}
}

func NewSQLiteInboundRepository(db interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	ExecContext(context.Context, string, ...any) (sql.Result, error)
},
) *SQLiteInboundRepository {
	return &SQLiteInboundRepository{db: db}
}

func (r *SQLiteInboundRepository) CreatePrice(ctx context.Context, value Price) (Price, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Price{}, err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `INSERT INTO x402_prices(id,resource_url,description,mime_type,network,scheme,amount,amount_minor,currency,pay_to,asset,asset_decimals,max_timeout_seconds,extra_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(id) DO NOTHING`, value.ID, value.ResourceURL, value.Description, value.MIMEType, value.Network, value.Scheme, value.Amount, value.AmountMinor, value.Currency, value.PayTo, value.Asset, value.AssetDecimals, value.MaxTimeoutSeconds, value.ExtraJSON, value.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Price{}, err
	}
	stored, err := scanPrice(tx.QueryRowContext(ctx, selectPrice+` WHERE id=?`, value.ID))
	if err != nil {
		return Price{}, err
	}
	if stored.ResourceURL != value.ResourceURL || stored.Description != value.Description || stored.MIMEType != value.MIMEType || stored.Network != value.Network || stored.Scheme != value.Scheme || stored.Amount != value.Amount || stored.AmountMinor != value.AmountMinor || stored.Currency != value.Currency || stored.PayTo != value.PayTo || stored.Asset != value.Asset || stored.AssetDecimals != value.AssetDecimals || stored.MaxTimeoutSeconds != value.MaxTimeoutSeconds || !jsonEqual([]byte(stored.ExtraJSON), []byte(value.ExtraJSON)) {
		return Price{}, fmt.Errorf("%w: price id already names different terms", ErrInvalid)
	}
	if err := tx.Commit(); err != nil {
		return Price{}, err
	}
	return stored, nil
}

func (r *SQLiteInboundRepository) GetPrice(ctx context.Context, id string) (Price, error) {
	return scanPrice(r.db.QueryRowContext(ctx, selectPrice+` WHERE id=?`, id))
}

func (r *SQLiteInboundRepository) GetAdmissionByDigest(ctx context.Context, digest string) (Admission, error) {
	return scanAdmission(r.db.QueryRowContext(ctx, selectAdmission+` WHERE payload_digest=?`, digest))
}

func (r *SQLiteInboundRepository) GetResourceAdmission(ctx context.Context, idOrDigest string) (Admission, error) {
	return scanResourceAdmission(r.db.QueryRowContext(ctx, selectResourceAdmission+` WHERE id=? OR payload_digest=? LIMIT 1`, idOrDigest, idOrDigest))
}

func (r *SQLiteInboundRepository) ClaimResourceAdmission(ctx context.Context, value Admission) (Admission, bool, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Admission{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := claimPaymentPayload(ctx, tx, value.PayloadDigest, value.ID, "resource", value.CreatedAt); err != nil {
		return Admission{}, false, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO x402_resource_admissions(id,price_id,payload_digest,status,payer,transaction_id,network,mode,recovery_reference,job_id,resource_id,order_id,detail,created_at,updated_at) VALUES(?,?,?,?,?,'',?,?,?,?,?,?,'',?,?) ON CONFLICT DO NOTHING`, value.ID, value.PriceID, value.PayloadDigest, value.Status, value.Payer, value.Network, value.Mode, value.RecoveryReference, value.JobID, value.ResourceID, value.OrderID, value.CreatedAt.Format(time.RFC3339Nano), value.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Admission{}, false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Admission{}, false, err
	}
	stored, err := scanResourceAdmission(tx.QueryRowContext(ctx, selectResourceAdmission+` WHERE payload_digest=? OR (job_id=? AND resource_id=? AND order_id=?) LIMIT 1`, value.PayloadDigest, value.JobID, value.ResourceID, value.OrderID))
	if err != nil {
		return Admission{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Admission{}, false, err
	}
	return stored, rows == 1, nil
}

func (r *SQLiteInboundRepository) BeginResourceSettlement(ctx context.Context, id string, updatedAt time.Time) (Admission, bool, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Admission{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	previous, err := scanResourceAdmission(tx.QueryRowContext(ctx, selectResourceAdmission+` WHERE id=?`, id))
	if err != nil {
		return Admission{}, false, err
	}
	staleBefore := updatedAt.UTC().Add(-90 * time.Second).Format(time.RFC3339Nano)
	result, err := tx.ExecContext(ctx, `UPDATE x402_resource_admissions SET status=CASE WHEN status IN ('settling','recovering') THEN 'recovering' ELSE 'settling' END,detail='',updated_at=? WHERE id=? AND (status IN ('verified','unknown') OR (status IN ('settling','recovering') AND updated_at<=?))`, updatedAt.UTC().Format(time.RFC3339Nano), id, staleBefore)
	if err != nil {
		return Admission{}, false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Admission{}, false, err
	}
	stored, err := scanResourceAdmission(tx.QueryRowContext(ctx, selectResourceAdmission+` WHERE id=?`, id))
	if err != nil {
		return Admission{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Admission{}, false, err
	}
	stored.Recovery = rows == 1 && (previous.Status == "unknown" || previous.Status == "settling" || previous.Status == "recovering")
	return stored, rows == 1, nil
}

func (r *SQLiteInboundRepository) CompleteResourceSettlement(ctx context.Context, id, status string, result SettleResult, updatedAt time.Time, price Price) (Admission, error) {
	if status != "settled" && status != "failed" && status != "unknown" {
		return Admission{}, fmt.Errorf("invalid resource admission status %q", status)
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Admission{}, err
	}
	defer func() { _ = tx.Rollback() }()
	execResult, err := tx.ExecContext(ctx, `UPDATE x402_resource_admissions SET status=?,payer=?,transaction_id=?,network=?,detail=?,updated_at=? WHERE id=? AND status IN ('settling','recovering')`, status, result.Payer, result.Transaction, result.Network, result.Reason, updatedAt.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return Admission{}, err
	}
	rows, err := execResult.RowsAffected()
	if err != nil || rows != 1 {
		return Admission{}, fmt.Errorf("complete resource admission: expected one settling row")
	}
	if status == "settled" {
		_, err = tx.ExecContext(ctx, `INSERT INTO ledger_emissions(id,settlement_id,external_id,adapter_id,account_id,book_id,amount_minor,currency,basis,occurred_at,fetched_at,description,status,attempts,last_error,created_at,accepted_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,0,'',?,'') ON CONFLICT DO NOTHING`, id+":ledger", id, "treasury-x402-inflow:"+result.Transaction, "treasury-x402", "", "", price.AmountMinor, price.Currency, "authoritative", updatedAt.Format(time.RFC3339Nano), updatedAt.Format(time.RFC3339Nano), "Treasury x402 receipt "+result.Transaction+" for "+price.ResourceURL, "queued", updatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return Admission{}, fmt.Errorf("queue inbound money-ledger emission: %w", err)
		}
	}
	stored, err := scanResourceAdmission(tx.QueryRowContext(ctx, selectResourceAdmission+` WHERE id=?`, id))
	if err != nil {
		return Admission{}, err
	}
	if err := tx.Commit(); err != nil {
		return Admission{}, err
	}
	return stored, nil
}

func (r *SQLiteInboundRepository) ClearResourceRecoveryReference(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE x402_resource_admissions SET recovery_reference='' WHERE id=? AND status IN ('settled','failed','cancelled')`, id)
	return err
}

func (r *SQLiteInboundRepository) CancelResourceAdmission(ctx context.Context, id, jobID, resourceID, orderID string, updatedAt time.Time) (Admission, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Admission{}, err
	}
	defer func() { _ = tx.Rollback() }()
	stored, err := scanResourceAdmission(tx.QueryRowContext(ctx, selectResourceAdmission+` WHERE id=?`, id))
	if err != nil {
		return Admission{}, err
	}
	if stored.JobID != jobID || stored.ResourceID != resourceID || stored.OrderID != orderID {
		return Admission{}, ErrBindingConflict
	}
	if stored.Status == "cancelled" {
		if err := tx.Commit(); err != nil {
			return Admission{}, err
		}
		return stored, nil
	}
	result, err := tx.ExecContext(ctx, `UPDATE x402_resource_admissions SET status='cancelled',detail='resource verifier declined delivery',updated_at=? WHERE id=? AND status='verified'`, updatedAt.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return Admission{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Admission{}, err
	}
	if rows != 1 {
		return Admission{}, fmt.Errorf("%w: only a verified admission can be cancelled", ErrStateConflict)
	}
	cancelled, err := scanResourceAdmission(tx.QueryRowContext(ctx, selectResourceAdmission+` WHERE id=?`, id))
	if err != nil {
		return Admission{}, err
	}
	if err := tx.Commit(); err != nil {
		return Admission{}, err
	}
	return cancelled, nil
}

func (r *SQLiteInboundRepository) ClaimAdmission(ctx context.Context, value Admission) (Admission, bool, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Admission{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := claimPaymentPayload(ctx, tx, value.PayloadDigest, value.ID, "inbound", value.CreatedAt); err != nil {
		return Admission{}, false, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO x402_inbound_admissions(id,price_id,payload_digest,status,payer,transaction_id,network,detail,created_at,updated_at) VALUES(?,?,?,?,?,'',?,'',?,?) ON CONFLICT(payload_digest) DO NOTHING`, value.ID, value.PriceID, value.PayloadDigest, "calling", value.Payer, value.Network, value.CreatedAt.Format(time.RFC3339Nano), value.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Admission{}, false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Admission{}, false, err
	}
	stored, err := scanAdmission(tx.QueryRowContext(ctx, selectAdmission+` WHERE payload_digest=?`, value.PayloadDigest))
	if err != nil {
		return Admission{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Admission{}, false, err
	}
	return stored, rows == 1, nil
}

// claimPaymentPayload makes the legacy immediate-settlement API and the
// resource-bound deferred API mutually exclusive for a signed payload. The
// transaction-local historical lookups also protect databases created before
// the shared claim table was introduced.
func claimPaymentPayload(ctx context.Context, tx *sql.Tx, digest, admissionID, kind string, createdAt time.Time) error {
	otherTable, otherKind := "x402_inbound_admissions", "inbound"
	sameTable := "x402_resource_admissions"
	if kind == "inbound" {
		otherTable, otherKind = "x402_resource_admissions", "resource"
		sameTable = "x402_inbound_admissions"
	}
	var historicalID string
	err := tx.QueryRowContext(ctx, `SELECT id FROM `+otherTable+` WHERE payload_digest=? LIMIT 1`, digest).Scan(&historicalID)
	if err == nil {
		return fmt.Errorf("%w: payment payload already belongs to %s admission %s", ErrBindingConflict, otherKind, historicalID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	// When upgrading an existing database, a same-kind admission may predate
	// this shared claim table. Preserve its original admission identity.
	err = tx.QueryRowContext(ctx, `SELECT id FROM `+sameTable+` WHERE payload_digest=? LIMIT 1`, digest).Scan(&historicalID)
	if err == nil {
		admissionID = historicalID
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO x402_payment_payload_claims(payload_digest,admission_id,admission_kind,created_at) VALUES(?,?,?,?) ON CONFLICT(payload_digest) DO NOTHING`, digest, admissionID, kind, createdAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	var storedKind string
	if err := tx.QueryRowContext(ctx, `SELECT admission_kind FROM x402_payment_payload_claims WHERE payload_digest=?`, digest).Scan(&storedKind); err != nil {
		return err
	}
	if storedKind != kind {
		return fmt.Errorf("%w: payment payload is already claimed by %s admission", ErrBindingConflict, storedKind)
	}
	return nil
}

func (r *SQLiteInboundRepository) CompleteAdmission(ctx context.Context, id, status string, result SettleResult, updatedAt time.Time, price Price) (Admission, error) {
	if status != "settled" && status != "failed" && status != "unknown" {
		return Admission{}, fmt.Errorf("invalid admission status %q", status)
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Admission{}, err
	}
	defer func() { _ = tx.Rollback() }()
	execResult, err := tx.ExecContext(ctx, `UPDATE x402_inbound_admissions SET status=?,payer=?,transaction_id=?,network=?,detail=?,updated_at=? WHERE id=? AND status='calling'`, status, result.Payer, result.Transaction, result.Network, result.Reason, updatedAt.Format(time.RFC3339Nano), id)
	if err != nil {
		return Admission{}, err
	}
	rows, err := execResult.RowsAffected()
	if err != nil || rows != 1 {
		return Admission{}, fmt.Errorf("complete inbound admission: expected one calling row")
	}
	stored, err := scanAdmission(tx.QueryRowContext(ctx, selectAdmission+` WHERE id=?`, id))
	if err != nil {
		return Admission{}, err
	}
	if status == "settled" {
		_, err = tx.ExecContext(ctx, `INSERT INTO ledger_emissions(id,settlement_id,external_id,adapter_id,account_id,book_id,amount_minor,currency,basis,occurred_at,fetched_at,description,status,attempts,last_error,created_at,accepted_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,0,'',?,'')`, id+":ledger", id, "treasury-x402-inflow:"+result.Transaction, "treasury-x402", "", "", price.AmountMinor, price.Currency, "authoritative", updatedAt.Format(time.RFC3339Nano), updatedAt.Format(time.RFC3339Nano), "Treasury x402 receipt "+result.Transaction+" for "+price.ResourceURL, "queued", updatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return Admission{}, fmt.Errorf("queue inbound money-ledger emission: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Admission{}, err
	}
	return stored, nil
}

const (
	selectPrice             = `SELECT id,resource_url,description,mime_type,network,scheme,amount,amount_minor,currency,pay_to,asset,asset_decimals,max_timeout_seconds,extra_json,created_at FROM x402_prices`
	selectAdmission         = `SELECT id,price_id,payload_digest,status,payer,transaction_id,network,detail,created_at,updated_at FROM x402_inbound_admissions`
	selectResourceAdmission = `SELECT id,price_id,payload_digest,status,payer,transaction_id,network,mode,recovery_reference,job_id,resource_id,order_id,detail,created_at,updated_at FROM x402_resource_admissions`
)

type rowScanner interface{ Scan(...any) error }

func scanPrice(row rowScanner) (Price, error) {
	var value Price
	var created string
	err := row.Scan(&value.ID, &value.ResourceURL, &value.Description, &value.MIMEType, &value.Network, &value.Scheme, &value.Amount, &value.AmountMinor, &value.Currency, &value.PayTo, &value.Asset, &value.AssetDecimals, &value.MaxTimeoutSeconds, &value.ExtraJSON, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Price{}, ErrNotFound
	}
	if err != nil {
		return Price{}, err
	}
	value.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	return value, err
}

func scanAdmission(row rowScanner) (Admission, error) {
	var value Admission
	var created, updated string
	err := row.Scan(&value.ID, &value.PriceID, &value.PayloadDigest, &value.Status, &value.Payer, &value.TransactionID, &value.Network, &value.Detail, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Admission{}, ErrNotFound
	}
	if err != nil {
		return Admission{}, err
	}
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return Admission{}, err
	}
	value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return value, err
}

func scanResourceAdmission(row rowScanner) (Admission, error) {
	var value Admission
	var created, updated string
	err := row.Scan(&value.ID, &value.PriceID, &value.PayloadDigest, &value.Status, &value.Payer, &value.TransactionID, &value.Network, &value.Mode, &value.RecoveryReference, &value.JobID, &value.ResourceID, &value.OrderID, &value.Detail, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return Admission{}, ErrNotFound
	}
	if err != nil {
		return Admission{}, err
	}
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return Admission{}, err
	}
	value.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return value, err
}

var _ InboundRepository = (*SQLiteInboundRepository)(nil)
