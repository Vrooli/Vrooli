package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/services/readiness"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
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
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.FlowDefinition == nil {
		return nil, fmt.Errorf("flow_definition is required")
	}

	executionID := uuid.New()
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

	// Resolve session profile if provided for authenticated execution
	var storageState json.RawMessage
	var profileBrowserSettings *sessionprofilepersistence.BrowserProfile
	var openTabs []sessionprofilepersistence.TabState
	if sessionProfileID != "" && s.sessionProfileService != nil {
		profile, err := s.sessionProfileService.GetProfile(sessionprofilepersistence.ProfileID(sessionProfileID))
		if err != nil {
			return nil, fmt.Errorf("session profile %s not found: %w", sessionProfileID, err)
		}
		storageState = profile.StorageState
		profileBrowserSettings = profile.BrowserProfile

		// Load open tabs if tab restoration is requested
		if restoreTabs && len(profile.OpenTabs) > 0 {
			openTabs = profile.OpenTabs
		}

		// Update usage tracking to keep profile LRU order current
		if _, err := s.sessionProfileService.Touch(sessionprofilepersistence.ProfileID(sessionProfileID)); err != nil && s.log != nil {
			s.log.WithError(err).WithField("session_profile_id", sessionProfileID).Warn("Failed to update session profile usage timestamp")
		}
	}

	// Extract adhoc workflow's default browser profile and merge with execution override
	// Priority order: execution params > session profile > workflow defaults
	var workflowBrowserProfile *sessionprofilepersistence.BrowserProfile
	if req.FlowDefinition != nil && req.FlowDefinition.Settings != nil && req.FlowDefinition.Settings.BrowserProfile != nil {
		workflowBrowserProfile = sessionprofilepersistence.BrowserProfileFromProto(req.FlowDefinition.Settings.BrowserProfile)
	}
	// First merge workflow defaults with session profile defaults
	baseBrowserProfile := sessionprofilepersistence.MergeBrowserProfiles(workflowBrowserProfile, profileBrowserSettings)
	// Then merge with execution-level overrides (highest priority)
	finalBrowserProfile := sessionprofilepersistence.MergeBrowserProfiles(baseBrowserProfile, execBrowserProfile)

	execIndex := &database.ExecutionIndex{
		ID:          executionID,
		WorkflowID:  workflowID,
		Status:      database.ExecutionStatusPending,
		TriggerType: "api",
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.CreateExecution(ctx, execIndex); err != nil {
		return nil, fmt.Errorf("create execution: %w", err)
	}

	// Use the standard async runner so status polling, stop requests, and result indexing work.
	s.startExecutionRunnerWithOptions(ctx, wf, executionID, store, params, env, artifactCfg, finalBrowserProfile, storageState, opts, projectRoot, startURL, saveSessionProfileID, restoreTabs, openTabs, navigationWaitUntil, continueOnError)

	if req.WaitForCompletion {
		// Mirror executions.go: poll the repo for completion so synchronous
		// callers (capture handler today; any future single-shot adhoc use
		// case) get a response only after the executor has produced its
		// artifacts. Without this branch WaitForCompletion was silently
		// ignored on the adhoc path.
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
				latest, err := s.repo.GetExecution(ctx, executionID)
				if err != nil {
					return nil, err
				}
				if latest.CompletedAt != nil {
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
			}
		}
	}

	// Async return — caller polls the execution ID separately.
	return &basexecution.ExecuteAdhocResponse{
		ExecutionId: executionID.String(),
		Status:      basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING,
		Message:     "Execution started (adhoc). Poll executions API for status.",
	}, nil
}
