// Package focus is the API handler for the FocusService — the gaps registry +
// prioritization domain. It is the proto translation edge over internal/focus;
// all business logic lives in internal/focus behind seams.
package focus

import (
	"log"

	"meta-optimization-manager/internal/clock"
	internalcoverage "meta-optimization-manager/internal/coverage"
	internalfocus "meta-optimization-manager/internal/focus"
	"meta-optimization-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	focusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/focus/focus_v1connect"
)

// Module returns the focus domain's contribution to the API: the generated
// FocusService Connect-RPC handler, backed by the SQLite gaps registry and a
// live GapSource derived from the owner space docs. The space reader is shared
// with the coverage domain (the same cross-scenario read seam) — wired here at
// the production edge, never imported into internal/focus.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	svc := internalfocus.NewService(internalfocus.Deps{
		Source: internalfocus.NewSpaceGapSource(internalcoverage.NewSpaceReader()),
		Repo:   internalfocus.NewSQLiteRepository(db, clk),
	})
	connectPath, connectHandler := focusconnect.NewFocusServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "focus",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalfocus.Schema so the modules registry collects
// endpoints and schema from one symbol per handler package.
func Schema() string { return internalfocus.Schema() }

// Endpoints is the machine-readable description of the focus module's public
// surface. The Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in focus.proto breaks this at compile time; the
// global parity test (registry_test.go) asserts every RPC has exactly one entry.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "focus_get",
		Path:        focusconnect.FocusServiceGetFocusProcedure,
		Method:      "POST",
		Summary:     "Ranked next-best gaps",
		Description: "Returns the ranked next-best gaps (impact × importance) with qualitative context (OT-P0-002). Optionally filtered by projection and capped by limit.",
		Category:    "focus",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"items": "array<FocusItem>"},
		},
	},
	{
		ID:          "focus_list_gaps",
		Path:        focusconnect.FocusServiceListGapsProcedure,
		Method:      "POST",
		Summary:     "List the gaps registry",
		Description: "Returns the full gaps registry (live-derived non-NOW cells overlaid with the owned registry), optionally filtered by projection/cell/status (OT-P0-003).",
		Category:    "focus",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"gaps": "array<Gap>"},
		},
	},
	{
		ID:          "focus_get_gap",
		Path:        focusconnect.FocusServiceGetGapProcedure,
		Method:      "POST",
		Summary:     "Get one gap",
		Description: "Returns one gap by id with its full qualitative context.",
		Category:    "focus",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"gap": "Gap"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No gap with that id"},
		},
	},
	{
		ID:          "focus_add_gap_note",
		Path:        focusconnect.FocusServiceAddGapNoteProcedure,
		Method:      "POST",
		Summary:     "Append an explored approach to a gap",
		Description: "Appends an explored-but-unbuilt approach to a gap — the one focus write verb (the 'store our thinking' primitive). Materializes a registry row for a derived gap.",
		Category:    "focus",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"gap": "Gap"},
		},
	},
}
