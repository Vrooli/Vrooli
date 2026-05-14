// Package themes is the HTTP-handler home for the themes domain. Exposes
// ThemesService (proto: packages/proto/schemas/react-component-library/v1/themes).
package themes

import (
	"database/sql"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	themesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/themes/themes_v1connect"

	"react-component-library/internal/module"
	"react-component-library/internal/themes"
)

// Module wires the themes domain. designs may be nil for installations
// that disable scenario theme resolution.
func Module(db *sql.DB, designs themes.DesignMDReader, logger *log.Logger) module.Module {
	svc := BuildService(db, designs)
	return ModuleFromService(svc, logger)
}

func ModuleFromService(svc themes.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := themesconnect.NewThemesServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "themes",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internal/themes.Schema for the modules registry.
func Schema() string { return themes.Schema() }

// BuildService constructs a themes.Service backed by SQLite.
func BuildService(db *sql.DB, designs themes.DesignMDReader) themes.Service {
	repo := themes.NewSQLiteRepository(db)
	return themes.NewService(repo, designs)
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "themes_list_builtin",
		Path:        themesconnect.ThemesServiceListBuiltinThemesProcedure,
		Method:      "POST",
		Summary:     "List built-in themes",
		Description: "Returns the seeded built-in themes (vrooli-default, neutral-light, neutral-dark, high-contrast).",
		Category:    "themes",
		Request:     &module.Schema{Type: "object"},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"themes": "array<Theme>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library themes list-builtin"},
	},
	{
		ID:          "themes_get_builtin",
		Path:        themesconnect.ThemesServiceGetBuiltinThemeProcedure,
		Method:      "POST",
		Summary:     "Get a built-in theme by id",
		Description: "Returns the named built-in theme including its flat token map.",
		Category:    "themes",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"theme": "Theme"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No built-in theme with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library themes get-builtin", Args: []string{"<id>"}},
	},
	{
		ID:          "themes_get_from_scenario",
		Path:        themesconnect.ThemesServiceGetThemeFromScenarioProcedure,
		Method:      "POST",
		Summary:     "Resolve a theme from a target scenario's DESIGN.md",
		Description: "Reads <scenarios-root>/<scenario_id>/DESIGN.md via the storage resolver, parses the YAML front-matter, and projects it into a flat CSS-custom-property token map. Client never builds the URL or parses DESIGN.md directly.",
		Category:    "themes",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario_id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"theme": "Theme"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "Target scenario's DESIGN.md missing"},
			{Status: 400, Code: "invalid_argument", Description: "DESIGN.md is malformed or does not project to a recognizable theme"},
			{Status: 500, Code: "internal", Description: "Filesystem read or parse failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library themes get-from-scenario", Args: []string{"<scenario_id>"}},
	},
}
