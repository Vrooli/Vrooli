// Package validation_record is the HTTP/Connect handler edge for the
// validation_record domain.
package validation_record

import (
	"log"

	"github.com/vrooli/api-core/database"

	"development-toolchain-validator/internal/clock"
	"development-toolchain-validator/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	vrconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record/validation_record_v1connect"

	vr "development-toolchain-validator/internal/validation_record"
)

// Module returns the validation_record domain's contribution to the API.
func Module(db *database.RoutedDB, clk clock.Clock, logger *log.Logger) module.Module {
	repo := vr.NewSQLiteRepository(db)
	svc := vr.NewService(repo, clk)
	connectPath, connectHandler := vrconnect.NewValidationRecordServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "validation_record",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the internal package schema.
func Schema() string { return vr.Schema() }

// Endpoints is the machine-readable description of the
// validation_record module's public surface.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_record_list",
		Path:        vrconnect.ValidationRecordServiceListRecordsProcedure,
		Method:      "POST",
		Summary:     "List validation records",
		Description: "Returns terminal validation records with cursor-paginated history, ordered by ended_at descending. Filters by golden_slug, subject_id, tuple_kind.",
		Category:    "validation_record",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"golden_slug": "string (optional)",
				"subject_id":  "string (optional)",
				"tuple_kind":  "TupleKind (optional)",
				"page_size":   "int32",
				"page_token":  "string",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"records":         "array<ValidationRecord>",
				"next_page_token": "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Bad page_token"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List records", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.validation_record.ValidationRecordService/ListRecords -H 'Content-Type: application/json' -d '{\"golden_slug\":\"reference-react-vite\"}'"},
		},
	},
	{
		ID:          "validation_record_get",
		Path:        vrconnect.ValidationRecordServiceGetRecordProcedure,
		Method:      "POST",
		Summary:     "Get a validation record by id",
		Description: "Returns the record with the given id.",
		Category:    "validation_record",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"record": "ValidationRecord"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "No record with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get record", Curl: "curl http://localhost:${API_PORT}/vrooli.development_toolchain_validator.v1.validation_record.ValidationRecordService/GetRecord -H 'Content-Type: application/json' -d '{\"id\":\"<uuid>\"}'"},
		},
	},
}
