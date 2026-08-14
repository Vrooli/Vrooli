package ingest

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
	"money-ledger/internal/ledger"
)

func newIngestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	sqlDB := databasetest.NewSQLite(t)
	db := database.NewFromPrimary(sqlDB)
	ctx := context.Background()
	require.NoError(t, database.EnsureSchemas(ctx, sqlDB, database.SchemaProviderFunc(func() string {
		return (&ledger.Store{}).Schema()
	})))
	return NewStore(db, nil), ctx
}

func TestManualAdapterUsesOperatorBasisAndDeduplicates(t *testing.T) { // [REQ:CTR-002] [REQ:CTR-003] [REQ:CTR-004]
	s, ctx := newIngestStore(t)
	book, err := s.journal.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	account, err := s.journal.CreateAccount(ctx, book.Id, "Cash", "cash")
	require.NoError(t, err)
	_, err = s.RegisterAdapter(ctx, &ingestpb.Adapter{Id: "manual", Name: "Hand entry", Kind: ingestpb.AdapterKind_ADAPTER_KIND_MANUAL, Enabled: true})
	require.NoError(t, err)
	event := &sharedpb.MoneyEvent{ExternalId: "cash-1", AdapterId: "manual", AccountId: account.Id, BookId: book.Id, AmountMinor: 900, Currency: "USD", OccurredAt: timestamppb.Now(), Basis: sharedpb.Basis_BASIS_AUTHORITATIVE}
	posting, duplicate, receipt, err := s.IngestEvent(ctx, event)
	require.NoError(t, err)
	require.False(t, duplicate)
	require.Equal(t, sharedpb.Basis_BASIS_OPERATOR_ASSERTED, posting.Event.Basis)
	require.EqualValues(t, 1, receipt.Written)
	_, duplicate, _, err = s.IngestEvent(ctx, event)
	require.NoError(t, err)
	require.True(t, duplicate)
}

func TestOperatorInputsImportPreservesPendingAsAbsent(t *testing.T) { // [REQ:CTR-006]
	s, ctx := newIngestStore(t)
	book, err := s.journal.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	account, err := s.journal.CreateAccount(ctx, book.Id, "Cash", "cash")
	require.NoError(t, err)
	_, err = s.RegisterAdapter(ctx, &ingestpb.Adapter{Id: "operator-inputs", Name: "Operator inputs", Kind: ingestpb.AdapterKind_ADAPTER_KIND_MANUAL, Enabled: true})
	require.NoError(t, err)
	field := func(value float64, status string) map[string]any {
		return map[string]any{"value": value, "status": status}
	}
	root := map[string]any{
		"cash":            field(1000, "current"),
		"monthlyBurn":     map[string]any{"aiApi": field(10, "current"), "infrastructure": field(20, "current"), "saas": field(30, "pending-operator"), "tooling": field(40, "current")},
		"timeAllocation":  map[string]any{"product": field(50, "current"), "services": field(60, "current"), "ops": field(70, "current")},
		"servicesRevenue": map[string]any{"leadGen": field(80, "current"), "doneForYou": field(90, "current"), "consulting": field(100, "current")},
		"servicesTime":    map[string]any{"hoursThisWindow": field(110, "current")},
		"subscriptions":   map[string]any{"mrr": field(120, "current")},
	}
	data, err := json.Marshal(root)
	require.NoError(t, err)
	report, err := s.ImportOperatorInputs(ctx, data, "operator-inputs", book.Id, account.Id)
	require.NoError(t, err)
	require.Equal(t, len(operatorPaths), report.Read)
	require.Equal(t, len(operatorPaths)-1, report.Written)
	var count int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM postings`).Scan(&count))
	require.Equal(t, report.Written, count)
}

func TestOperatorInputsFixtureImportCarriesSourceProvenance(t *testing.T) { // [REQ:CTR-006]
	s, ctx := newIngestStore(t)
	book, err := s.journal.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	account, err := s.journal.CreateAccount(ctx, book.Id, "Cash", "cash")
	require.NoError(t, err)
	_, err = s.RegisterAdapter(ctx, &ingestpb.Adapter{Id: "operator-inputs", Name: "Operator inputs", Kind: ingestpb.AdapterKind_ADAPTER_KIND_MANUAL, Enabled: true})
	require.NoError(t, err)
	report, err := s.ImportOperatorInputsFile(ctx, "testdata/operator-inputs.json", "operator-inputs", book.Id, account.Id)
	require.NoError(t, err)
	require.Equal(t, len(operatorPaths)-1, report.Written)
	var description string
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT description FROM postings ORDER BY occurred_at LIMIT 1`).Scan(&description))
	require.Contains(t, description, "testdata/operator-inputs.json")
}

func TestFileImportReportsMalformedInputWithoutWriting(t *testing.T) { // [REQ:CTR-003]
	s, ctx := newIngestStore(t)
	_, err := s.RegisterAdapter(ctx, &ingestpb.Adapter{Id: "file", Name: "CSV", Kind: ingestpb.AdapterKind_ADAPTER_KIND_FILE, Enabled: true})
	require.NoError(t, err)
	receipt, err := s.ImportFile(ctx, "file", []byte("external_id,account_id,book_id,amount_minor,currency,occurred_at,description,category\nfoo,account,book,nope,USD,2026-01-01T00:00:00Z,,"), nil, nil)
	require.Error(t, err)
	require.Equal(t, "failed", receipt.Status)
	require.EqualValues(t, 0, receipt.Written)
}

func TestFailedAdapterIsVisibleAndNeverWritesZero(t *testing.T) { // [REQ:POS-004]
	s, ctx := newIngestStore(t)
	book, err := s.journal.CreateBook(ctx, "Operating", "USD")
	require.NoError(t, err)
	_, err = s.RegisterAdapter(ctx, &ingestpb.Adapter{Id: "bank-feed", Name: "Bank feed", Kind: ingestpb.AdapterKind_ADAPTER_KIND_AGGREGATOR, Enabled: true, AvailabilityReason: "upstream timeout"})
	require.NoError(t, err)
	receipt, availability, err := s.RunAdapter(ctx, "bank-feed", nil, nil)
	require.ErrorContains(t, err, "upstream timeout")
	require.Equal(t, "failed", receipt.Status)
	require.Len(t, availability, 1)
	require.Equal(t, "upstream timeout", availability[0].Reason)
	position, err := s.journal.Position(ctx, book.Id)
	require.NoError(t, err)
	require.True(t, position.Partial)
	require.EqualValues(t, 0, position.CashMinor)
	require.Len(t, position.Availability, 1)
}
