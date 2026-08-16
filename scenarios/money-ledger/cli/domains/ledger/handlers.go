package ledger

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type handlers struct {
	clientB ledgerconnect.BooksServiceClient
	clientJ ledgerconnect.JournalServiceClient
	clientP ledgerconnect.PositionServiceClient
	clientI ingestconnect.IngestServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	hc, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{clientB: ledgerconnect.NewBooksServiceClient(hc, base), clientJ: ledgerconnect.NewJournalServiceClient(hc, base), clientP: ledgerconnect.NewPositionServiceClient(hc, base), clientI: ingestconnect.NewIngestServiceClient(hc, base)}
}

func (h *handlers) adaptersList(_ cliapp.OperationContext) (*ingestpb.ListAdaptersResponse, error) {
	r, e := h.clientI.ListAdapters(context.Background(), connect.NewRequest(&ingestpb.ListAdaptersRequest{}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) adapterRegister(c cliapp.OperationContext) (*ingestpb.RegisterAdapterResponse, error) {
	kind := ingestpb.AdapterKind_ADAPTER_KIND_FILE
	if strings.EqualFold(c.Flag("kind"), "manual") {
		kind = ingestpb.AdapterKind_ADAPTER_KIND_MANUAL
	}
	r, e := h.clientI.RegisterAdapter(context.Background(), connect.NewRequest(&ingestpb.RegisterAdapterRequest{Adapter: &ingestpb.Adapter{Id: c.Flag("id"), Name: c.Flag("name"), Kind: kind, Enabled: true}}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) ingestEvent(c cliapp.OperationContext) (*ingestpb.IngestEventResponse, error) {
	basis := sharedpb.Basis_BASIS_AUTHORITATIVE
	if strings.EqualFold(c.Flag("basis"), "derived") {
		basis = sharedpb.Basis_BASIS_DERIVED
	}
	if strings.EqualFold(c.Flag("basis"), "operator-asserted") {
		basis = sharedpb.Basis_BASIS_OPERATOR_ASSERTED
	}
	e := &sharedpb.MoneyEvent{ExternalId: c.Flag("external-id"), AdapterId: c.Flag("adapter-id"), AccountId: c.Flag("account-id"), BookId: c.Flag("book-id"), AmountMinor: parseInt(c.Flag("amount-minor")), Currency: strings.ToUpper(c.Flag("currency")), OccurredAt: timestamppb.Now(), FetchedAt: timestamppb.Now(), Basis: basis, Description: c.Flag("description"), Category: c.Flag("category")}
	r, err := h.clientI.IngestEvent(context.Background(), connect.NewRequest(&ingestpb.IngestEventRequest{Event: e}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func (h *handlers) adapterRun(c cliapp.OperationContext) (*ingestpb.RunAdapterResponse, error) {
	r, e := h.clientI.RunAdapter(context.Background(), connect.NewRequest(&ingestpb.RunAdapterRequest{AdapterId: c.Flag("adapter-id")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) fileImport(c cliapp.OperationContext) (*ingestpb.ImportFileResponse, error) {
	r, e := h.clientI.ImportFile(context.Background(), connect.NewRequest(&ingestpb.ImportFileRequest{AdapterId: c.Flag("adapter-id"), Csv: []byte(c.Flag("csv"))}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) operatorImport(c cliapp.OperationContext) (*ingestpb.OperatorImportResponse, error) {
	mode := ingestpb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED
	if strings.EqualFold(c.Flag("source-mode"), "fixture") {
		mode = ingestpb.SourceMode_SOURCE_MODE_FIXTURE
	}
	r, err := h.clientI.ImportOperatorInputs(context.Background(), connect.NewRequest(&ingestpb.OperatorImportRequest{SourcePath: c.Flag("source-path"), SourceMode: mode, Apply: strings.EqualFold(c.Flag("apply"), "true"), AdapterId: c.Flag("adapter-id"), BookId: c.Flag("book-id"), AccountId: c.Flag("account-id")}))
	if err != nil && r == nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("operator import returned no report")
	}
	return r.Msg, err
}

func (h *handlers) booksList(_ cliapp.OperationContext) (*ledgerpb.ListBooksResponse, error) {
	r, e := h.clientB.ListBooks(context.Background(), connect.NewRequest(&ledgerpb.ListBooksRequest{}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) booksCreate(c cliapp.OperationContext) (*ledgerpb.CreateBookResponse, error) {
	r, e := h.clientB.CreateBook(context.Background(), connect.NewRequest(&ledgerpb.CreateBookRequest{Name: c.Flag("name"), Currency: c.Flag("currency")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) accountsList(c cliapp.OperationContext) (*ledgerpb.ListAccountsResponse, error) {
	r, e := h.clientB.ListAccounts(context.Background(), connect.NewRequest(&ledgerpb.ListAccountsRequest{BookId: c.Flag("book-id")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) accountsCreate(c cliapp.OperationContext) (*ledgerpb.CreateAccountResponse, error) {
	r, e := h.clientB.CreateAccount(context.Background(), connect.NewRequest(&ledgerpb.CreateAccountRequest{BookId: c.Flag("book-id"), Name: c.Flag("name"), Kind: c.Flag("kind")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) postingsList(c cliapp.OperationContext) (*ledgerpb.ListPostingsResponse, error) {
	r, e := h.clientJ.ListPostings(context.Background(), connect.NewRequest(&ledgerpb.ListPostingsRequest{AccountId: c.Flag("account-id"), BookId: c.Flag("book-id"), Limit: int32(parseInt(c.Flag("limit")))}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) reverse(c cliapp.OperationContext) (*ledgerpb.ReversePostingResponse, error) {
	r, e := h.clientJ.ReversePosting(context.Background(), connect.NewRequest(&ledgerpb.ReversePostingRequest{PostingId: c.Flag("posting-id"), Reason: c.Flag("reason")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) postingGet(c cliapp.OperationContext) (*ledgerpb.GetPostingResponse, error) {
	r, e := h.clientJ.GetPosting(context.Background(), connect.NewRequest(&ledgerpb.GetPostingRequest{PostingId: c.Flag("posting-id")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) transfer(c cliapp.OperationContext) (*ledgerpb.TransferResponse, error) {
	r, e := h.clientJ.Transfer(context.Background(), connect.NewRequest(&ledgerpb.TransferRequest{FromAccountId: c.Flag("from-account-id"), ToAccountId: c.Flag("to-account-id"), AmountMinor: parseInt(c.Flag("amount-minor")), Currency: c.Flag("currency"), ExternalId: c.Flag("external-id"), Description: c.Flag("description"), OccurredAt: timestamppb.Now()}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) position(c cliapp.OperationContext) (*ledgerpb.PositionResponse, error) {
	r, e := h.clientP.GetPosition(context.Background(), connect.NewRequest(&ledgerpb.PositionRequest{BookId: c.Flag("book-id")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) statement(c cliapp.OperationContext) (*ledgerpb.StatementResponse, error) {
	r, e := h.clientP.GetStatement(context.Background(), connect.NewRequest(&ledgerpb.StatementRequest{BookId: c.Flag("book-id"), From: c.Flag("from"), To: c.Flag("to")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) goals(c cliapp.OperationContext) (*ledgerpb.ListGoalsResponse, error) {
	r, e := h.clientP.ListGoals(context.Background(), connect.NewRequest(&ledgerpb.ListGoalsRequest{BookId: c.Flag("book-id")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) goalDeclare(c cliapp.OperationContext) (*ledgerpb.DeclareGoalResponse, error) {
	g := &ledgerpb.Goal{BookId: c.Flag("book-id"), Name: c.Flag("name"), Metric: c.Flag("metric"), Comparator: c.Flag("comparator"), ThresholdMinor: parseInt(c.Flag("threshold-minor")), ThresholdRatio: parseFloat(c.Flag("threshold-ratio")), ComparandMetric: c.Flag("comparand-metric"), SustainPeriods: int32(parseInt(c.Flag("sustain-periods"))), SustainPeriodUnit: parsePeriodUnit(c.Flag("sustain-period-unit")), BufferMultiple: parseFloat(c.Flag("buffer-multiple"))}
	r, e := h.clientP.DeclareGoal(context.Background(), connect.NewRequest(&ledgerpb.DeclareGoalRequest{Goal: g}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func parsePeriodUnit(v string) ledgerpb.SustainPeriodUnit {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "day", "days":
		return ledgerpb.SustainPeriodUnit_DAY
	case "week", "weeks":
		return ledgerpb.SustainPeriodUnit_WEEK
	default:
		return ledgerpb.SustainPeriodUnit_MONTH
	}
}

func booksListReport(_ cliapp.OperationContext, m *ledgerpb.ListBooksResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d book(s).", len(m.Books))}, ResultsHeading: "Books", Results: mapStrings(len(m.Books), func(i int) string {
		return fmt.Sprintf("%s — %s (%s)", m.Books[i].Id, m.Books[i].Name, m.Books[i].Currency)
	})}
}

func booksCreateReport(_ cliapp.OperationContext, m *ledgerpb.CreateBookResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Created book " + m.Book.Id + "."}}
}

func accountsListReport(_ cliapp.OperationContext, m *ledgerpb.ListAccountsResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d account(s).", len(m.Accounts))}, ResultsHeading: "Accounts", Results: mapStrings(len(m.Accounts), func(i int) string {
		return fmt.Sprintf("%s — %s [%s]", m.Accounts[i].Id, m.Accounts[i].Name, m.Accounts[i].Kind)
	})}
}

func accountsCreateReport(_ cliapp.OperationContext, m *ledgerpb.CreateAccountResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Created account " + m.Account.Id + "."}}
}

func postingsListReport(_ cliapp.OperationContext, m *ledgerpb.ListPostingsResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d posting(s).", len(m.Postings))}, ResultsHeading: "Immutable postings", Results: mapStrings(len(m.Postings), func(i int) string {
		p := m.Postings[i]
		return fmt.Sprintf("%s amount=%d %s basis=%s reversal_of=%s", p.Id, p.Event.AmountMinor, p.Event.Currency, p.Event.Basis.String(), p.ReversalOf)
	})}
}

func reverseReport(_ cliapp.OperationContext, m *ledgerpb.ReversePostingResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Created reversing entry " + m.Posting.Id + "."}}
}

func postingGetReport(_ cliapp.OperationContext, m *ledgerpb.GetPostingResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Posting " + m.Posting.Id}, ResultsHeading: "Audit", Results: []string{fmt.Sprintf("basis=%s amount=%d", m.Posting.Event.Basis.String(), m.Posting.Event.AmountMinor)}}
}

func transferReport(_ cliapp.OperationContext, m *ledgerpb.TransferResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created %d paired transfer postings.", len(m.Postings))}}
}

func positionReport(_ cliapp.OperationContext, m *ledgerpb.PositionResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Cash=%d %s; revenue=%d; expense=%d; runway=%.2f months.", m.CashMinor, m.Currency, m.RevenueMinor, m.ExpenseMinor, m.RunwayMonths)}, ResultsHeading: "Position", Results: []string{fmt.Sprintf("basis=read-time partial=%t", m.Partial)}}
}

func statementReport(_ cliapp.OperationContext, m *ledgerpb.StatementResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Statement %s: inflow=%d, outflow=%d, closing cash=%d %s.", m.BookId, m.InflowMinor, m.OutflowMinor, m.ClosingCashMinor, m.Currency)}, ResultsHeading: "Statement", Results: []string{fmt.Sprintf("partial=%t revenue=%d expense=%d", m.Partial, m.RevenueMinor, m.ExpenseMinor)}}
}

func goalsReport(_ cliapp.OperationContext, m *ledgerpb.ListGoalsResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d goal(s).", len(m.Goals))}, ResultsHeading: "Goals", Results: mapStrings(len(m.Goals), func(i int) string { return m.Goals[i].Goal.Name + " — " + m.Goals[i].Explanation })}
}

func goalDeclareReport(_ cliapp.OperationContext, m *ledgerpb.DeclareGoalResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Declared goal " + m.Goal.Id + "."}}
}

func adaptersReport(_ cliapp.OperationContext, m *ingestpb.ListAdaptersResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d adapter(s).", len(m.Adapters))}, ResultsHeading: "Adapters", Results: mapStrings(len(m.Adapters), func(i int) string {
		a := m.Adapters[i]
		return fmt.Sprintf("%s — %s kind=%s enabled=%t", a.Id, a.Name, a.Kind.String(), a.Enabled)
	})}
}

func adapterRegisterReport(_ cliapp.OperationContext, m *ingestpb.RegisterAdapterResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Registered adapter " + m.Adapter.Id + "."}}
}

func ingestReport(_ cliapp.OperationContext, m *ingestpb.IngestEventResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Admitted posting %s basis=%s duplicate=%t.", m.Posting.Id, m.Posting.Event.Basis.String(), m.Duplicate)}}
}

func runReport(_ cliapp.OperationContext, m *ingestpb.RunAdapterResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Adapter receipt %s status=%s.", m.Receipt.Id, m.Receipt.Status)}, ResultsHeading: "Availability", Results: mapStrings(len(m.Availability), func(i int) string {
		a := m.Availability[i]
		return fmt.Sprintf("%s unavailable: %s last_success=%v", a.AdapterId, a.Reason, a.LastSuccessAt)
	})}
}

func fileImportReport(_ cliapp.OperationContext, m *ingestpb.ImportFileResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Imported %d row(s), wrote %d, skipped %d.", m.Receipt.Read, m.Receipt.Written, m.Receipt.SkippedDuplicates)}}
}

func operatorImportReport(_ cliapp.OperationContext, m *ingestpb.OperatorImportResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Inspected %d operator fields, applied=%t, findings=%d.", m.Read, m.Applied, m.Findings)}}
}
func parseInt(v string) int64     { var n int64; _, _ = fmt.Sscan(v, &n); return n }
func parseFloat(v string) float64 { var n float64; _, _ = fmt.Sscan(v, &n); return n }
func mapStrings(n int, f func(int) string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = strings.TrimSpace(f(i))
	}
	return out
}
