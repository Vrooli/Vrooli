// Package receiptfunding delivers retained chain evidence to Money Ledger and
// admits budget funding only from its immutable acceptance response.
package receiptfunding

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/database"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"treasury/internal/baseusdc"
	"treasury/internal/budget"
	"treasury/internal/ledger"
	"treasury/internal/wallet"
)

type CheckReader interface {
	GetCheck(context.Context, string, string, string, string) (baseusdc.Check, error)
}

type Wallets interface {
	Get(context.Context, string, string, string) (wallet.Wallet, error)
}

type Acceptor interface {
	Accept(context.Context, *ingestpb.AcceptCryptoReceiptRequest) (*ingestpb.AcceptedCryptoReceipt, error)
}

type ReceiptReader interface {
	Get(context.Context, string) (*ingestpb.AcceptedCryptoReceipt, error)
}

type CreditLotGranter interface {
	GrantAcceptedReceipt(context.Context, *ingestpb.AcceptedCryptoReceipt) (CreditLotLink, error)
}

type CreditLotLink struct{ ID, Owner, ReceiptID string }

type UnavailableAcceptor struct{}

func (UnavailableAcceptor) Accept(context.Context, *ingestpb.AcceptCryptoReceiptRequest) (*ingestpb.AcceptedCryptoReceipt, error) {
	return nil, errors.New("Treasury service identity or Money Ledger destination is not configured")
}

type SubmitInput struct {
	OperatorIdentity, WalletID, TransactionHash, CheckID string
	BudgetID, LedgerBookID, LedgerAccountID              string
	OriginalReceiptID, PreviousReceiptID, Reason         string
	LogIndex                                             uint64
	Action                                               ingestpb.CryptoReceiptAction
	Valuation                                            *ingestpb.CryptoValuation
	Attribution                                          *ingestpb.CryptoPayerAttribution
}

type Service struct {
	store      *Store
	checks     CheckReader
	wallets    Wallets
	budgets    *budget.SQLiteRepository
	acceptor   Acceptor
	creditLots CreditLotGranter
	now        func() time.Time
}

func (s *Service) SetCreditLotGranter(granter CreditLotGranter) { s.creditLots = granter }

func NewService(store *Store, checks CheckReader, wallets Wallets, budgets *budget.SQLiteRepository, acceptor Acceptor, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, checks: checks, wallets: wallets, budgets: budgets, acceptor: acceptor, now: now}
}

func (s *Service) Submit(ctx context.Context, in SubmitInput) (Delivery, error) {
	if s == nil || s.store == nil || s.checks == nil || s.wallets == nil || s.budgets == nil {
		return Delivery{}, errors.New("receipt funding dependencies unavailable")
	}
	if in.OperatorIdentity == "" || in.CheckID == "" || in.BudgetID == "" || in.LedgerBookID == "" || in.LedgerAccountID == "" || in.LogIndex > math.MaxUint32 || in.Valuation == nil {
		return Delivery{}, fmt.Errorf("%w: retained check, funding destination and explicit valuation are required", ErrInvalid)
	}
	w, err := s.wallets.Get(ctx, in.WalletID, baseusdc.Network, in.OperatorIdentity)
	if err != nil {
		return Delivery{}, err
	}
	policy, err := s.budgets.Get(ctx, in.BudgetID)
	if err != nil {
		return Delivery{}, err
	}
	if policy.BookID != w.BookID || policy.Currency != in.Valuation.Currency {
		return Delivery{}, fmt.Errorf("%w: wallet, budget and valuation must share the operator book and currency", ErrInvalid)
	}
	check, err := s.checks.GetCheck(ctx, in.OperatorIdentity, in.WalletID, in.TransactionHash, in.CheckID)
	if err != nil {
		return Delivery{}, err
	}
	transfers := check.Transfers
	state := ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_FINALIZED
	switch in.Action {
	case ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL,
		ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REINCLUSION,
		ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_CORRECTION:
		if check.Outcome != "verified" && !(in.Action == ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_CORRECTION && check.Outcome == "provider_observation_changed") {
			return Delivery{}, fmt.Errorf("%w: this check contains no verified transfer", ErrInvalid)
		}
	case ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_REMOVAL:
		if check.Relation != "provider_reports_not_canonical" || check.PriorCheckID == "" {
			return Delivery{}, fmt.Errorf("%w: removal requires retained noncanonicality linked to its prior check", ErrInvalid)
		}
		prior, err := s.checks.GetCheck(ctx, in.OperatorIdentity, in.WalletID, in.TransactionHash, check.PriorCheckID)
		if err != nil || prior.Outcome != "verified" {
			return Delivery{}, fmt.Errorf("%w: verified prior inclusion is unavailable", ErrInvalid)
		}
		transfers = prior.Transfers
		state = ingestpb.CryptoObservationState_CRYPTO_OBSERVATION_STATE_REMOVED
	default:
		return Delivery{}, fmt.Errorf("%w: receipt action is required", ErrInvalid)
	}
	var observation *baseusdc.Observation
	for i := range transfers {
		if transfers[i].LogIndex == in.LogIndex {
			observation = &transfers[i]
			break
		}
	}
	if observation == nil || observation.Finality != "finalized" || observation.WalletID != w.ID || observation.Network != baseusdc.Network || observation.AssetID != baseusdc.TransferAsset || !strings.EqualFold(observation.To, w.Address) {
		return Delivery{}, fmt.Errorf("%w: finalized native USDC transfer to this wallet is required", ErrInvalid)
	}
	attribution := &ingestpb.CryptoPayerAttribution{PayerAddress: observation.From}
	if in.Attribution != nil {
		attribution = proto.Clone(in.Attribution).(*ingestpb.CryptoPayerAttribution)
		if attribution.GetCreditLotId() != "" || attribution.GetCreditLotOwner() != "" {
			return Delivery{}, fmt.Errorf("%w: LPBS lot identifiers are assigned only after Money Ledger acceptance", ErrInvalid)
		}
		if attribution.PayerAddress != "" && !strings.EqualFold(attribution.PayerAddress, observation.From) {
			return Delivery{}, fmt.Errorf("%w: payer address does not match the retained transfer", ErrInvalid)
		}
		attribution.PayerAddress = observation.From
	}
	source := &ingestpb.TreasuryReceiptLink{CheckId: check.ID, WalletId: w.ID, OperatorIdentity: in.OperatorIdentity, SourceReference: "treasury://wallets/" + w.ID + "/checks/" + check.ID}
	request := &ingestpb.AcceptCryptoReceiptRequest{
		Observation: &ingestpb.PreviewCryptoObservationRequest{Network: observation.Network, AssetId: observation.AssetID, AtomicAmount: observation.AtomicAmount, Decimals: observation.Decimals, State: state, LegKind: ingestpb.CryptoLegKind_CRYPTO_LEG_KIND_DEPOSIT, SourceReference: source.SourceReference, TransactionHash: observation.TransactionHash, BlockHash: observation.BlockHash, LegIndex: uint32(in.LogIndex), BookId: in.LedgerBookID, AccountId: in.LedgerAccountID, Address: observation.To},
		Source:      source, Action: in.Action, OriginalReceiptId: in.OriginalReceiptID, PreviousReceiptId: in.PreviousReceiptID,
		Valuation: proto.Clone(in.Valuation).(*ingestpb.CryptoValuation), Attribution: attribution,
		OccurredAt: timestamppb.New(observation.BlockTime), CorrectionReason: strings.TrimSpace(in.Reason),
	}
	if in.Action != ingestpb.CryptoReceiptAction_CRYPTO_RECEIPT_ACTION_ORIGINAL && (in.OriginalReceiptID == "" || in.PreviousReceiptID == "" || request.CorrectionReason == "") {
		return Delivery{}, fmt.Errorf("%w: revisions require the accepted original, predecessor and reason", ErrInvalid)
	}
	if err := s.budgets.BindFunding(ctx, policy.ID, observation.Network, observation.AssetID, observation.Decimals, s.now()); err != nil {
		return Delivery{}, err
	}
	return s.store.Queue(ctx, Delivery{ID: check.ID + "/log/" + fmt.Sprint(in.LogIndex), OperatorIdentity: in.OperatorIdentity, WalletID: w.ID, BudgetID: policy.ID, SourceCheckID: check.ID, Request: request, CreatedAt: s.now().UTC()})
}

func (s *Service) Get(ctx context.Context, id, operator string) (Delivery, error) {
	return s.store.Get(ctx, id, operator)
}

func (s *Service) DrainPending(ctx context.Context) error {
	if s == nil || s.store == nil || s.acceptor == nil || s.budgets == nil {
		return errors.New("authenticated Money Ledger acceptance is unavailable")
	}
	values, err := s.store.Pending(ctx, 100)
	if err != nil {
		return err
	}
	var failures []error
	for _, value := range values {
		receipt := value.Receipt
		if value.Status == "queued" {
			receipt, err = s.acceptor.Accept(ctx, value.Request)
			if err == nil {
				err = validateAcceptance(value.Request, receipt)
			}
			if err == nil {
				err = s.budgets.CreditAccepted(ctx, budget.FundingCredit{LedgerReceiptID: receipt.Id, PostingID: receipt.PostingId, SourceCheckID: receipt.Source.CheckId, BudgetID: value.BudgetID, AtomicDelta: receipt.AtomicDelta, ValuationDeltaMinor: receipt.ValuationDeltaMinor, ValuationCurrency: receipt.Currency, AcceptedAt: receipt.AcceptedAt.AsTime()})
			}
			if err == nil {
				err = s.store.Accept(ctx, value.ID, receipt)
				if err == nil {
					value.Status = "accepted"
				}
			}
		}
		if err == nil && receipt != nil && value.Status == "accepted" {
			reader, ok := s.acceptor.(ReceiptReader)
			if !ok {
				err = errors.New("Money Ledger accepted receipt reader is unavailable")
			} else {
				var canonical *ingestpb.AcceptedCryptoReceipt
				canonical, err = reader.Get(ctx, receipt.GetId())
				if err == nil && !proto.Equal(receipt, canonical) {
					err = errors.New("Money Ledger receipt readback differs from retained acceptance")
				}
			}
		}
		if err == nil && receipt != nil && receipt.GetAttribution().GetPayerIdentity() != "" {
			if s.creditLots == nil {
				err = errors.New("LPBS accepted-receipt credit-lot owner is unavailable")
			} else {
				var link CreditLotLink
				link, err = s.creditLots.GrantAcceptedReceipt(ctx, receipt)
				if err == nil && (link.ID == "" || link.ReceiptID != receipt.GetId() || link.Owner != "landing-page-business-suite") {
					err = errors.New("LPBS returned an incomplete or mismatched credit-lot association")
				}
				if err == nil {
					err = s.store.AssociateCreditLot(ctx, value.ID, link.ReceiptID, link.Owner, link.ID, s.now())
				}
			}
		}
		if err != nil {
			_ = s.store.Failure(context.WithoutCancel(ctx), value.ID, err)
			failures = append(failures, fmt.Errorf("receipt %s: %w", value.ID, err))
		}
	}
	return errors.Join(failures...)
}

func validateAcceptance(request *ingestpb.AcceptCryptoReceiptRequest, receipt *ingestpb.AcceptedCryptoReceipt) error {
	if receipt == nil || receipt.Id == "" || receipt.PostingId == "" || receipt.Source == nil || receipt.Observation == nil || receipt.Source.CheckId == "" || receipt.AcceptedAt == nil || receipt.AcceptedAt.CheckValid() != nil || receipt.AcceptedAt.AsTime().IsZero() || receipt.Currency != request.Valuation.Currency || receipt.Source.WalletId != request.Source.WalletId || receipt.Source.OperatorIdentity != request.Source.OperatorIdentity || receipt.Action != request.Action || !receipt.NoChargeback {
		return errors.New("Money Ledger returned incomplete or mismatched receipt acceptance")
	}
	quantity, ok := new(big.Int).SetString(receipt.AtomicDelta, 10)
	if !ok || quantity.String() != receipt.AtomicDelta {
		return errors.New("Money Ledger returned an invalid atomic delta")
	}
	want, got := proto.Clone(request.Observation).(*ingestpb.PreviewCryptoObservationRequest), proto.Clone(receipt.Observation).(*ingestpb.PreviewCryptoObservationRequest)
	// A replay may have a new Treasury check while the ledger returns the first
	// accepted receipt. That provenance alias must not change economic content.
	want.SourceReference, got.SourceReference = "", ""
	if !proto.Equal(want, got) || !proto.Equal(request.Valuation, receipt.Valuation) || !proto.Equal(request.Attribution, receipt.Attribution) {
		return errors.New("Money Ledger acceptance differs from retained token, valuation or payer evidence")
	}
	return nil
}

func (s *Service) Run(ctx context.Context, interval time.Duration, logger *log.Logger) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	drain := func() {
		bounded, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := s.DrainPending(bounded); err != nil && logger != nil && !errors.Is(err, context.Canceled) {
			logger.Printf("receipt funding delivery deferred: %v", err)
		}
	}
	drain()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			drain()
		}
	}
}

type MoneyLedgerClient struct {
	client ingestconnect.IngestServiceClient
	tokens ledger.TokenSource
}

func NewMoneyLedgerClient(baseURL string, tokens ledger.TokenSource, client *http.Client) (*MoneyLedgerClient, error) {
	if strings.TrimSpace(baseURL) == "" || tokens == nil {
		return nil, errors.New("Money Ledger URL and Treasury service identity are required")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &MoneyLedgerClient{client: ingestconnect.NewIngestServiceClient(client, strings.TrimRight(baseURL, "/")), tokens: tokens}, nil
}

func (c *MoneyLedgerClient) Accept(ctx context.Context, value *ingestpb.AcceptCryptoReceiptRequest) (*ingestpb.AcceptedCryptoReceipt, error) {
	token, err := c.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return nil, errors.New("Treasury service identity is unavailable")
	}
	request := connect.NewRequest(value)
	request.Header().Set("Authorization", "Bearer "+token)
	if database.IsTestMode(ctx) {
		request.Header().Set("X-Vrooli-Test-Mode", "1")
	}
	response, err := c.client.AcceptCryptoReceipt(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.Msg.GetReceipt(), nil
}

func (c *MoneyLedgerClient) Get(ctx context.Context, id string) (*ingestpb.AcceptedCryptoReceipt, error) {
	token, err := c.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return nil, errors.New("Treasury service identity is unavailable")
	}
	request := connect.NewRequest(&ingestpb.GetCryptoReceiptRequest{Id: id})
	request.Header().Set("Authorization", "Bearer "+token)
	if database.IsTestMode(ctx) {
		request.Header().Set("X-Vrooli-Test-Mode", "1")
	}
	response, err := c.client.GetCryptoReceipt(ctx, request)
	if err != nil {
		return nil, err
	}
	return response.Msg.GetReceipt(), nil
}
