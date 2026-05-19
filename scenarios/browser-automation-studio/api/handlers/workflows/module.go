// Package workflows hosts the BAS WorkflowsService Connect-RPC handler.
//
// WorkflowsService owns the workflow lifecycle: CRUD, versioning, validation,
// AI-driven modification, and execution start (both persisted and adhoc).
// Execution state queries and per-execution artifact endpoints live in
// ExecutionsService (Phase 8).
package workflows

import (
	"context"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/api-core/connectx"

	"github.com/vrooli/browser-automation-studio/services/credits"
	workflowservice "github.com/vrooli/browser-automation-studio/services/workflow"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// Catalog is the narrow seam onto workflow.CatalogService for workflow CRUD,
// version queries, and AI modification.
type Catalog interface {
	CreateWorkflow(ctx context.Context, req *basapi.CreateWorkflowRequest) (*basapi.CreateWorkflowResponse, error)
	ListWorkflows(ctx context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error)
	GetWorkflowAPI(ctx context.Context, req *basapi.GetWorkflowRequest) (*basapi.GetWorkflowResponse, error)
	UpdateWorkflow(ctx context.Context, req *basapi.UpdateWorkflowRequest) (*basapi.UpdateWorkflowResponse, error)
	DeleteWorkflow(ctx context.Context, req *basapi.DeleteWorkflowRequest) (*basapi.DeleteWorkflowResponse, error)
	ListWorkflowVersionsAPI(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowVersionList, error)
	GetWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32) (*basapi.WorkflowVersion, error)
	RestoreWorkflowVersionAPI(ctx context.Context, workflowID uuid.UUID, version int32, changeDescription string) (*basapi.RestoreWorkflowVersionResponse, error)
	ModifyWorkflowAPI(ctx context.Context, workflowID uuid.UUID, prompt string, current *basworkflows.WorkflowDefinitionV2) (*basapi.UpdateWorkflowResponse, error)
}

// Executor is the narrow seam onto workflow.ExecutionService for execution
// kick-off operations.
type Executor interface {
	ExecuteWorkflowAPIWithOptions(ctx context.Context, req *basapi.ExecuteWorkflowRequest, opts *workflowservice.ExecuteOptions) (*basapi.ExecuteWorkflowResponse, error)
	ExecuteAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *workflowservice.ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error)
}

// Validator is the narrow seam onto the in-process workflow validator. The
// implementation is responsible for converting its internal Result into the
// proto shape returned by the service.
type Validator interface {
	ValidateDefinition(ctx context.Context, def *basworkflows.WorkflowDefinitionV2) *basapi.WorkflowValidationResult
}

// SeedRunner abstracts the test-genie seed apply/cleanup client.
type SeedRunner interface {
	ApplySeed(ctx context.Context, scenario string, retain bool) (cleanupToken string, seedState map[string]any, err error)
	CleanupSeed(ctx context.Context, scenario, cleanupToken string) error
}

// SeedScheduler abstracts deferred seed-cleanup scheduling used when the
// caller does not wait for execution completion.
type SeedScheduler interface {
	Schedule(executionID string, seedScenario string, cleanupToken string) error
}

// CreditCharger abstracts the per-execution credit charge. The implementation
// must tolerate transient failures and return them without aborting execution
// (the caller logs and continues).
type CreditCharger interface {
	Charge(ctx context.Context, req credits.ChargeRequest) (*credits.ChargeResult, error)
}

// UserIdentityResolver extracts the calling user identity from the request
// context. Tests inject a fixed identity; production uses the entitlement
// middleware's context extractor.
type UserIdentityResolver func(ctx context.Context) string

// Deps wires the workflows handler.
type Deps struct {
	Catalog          Catalog
	Executor         Executor
	Validator        Validator
	SeedRunner       SeedRunner
	SeedScheduler    SeedScheduler
	CreditService    CreditCharger
	UserIdentity     UserIdentityResolver
	Logger           *logrus.Logger
}

// Module builds the WorkflowsService Connect handler.
func Module(d Deps) connectx.ServiceMount {
	if d.Logger == nil {
		panic("workflows.Module requires Deps.Logger")
	}
	if d.Catalog == nil {
		panic("workflows.Module requires Deps.Catalog")
	}
	if d.Executor == nil {
		panic("workflows.Module requires Deps.Executor")
	}
	if d.Validator == nil {
		panic("workflows.Module requires Deps.Validator")
	}
	if d.UserIdentity == nil {
		d.UserIdentity = func(context.Context) string { return "" }
	}
	path, handler := apiconnect.NewWorkflowsServiceHandler(&service{deps: d})
	return connectx.ServiceMount{Path: path, Handler: handler}
}
