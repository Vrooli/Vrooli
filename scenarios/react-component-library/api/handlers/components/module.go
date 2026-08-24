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
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	componentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/components/components_v1connect"

	"react-component-library/internal/components"
	"react-component-library/internal/experience"
	"react-component-library/internal/module"
	previewdomain "react-component-library/internal/preview"
	"react-component-library/internal/versionledger"

	"github.com/vrooli/api-core/schedule"
)

// Module wires the components domain into the API server using the
// production-default source root. Tests should use ModuleWithRoot to
// inject a temp dir.
func Module(db *sql.DB, clk schedule.Clock, logger *log.Logger) module.Module {
	root, err := defaultSourceRoot()
	if err != nil {
		logger.Fatalf("components source root: %v", err)
	}
	return ModuleWithRoot(db, clk, root, logger)
}

// ModuleWithRoot is the explicit-injection variant used by tests and by
// callers that want to point the indexer at a custom path.
func ModuleWithRoot(db *sql.DB, clk schedule.Clock, sourceRoot string, logger *log.Logger) module.Module {
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

func WithVersionLedger(ledger *versionledger.Repository) ModuleOption {
	return func(d *Deps) { d.VersionLedger = ledger }
}

// WithIndexObserver installs the components.UpsertObserver the indexer
// calls after each successful upsert.
func WithIndexObserver(o components.UpsertObserver) ModuleOption {
	return func(d *Deps) { d.IndexObserver = o }
}

// WithExperienceReader installs the server-side contract/evidence projection.
func WithExperienceReader(reader experience.Reader) ModuleOption {
	return func(d *Deps) { d.ExperienceReader = reader }
}

func WithPreviewService(service previewdomain.Service) ModuleOption {
	return func(d *Deps) { d.Preview = service }
}

func ModuleFromService(svc components.Service, repo components.Repository, sourceRoot string, logger *log.Logger, opts ...ModuleOption) module.Module {
	d := Deps{
		Service:    svc,
		Repo:       repo,
		SourceRoot: sourceRoot,
		Logger:     logger,
	}
	if authoring, ok := svc.(components.AuthoringService); ok {
		d.Authoring = authoring
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
func BuildService(db *sql.DB, clk schedule.Clock, sourceRoot string) (components.Service, components.Repository) {
	repo := components.NewSQLiteRepository(db, clk)
	svc := components.NewServiceWithContent(repo, components.NewFSContentStore(sourceRoot))
	return svc, repo
}

// defaultSourceRoot resolves the on-disk root the indexer walks.
// Override via COMPONENT_SOURCE_ROOT env. Default is the scenario's
// Git-tracked library/ directory so component source is reviewed and
// versioned with normal repo changes.
func defaultSourceRoot() (string, error) {
	if path := strings.TrimSpace(os.Getenv("COMPONENT_SOURCE_ROOT")); path != "" {
		return path, nil
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve components source root: runtime caller unavailable")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "library"))
	err := os.MkdirAll(filepath.Join(path, "components"), 0o755)
	if err != nil {
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
	},
	{
		ID:          "components_get",
		Path:        componentsconnect.ComponentsServiceGetComponentProcedure,
		Method:      "POST",
		Summary:     "Get a component by id",
		Description: "Returns the component matching the request id. Set include_experience to include its declared contract and latest evidence projection.",
		Category:    "components",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string", "include_experience": "bool (optional contract and evidence projection)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"component": "Component", "experience": "ComponentExperience (when requested)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
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
	},
	{
		ID:          "components_initialize",
		Path:        componentsconnect.ComponentsServiceInitializeComponentProcedure,
		Method:      "POST",
		Summary:     "Initialize a component",
		Description: "Creates component.json plus an initial version TSX file under the Git-tracked library source root, indexes it, and returns the new registry row.",
		Category:    "components",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"library_id":      "string",
			"slug":            "string",
			"display_name":    "string",
			"description":     "string",
			"tags":            "array<string>",
			"initial_version": "string",
			"file_name":       "string",
			"initial_source":  "string",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"component": "Component", "manifest_path": "string", "source_path": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid slug, version, file name, or source header"},
			{Status: 409, Code: "already_exists", Description: "Component libraryId or slug already exists"},
			{Status: 500, Code: "internal", Description: "Filesystem or repository failure"},
		},
	},
	{
		ID:          "components_ingest",
		Path:        componentsconnect.ComponentsServiceIngestComponentProcedure,
		Method:      "POST",
		Summary:     "Ingest a scenario component",
		Description: "Copies a scenario-local TSX file into an indexed library component, creates a draft working version, and returns static de-scenario-ification findings.",
		Category:    "components",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"scenario": "string", "source_file": "string", "slug": "string", "display_name": "string", "description": "string", "tags": "array<string>", "slot": "string",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"component": "Component", "manifest_path": "string", "source_path": "string", "draft_version": "string", "findings": "array<IngestFinding>", "checklist_path": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid scenario, source file, or component metadata"},
			{Status: 409, Code: "already_exists", Description: "Component libraryId or slug already exists"},
			{Status: 500, Code: "internal", Description: "Filesystem or repository failure"},
		},
	},
	{
		ID:          "components_version_begin",
		Path:        componentsconnect.ComponentsServiceBeginComponentVersionProcedure,
		Method:      "POST",
		Summary:     "Begin a mutable component draft",
		Description: "Resolves a stable libraryId or UUID, computes the requested semver bump, copies the complete current version artifact set, indexes the draft, and returns its edit paths and preview path.",
		Category:    "components",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"component": "string (libraryId or UUID)", "bump": "major|minor|patch", "version": "explicit release semver"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"component": "Component", "version": "ComponentVersion", "artifact_paths": "array<string>", "preview_path": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid bump/version or an active draft already exists"},
			{Status: 404, Code: "not_found", Description: "No component with that selector"},
		},
	},
	{
		ID:          "components_version_check",
		Path:        componentsconnect.ComponentsServiceCheckComponentVersionProcedure,
		Method:      "POST",
		Summary:     "Check one component version",
		Description: "Runs source, dependency-closure, story-contract, enum-coverage, and preview-bundle checks for one explicit component version without scanning unrelated catalog assets.",
		Category:    "components",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"component": "string (libraryId or UUID)", "version": "string (defaults draft then latest)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"component": "Component", "version": "string", "passed": "bool", "checks": "array<ComponentVersionCheck>", "preview_path": "string"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No component or version with that selector"}},
	},
	{
		ID:          "components_version_publish",
		Path:        componentsconnect.ComponentsServicePublishComponentVersionProcedure,
		Method:      "POST",
		Summary:     "Publish a validated component draft",
		Description: "Checks the active draft, compiles its preview, preserves every version artifact, updates the release pointers, and rolls back the filesystem mutation if indexing fails.",
		Category:    "components",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"component": "string (libraryId or UUID)", "draft_version": "string", "version": "release semver", "changelog_md": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"component": "Component", "version": "ComponentVersion", "artifact_paths": "array<string>", "preview_path": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component or draft with that selector"},
			{Status: 412, Code: "failed_precondition", Description: "Draft checks, preview compilation, or parity policy failed"},
		},
	},
	{
		ID:          "components_version_create",
		Path:        componentsconnect.ComponentsServiceCreateComponentVersionProcedure,
		Method:      "POST",
		Summary:     "Create a component version",
		Description: "Creates a new version folder for an existing component, updates component.json latest/draft pointers, indexes the library, and returns the version row.",
		Category:    "components",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"component_id": "string",
			"version":      "string",
			"from_version": "string",
			"intent":       "ComponentVersionIntent",
			"file_name":    "string",
			"source":       "string",
			"changelog_md": "string",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"component": "Component", "version": "ComponentVersion", "source_path": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component with that id"},
			{Status: 400, Code: "invalid_argument", Description: "Invalid version, file name, or source header"},
			{Status: 500, Code: "internal", Description: "Filesystem or repository failure"},
		},
	},
	{
		ID:          "components_styles_list",
		Path:        componentsconnect.ComponentsServiceListDesignStylesProcedure,
		Method:      "POST",
		Summary:     "List design styles",
		Description: "Loads canonical design-style metadata from templates/design and returns the style IDs components may reference.",
		Category:    "components",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"styles": "array<DesignStyle>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Design metadata read failure"},
		},
	},
	{
		ID:          "components_style_fit_validate",
		Path:        componentsconnect.ComponentsServiceValidateStyleFitProcedure,
		Method:      "POST",
		Summary:     "Validate component design-style fit",
		Description: "Reads the target scenario's .vrooli/service.json generation.design.id and returns a style-fit verdict for the selected component version.",
		Category:    "components",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"component_id": "string",
			"scenario":     "string",
			"version":      "string",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"kind":           "StyleFitVerdictKind",
			"component_id":   "string",
			"version":        "string",
			"scenario":       "string",
			"scenario_style": "string",
			"affinity":       "DesignAffinity",
			"detail":         "string",
		}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component or component version with that id/version"},
			{Status: 500, Code: "internal", Description: "Scenario service.json read or parse failure"},
		},
	},
	{
		ID:          "components_manifest_update",
		Path:        componentsconnect.ComponentsServiceUpdateComponentManifestProcedure,
		Method:      "POST",
		Summary:     "Update a component manifest",
		Description: "Updates component.json metadata/latest/draft/deprecation fields, re-indexes the library, and returns the refreshed component row.",
		Category:    "components",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"component_id":        "string",
			"display_name":        "string",
			"description":         "string",
			"tags":                "array<string>",
			"latest_version":      "string",
			"draft_version":       "string",
			"deprecated_versions": "array<string>",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"component": "Component"}},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component with that id"},
			{Status: 400, Code: "invalid_argument", Description: "Invalid manifest values"},
			{Status: 500, Code: "internal", Description: "Filesystem or repository failure"},
		},
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
	},
	{
		ID:          "components_versions_list",
		Path:        componentsconnect.ComponentsServiceListComponentVersionsProcedure,
		Method:      "POST",
		Summary:     "List component versions",
		Description: "Returns release and draft artifacts indexed from the component manifest's version folders.",
		Category:    "components",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"component_id": "string", "limit": "int32"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"versions": "array<ComponentVersion>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
	},
	{
		ID:          "components_version_content_get",
		Path:        componentsconnect.ComponentsServiceGetComponentVersionContentProcedure,
		Method:      "POST",
		Summary:     "Read component version source",
		Description: "Returns the indexed source body for a specific component version.",
		Category:    "components",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"component_id": "string", "version": "string"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"version": "ComponentVersion", "content": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component version with that component_id/version pair"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
	},
	{
		ID:          "components_stories_list",
		Path:        componentsconnect.ComponentsServiceListComponentStoriesProcedure,
		Method:      "POST",
		Summary:     "List validated story contracts",
		Description: "Returns typed, indexed story contracts for one component and optional version.",
		Category:    "components",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"component_id": "string", "version": "string (optional)", "limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"stories": "array<ComponentStory>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository read failure"}},
	},
	{
		ID:          "components_preview_frames_list",
		Path:        componentsconnect.ComponentsServiceListPreviewFramesProcedure,
		Method:      "POST",
		Summary:     "List compatible Preview frames",
		Description: "Returns catalog-backed frame candidates with stable compatibility diagnostics. Selection is temporary until explicitly persisted in a story contract.",
		Category:    "components",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"component_id": "string", "version": "string (optional)", "story_id": "string (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"candidates": "array<PreviewFrameCandidate>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "component_id is required"}, {Status: 500, Code: "internal", Description: "Catalog or component registry read failure"}},
	},
	{
		ID:          "components_preview_frame_persist",
		Path:        componentsconnect.ComponentsServicePersistPreviewFrameProcedure,
		Method:      "POST",
		Summary:     "Persist a Preview frame choice",
		Description: "Creates or updates a draft story contract with an exact compatible frame reference; released versions remain immutable.",
		Category:    "components",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"component_id": "string", "version": "string (optional)", "story_id": "string", "asset": "string", "frame_version": "string", "region": "string", "capability": "string (optional)", "fixture": "string (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"component_id": "string", "version": "string", "story_id": "string", "story_json": "string", "source_path": "string"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "component, story, asset, frame version, and region are required"}, {Status: 412, Code: "failed_precondition", Description: "frame or story cannot be persisted"}, {Status: 500, Code: "internal", Description: "Catalog or source write failure"}},
	},
}
