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
	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/vrooli/api-core/schedule"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
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
	"treasury/internal/ledger"
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
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}
type Notification interface {
	Received(context.Context, string, string, string, string, string) error
}
type Config struct {
	DB                DB
	Wallets           *wallet.Service
	Books             book.Service
	Budgets           budget.Service
	Receipts          *baseusdc.Service
	Funding           *receiptfunding.Service
	Native            *nativeusdc.Adapter
	Mandates          mandate.Service
	Authorizations    *authorization.Service
	Approvals         *approval.Service
	Instruments       *instrument.Service
	Credentials       instrument.CredentialWriter
	Settlements       *settlement.Service
	LedgerDestination ledger.DestinationSource
	Notifications     Notification
	// FundingMode controls inbound funding intents only. `base-mainnet` uses
	// the read-only receipt verifier; it never enables native signing.
	FundingMode string
	Clock       schedule.Clock
}
type Service struct{ Config }

func New(c Config) *Service {
	if c.Clock == nil {
		c.Clock = schedule.System()
	}
	return &Service{Config: c}
}

func (s *Service) inboundFundingMode() string {
	if s == nil {
		return "disabled"
	}
	if strings.TrimSpace(s.FundingMode) != "" {
		return strings.TrimSpace(s.FundingMode)
	}
	// Preserve the existing explicitly constructed local-fork test/rehearsal
	// path for callers that predate the opt-in mode field.
	if s.Native != nil && s.Native.Mode() == "local-fork" {
		return "local-fork"
	}
	return "disabled"
}

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
	if s == nil || s.Wallets == nil || in == nil || in.BudgetId == "" || in.Network != baseusdc.Network || (in.AgentSubject == "") != (in.AgentRunId == "") {
		return nil, ErrInvalid
	}
	rotationKey := in.GetRotationIdempotencyKey()
	if in.GetRotateSession() {
		if in.GetMandateId() == "" || rotationKey == "" || strings.TrimSpace(rotationKey) != rotationKey || len(rotationKey) > 200 {
			return nil, ErrInvalid
		}
	} else if rotationKey != "" {
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
			session, err = s.RotateMandateSession(ctx, operator, in.GetMandateId(), rotationKey)
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

type fundingReceiptBinding struct {
	IntentID, OperatorIdentity, WalletID                    string
	Network, AssetID, TransactionHash                       string
	LogIndex                                                uint64
	AtomicAmount, Recipient, BlockHash, CheckID, DeliveryID string
}

func (s *Service) loadFundingReceiptBinding(ctx context.Context, intentID string) (fundingReceiptBinding, error) {
	var value fundingReceiptBinding
	var logIndex int64
	err := s.DB.QueryRowContext(ctx, `SELECT intent_id,operator_identity,wallet_id,network,asset_id,transaction_hash,log_index,atomic_amount,recipient,block_hash,check_id,delivery_id FROM funding_intent_receipt_bindings WHERE intent_id=?`, intentID).Scan(
		&value.IntentID, &value.OperatorIdentity, &value.WalletID, &value.Network, &value.AssetID, &value.TransactionHash, &logIndex, &value.AtomicAmount, &value.Recipient, &value.BlockHash, &value.CheckID, &value.DeliveryID,
	)
	if err != nil {
		return fundingReceiptBinding{}, err
	}
	if logIndex < 0 {
		return fundingReceiptBinding{}, ErrInvalid
	}
	value.LogIndex = uint64(logIndex)
	return value, nil
}

func (s *Service) bindFundingReceipt(ctx context.Context, intent fundingIntent, w wallet.Wallet, transactionHash string, check baseusdc.Check, observation baseusdc.Observation) (fundingReceiptBinding, error) {
	start, ok := new(big.Int).SetString(intent.startBlock, 10)
	if !ok || !start.IsUint64() || start.Sign() < 0 || observation.BlockNumber < start.Uint64() || observation.Finality != "finalized" ||
		check.Outcome != "verified" || check.OperatorIdentity != intent.operator || check.WalletID != w.ID || check.Network != baseusdc.Network ||
		!strings.EqualFold(check.TransactionHash, transactionHash) || observation.WalletID != w.ID || observation.Network != baseusdc.Network ||
		observation.AssetID != baseusdc.TransferAsset || observation.Decimals != baseusdc.USDCDecimals || observation.AtomicAmount != intent.atoms ||
		!strings.EqualFold(observation.To, w.Address) || !strings.EqualFold(observation.TransactionHash, transactionHash) ||
		observation.LogIndex > uint64(^uint32(0)) || observation.BlockHash == "" || check.ID == "" {
		return fundingReceiptBinding{}, fmt.Errorf("%w: receipt evidence does not match the exact finalized Base USDC intent", ErrInvalid)
	}
	deliveryID := fmt.Sprintf("%s/log/%d", check.ID, observation.LogIndex)
	createdAt := s.now().Format(time.RFC3339Nano)
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fundingReceiptBinding{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var state, storedHash, storedDeliveryID string
	if err = tx.QueryRowContext(ctx, `SELECT state,transaction_hash,receipt_delivery_id FROM funding_intents WHERE id=? AND operator_identity=?`, intent.id, intent.operator).Scan(&state, &storedHash, &storedDeliveryID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fundingReceiptBinding{}, ErrNotFound
		}
		return fundingReceiptBinding{}, err
	}
	if state == "credited" || (storedHash != "" && !strings.EqualFold(storedHash, transactionHash)) {
		return fundingReceiptBinding{}, ErrConflict
	}
	canonicalHash := strings.ToLower(transactionHash)
	_, err = tx.ExecContext(ctx, `INSERT INTO funding_intent_receipt_log_claims(network,asset_id,transaction_hash,log_index,intent_id,created_at) VALUES(?,?,?,?,?,?) ON CONFLICT(network,asset_id,transaction_hash,log_index) DO NOTHING`, baseusdc.Network, baseusdc.TransferAsset, canonicalHash, observation.LogIndex, intent.id, createdAt)
	if err != nil {
		return fundingReceiptBinding{}, fmt.Errorf("claim finalized Base USDC transfer log: %w", err)
	}
	var claimedIntentID string
	if err = tx.QueryRowContext(ctx, `SELECT intent_id FROM funding_intent_receipt_log_claims WHERE network=? AND asset_id=? AND transaction_hash=? AND log_index=?`, baseusdc.Network, baseusdc.TransferAsset, canonicalHash, observation.LogIndex).Scan(&claimedIntentID); err != nil {
		return fundingReceiptBinding{}, err
	}
	if claimedIntentID != intent.id {
		return fundingReceiptBinding{}, ErrConflict
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO funding_intent_receipt_bindings(intent_id,operator_identity,wallet_id,network,asset_id,transaction_hash,log_index,atomic_amount,recipient,block_hash,check_id,delivery_id,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(intent_id) DO NOTHING`, intent.id, intent.operator, w.ID, baseusdc.Network, baseusdc.TransferAsset, transactionHash, observation.LogIndex, intent.atoms, strings.ToLower(w.Address), strings.ToLower(observation.BlockHash), check.ID, deliveryID, createdAt)
	if err != nil {
		return fundingReceiptBinding{}, fmt.Errorf("%w: receipt log is already bound to another intent: %v", ErrConflict, err)
	}
	var binding fundingReceiptBinding
	var index int64
	if err = tx.QueryRowContext(ctx, `SELECT intent_id,operator_identity,wallet_id,network,asset_id,transaction_hash,log_index,atomic_amount,recipient,block_hash,check_id,delivery_id FROM funding_intent_receipt_bindings WHERE intent_id=?`, intent.id).Scan(
		&binding.IntentID, &binding.OperatorIdentity, &binding.WalletID, &binding.Network, &binding.AssetID, &binding.TransactionHash, &index, &binding.AtomicAmount, &binding.Recipient, &binding.BlockHash, &binding.CheckID, &binding.DeliveryID,
	); err != nil {
		return fundingReceiptBinding{}, err
	}
	if index < 0 {
		return fundingReceiptBinding{}, ErrInvalid
	}
	binding.LogIndex = uint64(index)
	if binding.IntentID != intent.id || binding.OperatorIdentity != intent.operator || binding.WalletID != w.ID || binding.Network != baseusdc.Network || binding.AssetID != baseusdc.TransferAsset ||
		!strings.EqualFold(binding.TransactionHash, transactionHash) || binding.AtomicAmount != intent.atoms || !strings.EqualFold(binding.Recipient, w.Address) {
		return fundingReceiptBinding{}, ErrConflict
	}
	result, err := tx.ExecContext(ctx, `UPDATE funding_intents SET state=CASE WHEN state='waiting' THEN 'observed' ELSE state END,transaction_hash=?,receipt_delivery_id=?,last_error='',updated_at=? WHERE id=? AND operator_identity=? AND state<>'credited' AND (transaction_hash='' OR LOWER(transaction_hash)=LOWER(?)) AND (receipt_delivery_id='' OR receipt_delivery_id=?)`, binding.TransactionHash, binding.DeliveryID, createdAt, intent.id, intent.operator, binding.TransactionHash, binding.DeliveryID)
	if err != nil {
		return fundingReceiptBinding{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fundingReceiptBinding{}, err
	}
	if changed != 1 && storedDeliveryID != binding.DeliveryID {
		return fundingReceiptBinding{}, ErrConflict
	}
	if err = tx.Commit(); err != nil {
		return fundingReceiptBinding{}, err
	}
	return binding, nil
}

func (s *Service) now() time.Time {
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
	if s == nil || s.DB == nil || s.Wallets == nil || s.Receipts == nil || s.Funding == nil || operator == "" || in == nil {
		return nil, ErrInvalid
	}
	if s.LedgerDestination != nil {
		destination, err := s.LedgerDestination.Destination(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve Treasury Money Ledger destination: %w", err)
		}
		in = proto.Clone(in).(*fundingpb.CreateFundingIntentRequest)
		in.LedgerBookId = destination.BookID
		in.LedgerAccountId = destination.AccountID
	}
	if in.BudgetId == "" || !positiveAtoms.MatchString(in.AtomicAmount) || in.Valuation == nil || in.Valuation.Currency == "" || in.LedgerBookId == "" || in.LedgerAccountId == "" {
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
	var start *big.Int
	switch s.inboundFundingMode() {
	case "local-fork":
		if s.Native == nil || s.Native.Mode() != "local-fork" {
			return nil, fmt.Errorf("%w: local-fork funding requires the explicitly configured Base fork", ErrInvalid)
		}
		var head string
		if err = s.Native.Call(ctx, "eth_blockNumber", []any{}, &head); err != nil {
			return nil, err
		}
		start, err = rpcUint(head)
		if err != nil {
			return nil, err
		}
	case "base-mainnet":
		// This mode only creates receive intents. Native must stay absent or
		// disabled so enabling inbound confirmation cannot expose a signing rail.
		if s.Native != nil && s.Native.Mode() != "disabled" {
			return nil, fmt.Errorf("%w: mainnet funding refuses an enabled native signer", ErrInvalid)
		}
		start, err = s.Receipts.LatestBlockNumber(ctx)
		if err != nil {
			return nil, fmt.Errorf("read Base mainnet funding-intent start block: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w: inbound Base funding is disabled", ErrInvalid)
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
	// Funding-intent values come from the authenticated operator's declaration.
	// Persist the basis explicitly and timestamp the declaration at the Treasury
	// boundary so downstream receipt acceptance has complete valuation evidence.
	now := s.now()
	copy.Valuation.Basis = sharedpb.Basis_BASIS_OPERATOR_ASSERTED
	copy.Valuation.ValuedAt = timestamppb.New(now)
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(copy)
	if err != nil {
		return nil, err
	}
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
	if _, err = s.DB.ExecContext(ctx, `INSERT INTO funding_intent_watch_cursors(intent_id,next_block,updated_at) VALUES(?,?,?) ON CONFLICT(intent_id) DO NOTHING`, id, start.String(), now.Format(time.RFC3339Nano)); err != nil {
		return nil, err
	}
	return s.publicIntent(ctx, v)
}

const (
	fundingWatchChunkBlocks = 2048
	fundingWatchMaxChunks   = 4
)

type fundingTransferLog struct {
	Address         string   `json:"address"`
	Topics          []string `json:"topics"`
	Data            string   `json:"data"`
	TransactionHash string   `json:"transactionHash"`
}

// WatchFundingIntents searches finalized local-fork Base blocks, then routes a
// matching transfer through the existing receipt verification and credit path.
func (s *Service) WatchFundingIntents(ctx context.Context) error {
	if s == nil || s.DB == nil || s.inboundFundingMode() != "local-fork" || s.Native == nil || s.Native.Mode() != "local-fork" {
		return nil
	}
	var finalized struct {
		Number string `json:"number"`
	}
	if err := s.Native.Call(ctx, "eth_getBlockByNumber", []any{"finalized", false}, &finalized); err != nil {
		return fmt.Errorf("read finalized Base head for funding watcher: %w", err)
	}
	head, err := rpcUint(finalized.Number)
	if err != nil {
		return fmt.Errorf("decode finalized Base head for funding watcher: %w", err)
	}
	// New intents do not have a cursor until the first watcher pass. Keep them
	// eligible here so the watcher can bootstrap that durable cursor from the
	// intent's creation block instead of requiring a separate enrollment step.
	rows, err := s.DB.QueryContext(ctx, `SELECT i.id,i.operator_identity,i.transaction_hash FROM funding_intents i LEFT JOIN funding_intent_watch_cursors c ON c.intent_id=i.id WHERE i.state IN ('waiting','observed','delivering') ORDER BY COALESCE(c.updated_at,i.created_at),i.created_at LIMIT 25`)
	if err != nil {
		return err
	}
	type pendingIntent struct{ id, operator, txHash string }
	var intents []pendingIntent
	for rows.Next() {
		var pending pendingIntent
		if err = rows.Scan(&pending.id, &pending.operator, &pending.txHash); err != nil {
			rows.Close()
			return err
		}
		intents = append(intents, pending)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err = rows.Close(); err != nil {
		return err
	}
	var failures []error
	for _, pending := range intents {
		if pending.txHash != "" {
			_, err = s.ConfirmFundingTransfer(ctx, pending.operator, &fundingpb.ConfirmFundingTransferRequest{Id: pending.id, TransactionHash: pending.txHash})
			if err != nil {
				_, _ = s.DB.ExecContext(context.WithoutCancel(ctx), `UPDATE funding_intent_watch_cursors SET updated_at=? WHERE intent_id=?`, s.now().Format(time.RFC3339Nano), pending.id)
			}
		} else {
			err = s.watchFundingIntent(ctx, pending.id, pending.operator, head)
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("watch funding intent %s: %w", pending.id, err))
		}
	}
	return errors.Join(failures...)
}

func (s *Service) watchFundingIntent(ctx context.Context, id, operator string, finalizedHead *big.Int) error {
	intent, err := s.loadIntent(ctx, operator, id)
	if err != nil {
		return err
	}
	if intent.state != "waiting" || intent.txHash != "" {
		return nil
	}
	b, err := s.load(ctx, intent.budgetID)
	if err != nil {
		return err
	}
	w, err := s.Wallets.Get(ctx, b.walletID, b.network, operator)
	if err != nil {
		return err
	}
	var cursorText string
	if err = s.DB.QueryRowContext(ctx, `SELECT next_block FROM funding_intent_watch_cursors WHERE intent_id=?`, id).Scan(&cursorText); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if _, err = s.DB.ExecContext(ctx, `INSERT INTO funding_intent_watch_cursors(intent_id,next_block,updated_at) VALUES(?,?,?) ON CONFLICT(intent_id) DO NOTHING`, id, intent.startBlock, s.now().Format(time.RFC3339Nano)); err != nil {
			return err
		}
		if err = s.DB.QueryRowContext(ctx, `SELECT next_block FROM funding_intent_watch_cursors WHERE intent_id=?`, id).Scan(&cursorText); err != nil {
			return err
		}
	}
	next, ok := new(big.Int).SetString(cursorText, 10)
	if !ok || next.Sign() < 0 {
		return ErrInvalid
	}
	if next.Cmp(finalizedHead) > 0 {
		_, err = s.DB.ExecContext(ctx, `UPDATE funding_intent_watch_cursors SET updated_at=? WHERE intent_id=?`, s.now().Format(time.RFC3339Nano), id)
		return err
	}
	transferTopic := gethcrypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	toTopic := common.BytesToHash(common.HexToAddress(w.Address).Bytes())
	for chunk := 0; chunk < fundingWatchMaxChunks && next.Cmp(finalizedHead) <= 0; chunk++ {
		end := new(big.Int).Add(next, big.NewInt(fundingWatchChunkBlocks-1))
		if end.Cmp(finalizedHead) > 0 {
			end.Set(finalizedHead)
		}
		filter := map[string]any{
			"address": baseusdc.USDCAddress, "fromBlock": fmt.Sprintf("0x%x", next), "toBlock": fmt.Sprintf("0x%x", end),
			"topics": []any{transferTopic.Hex(), nil, toTopic.Hex()},
		}
		var logs []fundingTransferLog
		if err = s.Native.Call(ctx, "eth_getLogs", []any{filter}, &logs); err != nil {
			return fmt.Errorf("read finalized USDC transfers: %w", err)
		}
		var matches []fundingTransferLog
		for _, item := range logs {
			if !strings.EqualFold(item.Address, baseusdc.USDCAddress) || len(item.Topics) != 3 || !strings.EqualFold(item.Topics[0], transferTopic.Hex()) || !strings.EqualFold(item.Topics[2], toTopic.Hex()) {
				continue
			}
			amount, parsed := new(big.Int).SetString(strings.TrimPrefix(strings.TrimPrefix(item.Data, "0x"), "0X"), 16)
			if parsed && amount.String() == intent.atoms {
				matches = append(matches, item)
			}
		}
		if len(matches) > 1 {
			return fmt.Errorf("%w: multiple exact-amount Base transfers match one funding intent", ErrInvalid)
		}
		if len(matches) == 1 {
			if _, err = s.ConfirmFundingTransfer(ctx, operator, &fundingpb.ConfirmFundingTransferRequest{Id: id, TransactionHash: matches[0].TransactionHash}); err != nil {
				return err
			}
		}
		advanced := new(big.Int).Add(end, big.NewInt(1))
		result, updateErr := s.DB.ExecContext(ctx, `UPDATE funding_intent_watch_cursors SET next_block=?,updated_at=? WHERE intent_id=? AND next_block=?`, advanced.String(), s.now().Format(time.RFC3339Nano), id, next.String())
		if updateErr != nil {
			return updateErr
		}
		changed, updateErr := result.RowsAffected()
		if updateErr != nil {
			return updateErr
		}
		if changed == 0 {
			return nil
		}
		next = advanced
	}
	return nil
}

func (s *Service) ConfirmFundingTransfer(ctx context.Context, operator string, in *fundingpb.ConfirmFundingTransferRequest) (*fundingpb.FundingIntent, error) {
	if in == nil || operator == "" || in.Id == "" || strings.TrimSpace(in.TransactionHash) == "" || s.Receipts == nil || s.Funding == nil {
		return nil, ErrInvalid
	}
	transactionHash := strings.ToLower(strings.TrimSpace(in.TransactionHash))
	v, err := s.loadIntent(ctx, operator, in.Id)
	if err != nil {
		return nil, err
	}
	if v.state == "credited" {
		if !strings.EqualFold(v.txHash, transactionHash) {
			return nil, ErrConflict
		}
		_ = s.retryNotification(ctx, v)
		v, err = s.loadIntent(ctx, operator, in.Id)
		if err != nil {
			return nil, err
		}
		return s.publicIntent(ctx, v)
	}
	if v.txHash != "" && !strings.EqualFold(v.txHash, transactionHash) {
		return nil, ErrConflict
	}
	b, err := s.load(ctx, v.budgetID)
	if err != nil {
		return nil, err
	}
	w, err := s.Wallets.Get(ctx, b.walletID, b.network, operator)
	if err != nil {
		return nil, err
	}
	binding, err := s.loadFundingReceiptBinding(ctx, v.id)
	if err == nil {
		if !fundingBindingMatches(v, w, transactionHash, binding) {
			return nil, ErrConflict
		}
		return s.reconcileBoundFunding(ctx, v, w, binding)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if v.deliveryID != "" {
		// A pre-binding intent may represent a legacy partial attempt. Its
		// delivery is not enough to prove which check/log was accepted, so leave
		// it fail-closed for an explicit reconciliation instead of guessing.
		return nil, fmt.Errorf("%w: funding intent has a delivery reference without its exact receipt binding", ErrConflict)
	}
	check, err := s.Receipts.Verify(ctx, operator, w, transactionHash)
	if err != nil {
		return nil, err
	}
	matched, err := exactFundingObservation(v, w, transactionHash, check)
	if err != nil {
		return nil, err
	}
	binding, err = s.bindFundingReceipt(ctx, v, w, transactionHash, check, *matched)
	if err != nil {
		return nil, err
	}
	v, err = s.loadIntent(ctx, operator, in.Id)
	if err != nil {
		return nil, err
	}
	return s.reconcileBoundFunding(ctx, v, w, binding)
}

func fundingBindingMatches(intent fundingIntent, w wallet.Wallet, transactionHash string, binding fundingReceiptBinding) bool {
	return binding.IntentID == intent.id && binding.OperatorIdentity == intent.operator && binding.WalletID == w.ID &&
		binding.Network == baseusdc.Network && binding.AssetID == baseusdc.TransferAsset && strings.EqualFold(binding.TransactionHash, transactionHash) &&
		binding.AtomicAmount == intent.atoms && strings.EqualFold(binding.Recipient, w.Address) && binding.CheckID != "" && binding.DeliveryID != ""
}

func exactFundingObservation(intent fundingIntent, w wallet.Wallet, transactionHash string, check baseusdc.Check) (*baseusdc.Observation, error) {
	if check.Outcome != "verified" || check.OperatorIdentity != intent.operator || check.WalletID != w.ID || check.Network != baseusdc.Network || !strings.EqualFold(check.TransactionHash, transactionHash) {
		return nil, fmt.Errorf("%w: retained receipt check is outside the requested Base wallet and transaction scope", ErrInvalid)
	}
	start, ok := new(big.Int).SetString(intent.startBlock, 10)
	if !ok || !start.IsUint64() || start.Sign() < 0 {
		return nil, ErrInvalid
	}
	var matched *baseusdc.Observation
	for i := range check.Transfers {
		obs := &check.Transfers[i]
		if !strings.EqualFold(obs.TransactionHash, transactionHash) || obs.Network != baseusdc.Network || obs.AssetID != baseusdc.TransferAsset ||
			obs.Decimals != baseusdc.USDCDecimals || !strings.EqualFold(obs.To, w.Address) {
			return nil, fmt.Errorf("%w: verifier returned a transfer outside the requested Base USDC recipient scope", ErrInvalid)
		}
		if obs.BlockNumber < start.Uint64() || obs.AtomicAmount != intent.atoms {
			continue
		}
		if obs.Finality != "finalized" {
			return nil, fmt.Errorf("%w: matching Base USDC transfer is not finalized", ErrInvalid)
		}
		if matched != nil {
			return nil, fmt.Errorf("%w: multiple exact Base USDC logs in one transaction match this intent", ErrInvalid)
		}
		matched = obs
	}
	if matched == nil {
		return nil, fmt.Errorf("%w: no exact finalized native-USDC transfer at or after intent creation", ErrInvalid)
	}
	if intent.request.GetAttribution().GetPayerAddress() != "" && !strings.EqualFold(intent.request.GetAttribution().GetPayerAddress(), matched.From) {
		return nil, fmt.Errorf("%w: declared payer address does not match the verified Transfer sender", ErrInvalid)
	}
	return matched, nil
}

func (s *Service) reconcileBoundFunding(ctx context.Context, intent fundingIntent, w wallet.Wallet, binding fundingReceiptBinding) (*fundingpb.FundingIntent, error) {
	if !fundingBindingMatches(intent, w, intent.txHash, binding) || intent.deliveryID != binding.DeliveryID {
		return nil, ErrConflict
	}
	check, err := s.Receipts.GetCheck(ctx, intent.operator, w.ID, binding.TransactionHash, binding.CheckID)
	if err != nil {
		return nil, err
	}
	matched, err := exactFundingObservation(intent, w, binding.TransactionHash, check)
	if err != nil || matched.LogIndex != binding.LogIndex || matched.AtomicAmount != binding.AtomicAmount || !strings.EqualFold(matched.BlockHash, binding.BlockHash) {
		if err != nil {
			return nil, err
		}
		return nil, ErrConflict
	}
	delivery, err := s.Funding.Get(ctx, binding.DeliveryID, intent.operator)
	if errors.Is(err, receiptfunding.ErrNotFound) {
		attribution := intent.request.Attribution
		if attribution != nil {
			attribution = proto.Clone(attribution).(*ingestpb.CryptoPayerAttribution)
		}
		if attribution == nil {
			attribution = &ingestpb.CryptoPayerAttribution{}
		}
		attribution.PayerAddress = strings.ToLower(matched.From)
		delivery, err = s.Funding.Submit(ctx, receiptfunding.SubmitInput{
			OperatorIdentity: intent.operator, WalletID: w.ID, TransactionHash: binding.TransactionHash, CheckID: binding.CheckID,
			BudgetID: intent.budgetID, LedgerBookID: intent.request.LedgerBookId, LedgerAccountID: intent.request.LedgerAccountId,
			LogIndex: binding.LogIndex, Action: ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL,
			Valuation: intent.request.Valuation, Attribution: attribution,
		})
		if err != nil {
			_ = s.updateIntent(ctx, intent.id, "observed", binding.TransactionHash, binding.DeliveryID, intent.notificationState, err)
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if err := validateFundingDeliveryBinding(delivery, intent, w, binding); err != nil {
		return nil, err
	}
	_ = s.updateIntent(ctx, intent.id, "delivering", binding.TransactionHash, binding.DeliveryID, intent.notificationState, nil)
	drainErr := s.Funding.DrainPending(ctx)
	delivery, err = s.Funding.Get(ctx, binding.DeliveryID, intent.operator)
	if err != nil {
		if drainErr != nil {
			return nil, errors.Join(drainErr, err)
		}
		return nil, err
	}
	if err = validateFundingDeliveryBinding(delivery, intent, w, binding); err != nil {
		return nil, err
	}
	if delivery.Status != "accepted" || delivery.Receipt == nil || delivery.Receipt.Id == "" {
		if drainErr != nil {
			return nil, drainErr
		}
		return nil, fmt.Errorf("%w: accepted Ledger receipt is not available", ErrInvalid)
	}
	if err = validateAcceptedFundingBinding(delivery.Receipt, intent, w, binding); err != nil {
		return nil, err
	}
	if drainErr != nil {
		return nil, drainErr
	}
	if err = s.updateIntent(ctx, intent.id, "credited", binding.TransactionHash, binding.DeliveryID, "pending", nil); err != nil {
		return nil, err
	}
	intent, err = s.loadIntent(ctx, intent.operator, intent.id)
	if err != nil {
		return nil, err
	}
	_ = s.retryNotification(ctx, intent)
	intent, err = s.loadIntent(ctx, intent.operator, intent.id)
	if err != nil {
		return nil, err
	}
	return s.publicIntent(ctx, intent)
}

func validateFundingDeliveryBinding(delivery receiptfunding.Delivery, intent fundingIntent, w wallet.Wallet, binding fundingReceiptBinding) error {
	request := delivery.Request
	observation := request.GetObservation()
	if delivery.ID != binding.DeliveryID || delivery.OperatorIdentity != intent.operator || delivery.WalletID != w.ID || delivery.BudgetID != intent.budgetID ||
		request == nil || request.GetAction() != ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL || request.GetSource().GetCheckId() != binding.CheckID ||
		request.GetSource().GetWalletId() != w.ID || request.GetSource().GetOperatorIdentity() != intent.operator || observation == nil ||
		observation.GetNetwork() != baseusdc.Network || observation.GetAssetId() != baseusdc.TransferAsset || observation.GetDecimals() != baseusdc.USDCDecimals ||
		observation.GetAtomicAmount() != intent.atoms || !strings.EqualFold(observation.GetTransactionHash(), binding.TransactionHash) ||
		observation.GetLegIndex() != uint32(binding.LogIndex) || !strings.EqualFold(observation.GetAddress(), w.Address) ||
		!strings.EqualFold(observation.GetBlockHash(), binding.BlockHash) || observation.GetBookId() != intent.request.GetLedgerBookId() ||
		observation.GetAccountId() != intent.request.GetLedgerAccountId() || request.GetValuation().GetCurrency() != intent.request.GetValuation().GetCurrency() {
		return fmt.Errorf("%w: queued receipt delivery does not match its immutable Base funding binding", ErrConflict)
	}
	return nil
}

func validateAcceptedFundingBinding(receipt *ingestpb.AcceptedCryptoReceipt, intent fundingIntent, w wallet.Wallet, binding fundingReceiptBinding) error {
	if receipt == nil || receipt.GetId() == "" || receipt.GetPostingId() == "" || receipt.GetAction() != ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL ||
		!receipt.GetNoChargeback() || receipt.GetSource().GetCheckId() != binding.CheckID || receipt.GetSource().GetWalletId() != w.ID ||
		receipt.GetSource().GetOperatorIdentity() != intent.operator || receipt.GetObservation().GetNetwork() != baseusdc.Network ||
		receipt.GetObservation().GetAssetId() != baseusdc.TransferAsset || receipt.GetObservation().GetAtomicAmount() != intent.atoms ||
		receipt.GetObservation().GetDecimals() != baseusdc.USDCDecimals || !strings.EqualFold(receipt.GetObservation().GetTransactionHash(), binding.TransactionHash) ||
		receipt.GetObservation().GetLegIndex() != uint32(binding.LogIndex) || !strings.EqualFold(receipt.GetObservation().GetAddress(), w.Address) ||
		!strings.EqualFold(receipt.GetObservation().GetBlockHash(), binding.BlockHash) || receipt.GetCurrency() != intent.request.GetValuation().GetCurrency() {
		return fmt.Errorf("%w: accepted Ledger receipt does not reconcile to the exact Base funding intent", ErrConflict)
	}
	return nil
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
	return &fundingpb.FundingIntent{
		Id: v.id, BudgetId: v.budgetID, Wallet: publicWallet(b, w), AtomicAmount: v.atoms,
		PaymentUri: fmt.Sprintf("ethereum:%s@8453/transfer?address=%s&uint256=%s", baseusdc.USDCAddress, w.Address, v.atoms),
		State:      v.state, TransactionHash: v.txHash, ReceiptDeliveryId: v.deliveryID,
		CreatedAt: timestamppb.New(v.createdAt), UpdatedAt: timestamppb.New(v.updatedAt),
		NotificationState: v.notificationState, LastError: v.lastError,
		FundingMode: s.inboundFundingMode(), PaymentValueMinor: v.request.GetValuation().GetAmountMinor(),
		PaymentCurrency: strings.ToUpper(v.request.GetValuation().GetCurrency()),
	}, nil
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
	instrumentID := stableID("withdrawal-instrument", operator, in.IdempotencyKey)
	withdrawalID := stableID("withdrawal", operator, in.IdempotencyKey)
	authRecord, err := s.Authorizations.Propose(ctx, authorization.ProposeInput{
		ID: authorizationID, IdempotencyKey: authorizationKey, MandateID: grant.ID,
		IdentityToken: identityToken, AmountMinor: amountMinor, Currency: quote.GetCurrency(), Counterparty: quote.GetDestination(),
		SettlementID: settlementID, InstrumentID: instrumentID, SettlementIdempotencyKey: settlementKey,
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
