package reconcile

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/serviceprincipal"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const InvoiceAdapterID = "compute-manager-provider-invoices"
const InvoiceWriterPrincipal = "compute-manager-money-ledger-invoice-writer"
const InvoiceWriterScope = "money-ledger:compute-invoice"
const MoneyLedgerAudience = "scenario-authenticator:money-ledger"

type LedgerTokenSource func(context.Context) (string, error)

// NewInvoiceWriterTokens reads the separately approved Ledger principal from
// Credential Authority and exchanges it for a short-lived token whose complete
// identity is pinned. Missing grant/credential fails closed with Onboarding
// remediation; no command or ambient-token fallback is offered.
func NewInvoiceWriterTokens() *servicetoken.Tokens {
	identity := serviceprincipal.CredentialIdentity(InvoiceWriterPrincipal, "money-ledger")
	return servicetoken.New(servicetoken.Config{
		Name:       "Compute Manager provider invoice writer",
		Credential: credentialauthority.ServiceCredential(identity, serviceprincipal.CredentialField),
		Expect:     servicetoken.ExpectIdentity(InvoiceWriterPrincipal, MoneyLedgerAudience, InvoiceWriterScope),
		IssueHint:  "review the Compute Manager provider invoice writer in Vrooli Onboarding",
	})
}

func InvoiceWriterTokenSource(tokens *servicetoken.Tokens) LedgerTokenSource {
	return func(ctx context.Context) (string, error) {
		if tokens == nil {
			return "", fmt.Errorf("Money Ledger invoice writer identity is not configured")
		}
		return tokens.Token(ctx)
	}
}

// ConnectInvoiceLedger calls Money Ledger's generated Connect API. Token is
// resolved at use from credential authority; this type never accepts an
// ambient env/token fallback. The configured book/account are explicit to
// prevent an accidental post into an unrelated ledger.
type ConnectInvoiceLedger struct {
	BaseURL, BookID, AccountID string
	HTTPClient                 connect.HTTPClient
	Token                      LedgerTokenSource
	Now                        func() time.Time
	ingest                     ingestconnect.IngestServiceClient
	journal                    ledgerconnect.JournalServiceClient
}

func NewConnectInvoiceLedger(baseURL, bookID, accountID string, client connect.HTTPClient, token LedgerTokenSource) (*ConnectInvoiceLedger, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || bookID == "" || accountID == "" || client == nil || token == nil {
		return nil, fmt.Errorf("Money Ledger Connect URL, book, account, HTTP client, and credential-authority token source are required")
	}
	return &ConnectInvoiceLedger{BaseURL: baseURL, BookID: bookID, AccountID: accountID, HTTPClient: client, Token: token,
		ingest: ingestconnect.NewIngestServiceClient(client, baseURL), journal: ledgerconnect.NewJournalServiceClient(client, baseURL)}, nil
}

func (c *ConnectInvoiceLedger) PostInvoice(ctx context.Context, posting InvoicePosting) (string, error) {
	line, externalID := posting.Line, posting.ExternalID
	if err := posting.Destination.Validate(); err != nil {
		return "", err
	}
	if posting.Destination.BookID != c.BookID || posting.Destination.AccountID != c.AccountID {
		return "", fmt.Errorf("invoice destination does not match the configured Money Ledger target")
	}
	token, err := c.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve Money Ledger invoice-writer credential: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("Money Ledger invoice-writer credential is unavailable")
	}
	now := time.Now().UTC()
	if c.Now != nil {
		now = c.Now().UTC()
	}
	description := fmt.Sprintf("Compute Manager provider invoice %s line %s (%s)", line.InvoiceID, line.LineID, line.Provider)
	request := connect.NewRequest(&ingestpb.IngestEventRequest{Event: &sharedpb.MoneyEvent{
		ExternalId: externalID, AdapterId: InvoiceAdapterID, AccountId: posting.Destination.AccountID, BookId: posting.Destination.BookID,
		AmountMinor: line.AmountMinor, Currency: strings.ToUpper(strings.TrimSpace(line.Currency)),
		OccurredAt: timestamppb.New(line.OccurredAt.UTC()), FetchedAt: timestamppb.New(now),
		Basis: sharedpb.Basis_BASIS_AUTHORITATIVE, Description: description, Category: "compute.provider_invoice",
	}})
	request.Header().Set("Authorization", "Bearer "+token)
	response, err := c.ingest.IngestEvent(ctx, request)
	if err != nil {
		return "", fmt.Errorf("Money Ledger invoice event-ingest failed: %w", err)
	}
	if response.Msg.GetPosting() == nil || strings.TrimSpace(response.Msg.GetPosting().GetId()) == "" {
		return "", fmt.Errorf("Money Ledger returned no invoice posting identity")
	}
	return response.Msg.GetPosting().GetId(), nil
}

func (c *ConnectInvoiceLedger) ReverseInvoice(ctx context.Context, postingID, reason string) (string, error) {
	token, err := c.Token(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve Money Ledger invoice-writer credential: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("Money Ledger invoice-writer credential is unavailable")
	}
	request := connect.NewRequest(&ledgerpb.ReversePostingRequest{PostingId: postingID, Reason: reason})
	request.Header().Set("Authorization", "Bearer "+token)
	response, err := c.journal.ReversePosting(ctx, request)
	if err != nil {
		return "", fmt.Errorf("Money Ledger invoice reversal failed: %w", err)
	}
	if response.Msg.GetPosting() == nil || strings.TrimSpace(response.Msg.GetPosting().GetId()) == "" {
		return "", fmt.Errorf("Money Ledger returned no invoice reversal identity")
	}
	return response.Msg.GetPosting().GetId(), nil
}

var _ InvoiceLedger = (*ConnectInvoiceLedger)(nil)
