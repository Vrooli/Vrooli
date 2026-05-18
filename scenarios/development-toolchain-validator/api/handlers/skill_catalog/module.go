// Package skill_catalog is the HTTP/Connect handler edge for the
// skill_catalog domain. It owns proto ↔ domain translation (adapter.go)
// and Connect-RPC method implementations (connect_handler.go) but
// contains no business logic — that all lives in internal/skill_catalog/.
package skill_catalog

import (
	"database/sql"
	"log"

	"development-toolchain-validator/internal/clock"
	"development-toolchain-validator/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	skillcatalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog/skill_catalog_v1connect"

	internalskillcatalog "development-toolchain-validator/internal/skill_catalog"
)

// Module returns the skill_catalog domain's contribution to the API.
// Production callers pass a real SkillCatalogSource (REST adapter for
// prompt-manager); tests use ModuleWithSource with a fake source.
func Module(db *sql.DB, clk clock.Clock, source internalskillcatalog.SkillCatalogSource, logger *log.Logger) module.Module {
	return ModuleWithSource(db, clk, source, logger)
}

// ModuleWithSource is the explicit-injection variant. Used by tests to
// supply a deterministic SkillCatalogSource.
func ModuleWithSource(db *sql.DB, clk clock.Clock, source internalskillcatalog.SkillCatalogSource, logger *log.Logger) module.Module {
	repo := internalskillcatalog.NewSQLiteRepository(db, clk)
	svc := internalskillcatalog.NewService(repo, source, clk)
	connectPath, connectHandler := skillcatalogconnect.NewSkillCatalogServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "skill_catalog",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the internal package schema so the modules registry
// can collect schema from one symbol per handler package.
func Schema() string { return internalskillcatalog.Schema() }

// Endpoints is the machine-readable description of the skill_catalog
// module's public surface. Connect-RPC method paths reference generated
// *Procedure constants so renaming an RPC in skill_catalog.proto breaks
// this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "skill_catalog_sync",
		Path:        skillcatalogconnect.SkillCatalogServiceSyncProcedure,
		Method:      "POST",
		Summary:     "Sync skill catalog from prompt-manager",
		Description: "Pulls the current skill set from prompt-manager and reconciles the local mirror (insert new, update changed, remove missing). Returns the post-reconcile catalog plus counts.",
		Category:    "skill_catalog",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"skills":  "array<Skill>",
				"added":   "int32",
				"updated": "int32",
				"removed": "int32",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 503, Code: "unavailable", Description: "prompt-manager dependency is not running"},
			{Status: 500, Code: "internal", Description: "Upstream fetch or repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Sync catalog", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.skill_catalog.SkillCatalogService/Sync -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "development-toolchain-validator skill-catalog sync"},
	},
	{
		ID:          "skill_catalog_list",
		Path:        skillcatalogconnect.SkillCatalogServiceListSkillsProcedure,
		Method:      "POST",
		Summary:     "List mirrored skills",
		Description: "Returns the local mirror of the skill catalog ordered by id ascending.",
		Category:    "skill_catalog",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"skills": "array<Skill>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List skills", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.skill_catalog.SkillCatalogService/ListSkills -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "development-toolchain-validator skill-catalog list"},
	},
	{
		ID:          "skill_catalog_get",
		Path:        skillcatalogconnect.SkillCatalogServiceGetSkillProcedure,
		Method:      "POST",
		Summary:     "Get a mirrored skill by id",
		Description: "Returns the skill whose id matches the request.",
		Category:    "skill_catalog",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"skill": "Skill"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "No skill with that id exists in the local mirror"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get skill", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.skill_catalog.SkillCatalogService/GetSkill -H 'Content-Type: application/json' -d '{\"id\":\"plan-skill-discovery\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "development-toolchain-validator skill-catalog get", Args: []string{"<id>"}},
	},
}
