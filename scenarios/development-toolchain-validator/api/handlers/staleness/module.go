// Package staleness is the HTTP/Connect handler edge for the staleness
// domain. The staleness service has no SQL of its own — it composes
// reads over manifest, golden, skill_catalog.
package staleness

import (
	"database/sql"
	"log"

	"development-toolchain-validator/internal/clock"
	golden "development-toolchain-validator/internal/golden"
	manifest "development-toolchain-validator/internal/manifest"
	"development-toolchain-validator/internal/module"
	skillcatalog "development-toolchain-validator/internal/skill_catalog"
	stalenessdom "development-toolchain-validator/internal/staleness"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	stalenessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/staleness/staleness_v1connect"
)

// Module returns the staleness domain's contribution to the API.
//
// The staleness service depends on three peer domains; their
// repositories are constructed inline to read directly without
// inverting through services.
func Module(db *sql.DB, clk clock.Clock, logger *log.Logger) module.Module {
	manifestRepo := manifest.NewSQLiteRepository(db, clk)
	manifestSvc := manifest.NewService(manifestRepo, clk)
	goldenRepo := golden.NewSQLiteRepository(db, clk)
	skillRepo := skillcatalog.NewSQLiteRepository(db, clk)

	svc := stalenessdom.NewService(
		stalenessdom.ManifestSourceFromService{Svc: manifestSvc, Repo: manifestRepo},
		stalenessdom.GoldenSourceFromRepo{Repo: goldenRepo},
		stalenessdom.SkillSourceFromRepo{Repo: skillRepo},
	)

	connectPath, connectHandler := stalenessconnect.NewStalenessServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "staleness",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema is the empty-string contribution — staleness owns no SQL.
// Returning empty keeps the registry symmetric without requiring a
// special-case "modules without schema" path.
func Schema() string { return "" }

// Endpoints is the machine-readable description of the staleness
// module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "staleness_list",
		Path:        stalenessconnect.StalenessServiceListStaleProcedure,
		Method:      "POST",
		Summary:     "List stale (skill_id, golden_slug) tuples",
		Description: "Returns every manifest where the pinned template or skill version differs from current, with manual ClearStale overrides applied.",
		Category:    "staleness",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"entries": "array<StaleEntry>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List stale", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.staleness.StalenessService/ListStale -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "development-toolchain-validator staleness list"},
	},
}
