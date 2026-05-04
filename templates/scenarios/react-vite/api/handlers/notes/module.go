package notes

import (
	"database/sql"
	"log"

	"github.com/gorilla/mux"

	"{{SCENARIO_ID}}/internal/clock"
	"{{SCENARIO_ID}}/internal/module"
	internalnotes "{{SCENARIO_ID}}/internal/notes"
)

// Module returns the notes domain's contribution to the API: a wired
// handler subrouter (PathPrefix /api/v1/notes) plus the endpoint
// descriptors the codegen reads. Callers from main.go just pass the
// db handle, clock, and logger; the module owns the rest of the
// repo + service + handler chain.
//
// Adding a real domain to a scenario means copying this file into
// handlers/<dom>/module.go and pointing it at <dom>'s Repository /
// Service / NewHandler. The center (server.New) does not change.
func Module(db *sql.DB, clk clock.Clock, logger *log.Logger) module.Module {
	repo := internalnotes.NewSQLiteRepository(db, clk)
	svc := internalnotes.NewService(repo)
	h := NewHandler(Deps{Service: svc, Logger: logger})
	return module.Module{
		Name: "notes",
		Mount: func(r *mux.Router) {
			r.PathPrefix("/api/v1/notes").Handler(h)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalnotes.Schema so the modules registry can
// collect both endpoint descriptors and schema from one symbol per
// handler package. Keeps the registry's per-domain shape uniform:
// handlers/<dom>/{Module, Endpoints, Schema}.
func Schema() string { return internalnotes.Schema() }
