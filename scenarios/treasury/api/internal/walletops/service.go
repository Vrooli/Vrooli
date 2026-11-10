// Package walletops connects budget identities, funding and withdrawals to
// Treasury's existing custody, receipt, authorization and settlement owners.
package walletops

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/vrooli/api-core/schedule"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	fundingpb "github.com/vrooli/vrooli/packages/proto/gen/go/treasury/v1/funding"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/approval"
	"treasury/internal/authorization"
	"treasury/internal/baseusdc"
	"treasury/internal/book"
	"treasury/internal/budget"
	"treasury/internal/identity"
	"treasury/internal/instrument"
	"treasury/internal/mandate"
	"treasury/internal/nativeusdc"
	"treasury/internal/receiptfunding"
	"treasury/internal/settlement"
	"treasury/internal/wallet"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

var (
	ErrInvalid  = errors.New("invalid wallet operation")
	ErrNotFound = errors.New("wallet operation not found")
	ErrConflict = errors.New("wallet operation conflicts with retained scope")
)

type DB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}
type Notification interface {
	Received(context.Context, string, string, string, string, string) error
}
type Config struct {
	DB             DB
	Wallets        *wallet.Service
	Books          book.Service
	Budgets        budget.Service
	Receipts       *baseusdc.Service
	Funding        *receiptfunding.Service
	Native         *nativeusdc.Adapter
	Mandates       mandate.Service
	Authorizations *authorization.Service
	Approvals      *approval.Service
	Instruments    *instrument.Service
	Credentials    instrument.CredentialWriter
	Settlements    *settlement.Service
	Notifications  Notification
	Clock          schedule.Clock
}
type Service struct{ Config }

func New(c Config) *Service { return &Service{Config: c} }

type binding struct{ budgetID, walletID, parentID, operator, network, agent, run string }

func (s *Service) load(ctx context.Context, id string) (binding, error) {
	var b binding
	err := s.DB.QueryRowContext(ctx, `SELECT budget_id,wallet_id,parent_wallet_id,operator_identity,network,agent_subject,agent_run_id FROM budget_wallets WHERE budget_id=?`, id).Scan(&b.budgetID, &b.walletID, &b.parentID, &b.operator, &b.network, &b.agent, &b.run)
	if errors.Is(err, sql.ErrNoRows) {
		return b, ErrNotFound
	}
	return b, err
}

func stableID(prefix string, parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return fmt.Sprintf("%s-%x", prefix, hash[:16])
}

func (s *Service) Provision(ctx context.Context, operator string, in *fundingpb.ProvisionBudgetWalletRequest) (*fundingpb.BudgetWallet, error) {
	if s.Wallets == nil || in == nil || in.BudgetId == "" || in.Network != baseusdc.Network || (in.AgentSubject == "") != (in.AgentRunId == "") {
		return nil, ErrInvalid
	}
	policy, err := s.Budgets.Get(ctx, in.BudgetId)
	if err != nil {
		return nil, err
	}
	owner, err := s.Books.Get(ctx, policy.BookID)
	if err != nil {
		return nil, err
	}
	if owner.BeneficiaryIdentity != operator {
		return nil, ErrInvalid
	}
	rootID := stableID("budget-root", operator, policy.BookID, in.Network)
	root, err := s.Wallets.Create(ctx, wallet.CreateInput{ID: rootID, BookID: policy.BookID, Network: in.Network, OperatorIdentity: operator})
	if err != nil {
		return nil, err
	}
	childID := stableID("budget", policy.ID, in.Network)
	child, err := s.Wallets.CreateDerived(ctx, root.ID, "budget:"+policy.ID, wallet.CreateInput{ID: childID, BookID: policy.BookID, Network: in.Network, OperatorIdentity: operator})
	if err != nil {
		return nil, err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO budget_wallets(budget_id,wallet_id,parent_wallet_id,operator_identity,network,agent_subject,agent_run_id,created_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(budget_id) DO NOTHING`, policy.ID, child.ID, root.ID, operator, in.Network, in.AgentSubject, in.AgentRunId, s.Clock.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	b, err := s.load(ctx, policy.ID)
	if err != nil {
		return nil, err
	}
	if b.walletID != child.ID || b.operator != operator || b.network != in.Network || b.agent != in.AgentSubject || b.run != in.AgentRunId {
		return nil, ErrConflict
	}
	if in.GetMandateId() != "" {
		var session MandateSessionWallet
		if in.GetRotateSession() {
			session, err = s.RotateMandateSession(ctx, operator, in.GetMandateId())
		} else {
			session, err = s.ProvisionMandateSession(ctx, operator, in.GetMandateId())
		}
		if err != nil {
			return nil, err
		}
		return sessionBudgetWallet(session), nil
	}
	return publicWallet(b, child), nil
}

func (s *Service) Get(ctx context.Context, operator, budgetID string) (*fundingpb.BudgetWallet, error) {
	b, err := s.load(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	if b.operator != operator {
		return nil, ErrNotFound
	}
	w, err := s.Wallets.Get(ctx, b.walletID, b.network, operator)
	if err != nil {
		return nil, err
	}
	return publicWallet(b, w), nil
}

func (s *Service) GetForOperator(ctx context.Context, operator, budgetID, mandateID string) (*fundingpb.BudgetWallet, error) {
	if mandateID == "" {
		return s.Get(ctx, operator, budgetID)
	}
	record, err := s.sessionByMandate(ctx, mandateID, "active")
	if err != nil || record.budgetID != budgetID || record.operator != operator || !s.Clock.Now().Before(record.expiresAt) {
		return nil, ErrNotFound
	}
	if _, err := s.Mandates.RequireLive(ctx, mandateID); err != nil {
		return nil, ErrSessionInactive
	}
	session, err := s.publicSession(ctx, record)
	if err != nil {
		return nil, err
	}
	return sessionBudgetWallet(session), nil
}

func (s *Service) GetForAgent(ctx context.Context, budgetID, mandateID string, claims identity.Claims) (*fundingpb.BudgetWallet, error) {
	if err := identity.Authorize(claims, "treasury.read", "treasury:budget/"+budgetID, s.Clock.Now()); err != nil {
		return nil, err
	}
	b, err := s.load(ctx, budgetID)
	if err != nil {
		return nil, err
	}
	if b.agent == "" || b.agent != claims.Subject || b.run != claims.RunID {
		return nil, ErrNotFound
	}
	if mandateID != "" {
		session, sessionErr := s.GetMandateSessionForAgent(ctx, budgetID, mandateID, claims)
		if sessionErr != nil {
			return nil, sessionErr
		}
		return sessionBudgetWallet(session), nil
	}
	return s.Get(ctx, b.operator, budgetID)
}

func publicWallet(b binding, w wallet.Wallet) *fundingpb.BudgetWallet {
	return &fundingpb.BudgetWallet{BudgetId: b.budgetID, WalletId: w.ID, Network: w.Network, Address: w.Address, AgentSubject: b.agent, AgentRunId: b.run}
}

func sessionBudgetWallet(session MandateSessionWallet) *fundingpb.BudgetWallet {
	return &fundingpb.BudgetWallet{
		BudgetId: session.BudgetID, WalletId: session.WalletID, Network: session.Network, Address: session.Address,
		AgentSubject: session.AgentSubject, AgentRunId: session.AgentRunID, MandateId: session.MandateID,
		SessionGeneration: session.Generation, SessionExpiresAt: timestamppb.New(session.ExpiresAt),
	}
}

var positiveAtoms = regexp.MustCompile(`^[1-9][0-9]*$`)

type fundingIntent struct {
	id, budgetID, operator, atoms, startBlock, state string
	txHash, deliveryID, notificationState, lastError string
	request                                          *fundingpb.CreateFundingIntentRequest
	createdAt, updatedAt                             time.Time
}

func (s *Service) now() time.Time {
	if s.Clock == nil {
		return time.Now().UTC()
	}
	return s.Clock.Now().UTC()
}

func newID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + "-" + hex.EncodeToString(b[:]), nil
}

// CreateFundingIntent snapshots the chain head before returning a payment URI.
// Only a finalized transfer at or after that block can satisfy the intent.
func (s *Service) CreateFundingIntent(ctx context.Context, operator string, in *fundingpb.CreateFundingIntentRequest) (*fundingpb.FundingIntent, error) {
	if s == nil || s.DB == nil || s.Wallets == nil || s.Receipts == nil || s.Funding == nil || operator == "" || in == nil || in.BudgetId == "" || !positiveAtoms.MatchString(in.AtomicAmount) || in.Valuation == nil || in.Valuation.Currency == "" || in.LedgerBookId == "" || in.LedgerAccountId == "" {
		return nil, ErrInvalid
	}
	b, err := s.load(ctx, in.BudgetId)
	if err != nil {
		return nil, err
	}
	if b.operator != operator || b.network != baseusdc.Network {
		return nil, ErrNotFound
	}
	policy, err := s.Budgets.Get(ctx, b.budgetID)
	if err != nil {
		return nil, err
	}
	if policy.Currency != strings.ToUpper(in.Valuation.Currency) {
		return nil, ErrInvalid
	}
	if s.Native == nil || s.Native.Mode() != "local-fork" {
		return nil, fmt.Errorf("%w: funding intents require the explicitly configured local Base fork", ErrInvalid)
	}
	var head string
	if err = s.Native.Call(ctx, "eth_blockNumber", []any{}, &head); err != nil {
		return nil, err
	}
	start, err := rpcUint(head)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(in.Id)
	if id == "" {
		id, err = newID("funding")
		if err != nil {
			return nil, err
		}
	}
	if len(id) > 120 || strings.ContainsAny(id, " \r\n\t") {
		return nil, ErrInvalid
	}
	copy := proto.Clone(in).(*fundingpb.CreateFundingIntentRequest)
	copy.Id = id
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(copy)
	if err != nil {
		return nil, err
	}
	now := s.now()
	_, err = s.DB.ExecContext(ctx, `INSERT INTO funding_intents(id,budget_id,operator_identity,atomic_amount,request_proto,start_block,state,created_at,updated_at) VALUES(?,?,?,?,?,?,'waiting',?,?) ON CONFLICT(id) DO NOTHING`, id, b.budgetID, operator, in.AtomicAmount, encoded, start.String(), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	v, err := s.loadIntent(ctx, operator, id)
	if err != nil {
		return nil, err
	}
	if v.budgetID != b.budgetID || v.atoms != in.AtomicAmount || !proto.Equal(v.request, copy) {
		return nil, ErrConflict
	}
	return s.publicIntent(ctx, v)
}

func (s *Service) ConfirmFundingTransfer(ctx context.Context, operator string, in *fundingpb.ConfirmFundingTransferRequest) (*fundingpb.FundingIntent, error) {
	if in == nil || operator == "" || in.Id == "" || in.TransactionHash == "" || s.Receipts == nil || s.Funding == nil {
		return nil, ErrInvalid
	}
	v, err := s.loadIntent(ctx, operator, in.Id)
	if err != nil {
		return nil, err
	}
	if v.state == "credited" {
		if !strings.EqualFold(v.txHash, in.TransactionHash) {
			return nil, ErrConflict
		}
		_ = s.retryNotification(ctx, v)
		v, err = s.loadIntent(ctx, operator, in.Id)
		if err != nil {
			return nil, err
		}
		return s.publicIntent(ctx, v)
	}
	b, err := s.load(ctx, v.budgetID)
	if err != nil {
		return nil, err
	}
	w, err := s.Wallets.Get(ctx, b.walletID, b.network, operator)
	if err != nil {
		return nil, err
	}
	check, err := s.Receipts.Verify(ctx, operator, w, in.TransactionHash)
	if err != nil {
		_ = s.updateIntent(ctx, v.id, "waiting", "", "", v.notificationState, err)
		return nil, err
	}
	start, ok := new(big.Int).SetString(v.startBlock, 10)
	if !ok {
		return nil, ErrInvalid
	}
	var matched *baseusdc.Observation
	for i := range check.Transfers {
		obs := &check.Transfers[i]
		if new(big.Int).SetUint64(obs.BlockNumber).Cmp(start) < 0 || obs.AtomicAmount != v.atoms {
			continue
		}
		if matched != nil {
			return nil, fmt.Errorf("%w: multiple exact amount transfer logs in transaction", ErrInvalid)
		}
		matched = obs
	}
	if matched == nil {
		return nil, fmt.Errorf("%w: no exact native-USDC transfer at or after intent start block", ErrInvalid)
	}
	if v.txHash != "" && !strings.EqualFold(v.txHash, in.TransactionHash) {
		return nil, ErrConflict
	}
	if v.deliveryID == "" {
		attribution := v.request.Attribution
		if attribution != nil {
			if attribution.PayerAddress != "" && !strings.EqualFold(attribution.PayerAddress, matched.From) {
				return nil, fmt.Errorf("%w: claimed payer address does not match the verified Transfer sender", ErrInvalid)
			}
			attribution = proto.Clone(attribution).(*ingestpb.CryptoPayerAttribution)
			attribution.PayerAddress = strings.ToLower(matched.From)
		}
		delivery, e := s.Funding.Submit(ctx, receiptfunding.SubmitInput{OperatorIdentity: operator, WalletID: w.ID, TransactionHash: matched.TransactionHash, CheckID: check.ID, BudgetID: v.budgetID, LedgerBookID: v.request.LedgerBookId, LedgerAccountID: v.request.LedgerAccountId, LogIndex: matched.LogIndex, Action: ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL, Valuation: v.request.Valuation, Attribution: attribution})
		if e != nil {
			_ = s.updateIntent(ctx, v.id, "observed", matched.TransactionHash, "", v.notificationState, e)
			return nil, e
		}
		v.deliveryID = delivery.ID
	}
	_ = s.updateIntent(ctx, v.id, "delivering", matched.TransactionHash, v.deliveryID, v.notificationState, nil)
	if err = s.Funding.DrainPending(ctx); err != nil {
		_ = s.updateIntent(ctx, v.id, "delivering", matched.TransactionHash, v.deliveryID, v.notificationState, err)
		return nil, err
	}
	delivery, err := s.Funding.Get(ctx, v.deliveryID, operator)
	if err != nil {
		return nil, err
	}
	if delivery.Status != "accepted" || delivery.Receipt == nil || delivery.Receipt.Id == "" {
		return nil, fmt.Errorf("%w: accepted ledger receipt is not available", ErrInvalid)
	}
	if err = s.updateIntent(ctx, v.id, "credited", matched.TransactionHash, v.deliveryID, "pending", nil); err != nil {
		return nil, err
	}
	v, err = s.loadIntent(ctx, operator, v.id)
	if err != nil {
		return nil, err
	}
	_ = s.retryNotification(ctx, v)
	v, err = s.loadIntent(ctx, operator, v.id)
	if err != nil {
		return nil, err
	}
	return s.publicIntent(ctx, v)
}

func (s *Service) loadIntent(ctx context.Context, operator, id string) (fundingIntent, error) {
	var v fundingIntent
	var raw []byte
	var created, updated string
	err := s.DB.QueryRowContext(ctx, `SELECT id,budget_id,operator_identity,atomic_amount,request_proto,start_block,state,transaction_hash,receipt_delivery_id,notification_state,last_error,created_at,updated_at FROM funding_intents WHERE id=? AND operator_identity=?`, id, operator).Scan(&v.id, &v.budgetID, &v.operator, &v.atoms, &raw, &v.startBlock, &v.state, &v.txHash, &v.deliveryID, &v.notificationState, &v.lastError, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return v, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	v.request = new(fundingpb.CreateFundingIntentRequest)
	if err = proto.Unmarshal(raw, v.request); err != nil {
		return v, err
	}
	v.createdAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return v, err
	}
	v.updatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return v, err
}

func (s *Service) updateIntent(ctx context.Context, id, state, txHash, deliveryID, notification string, cause error) error {
	last := ""
	if cause != nil {
		last = cause.Error()
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE funding_intents SET state=?,transaction_hash=CASE WHEN ?='' THEN transaction_hash ELSE ? END,receipt_delivery_id=CASE WHEN ?='' THEN receipt_delivery_id ELSE ? END,notification_state=?,last_error=?,updated_at=? WHERE id=?`, state, txHash, txHash, deliveryID, deliveryID, notification, last, s.now().Format(time.RFC3339Nano), id)
	return err
}

func (s *Service) publicIntent(ctx context.Context, v fundingIntent) (*fundingpb.FundingIntent, error) {
	b, err := s.load(ctx, v.budgetID)
	if err != nil {
		return nil, err
	}
	w, err := s.Wallets.Get(ctx, b.walletID, b.network, v.operator)
	if err != nil {
		return nil, err
	}
	return &fundingpb.FundingIntent{Id: v.id, BudgetId: v.budgetID, Wallet: publicWallet(b, w), AtomicAmount: v.atoms, PaymentUri: fmt.Sprintf("ethereum:%s@8453/transfer?address=%s&uint256=%s", baseusdc.USDCAddress, w.Address, v.atoms), State: v.state, TransactionHash: v.txHash, ReceiptDeliveryId: v.deliveryID, CreatedAt: timestamppb.New(v.createdAt), UpdatedAt: timestamppb.New(v.updatedAt), NotificationState: v.notificationState, LastError: v.lastError}, nil
}

func (s *Service) GetFundingIntent(ctx context.Context, operator, id string) (*fundingpb.FundingIntent, error) {
	v, err := s.loadIntent(ctx, operator, id)
	if err != nil {
		return nil, err
	}
	if v.state == "credited" && v.notificationState != "sent" {
		_ = s.retryNotification(ctx, v)
		v, err = s.loadIntent(ctx, operator, id)
		if err != nil {
			return nil, err
		}
	}
	return s.publicIntent(ctx, v)
}

func (s *Service) ListFundingIntents(ctx context.Context, operator, budgetID string) ([]*fundingpb.FundingIntent, error) {
	if operator == "" || budgetID == "" {
		return nil, ErrInvalid
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM funding_intents WHERE operator_identity=? AND budget_id=? ORDER BY created_at DESC,id DESC`, operator, budgetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	out := make([]*fundingpb.FundingIntent, 0, len(ids))
	for _, id := range ids {
		v, e := s.loadIntent(ctx, operator, id)
		if e != nil {
			return nil, e
		}
		item, e := s.publicIntent(ctx, v)
		if e != nil {
			return nil, e
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) retryNotification(ctx context.Context, v fundingIntent) error {
	if v.state != "credited" || v.notificationState == "sent" {
		return nil
	}
	var err error
	if s.Notifications == nil {
		err = errors.New("funding notification is not configured")
	} else {
		delivery, e := s.Funding.Get(ctx, v.deliveryID, v.operator)
		err = e
		if err == nil && (delivery.Receipt == nil || delivery.Receipt.Id == "") {
			err = ErrInvalid
		}
		if err == nil {
			b, e := s.load(ctx, v.budgetID)
			if e != nil {
				err = e
			} else if w, e := s.Wallets.Get(ctx, b.walletID, b.network, v.operator); e != nil {
				err = e
			} else {
				err = s.Notifications.Received(ctx, v.operator, v.budgetID, w.Address, v.atoms, delivery.Receipt.Id)
			}
		}
	}
	state := "sent"
	if err != nil {
		state = "failed"
	}
	_ = s.updateIntent(ctx, v.id, "credited", v.txHash, v.deliveryID, state, err)
	return err
}

func (s *Service) RetryNotifications(ctx context.Context) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,operator_identity FROM funding_intents WHERE state='credited' AND notification_state IN ('pending','failed') ORDER BY updated_at,id LIMIT 100`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type entry struct{ id, operator string }
	var entries []entry
	for rows.Next() {
		var e entry
		if err = rows.Scan(&e.id, &e.operator); err != nil {
			return err
		}
		entries = append(entries, e)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	var failures []error
	for _, e := range entries {
		v, x := s.loadIntent(ctx, e.operator, e.id)
		if x == nil {
			x = s.retryNotification(ctx, v)
		}
		if x != nil {
			failures = append(failures, x)
		}
	}
	return errors.Join(failures...)
}

func rpcUint(value string) (*big.Int, error) {
	if !strings.HasPrefix(value, "0x") {
		return nil, ErrInvalid
	}
	n, ok := new(big.Int).SetString(value[2:], 16)
	if !ok || n.Sign() < 0 {
		return nil, ErrInvalid
	}
	return n, nil
}

func (s *Service) SetWithdrawalDestination(ctx context.Context, operator string, in *fundingpb.SetWithdrawalDestinationRequest) (*fundingpb.WithdrawalDestination, error) {
	if in == nil || in.Destination == nil || operator == "" || in.Destination.BudgetId == "" || !common.IsHexAddress(in.Destination.Address) || common.HexToAddress(in.Destination.Address) == (common.Address{}) || strings.TrimSpace(in.Destination.Label) == "" {
		return nil, ErrInvalid
	}
	b, err := s.load(ctx, in.Destination.BudgetId)
	if err != nil {
		return nil, err
	}
	if b.operator != operator {
		return nil, ErrNotFound
	}
	address := strings.ToLower(common.HexToAddress(in.Destination.Address).Hex())
	_, err = s.DB.ExecContext(ctx, `INSERT INTO withdrawal_destinations(budget_id,address,label,operator_identity,created_at) VALUES(?,?,?,?,?) ON CONFLICT(budget_id,address) DO UPDATE SET label=excluded.label WHERE operator_identity=excluded.operator_identity`, b.budgetID, address, strings.TrimSpace(in.Destination.Label), operator, s.now().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return &fundingpb.WithdrawalDestination{BudgetId: b.budgetID, Address: address, Label: strings.TrimSpace(in.Destination.Label)}, nil
}

func (s *Service) PreviewWithdrawal(ctx context.Context, operator string, in *fundingpb.PreviewWithdrawalRequest) (*fundingpb.WithdrawalQuote, error) {
	if s == nil || in == nil || operator == "" || in.BudgetId == "" || in.MandateId == "" || !positiveAtoms.MatchString(in.AtomicAmount) || in.Valuation == nil || in.Valuation.Currency == "" || in.Valuation.AmountMinor <= 0 || in.FeeValuationMinor < 0 || s.Native == nil || s.Native.Mode() != "local-fork" || !common.IsHexAddress(in.Destination) {
		return nil, ErrInvalid
	}
	b, err := s.load(ctx, in.BudgetId)
	if err != nil {
		return nil, err
	}
	if b.operator != operator {
		return nil, ErrNotFound
	}
	if b.agent == "" || b.run == "" || s.Mandates == nil {
		return nil, fmt.Errorf("%w: a run-bound budget agent and live mandate are required for withdrawal", ErrInvalid)
	}
	grant, err := s.Mandates.RequireLive(ctx, in.MandateId)
	if err != nil {
		return nil, err
	}
	if grant.BudgetID != b.budgetID || grant.BookID == "" || grant.AgentSubject != b.agent || grant.AgentRunID != b.run {
		return nil, fmt.Errorf("%w: withdrawal mandate must match this budget's bound agent run", ErrInvalid)
	}
	destination := strings.ToLower(common.HexToAddress(in.Destination).Hex())
	var allowlisted string
	if err = s.DB.QueryRowContext(ctx, `SELECT address FROM withdrawal_destinations WHERE budget_id=? AND operator_identity=? AND address=?`, b.budgetID, operator, destination).Scan(&allowlisted); err != nil {
		return nil, ErrInvalid
	}
	policy, err := s.Budgets.Get(ctx, b.budgetID)
	if err != nil {
		return nil, err
	}
	if !grant.AllowsCounterparty(destination) {
		return nil, fmt.Errorf("%w: withdrawal destination is outside the mandate allowlist", ErrInvalid)
	}
	if policy.Currency != strings.ToUpper(in.Valuation.Currency) {
		return nil, ErrInvalid
	}
	w, err := s.Wallets.Get(ctx, b.walletID, b.network, operator)
	if err != nil {
		return nil, err
	}
	q, err := s.Native.Preview(ctx, w, allowlisted, in.AtomicAmount, in.Valuation.AmountMinor, in.Valuation.Currency, in.MaxFeeWei)
	if err != nil {
		return nil, err
	}
	q.SigningPurpose = "operator-withdrawal"
	now := s.now()
	if !q.ExpiresAt.After(now) {
		return nil, ErrInvalid
	}
	pub := &fundingpb.WithdrawalQuote{BudgetId: b.budgetID, Destination: allowlisted, AtomicAmount: in.AtomicAmount, Network: baseusdc.Network, MaxFeeWei: q.MaxFee, ValueMinor: in.Valuation.AmountMinor, FeeValuationMinor: in.FeeValuationMinor, Currency: strings.ToUpper(in.Valuation.Currency), EstimatedSeconds: 120, ExpiresAt: timestamppb.New(q.ExpiresAt), ExecutionMode: s.Native.Mode(), MandateId: grant.ID}
	encoded, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}
	id, err := newID("withdrawal-quote")
	if err != nil {
		return nil, err
	}
	pub.Id = id
	raw, err := proto.MarshalOptions{Deterministic: true}.Marshal(pub)
	if err != nil {
		return nil, err
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO withdrawal_quotes(id,budget_id,operator_identity,public_proto,execution_quote,expires_at,created_at) VALUES(?,?,?,?,?,?,?)`, id, b.budgetID, operator, raw, string(encoded), q.ExpiresAt.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return pub, nil
}

// ExecuteWithdrawal requires both an authenticated operator request and the
// exact run-bound agent identity that owns the mandate. Authorization and
// settlement remain in their canonical services; the native rail stays local
// fork only.
func (s *Service) ExecuteWithdrawal(ctx context.Context, operator, identityToken string, in *fundingpb.ExecuteWithdrawalRequest) (*fundingpb.Withdrawal, error) {
	if s == nil || in == nil || operator == "" || strings.TrimSpace(identityToken) == "" || in.QuoteId == "" || in.IdempotencyKey == "" || s.Native == nil || s.Native.Mode() != "local-fork" || s.Authorizations == nil || s.Instruments == nil || s.Credentials == nil || s.Settlements == nil || s.Mandates == nil {
		return nil, fmt.Errorf("%w: local-fork withdrawal requires a mandate identity, credential authority, instrument and settlement owners", ErrInvalid)
	}
	if existing, getErr := s.withdrawalByIdempotencyKey(ctx, operator, in.IdempotencyKey); getErr == nil {
		if existing.QuoteId != in.QuoteId {
			return nil, ErrConflict
		}
		switch existing.State {
		case "settled", "failed", "refused", "unknown", "in_progress":
			return existing, nil
		}
	} else if !errors.Is(getErr, ErrNotFound) {
		return nil, getErr
	}
	var quote fundingpb.WithdrawalQuote
	var quoteProto []byte
	var executionQuote string
	err := s.DB.QueryRowContext(ctx, `SELECT q.public_proto,q.execution_quote FROM withdrawal_quotes q JOIN budget_wallets b ON b.budget_id=q.budget_id WHERE q.id=? AND q.operator_identity=? AND b.operator_identity=?`, in.QuoteId, operator, operator).Scan(&quoteProto, &executionQuote)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if err = proto.Unmarshal(quoteProto, &quote); err != nil || quote.GetId() != in.QuoteId || quote.GetExecutionMode() != "local-fork" || quote.GetMandateId() == "" || quote.GetAtomicAmount() == "" || quote.GetValueMinor() <= 0 || quote.GetFeeValuationMinor() < 0 || quote.GetCurrency() == "" || !common.IsHexAddress(quote.GetDestination()) {
		return nil, fmt.Errorf("%w: persisted withdrawal quote is incomplete or outside local-fork mode", ErrInvalid)
	}
	if quote.ExpiresAt == nil || !s.now().Before(quote.ExpiresAt.AsTime()) {
		return nil, fmt.Errorf("%w: withdrawal quote expired", ErrInvalid)
	}
	b, err := s.load(ctx, quote.GetBudgetId())
	if err != nil {
		return nil, err
	}
	if b.operator != operator || b.agent == "" || b.run == "" {
		return nil, ErrNotFound
	}
	grant, err := s.Mandates.RequireLive(ctx, quote.GetMandateId())
	if err != nil {
		return nil, err
	}
	if grant.ID != quote.GetMandateId() || grant.BudgetID != b.budgetID || grant.AgentSubject != b.agent || grant.AgentRunID != b.run || !grant.AllowsCounterparty(quote.GetDestination()) {
		return nil, fmt.Errorf("%w: mandate principal, run, budget or destination changed after quote", ErrInvalid)
	}
	var allowed string
	if err = s.DB.QueryRowContext(ctx, `SELECT address FROM withdrawal_destinations WHERE budget_id=? AND operator_identity=? AND address=?`, b.budgetID, operator, quote.GetDestination()).Scan(&allowed); err != nil {
		return nil, fmt.Errorf("%w: withdrawal destination is no longer allowlisted", ErrInvalid)
	}
	var nativeQuote nativeusdc.Quote
	if err = json.Unmarshal([]byte(executionQuote), &nativeQuote); err != nil || nativeQuote.WalletID != b.walletID || nativeQuote.Operator != operator || nativeQuote.Network != b.network || nativeQuote.SigningPurpose != "operator-withdrawal" || !strings.EqualFold(nativeQuote.Recipient, quote.GetDestination()) || nativeQuote.AtomicAmount != quote.GetAtomicAmount() || nativeQuote.ValueMinor != quote.GetValueMinor() || nativeQuote.Currency != quote.GetCurrency() || !s.now().Before(nativeQuote.ExpiresAt) {
		return nil, fmt.Errorf("%w: local rail quote no longer matches the persisted withdrawal scope", ErrInvalid)
	}
	if quote.GetValueMinor() > int64(^uint64(0)>>1)-quote.GetFeeValuationMinor() {
		return nil, fmt.Errorf("%w: withdrawal value and fee exceed the authorization amount range", ErrInvalid)
	}
	amountMinor := quote.GetValueMinor() + quote.GetFeeValuationMinor()
	authorizationID := stableID("withdrawal-authorization", operator, in.IdempotencyKey)
	authorizationKey := stableID("withdrawal-authorization-key", operator, in.IdempotencyKey)
	settlementID := stableID("withdrawal-settlement", operator, in.IdempotencyKey)
	settlementKey := stableID("withdrawal-settlement-key", operator, in.IdempotencyKey)
	withdrawalID := stableID("withdrawal", operator, in.IdempotencyKey)
	authRecord, err := s.Authorizations.Propose(ctx, authorization.ProposeInput{
		ID: authorizationID, IdempotencyKey: authorizationKey, MandateID: grant.ID,
		IdentityToken: identityToken, AmountMinor: amountMinor, Currency: quote.GetCurrency(), Counterparty: quote.GetDestination(),
		SettlementID: settlementID, InstrumentID: b.walletID, SettlementIdempotencyKey: settlementKey,
	})
	if err != nil {
		return nil, fmt.Errorf("propose withdrawal through Treasury authorization: %w", err)
	}
	state := "authorizing"
	lastError := ""
	switch authRecord.Verdict {
	case authorization.VerdictApproved:
		state = "authorized"
	case authorization.VerdictSettled:
		// A retry after successful settlement is forwarded to the same
		// idempotent settlement record below.
		state = "authorized"
	case authorization.VerdictPending:
		state, lastError = "pending_approval", authRecord.Remediation
	case authorization.VerdictRefused:
		state, lastError = "refused", authRecord.ViolatedConstraint+": "+authRecord.Remediation
	default:
		return nil, fmt.Errorf("%w: authorization owner returned unsupported withdrawal verdict %q", ErrInvalid, authRecord.Verdict)
	}
	_, err = s.DB.ExecContext(ctx, `INSERT INTO withdrawals(id,quote_id,operator_identity,idempotency_key,authorization_id,settlement_id,state,last_error,created_at) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(idempotency_key) DO NOTHING`, withdrawalID, quote.GetId(), operator, in.IdempotencyKey, authRecord.ID, settlementID, state, lastError, s.now().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	if authRecord.Verdict == authorization.VerdictApproved || authRecord.Verdict == authorization.VerdictSettled {
		if _, err = s.DB.ExecContext(ctx, `UPDATE withdrawals SET state='authorized',last_error='' WHERE id=? AND operator_identity=? AND authorization_id=? AND state IN ('pending_approval','authorizing','authorized')`, withdrawalID, operator, authRecord.ID); err != nil {
			return nil, fmt.Errorf("persist operator-authorized withdrawal state: %w", err)
		}
	}
	value, err := s.GetWithdrawal(ctx, operator, withdrawalID)
	if err != nil {
		return nil, err
	}
	if value.QuoteId != quote.GetId() || value.AuthorizationId != authRecord.ID || value.SettlementId != settlementID {
		return nil, ErrConflict
	}
	if authRecord.Verdict != authorization.VerdictApproved && authRecord.Verdict != authorization.VerdictSettled {
		_, err = s.DB.ExecContext(ctx, `UPDATE withdrawals SET state=?,last_error=? WHERE id=? AND operator_identity=?`, state, lastError, withdrawalID, operator)
		return value, err
	}
	if s.Credentials == nil {
		return nil, fmt.Errorf("%w: Credential Authority writer is required for native-USDC execution", ErrInvalid)
	}
	// The mandate hold covers both the outgoing value and the operator's
	// conservative fee valuation; settlement evidence records that full charge.
	nativeQuote.AuthorizationValueMinor = amountMinor
	instrumentID := stableID("withdrawal-instrument", operator, in.IdempotencyKey)
	credentialReference := "vrooli/treasury/withdrawals/" + strings.TrimPrefix(instrumentID, "withdrawal-instrument-")
	quoteBytes, err := json.Marshal(nativeQuote)
	if err != nil {
		return nil, err
	}
	if err = s.Credentials.Store(ctx, credentialReference, "value", string(quoteBytes)); err != nil {
		return nil, fmt.Errorf("store one-use native quote in Credential Authority: %w", err)
	}
	scopedInstrument, err := s.Instruments.ResolveForUse(ctx, instrumentID)
	if errors.Is(err, instrument.ErrNotFound) {
		registered, registerErr := s.Instruments.Register(ctx, instrument.RegisterInput{ID: instrumentID, MandateID: grant.ID, Rail: "base-usdc", CredentialReference: credentialReference, Counterparty: quote.GetDestination()})
		if registerErr != nil {
			return nil, fmt.Errorf("register mandate-scoped withdrawal instrument: %w", registerErr)
		}
		scopedInstrument.Instrument = registered
	} else if err != nil {
		return nil, fmt.Errorf("resolve mandate-scoped withdrawal instrument: %w", err)
	} else if scopedInstrument.Instrument.MandateID != grant.ID || scopedInstrument.Instrument.Rail != "base-usdc" || scopedInstrument.Instrument.Counterparty != quote.GetDestination() || scopedInstrument.Instrument.CredentialReference != credentialReference {
		return nil, fmt.Errorf("%w: retained withdrawal instrument no longer matches quote scope", ErrConflict)
	}
	registered := scopedInstrument.Instrument
	settled, settleErr := s.Settlements.Settle(ctx, settlement.SettleInput{ID: settlementID, AuthorizationID: authRecord.ID, InstrumentID: registered.ID, IdempotencyKey: settlementKey, IdentityToken: identityToken})
	if settleErr != nil && settled.ID == "" {
		_, _ = s.DB.ExecContext(ctx, `UPDATE withdrawals SET state='in_progress',last_error=? WHERE id=? AND operator_identity=?`, settleErr.Error(), withdrawalID, operator)
		return nil, settleErr
	}
	state = string(settled.Outcome)
	_, err = s.DB.ExecContext(ctx, `UPDATE withdrawals SET state=?,transaction_hash=?,last_error=? WHERE id=? AND operator_identity=?`, state, settled.ExternalID, settled.Detail, withdrawalID, operator)
	if err != nil {
		return nil, err
	}
	if settled.Outcome == settlement.OutcomeSettled || settled.Outcome == settlement.OutcomeFailed {
		if deleter, ok := s.Credentials.(interface {
			Delete(context.Context, string, string) error
		}); ok {
			if deleteErr := deleter.Delete(ctx, credentialReference, "value"); deleteErr != nil {
				return nil, fmt.Errorf("withdrawal outcome recorded but quote credential cleanup failed: %w", deleteErr)
			}
		}
	}
	value, err = s.GetWithdrawal(ctx, operator, withdrawalID)
	if err != nil {
		return nil, err
	}
	if settleErr != nil {
		return value, settleErr
	}
	return value, nil
}

func (s *Service) GetWithdrawal(ctx context.Context, operator, id string) (*fundingpb.Withdrawal, error) {
	if operator == "" || id == "" {
		return nil, ErrInvalid
	}
	var v fundingpb.Withdrawal
	err := s.DB.QueryRowContext(ctx, `SELECT id,quote_id,authorization_id,settlement_id,state,transaction_hash,last_error FROM withdrawals WHERE id=? AND operator_identity=?`, id, operator).Scan(&v.Id, &v.QuoteId, &v.AuthorizationId, &v.SettlementId, &v.State, &v.TransactionHash, &v.LastError)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *Service) withdrawalByIdempotencyKey(ctx context.Context, operator, key string) (*fundingpb.Withdrawal, error) {
	if operator == "" || key == "" {
		return nil, ErrInvalid
	}
	var v fundingpb.Withdrawal
	err := s.DB.QueryRowContext(ctx, `SELECT id,quote_id,authorization_id,settlement_id,state,transaction_hash,last_error FROM withdrawals WHERE idempotency_key=? AND operator_identity=?`, key, operator).Scan(&v.Id, &v.QuoteId, &v.AuthorizationId, &v.SettlementId, &v.State, &v.TransactionHash, &v.LastError)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}
