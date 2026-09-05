// Package preview is the HTTP-handler home for the preview domain. It
// exposes the generated Connect-RPC PreviewService alongside a static
// HTML route (/preview/{id}/harness.html) the host UI loads into the
// live-preview iframe.
//
// The Connect surface is the canonical seam (CLI + UI consume it for
// diagnostics and machine reads); the static route is browser-only
// plumbing that wraps the same service call so iframes can load it as
// a URL. See docs/RESEARCH.md for the build-approach decision.
package preview

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	previewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview/preview_v1connect"

	"react-component-library/internal/components"
	"react-component-library/internal/deps"
	"react-component-library/internal/module"
	"react-component-library/internal/preview"
)

// Module wires the preview domain. Depends on a components.Service so
// the bundler's content read goes through the same path-traversal
// guard the components handler uses.
func Module(comp components.Service, logger *log.Logger) module.Module {
	return ModuleWithDeps(comp, nil, logger)
}

func ModuleWithDeps(comp components.Service, depsSvc deps.Service, logger *log.Logger) module.Module {
	return ModuleWithDepsAtRoot(comp, depsSvc, logger, "")
}

func ModuleWithDepsAtRoot(comp components.Service, depsSvc deps.Service, logger *log.Logger, repoRoot string) module.Module {
	svc := BuildServiceAtRoot(comp, depsSvc, repoRoot)
	return ModuleFromService(svc, comp, logger, repoRoot)
}

func BuildServiceAtRoot(comp components.Service, depsSvc deps.Service, repoRoot string) preview.Service {
	return preview.NewServiceWithDepsAtRoot(comp, preview.NewEsbuilder(), depsSvc, repoRoot)
}

func ModuleFromService(svc preview.Service, comp components.Service, logger *log.Logger, repoRoot string) module.Module {
	connectPath, connectHandler := previewconnect.NewPreviewServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	harness := NewHarnessHandlerWithStoriesAtRoot(svc, comp, logger, repoRoot)
	runtime := NewRuntimeHandlerAtRoot(logger, repoRoot)
	return module.Module{
		Name: "preview",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			r.HandleFunc("/preview/runtime/{module:react|react-dom}@{version}", runtime.ServeHTTP).Methods("GET")
			r.HandleFunc("/preview/runtime/{module:react|react-dom}@{version}/{path:.*}", runtime.ServeHTTP).Methods("GET")
			r.HandleFunc("/preview/runtime/npm/{module:.+}@{version}", runtime.ServeHTTP).Methods("GET")
			r.HandleFunc("/preview/runtime/npm/{module:.+}@{version}/{path:.*}", runtime.ServeHTTP).Methods("GET")
			r.HandleFunc("/preview/foundations/base-styles-fixture.html", harness.ServeBaseStylesFixture).Methods("GET")
			r.HandleFunc("/preview/{id}/harness.html", harness.ServeHTTP).Methods("GET")
		},
		Endpoints: Endpoints,
	}
}

// Endpoints describes the preview module's machine-readable surface.
// The static harness route is intentionally NOT listed — it's a
// browser-only wrapper around the Connect RPC, not a separate logical
// endpoint. The parity test in module_test.go enforces 1:1 between
// proto RPCs and entries here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "preview_get_bundle",
		Path:        previewconnect.PreviewServiceGetPreviewBundleProcedure,
		Method:      "POST",
		Summary:     "Bundle a component for preview",
		Description: "Transpiles a component's TSX source to an ES module via esbuild. Used by the CLI for diagnostics and by the harness HTML route for iframe rendering. Externals: react, react-dom, react-dom/client.",
		Category:    "preview",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"js":          "string (ES module text)",
				"source_path": "string",
				"sha256":      "string (hex digest of js)",
				"warnings":    "array<string>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No component with that id"},
			{Status: 400, Code: "invalid_argument", Description: "Bundler rejected the file (syntax error or unresolved import) or source_path escapes root"},
			{Status: 500, Code: "internal", Description: "Filesystem read or bundler internal failure"},
		},
	},
}
