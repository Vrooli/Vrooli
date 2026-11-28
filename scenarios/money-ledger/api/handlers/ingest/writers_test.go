package ingest_test

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"

	"money-ledger/internal/access"
	"money-ledger/internal/testutil/authfixture"
	"money-ledger/internal/testutil/ledgertest"
)

func adapterRequest(id string, kind ingestpb.AdapterKind) *ingestpb.RegisterAdapterRequest {
	return &ingestpb.RegisterAdapterRequest{Adapter: &ingestpb.Adapter{Id: id, Name: id, Kind: kind, Enabled: true}}
}

func event(l *ledgertest.Ledger, adapter, external string) *ingestpb.IngestEventRequest {
	at := timestamppb.New(time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC))
	return &ingestpb.IngestEventRequest{Event: &sharedpb.MoneyEvent{
		Id: external, ExternalId: external, AdapterId: adapter, AccountId: l.Account.Id, BookId: l.Book.Id,
		AmountMinor: -1000, Currency: "USD", OccurredAt: at, FetchedAt: at, Basis: sharedpb.Basis_BASIS_AUTHORITATIVE,
	}}
}

func as[T any](msg *T, headers map[string]string) *connect.Request[T] {
	request := connect.NewRequest(msg)
	for key, value := range headers {
		request.Header().Set(key, value)
	}
	return request
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// The journal names each verified writer by its role, whichever credential
// carried it, and each writer journals under its own adapters.
func TestWritersJournalUnderTheirOwnName(t *testing.T) {
	ctx := context.Background()
	l := ledgertest.New(t)
	owner := bearer(l.Identity.Owner())
	tunnelOwner := map[string]string{"Cf-Access-Jwt-Assertion": l.Identity.OwnerAccess()}
	treasury := bearer(l.Identity.Service(authfixture.TreasuryWriter, authfixture.WriteScope))
	landing := bearer(l.Identity.Service(authfixture.LandingPageWriter, authfixture.WriteScope))
	cm := bearer(l.Identity.Service(authfixture.ComputeManagerInvoiceWriter, authfixture.ComputeInvoiceScope))
	_, err := l.Ingest.RegisterAdapter(ctx, as(adapterRequest("manual", ingestpb.AdapterKind_ADAPTER_KIND_MANUAL), tunnelOwner))
	require.NoError(t, err)
	_, err = l.Ingest.RegisterAdapter(ctx, as(adapterRequest("treasury", ingestpb.AdapterKind_ADAPTER_KIND_AGGREGATOR), treasury))
	require.NoError(t, err)
	_, err = l.Ingest.RegisterAdapter(ctx, as(adapterRequest("lpbs-funding", ingestpb.AdapterKind_ADAPTER_KIND_FILE), owner))
	require.NoError(t, err)
	_, err = l.Ingest.RegisterAdapter(ctx, as(&ingestpb.RegisterAdapterRequest{Adapter: &ingestpb.Adapter{Id: access.ComputeManagerInvoiceAdapter, Name: "Compute Manager provider invoices", Kind: ingestpb.AdapterKind_ADAPTER_KIND_AGGREGATOR, Enabled: true}}, owner))
	require.NoError(t, err)
	for _, test := range []struct {
		name, adapter, actor string
		headers              map[string]string
	}{
		{"owner in the browser through the tunnel", "manual", "operator", tunnelOwner},
		{"owner on the CLI", "lpbs-funding", "operator", owner},
		{"treasury on its adapter", "treasury", "treasury", treasury},
		{"landing page on an adapter the owner registered", "lpbs-funding", "landing-page-business-suite", landing},
		{"compute manager on its dedicated invoice adapter with its narrow scope", access.ComputeManagerInvoiceAdapter, string(access.ComputeManagerInvoiceWriter), cm},
	} {
		if test.adapter == access.ComputeManagerInvoiceAdapter {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			out, err := l.Ingest.IngestEvent(ctx, as(event(l, test.adapter, test.name), test.headers))
			require.NoError(t, err)
			var actor string
			require.NoError(t, l.DB.QueryRowContext(ctx, `SELECT actor FROM postings WHERE id=?`, out.Msg.Posting.Id).Scan(&actor))
			require.Equal(t, test.actor, actor)
		})
	}
	_, err = l.Ingest.IngestEvent(ctx, as(event(l, access.ComputeManagerInvoiceAdapter, "owner-forbidden"), owner))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
	_, err = l.Ingest.IngestEvent(ctx, as(event(l, access.ComputeManagerInvoiceAdapter, "treasury-forbidden"), treasury))
	require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

// A reserved adapter belongs to one writer: no other writer, the owner
// included, can register or journal under it, and the refusal stores nothing.
func TestReservedAdaptersBelongToOneWriter(t *testing.T) {
	ctx := context.Background()
	l := ledgertest.New(t)
	owner := bearer(l.Identity.Owner())
	treasury := bearer(l.Identity.Service(authfixture.TreasuryWriter, authfixture.WriteScope))
	landing := bearer(l.Identity.Service(authfixture.LandingPageWriter, authfixture.WriteScope))
	for _, test := range []struct {
		name, adapter string
		headers       map[string]string
	}{
		{"owner on treasury's adapter", "treasury", owner},
		{"landing page on treasury's adapter", "treasury", landing},
		{"treasury on the owner's manual entries", "manual", treasury},
		{"landing page on the owner's manual entries", "manual", landing},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := l.Ingest.RegisterAdapter(ctx, as(adapterRequest(test.adapter, ingestpb.AdapterKind_ADAPTER_KIND_MANUAL), test.headers))
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "RegisterAdapter")
			_, err = l.Ingest.IngestEvent(ctx, as(event(l, test.adapter, test.name), test.headers))
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "IngestEvent")
		})
	}
	require.Equal(t, 1, l.Count(t, "adapters"), "the built-in Compute Manager invoice adapter remains seeded")
	require.Zero(t, l.Count(t, "postings"))
}

// Importing a file journals in the owner's name: the actor comes from the
// gate, not from the importer.
func TestFileImportJournalsTheOwner(t *testing.T) {
	ctx := context.Background()
	l := ledgertest.New(t)
	owner := bearer(l.Identity.Owner())
	_, err := l.Ingest.RegisterAdapter(ctx, as(adapterRequest("bank-csv", ingestpb.AdapterKind_ADAPTER_KIND_FILE), owner))
	require.NoError(t, err)
	csv := "external_id,account_id,book_id,amount_minor,currency,occurred_at,description,category\nrow-1," + l.Account.Id + "," + l.Book.Id + ",-250,USD,2026-11-01T00:00:00Z,coffee,food\n"
	receipt, err := l.Ingest.ImportFile(ctx, as(&ingestpb.ImportFileRequest{AdapterId: "bank-csv", Csv: []byte(csv)}, owner))
	require.NoError(t, err)
	require.EqualValues(t, 1, receipt.Msg.Receipt.Written)
	var actor string
	require.NoError(t, l.DB.QueryRowContext(ctx, `SELECT actor FROM postings WHERE external_id='row-1'`).Scan(&actor))
	require.Equal(t, "operator", actor)
}
