// Package access is Money Ledger's one access policy: which callers may reach
// each route, who the journal names as the actor of a write, and which
// adapters belong to one writer. Gate enforces it in front of every route, so
// a route that is not declared in Policy is refused before it runs.
package access

import (
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/access/access_v1connect"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
)

// Actor is who the journal records as having made a write. It names a role
// rather than a credential, so attribution survives a principal rename.
type Actor string

const (
	// Operator is the owner of these records: a verified person the host's
	// owner-binding policy names as the owner, holding WriteCapability.
	Operator Actor = "operator"
	// Treasury writes its own wallet movements and accepts crypto receipts.
	Treasury Actor = "treasury"
	// LandingPageBusinessSuite posts commerce money events.
	LandingPageBusinessSuite Actor = "landing-page-business-suite"
	// ComputeManagerInvoiceWriter can ingest provider invoice lines and reverse
	// only its own invoice postings. The route handler enforces the source adapter.
	ComputeManagerInvoiceWriter Actor = "compute-manager-invoice-writer"
)

// WriteCapability is the coarse capability every writer must hold, person or
// service, in addition to being named here.
const WriteCapability = "money-ledger:write"

// ComputeInvoiceCapability is separate from the general Ledger write scope.
const ComputeInvoiceCapability = "money-ledger:compute-invoice"
const ComputeManagerInvoiceAdapter = "compute-manager-provider-invoices"

// Effect says whether a route changes the records.
type Effect string

const (
	Read  Effect = "read"
	Write Effect = "write"
	// Session signs a browser in or out. It changes no records and proves
	// nothing by itself: the signed-in person is still checked on each write.
	Session Effect = "session"
)

// Rule is one route's access. Callers is the set of actors admitted; an empty
// set is an open read that any caller may make, including one with no
// identity. A write always names its callers.
type Rule struct {
	Effect  Effect
	Callers []Actor
}

// Open reports whether any caller may use the route.
func (r Rule) Open() bool { return len(r.Callers) == 0 }

// Admits reports whether actor may use the route.
func (r Rule) Admits(actor Actor) bool {
	if r.Open() {
		return true
	}
	for _, caller := range r.Callers {
		if caller == actor {
			return true
		}
	}
	return false
}

func openSession() Rule               { return Rule{Effect: Session} }
func openRead() Rule                  { return Rule{Effect: Read} }
func readBy(callers ...Actor) Rule    { return Rule{Effect: Read, Callers: callers} }
func writtenBy(callers ...Actor) Rule { return Rule{Effect: Write, Callers: callers} }
func operatorOnly() Rule              { return writtenBy(Operator) }

// Policy is every route the API serves, keyed by request path.
//
// Reads are open to any local caller: the API listens only on this host,
// local consumers (Offer Desk, Treasury) read positions and books without a
// credential, and any process running as the operator can read the SQLite
// file directly, so requiring identity to read would add no confidentiality
// against local programs. Browsers are the exception, because the operator's
// browser also runs other websites' script: the gate refuses every cross-site
// browser request (Sec-Fetch-Site) before Policy is consulted, so another
// website cannot read the ledger through localhost. Through the tunnel,
// Cloudflare Access admits only the people its policy names to the UI at all.
//
// Writes change records that cannot be regenerated, so each names exactly who
// may make it. See docs/internal/SECURITY.md.
var Policy = map[string]Rule{
	// Operational probes.
	"/health":                       openRead(),
	"/api/v1/health":                openRead(),
	"/api/v1/capabilities/describe": openRead(),

	accessconnect.AccessServiceGetWriteAccessProcedure: openRead(),
	accessconnect.AccessServiceSignInProcedure:         openSession(),
	accessconnect.AccessServiceSignOutProcedure:        openSession(),

	ledgerconnect.BooksServiceListBooksProcedure:     openRead(),
	ledgerconnect.BooksServiceListAccountsProcedure:  openRead(),
	ledgerconnect.BooksServiceCreateBookProcedure:    operatorOnly(),
	ledgerconnect.BooksServiceArchiveBookProcedure:   operatorOnly(),
	ledgerconnect.BooksServiceCreateAccountProcedure: operatorOnly(),

	ledgerconnect.JournalServiceGetPostingProcedure:   openRead(),
	ledgerconnect.JournalServiceListPostingsProcedure: openRead(),
	// The owner can reverse any accepted posting. Compute Manager can reach this
	// route only because its handler further restricts it to an original posting
	// on the reserved provider-invoice adapter. Transfers remain owner-only.
	ledgerconnect.JournalServiceReversePostingProcedure: writtenBy(Operator, ComputeManagerInvoiceWriter),
	ledgerconnect.JournalServiceTransferProcedure:       operatorOnly(),

	ledgerconnect.PositionServiceGetPositionProcedure:  openRead(),
	ledgerconnect.PositionServiceGetStatementProcedure: openRead(),
	ledgerconnect.PositionServiceListGoalsProcedure:    openRead(),
	ledgerconnect.PositionServiceDeclareGoalProcedure:  operatorOnly(),
	ledgerconnect.PositionServiceArchiveGoalProcedure:  operatorOnly(),
	ledgerconnect.PositionServiceReparentGoalProcedure: operatorOnly(),

	ingestconnect.IngestServiceListAdaptersProcedure:        openRead(),
	ingestconnect.IngestServiceOperatorInputStatusProcedure: openRead(),
	// Preview computes effects without posting or crediting anything.
	ingestconnect.IngestServicePreviewCryptoObservationProcedure: openRead(),
	// Accepted receipts carry customer credit attribution; they are read by
	// the service that accepted them and by the owner.
	ingestconnect.IngestServiceGetCryptoReceiptProcedure:    readBy(Treasury, Operator),
	ingestconnect.IngestServiceListCryptoReceiptsProcedure:  readBy(Treasury, Operator),
	ingestconnect.IngestServiceAcceptCryptoReceiptProcedure: writtenBy(Treasury),
	ingestconnect.IngestServiceRegisterAdapterProcedure:     writtenBy(Operator, Treasury),
	ingestconnect.IngestServiceIngestEventProcedure:         writtenBy(Operator, Treasury, LandingPageBusinessSuite, ComputeManagerInvoiceWriter),
	// Running an adapter, importing a file and applying operator inputs each
	// journal postings in the owner's name.
	ingestconnect.IngestServiceRunAdapterProcedure: operatorOnly(),
	ingestconnect.IngestServiceImportFileProcedure: operatorOnly(),
	// A rehearsal reads a host file the caller names, so it is the owner's
	// operation even when it does not apply.
	ingestconnect.IngestServiceImportOperatorInputsProcedure: operatorOnly(),
}

// serviceWriters names each declared service principal and the actor it
// writes as. Each principal is declared in its scenario's
// .vrooli/service-principals.json with WriteCapability and issued by
// `scenario-authenticator service-principals sync --apply`.
type serviceWriter struct {
	actor Actor
	scope string
}

var serviceWriters = map[string]serviceWriter{
	"treasury-money-ledger":                           {actor: Treasury, scope: WriteCapability},
	"landing-page-business-suite-money-ledger-writer": {actor: LandingPageBusinessSuite, scope: WriteCapability},
	"compute-manager-money-ledger-invoice-writer":     {actor: ComputeManagerInvoiceWriter, scope: ComputeInvoiceCapability},
}

// reservedAdapters gives an adapter id to one actor, so no other writer can
// journal under it: Treasury's wallet feed, the owner's manual entries, and the
// paired postings the owner's transfers create. Every other adapter id is open
// to any actor the route admits.
var reservedAdapters = map[string]Actor{
	"treasury":                   Treasury,
	"manual":                     Operator,
	"transfer":                   Operator,
	ComputeManagerInvoiceAdapter: ComputeManagerInvoiceWriter,
}
