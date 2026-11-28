package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	apiidentity "github.com/vrooli/api-core/identity"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/services/readiness"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
)

const (
	adhocWorkflowIndexName       = "adhoc"
	adhocWorkflowIndexFolderPath = "/__adhoc"
)

func (s *WorkflowService) ensureAdhocWorkflowIndex(ctx context.Context) (uuid.UUID, error) {
	existing, err := s.repo.GetWorkflowByName(ctx, adhocWorkflowIndexName, adhocWorkflowIndexFolderPath)
	if err == nil && existing != nil && existing.ID != uuid.Nil {
		return existing.ID, nil
	}
	if err != nil && !errors.Is(err, database.ErrNotFound) {
		return uuid.Nil, err
	}

	index := &database.WorkflowIndex{
		ID:         uuid.New(),
		ProjectID:  nil,
		Name:       adhocWorkflowIndexName,
		FolderPath: adhocWorkflowIndexFolderPath,
		FilePath:   "",
		Version:    1,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if createErr := s.repo.CreateWorkflow(ctx, index); createErr != nil {
		// Possible concurrent creation or unique constraint; attempt to read it back.
		latest, getErr := s.repo.GetWorkflowByName(ctx, adhocWorkflowIndexName, adhocWorkflowIndexFolderPath)
		if getErr == nil && latest != nil && latest.ID != uuid.Nil {
			return latest.ID, nil
		}
		if getErr != nil {
			return uuid.Nil, getErr
		}
		return uuid.Nil, createErr
	}
	return index.ID, nil
}

// ExecuteAdhocWorkflowAPI executes a provided workflow definition without persisting it as a workflow index.
// Artifacts are still recorded to disk using the standard recorder.
func (s *WorkflowService) ExecuteAdhocWorkflowAPI(ctx context.Context, req *basexecution.ExecuteAdhocRequest) (*basexecution.ExecuteAdhocResponse, error) {
	return s.ExecuteAdhocWorkflowAPIWithOptions(ctx, req, nil)
}

// ExecuteAdhocWorkflowAPIWithOptions executes a provided workflow definition without persisting it as a workflow index.
// Artifacts are still recorded to disk using the standard recorder.
func (s *WorkflowService) ExecuteAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
	return s.executeAdhocWorkflowAPIWithOptions(ctx, req, opts, false)
}

// ExecuteSystemCaptureAdhocWorkflowAPIWithOptions is the accountless entry
// reserved for CaptureService after its exact service principal has been
// authenticated. The ordinary adhoc route never receives this admission flag.
func (s *WorkflowService) ExecuteSystemCaptureAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *ExecuteOptions) (*basexecution.ExecuteAdhocResponse, error) {
	principal, ok := apiidentity.PrincipalFromContext(ctx)
	if !ok || !principal.Verified || principal.Kind != apiidentity.ActorService ||
		principal.Subject != "experience-manager-bas-capture" || principal.Issuer != "scenario-authenticator" ||
		principal.Audience != "scenario-authenticator:browser-automation-studio" ||
		len(principal.Scopes) != 1 || principal.Scopes[0] != "browser-automation-studio:capture" {
		return nil, errors.New("verified BAS capture service principal is required")
	}
	return s.executeAdhocWorkflowAPIWithOptions(ctx, req, opts, true)
}

func (s *WorkflowService) executeAdhocWorkflowAPIWithOptions(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *ExecuteOptions, systemCapture bool) (*basexecution.ExecuteAdhocResponse, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.FlowDefinition == nil {
		return nil, fmt.Errorf("flow_definition is required")
	}

	executionID := uuid.New()
	if opts != nil && opts.ExecutionID != nil {
		executionID = *opts.ExecutionID
	}
	workflowID, err := s.ensureAdhocWorkflowIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("ensure adhoc workflow index: %w", err)
	}

	now := time.Now().UTC()
	wf := &basapi.WorkflowSummary{
		Id:                    workflowID.String(),
		ProjectId:             "",
		Name:                  strings.TrimSpace(req.GetMetadata().GetName()),
		FolderPath:            "/",
		Description:           strings.TrimSpace(req.GetMetadata().GetDescription()),
		Version:               1,
		CreatedAt:             autocontracts.TimeToTimestamp(now),
		UpdatedAt:             autocontracts.TimeToTimestamp(now),
		FlowDefinition:        req.FlowDefinition,
		LastChangeSource:      basbase.ChangeSource_CHANGE_SOURCE_MANUAL,
		LastChangeDescription: "Adhoc execution",
	}
	if strings.TrimSpace(wf.Name) == "" {
		wf.Name = "adhoc"
	}

	store, params, env, artifactCfg, execBrowserProfile, projectRoot, startURL, sessionProfileID, saveSessionProfileID, restoreTabs, navigationWaitUntil, continueOnError := executionParametersToMaps(req.Parameters)
	if err := validateProjectRoot(projectRoot); err != nil {
		return nil, err
	}
	if err := validateSeedRequirements(req.FlowDefinition, store, params, env); err != nil {
		return nil, err
	}
	if err := s.validateAdhocSubflows(ctx, req.FlowDefinition, workflowID, projectRoot); err != nil {
		return nil, err
	}

	// Settle the opening navigation on the target scenario's declared surfaces.
	// This is the path bas/ cases actually take — Workflow Health runs every
	// case through ExecuteAdhocWorkflow — so without it the declared contract
	// would be resolved for captures and ignored for the suite it exists to fix.
	// A caller that already injected its own wait (Capture does) reports
	// explicit-wait here and is left untouched.
	settle := readiness.Settle(ctx, s.readinessResolver, req.FlowDefinition)
	if s.log != nil {
		entry := s.log.WithFields(logrus.Fields{
			"execution_id":       executionID,
			"readiness_strategy": settle.Strategy,
		})
		if settle.Strategy == readiness.StrategyDeclaredSurface {
			entry.WithFields(logrus.Fields{
				"profile_version":   settle.ProfileVersion,
				"route":             settle.Route,
				"required_surfaces": settle.RequiredSurfaceIDs,
			}).Info("Adhoc execution settles on declared readiness surfaces")
		} else if settle.FallbackReason != "" {
			// Info, not Debug: a silent fallback is indistinguishable from a
			// fast pass, which is the failure mode this whole contract exists
			// to remove. Say which rung was used and why.
			entry.WithField("fallback_reason", settle.FallbackReason).
				Info("Adhoc execution uses generic navigation readiness")
		}
	}
	wf.FlowDefinition = req.FlowDefinition

	// Saved and ad-hoc executions resolve persisted state and browser-profile
	// precedence through one owner so authenticated sessions behave identically.
	sessionProfile, err := s.resolveExecutionSessionProfile(req.FlowDefinition, sessionProfileID, saveSessionProfileID, restoreTabs, execBrowserProfile)
	if err != nil {
		return nil, fmt.Errorf("session profile %s not found: %w", sessionProfileID, err)
	}
	releaseAdmission, releaseSlot, err := s.beginExecutionAdmission()
	if err != nil {
		return nil, err
	}
	defer releaseAdmission()
	slotTransferred := false
	defer func() {
		if !slotTransferred {
			releaseSlot()
		}
	}()
	var hold *monetization.QuotaHold
	if !systemCapture {
		hold, err = s.reserveExecution(ctx)
		if err != nil {
			return nil, err
		}
	}

	execIndex := &database.ExecutionIndex{
		ID:          executionID,
		WorkflowID:  workflowID,
		Status:      database.ExecutionStatusPending,
		TriggerType: "api",
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.createExecution(ctx, execIndex, &basexecution.Execution{
		WorkflowVersion: wf.Version,
		TriggerType:     basbase.TriggerType_TRIGGER_TYPE_API,
		Parameters:      req.Parameters,
	}); err != nil {
		s.releaseExecutionHold(ctx, hold)
		return nil, fmt.Errorf("create execution: %w", err)
	}
	if err := s.commitExecutionHold(ctx, hold, executionID); err != nil {
		return nil, fmt.Errorf("commit workflow execution quota: %w", err)
	}

	// Use the standard async runner so status polling, stop requests, and result indexing work.
	completion := s.startExecutionRunnerWithOptions(ctx, wf, executionID, store, params, env, artifactCfg, sessionProfile.browserProfile, sessionProfile.storageState, sessionProfile.sessionProfileVersion, opts, projectRoot, startURL, saveSessionProfileID, restoreTabs, sessionProfile.openTabs, navigationWaitUntil, continueOnError, releaseSlot)
	slotTransferred = true
	releaseAdmission()

	if req.WaitForCompletion {
		latest, err := s.waitForExecutionCompletion(ctx, executionID, completion)
		if err != nil {
			return nil, err
		}
		resp := &basexecution.ExecuteAdhocResponse{
			ExecutionId: latest.ID.String(),
			Status:      enums.StringToExecutionStatus(latest.Status),
			CompletedAt: autocontracts.TimePtrToTimestamp(latest.CompletedAt),
		}
		if strings.TrimSpace(latest.ErrorMessage) != "" {
			msg := latest.ErrorMessage
			resp.Error = &msg
		}
		return resp, nil
	}

	// Async return — caller polls the execution ID separately.
	return &basexecution.ExecuteAdhocResponse{
		ExecutionId: executionID.String(),
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING,
		Message:     "Execution started (adhoc). Poll executions API for status.",
	}, nil
}
