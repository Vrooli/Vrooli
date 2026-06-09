package findings

import (
	"log"

	"web-search/internal/clock"
	"web-search/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	findingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings/findings_v1connect"

	internalfindings "web-search/internal/findings"
)

// Module returns the findings domain's contribution to the API: the generated
// Connect-RPC service handler. The semantic searcher is injected by main.go
// because it owns the qdrant/embedder wiring; tests pass a fake (or nil to
// exercise the SQL-only paths).
func Module(db *database.RoutedDB, clk clock.Clock, searcher Searcher, logger *log.Logger) module.Module {
	repo := internalfindings.NewSQLiteRepository(db, clk)
	svc := internalfindings.NewService(repo)
	connectPath, connectHandler := findingsconnect.NewFindingsServiceHandler(NewConnectHandler(Deps{
		Service:  svc,
		Searcher: searcher,
		Clock:    clk,
		Logger:   logger,
	}))
	return module.Module{
		Name: "findings",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the findings domain schema so the modules registry can
// collect endpoints + schema from one symbol per handler package.
func Schema() string { return internalfindings.Schema() }

// Endpoints describes the findings module's public surface. Connect-RPC method
// paths reference the generated *Procedure constants, so renaming an RPC in
// findings.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "findings_list",
		Path:        findingsconnect.FindingsServiceListFindingsProcedure,
		Method:      "POST",
		Summary:     "List findings",
		Description: "Lists findings newest-first. Superseded are excluded unless include_archived; an explicit status filters to one lifecycle state.",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search findings list"},
	},
	{
		ID:          "findings_get",
		Path:        findingsconnect.FindingsServiceGetFindingProcedure,
		Method:      "POST",
		Summary:     "Get a finding by id",
		Description: "Returns the finding (with citations) matching the request id.",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search findings get", Args: []string{"<id>"}},
	},
	{
		ID:          "findings_add",
		Path:        findingsconnect.FindingsServiceAddFindingProcedure,
		Method:      "POST",
		Summary:     "Add a finding",
		Description: "Persists a citation-backed claim. Claim is required; confidence is clamped to [0,1].",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search findings add", Args: []string{"--claim", "<claim>", "--confidence", "<c>", "--citations", "<url|title,...>"}},
	},
	{
		ID:          "findings_edit",
		Path:        findingsconnect.FindingsServiceEditFindingProcedure,
		Method:      "POST",
		Summary:     "Edit a finding",
		Description: "Overwrites the claim and confidence of an existing finding; appends an audit row.",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search findings edit", Args: []string{"<id>", "--claim", "<claim>", "--confidence", "<c>"}},
	},
	{
		ID:          "findings_supersede",
		Path:        findingsconnect.FindingsServiceSupersedeFindingProcedure,
		Method:      "POST",
		Summary:     "Supersede a finding",
		Description: "Soft-retires a finding (status=superseded, superseded_by=replacement). Never deletes; excluded from default search.",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search findings supersede", Args: []string{"<id>", "--replacement", "<id>", "--reason", "<reason>"}},
	},
	{
		ID:          "findings_flag",
		Path:        findingsconnect.FindingsServiceFlagFindingProcedure,
		Method:      "POST",
		Summary:     "Flag a finding as disputed",
		Description: "Moves a finding to DISPUTED with a dispute note. Disputed findings are returned by search with a flag.",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search findings flag", Args: []string{"<id>", "--reason", "<reason>"}},
	},
	{
		ID:          "findings_disputes_list",
		Path:        findingsconnect.FindingsServiceListDisputesProcedure,
		Method:      "POST",
		Summary:     "List disputed findings",
		Description: "Returns the DISPUTED findings — the dispute review queue's read side. Convenience over ListFindings(status=disputed).",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search disputes list", Args: []string{"--limit", "<n>"}},
	},
	{
		ID:          "findings_resolve",
		Path:        findingsconnect.FindingsServiceResolveDisputeProcedure,
		Method:      "POST",
		Summary:     "Resolve a disputed finding",
		Description: "Closes a DISPUTED finding: --resolution keep returns it to ACTIVE (clearing the dispute); --resolution supersede retires it in favor of --replacement. Writes an audit row.",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search disputes resolve", Args: []string{"<id>", "--resolution", "<keep|supersede>", "--replacement", "<id>", "--reason", "<reason>"}},
	},
	{
		ID:          "findings_prune",
		Path:        findingsconnect.FindingsServicePruneFindingsProcedure,
		Method:      "POST",
		Summary:     "Prune superseded findings",
		Description: "Archives (hard-deletes) superseded findings. --dry-run reports what would be pruned.",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search findings prune", Args: []string{"--dry-run"}},
	},
	{
		ID:          "findings_search",
		Path:        findingsconnect.FindingsServiceSearchFindingsProcedure,
		Method:      "POST",
		Summary:     "Semantic search over findings",
		Description: "Runs the aisearch-go read path over the findings corpus. Default excludes superseded; disputed returned flagged.",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search findings search", Args: []string{"<query>", "--limit", "<n>", "--include-archived"}},
	},
	{
		ID:          "findings_count",
		Path:        findingsconnect.FindingsServiceCountFindingsProcedure,
		Method:      "POST",
		Summary:     "Count findings in a time window",
		Description: "Counts findings captured within the requested canonical time window. Bound to the `findings count` measure.",
		Category:    "findings",
		CLIMapping:  &module.CLIMapping{Command: "web-search findings count", Args: []string{"--window", "<window>"}},
	},
}
