// Package exports_service hosts the BAS ExportsService Connect-RPC handler.
//
// ExportsService owns the persisted export-library record set: rendered
// replays (mp4/gif), JSON export packages, and HTML interactive replays.
// Each row is metadata about the produced artifact (name, format, storage
// URL, duration, file size, AI-generated caption, status).
//
// The replay-export bytes path (POST /executions/{id}/export) intentionally
// stays on chi as a RESTException because it streams binary content and
// writes to a caller-supplied output_dir. ExportsService is exclusively the
// metadata/library + lifecycle surface.
package exports_service

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/ai"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	exportsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports/exportsconnect"
)

// Repo is the narrow seam over database.Repository for export rows. The
// production wiring satisfies it with *database.repository.
type Repo interface {
	CreateExport(ctx context.Context, export *database.ExportIndex) error
	GetExport(ctx context.Context, id uuid.UUID) (*database.ExportIndex, error)
	UpdateExport(ctx context.Context, export *database.ExportIndex) error
	DeleteExport(ctx context.Context, id uuid.UUID) error
	ListExports(ctx context.Context, limit, offset int) ([]*database.ExportIndex, error)
	ListExportsByExecution(ctx context.Context, executionID uuid.UUID) ([]*database.ExportIndex, error)
	ListExportsByWorkflow(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*database.ExportIndex, error)
}

// ExecutionLookup is the narrow seam used to verify an execution exists
// before creating an export. Satisfied by workflow.ExecutionService.
type ExecutionLookup interface {
	GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
}

// CatalogLookup fetches a workflow summary by id for caption generation.
// Satisfied by workflow.CatalogService.
type CatalogLookup interface {
	GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error)
}

// AIClientFactory is the narrow seam used to build a per-request AI
// client for caption generation. Optional — when nil, the service falls
// back to a direct OpenRouter client (legacy path).
type AIClientFactory interface {
	CreateClient(opts ai.ClientOptions) ai.AIClient
}

// SystemOpener wraps reveal-in-file-manager / open-folder shell-outs so
// tests can intercept them without invoking real OS commands.
type SystemOpener interface {
	Reveal(path string) error
	OpenFolder(path string) error
}

// Deps wires the ExportsService handler.
type Deps struct {
	Repo            Repo
	Executor        ExecutionLookup
	Catalog         CatalogLookup
	AIClientFactory AIClientFactory // optional
	Opener          SystemOpener    // optional; defaults to OS-native
	Logger          *logrus.Logger
}

// Module builds the ExportsService Connect handler.
//
// Required deps: Repo, Executor, Catalog, Logger. AIClientFactory and
// Opener are optional; AIClientFactory missing means the legacy direct
// OpenRouter path is used, and Opener missing means the production
// os/exec-based opener is installed.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("exports_service.Module requires Deps.Logger")
	}
	if d.Repo == nil {
		panic("exports_service.Module requires Deps.Repo")
	}
	if d.Executor == nil {
		panic("exports_service.Module requires Deps.Executor")
	}
	if d.Catalog == nil {
		panic("exports_service.Module requires Deps.Catalog")
	}
	if d.Opener == nil {
		d.Opener = osOpener{}
	}
	path, handler := exportsconnect.NewExportsServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}

var _ exportsconnect.ExportsServiceHandler = (*service)(nil)
