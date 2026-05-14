// Package components is the HTTP-handler home for the components
// domain. It exposes the generated Connect-RPC ComponentsService
// (proto schema: packages/proto/schemas/react-component-library/v1/components).
//
// RPCs (mounted at /vrooli.react_component_library.v1.components.ComponentsService/...):
//
//	ListComponents          — filtered list (match / tag / limit)
//	GetComponent            — fetch by primary id
//	GetComponentByLibraryId — fetch by @libraryId header value
//	IndexComponents         — re-walk the configured source root
package components

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/storage"

	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"

	"react-component-library/internal/clock"
	"react-component-library/internal/components"
	"react-component-library/internal/module"
)

// Module wires the components domain into the API server using the
// production-default source root. Tests should use ModuleWithRoot to
// inject a temp dir.
func Module(db *sql.DB, clk clock.Clock, logger *log.Logger) module.Module {
	root, err := defaultSourceRoot()
	if err != nil {
		logger.Fatalf("components source root: %v", err)
	}
	return ModuleWithRoot(db, clk, root, logger)
}

// ModuleWithRoot is the explicit-injection variant used by tests and by
// callers that want to point the indexer at a custom path.
func ModuleWithRoot(db *sql.DB, clk clock.Clock, sourceRoot string, logger *log.Logger) module.Module {
	svc, repo := BuildService(db, clk, sourceRoot)
	return ModuleFromService(svc, repo, sourceRoot, logger)
}

// ModuleFromService mounts a prebuilt components.Service. main.go uses
// this variant so sibling modules (preview, versions) and the content-
// change listener bind to the same service instance.
// ModuleOption mutates the Deps the components handler is constructed
// with. Use WithIndexObserver to plug in a cross-domain post-upsert
// hook (req 10's deps recorder).
type ModuleOption func(*Deps)

// WithIndexObserver installs the components.UpsertObserver the indexer
// calls after each successful upsert.
func WithIndexObserver(o components.UpsertObserver) ModuleOption {
	return func(d *Deps) { d.IndexObserver = o }
}

func ModuleFromService(svc components.Service, repo components.Repository, sourceRoot string, logger *log.Logger, opts ...ModuleOption) module.Module {
	d := Deps{
		Service:    svc,
		Repo:       repo,
		SourceRoot: sourceRoot,
		Logger:     logger,
	}
	for _, opt := range opts {
		opt(&d)
	}
	connectPath, connectHandler := componentsconnect.NewComponentsServiceHandler(NewConnectHandler(d))
	return module.Module{
		Name: "components",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internal/components.Schema for the modules
// registry. Keeps the registry shape uniform.
func Schema() string { return components.Schema() }

// DefaultSourceRoot exposes the resolved on-disk components root that
// the default Module(...) wiring uses. main.go reads it once to share
// the same root with sibling modules (e.g. preview).
func DefaultSourceRoot() (string, error) { return defaultSourceRoot() }

// BuildService is the shared seam main.go calls to construct one
// components.Service that multiple modules can read from. Keeps a
// single Service per process so preview reads see the same content
// the components handler does.
func BuildService(db *sql.DB, clk clock.Clock, sourceRoot string) (components.Service, components.Repository) {
	repo := components.NewSQLiteRepository(db, clk)
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(sourceRoot))
	return svc, repo
}

// defaultSourceRoot resolves the on-disk root the indexer walks.
// Override via COMPONENT_SOURCE_ROOT env. Default lives under the
// scenario's storage data class so re-running setup doesn't clobber it.
func defaultSourceRoot() (string, error) {
	if path := strings.TrimSpace(os.Getenv("COMPONENT_SOURCE_ROOT")); path != "" {
		return path, nil
	}
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return "", fmt.Errorf("storage resolver: %w", err)
	}
	path, err := resolver.Path(
		storage.Options{ScenarioID: "react-component-library"},
		storage.ClassData,
		"components",
	)
	if err != nil {
		return "", fmt.Errorf("resolve components root: %w", err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("create components root: %w", err)
	}
	return path, nil
}

// Endpoints is the machine-readable description of the components
// module's public surface. Connect-RPC method paths reference generated
// *Procedure constants so adding/renaming an RPC in components.proto
// breaks this file at compile time. The parity test in module_test.go
// enforces that every RPC has exactly one matching entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "components_list",
		Path:        componentsconnect.ComponentsServiceListComponentsProcedure,
		Method:      "POST",
		Summary:     "List components",
		Description: "Returns components matching optional match / tag / limit filters through the generated Connect-RPC ComponentsService.",
		Category:    "components",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"match": "string (case-insensitive substring filter)",
				"tag":   "string (exact tag filter)",
				"limit": "int32 (max rows, 0 = server default)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"components": "array<Component>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library components list"},
	},
	{
		ID:          "components_get",
		Path:        componentsconnect.ComponentsServiceGetComponentProcedure,
		Method:      "POST",
		Summary:     "Get a component by id",
		Description: "Returns the component matching the request id.",
		Category:    "components",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"component": "Component"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library components get", Args: []string{"<id>"}},
	},
	{
		ID:          "components_get_by_library_id",
		Path:        componentsconnect.ComponentsServiceGetComponentByLibraryIdProcedure,
		Method:      "POST",
		Summary:     "Get a component by libraryId",
		Description: "Returns the component matching the @libraryId header value.",
		Category:    "components",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"library_id": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"component": "Component"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component with that libraryId"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library components get-by-library-id", Args: []string{"<libraryId>"}},
	},
	{
		ID:          "components_index",
		Path:        componentsconnect.ComponentsServiceIndexComponentsProcedure,
		Method:      "POST",
		Summary:     "Re-index components from disk",
		Description: "Walks the configured source root for *.tsx files with @libraryId headers and upserts them into the registry. Removes rows whose files have been deleted.",
		Category:    "components",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scanned":     "int32",
				"indexed":     "int32",
				"skipped":     "int32",
				"deleted":     "int32",
				"errors":      "array<string>",
				"library_ids": "array<string>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Walk or upsert failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library components index"},
	},
	{
		ID:          "components_content_get",
		Path:        componentsconnect.ComponentsServiceGetComponentContentProcedure,
		Method:      "POST",
		Summary:     "Read a component's source file",
		Description: "Returns the raw TSX content of the component's source file plus a SHA-256 digest. Path-traversal guard rejects rows whose source_path escapes the configured root.",
		Category:    "components",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"content":     "string (utf-8 source)",
				"source_path": "string",
				"sha256":      "string (hex digest)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component with that id"},
			{Status: 400, Code: "invalid_argument", Description: "Component source_path escapes configured root"},
			{Status: 500, Code: "internal", Description: "Filesystem read failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library components content-get", Args: []string{"<id>"}},
	},
	{
		ID:          "components_content_set",
		Path:        componentsconnect.ComponentsServiceUpdateComponentContentProcedure,
		Method:      "POST",
		Summary:     "Write a component's source file",
		Description: "Overwrites the component's TSX file in place. Pass expected_sha256 from a prior read for optimistic concurrency. Re-run `index` afterward if the @libraryId header changed.",
		Category:    "components",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":              "string",
				"content":         "string (utf-8 source)",
				"expected_sha256": "string (optional)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"sha256":      "string (hex digest of written content)",
				"source_path": "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component with that id"},
			{Status: 400, Code: "invalid_argument", Description: "Component source_path escapes configured root"},
			{Status: 412, Code: "failed_precondition", Description: "expected_sha256 did not match on-disk digest"},
			{Status: 500, Code: "internal", Description: "Filesystem write failure"},
		},
		CLIMapping: &module.CLIMapping{Command: "react-component-library components content-set", Args: []string{"<id>", "<file>"}},
	},
}
