package findings

import (
	"log"

	"web-search/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	findingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings/findings_v1connect"

	internalfindings "web-search/internal/findings"
)

// Module returns the findings domain's contribution to the API: the generated
// Connect-RPC service handler. The semantic searcher is injected by main.go
// because it owns the qdrant/embedder wiring; tests pass a fake (or nil to
// exercise the SQL-only paths). indexKick (nil-safe) fires after successful
// content mutations so the semantic index reconciles within seconds of a
// write instead of waiting out the periodic sync interval.
func Module(db *database.RoutedDB, clk schedule.Clock, searcher Searcher, surfacer Surfacer, indexKick func(), logger *log.Logger) module.Module {
	repo := internalfindings.NewSQLiteRepository(db, clk)
	svc := internalfindings.WithMutationNotify(internalfindings.NewService(repo), indexKick)
	gc := internalfindings.NewGCService(svc, clk, internalfindings.GCConfig{})
	connectPath, connectHandler := findingsconnect.NewFindingsServiceHandler(NewConnectHandler(Deps{
		Service:  svc,
		Searcher: searcher,
		Surfacer: surfacer,
		GC:       gc,
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
	},
	{
		ID:          "findings_get",
		Path:        findingsconnect.FindingsServiceGetFindingProcedure,
		Method:      "POST",
		Summary:     "Get a finding by id",
		Description: "Returns the finding (with citations) matching the request id.",
		Category:    "findings",
	},
	{
		ID:          "findings_add",
		Path:        findingsconnect.FindingsServiceAddFindingProcedure,
		Method:      "POST",
		Summary:     "Add a finding",
		Description: "Persists a citation-backed claim. Claim is required; confidence is clamped to [0,1].",
		Category:    "findings",
	},
	{
		ID:          "findings_edit",
		Path:        findingsconnect.FindingsServiceEditFindingProcedure,
		Method:      "POST",
		Summary:     "Edit a finding",
		Description: "Overwrites the claim and confidence of an existing finding; appends an audit row.",
		Category:    "findings",
	},
	{
		ID:          "findings_supersede",
		Path:        findingsconnect.FindingsServiceSupersedeFindingProcedure,
		Method:      "POST",
		Summary:     "Supersede a finding",
		Description: "Soft-retires a finding (status=superseded, superseded_by=replacement). Never deletes; excluded from default search.",
		Category:    "findings",
	},
	{
		ID:          "findings_flag",
		Path:        findingsconnect.FindingsServiceFlagFindingProcedure,
		Method:      "POST",
		Summary:     "Flag a finding as disputed",
		Description: "Moves a finding to DISPUTED with a dispute note. Disputed findings are returned by search with a flag.",
		Category:    "findings",
	},
	{
		ID:          "findings_disputes_list",
		Path:        findingsconnect.FindingsServiceListDisputesProcedure,
		Method:      "POST",
		Summary:     "List disputed findings",
		Description: "Returns the DISPUTED findings — the dispute review queue's read side. Convenience over ListFindings(status=disputed).",
		Category:    "findings",
	},
	{
		ID:          "findings_resolve",
		Path:        findingsconnect.FindingsServiceResolveDisputeProcedure,
		Method:      "POST",
		Summary:     "Resolve a disputed finding",
		Description: "Closes a DISPUTED finding: --resolution keep returns it to ACTIVE (clearing the dispute); --resolution supersede retires it in favor of --replacement. Writes an audit row.",
		Category:    "findings",
	},
	{
		ID:          "findings_prune",
		Path:        findingsconnect.FindingsServicePruneFindingsProcedure,
		Method:      "POST",
		Summary:     "Prune superseded findings",
		Description: "Archives (hard-deletes) superseded findings. --dry-run reports what would be pruned.",
		Category:    "findings",
	},
	{
		ID:          "findings_search",
		Path:        findingsconnect.FindingsServiceSearchFindingsProcedure,
		Method:      "POST",
		Summary:     "Semantic search over findings",
		Description: "Runs the aisearch-go read path over the findings corpus. Default excludes superseded; disputed returned flagged.",
		Category:    "findings",
	},
	{
		ID:          "findings_count",
		Path:        findingsconnect.FindingsServiceCountFindingsProcedure,
		Method:      "POST",
		Summary:     "Count findings in a time window",
		Description: "Counts findings captured within the requested canonical time window. Bound to the `findings count` measure.",
		Category:    "findings",
	},
	{
		ID: "findings_used_rate", Path: findingsconnect.FindingsServiceUsedRateProcedure, Method: "POST",
		Summary: "Measure finding usage rate", Description: "Ratio of used to surfaced findings over the declared window.", Category: "findings",
	},
	{
		ID: "findings_never_surfaced", Path: findingsconnect.FindingsServiceNeverSurfacedProcedure, Method: "POST",
		Summary: "Count never-used findings", Description: "Counts findings without an explicit use record over the declared creation window.", Category: "findings",
	},
	{
		ID:          "findings_effectiveness",
		Path:        findingsconnect.FindingsServiceListEffectivenessProcedure,
		Method:      "POST",
		Summary:     "List findings by usage effectiveness",
		Description: "Pairs each finding with its usage telemetry (surfaced/used counts, last-surfaced age) and the blended effective score (age-decayed confidence × usage factor). The OT-P2-001 curation signal.",
		Category:    "findings",
	},
	{
		ID:          "findings_record_usage",
		Path:        findingsconnect.FindingsServiceRecordUsageProcedure,
		Method:      "POST",
		Summary:     "Record explicit finding usage",
		Description: "Records an explicit 'this finding was used' signal (distinct from the implicit surfacing counted automatically on search).",
		Category:    "findings",
	},
	{
		ID:          "findings_gc",
		Path:        findingsconnect.FindingsServiceRunGCProcedure,
		Method:      "POST",
		Summary:     "Run the store-consistency GC",
		Description: "Runs the periodic full-store consistency pass: soft-retires never-surfaced, fully-decayed findings (confidence-gated) and reports cold-archive candidates, stale disputes, and orphans. --dry-run reports without mutating. Never hard-deletes; never auto-resolves a dispute.",
		Category:    "findings",
	},
}
