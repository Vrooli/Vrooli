// Package toolexecution implements the Tool Execution Protocol for scenario-to-cloud.
package toolexecution

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"scenario-to-cloud/deployment"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/toolregistry"
	"scenario-to-cloud/vps"
	"scenario-to-cloud/vps/preflight"
)

// ToolExecutor executes tools from the Tool Execution Protocol.
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error)
}

// Repository is the interface for database operations.
type Repository interface {
	toolregistry.DeploymentRepository
	CreateDeployment(ctx context.Context, d *domain.Deployment) error
	UpdateDeployment(ctx context.Context, d *domain.Deployment) error
	GetDeploymentByHostAndScenario(ctx context.Context, host, scenarioID string) (*domain.Deployment, error)
	UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, errorMessage, errorStep *string) error
	StartDeploymentRun(ctx context.Context, id, runID string) error
	StartDeploymentStart(ctx context.Context, id, runID string) error
	UpdateDeploymentInspectResult(ctx context.Context, id string, result json.RawMessage) error
	AppendHistoryEvent(ctx context.Context, id string, event domain.HistoryEvent) error
}

// ServerExecutorConfig holds dependencies for creating a ServerExecutor.
type ServerExecutorConfig struct {
	Repo         Repository
	Resolver     toolregistry.DeploymentResolver
	Orchestrator *deployment.Orchestrator
	SSHRunner    ssh.Runner
	DNSService   dns.Service
	Logger       func(msg string, fields map[string]interface{})
}

// ServerExecutor implements ToolExecutor using the Server's dependencies.
type ServerExecutor struct {
	repo         Repository
	resolver     toolregistry.DeploymentResolver
	orchestrator *deployment.Orchestrator
	sshRunner    ssh.Runner
	dnsService   dns.Service
	logger       func(msg string, fields map[string]interface{})
}

// NewServerExecutor creates a new ServerExecutor.
func NewServerExecutor(cfg ServerExecutorConfig) *ServerExecutor {
	logger := cfg.Logger
	if logger == nil {
		logger = func(msg string, fields map[string]interface{}) {}
	}
	return &ServerExecutor{
		repo:         cfg.Repo,
		resolver:     cfg.Resolver,
		orchestrator: cfg.Orchestrator,
		sshRunner:    cfg.SSHRunner,
		dnsService:   cfg.DNSService,
		logger:       logger,
	}
}

// Execute dispatches tool execution to the appropriate handler method.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	switch toolName {
	case "list_deployments":
		return e.listDeployments(ctx, args)
	case "create_deployment":
		return e.createDeployment(ctx, args)
	case "execute_deployment":
		return e.executeDeployment(ctx, args)
	case "stop_deployment":
		return e.stopDeployment(ctx, args)
	case "start_deployment":
		return e.startDeployment(ctx, args)
	case "check_deployment_status":
		return e.checkDeploymentStatus(ctx, args)
	case "inspect_deployment":
		return e.inspectDeployment(ctx, args)
	case "get_deployment_logs":
		return e.getDeploymentLogs(ctx, args)
	case "get_live_state":
		return e.getLiveState(ctx, args)
	case "validate_manifest":
		return e.validateManifest(ctx, args)
	case "run_preflight":
		return e.runPreflight(ctx, args)
	default:
		return ErrorResult(fmt.Sprintf("unknown tool: %s", toolName), CodeUnknownTool), nil
	}
}

// -----------------------------------------------------------------------------
// Tool Implementations
// -----------------------------------------------------------------------------

// listDeployments returns all deployment records.
// Arguments: status (optional), scenario_id (optional), limit (optional)
func (e *ServerExecutor) listDeployments(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	filter := domain.ListFilter{}

	if status, ok := args["status"].(string); ok && status != "" {
		st := domain.DeploymentStatus(status)
		filter.Status = &st
	}
	if scenarioID, ok := args["scenario_id"].(string); ok && scenarioID != "" {
		filter.ScenarioID = &scenarioID
	}
	if limit, ok := args["limit"].(float64); ok {
		filter.Limit = int(limit)
	}

	deployments, err := e.repo.ListDeployments(ctx, filter)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to list deployments: %v", err), CodeInternalError), nil
	}

	// Convert to summaries for list view
	summaries := make([]domain.DeploymentSummary, len(deployments))
	for i, d := range deployments {
		summary := domain.DeploymentSummary{
			ID:              d.ID,
			Name:            d.Name,
			ScenarioID:      d.ScenarioID,
			Status:          d.Status,
			ErrorMessage:    d.ErrorMessage,
			ProgressStep:    d.ProgressStep,
			ProgressPercent: d.ProgressPercent,
			CreatedAt:       d.CreatedAt,
			LastDeployedAt:  d.LastDeployedAt,
		}
		// Extract domain and host from manifest
		if d.Manifest != nil {
			var m domain.CloudManifest
			if err := json.Unmarshal(d.Manifest, &m); err == nil {
				summary.Domain = m.Edge.Domain
				if m.Target.VPS != nil {
					summary.Host = m.Target.VPS.Host
				}
			}
		}
		summaries[i] = summary
	}

	return SuccessResult(map[string]interface{}{
		"deployments": summaries,
		"count":       len(summaries),
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}), nil
}

// createDeployment creates a new deployment record.
// Arguments: manifest (required), name (optional)
func (e *ServerExecutor) createDeployment(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	manifestArg, ok := args["manifest"]
	if !ok {
		return ErrorResult("manifest is required", CodeInvalidArgs), nil
	}

	var manifestJSON []byte
	switch m := manifestArg.(type) {
	case string:
		manifestJSON = []byte(m)
	case map[string]interface{}:
		var err error
		manifestJSON, err = json.Marshal(m)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to marshal manifest: %v", err), CodeInvalidArgs), nil
		}
	default:
		return ErrorResult("manifest must be a JSON object or string", CodeInvalidArgs), nil
	}

	var cloudManifest domain.CloudManifest
	if err := json.Unmarshal(manifestJSON, &cloudManifest); err != nil {
		return ErrorResult(fmt.Sprintf("invalid manifest JSON: %v", err), CodeInvalidArgs), nil
	}

	normalized, issues := manifest.ValidateAndNormalize(cloudManifest)
	if manifest.HasBlockingIssues(issues) {
		return ErrorResult("manifest has blocking validation issues", CodeValidation), nil
	}

	// Re-marshal the normalized manifest
	normalizedJSON, err := json.Marshal(normalized)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to marshal normalized manifest: %v", err), CodeInternalError), nil
	}

	// Check for existing deployment with same host+scenario
	var host string
	if normalized.Target.VPS != nil {
		host = normalized.Target.VPS.Host
	}
	existing, err := e.repo.GetDeploymentByHostAndScenario(ctx, host, normalized.Scenario.ID)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to check existing deployment: %v", err), CodeInternalError), nil
	}

	if existing != nil {
		// Update existing deployment
		existing.Manifest = normalizedJSON
		existing.Status = domain.StatusPending
		existing.ErrorMessage = nil
		existing.ErrorStep = nil
		existing.UpdatedAt = time.Now()

		if name, ok := args["name"].(string); ok && name != "" {
			existing.Name = name
		}

		if err := e.repo.UpdateDeployment(ctx, existing); err != nil {
			return ErrorResult(fmt.Sprintf("failed to update deployment: %v", err), CodeInternalError), nil
		}

		return SuccessResult(map[string]interface{}{
			"deployment": existing,
			"updated":    true,
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		}), nil
	}

	// Generate name if not provided
	name, _ := args["name"].(string)
	if name == "" {
		name = fmt.Sprintf("%s @ %s", normalized.Scenario.ID, normalized.Edge.Domain)
	}

	// Create new deployment
	now := time.Now()
	dep := &domain.Deployment{
		ID:         uuid.New().String(),
		Name:       name,
		ScenarioID: normalized.Scenario.ID,
		Status:     domain.StatusPending,
		Manifest:   normalizedJSON,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := e.repo.CreateDeployment(ctx, dep); err != nil {
		return ErrorResult(fmt.Sprintf("failed to create deployment: %v", err), CodeInternalError), nil
	}

	e.appendHistoryEvent(ctx, dep.ID, domain.HistoryEvent{
		Type:      domain.EventDeploymentCreated,
		Timestamp: time.Now().UTC(),
		Message:   "Deployment created via tool execution",
		Success:   boolPtr(true),
	})

	return SuccessResult(map[string]interface{}{
		"deployment": dep,
		"created":    true,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}), nil
}

// executeDeployment starts the deployment pipeline (async).
// Arguments: deployment (required - name or ID), run_preflight (optional), force_bundle_build (optional)
func (e *ServerExecutor) executeDeployment(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	depID, err := e.resolveDeployment(ctx, args)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	dep, err := e.repo.GetDeployment(ctx, depID)
	if err != nil || dep == nil {
		return ErrorResult("deployment not found", CodeNotFound), nil
	}

	// Parse manifest
	var m domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &m); err != nil {
		return ErrorResult(fmt.Sprintf("failed to parse manifest: %v", err), CodeInternalError), nil
	}

	normalized, issues := manifest.ValidateAndNormalize(m)
	if manifest.HasBlockingIssues(issues) {
		return ErrorResult("manifest has blocking validation issues", CodeValidation), nil
	}

	// Generate run ID
	runID := uuid.New().String()

	// Atomic status transition
	if err := e.repo.StartDeploymentRun(ctx, depID, runID); err != nil {
		return ErrorResult("deployment is already in progress or not found", CodeConflict), nil
	}

	e.logger("deployment run started via tool execution", map[string]interface{}{
		"deployment_id": depID,
		"run_id":        runID,
	})

	// Parse options
	runPreflight, _ := args["run_preflight"].(bool)
	forceBundleBuild, _ := args["force_bundle_build"].(bool)
	providedSecrets, _ := args["provided_secrets"].(map[string]interface{})

	// Convert provided_secrets to map[string]string
	secrets := make(map[string]string)
	for k, v := range providedSecrets {
		if s, ok := v.(string); ok {
			secrets[k] = s
		}
	}

	// Start deployment in background
	options := deployment.ExecuteOptions{
		RunPreflight:     runPreflight,
		ForceBundleBuild: forceBundleBuild,
	}
	go e.orchestrator.RunPipeline(depID, runID, normalized, dep.BundlePath, secrets, options)

	return AsyncResult(map[string]interface{}{
		"deployment_id": depID,
		"message":       "Deployment started. Use check_deployment_status to monitor progress.",
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}, runID), nil
}

// stopDeployment stops a deployment on the VPS.
// Arguments: deployment (required - name or ID)
func (e *ServerExecutor) stopDeployment(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	depID, err := e.resolveDeployment(ctx, args)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	dep, m, err := e.getDeploymentWithManifest(ctx, depID)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	if m.Target.VPS == nil {
		return ErrorResult("deployment does not have a VPS target", CodeInvalidArgs), nil
	}

	result := e.stopDeploymentOnVPS(ctx, m)

	if result.OK {
		if err := e.repo.UpdateDeploymentStatus(ctx, depID, domain.StatusStopped, nil, nil); err != nil {
			e.logger("failed to update status after stop", map[string]interface{}{"error": err.Error()})
		}
	}

	e.appendHistoryEvent(ctx, dep.ID, domain.HistoryEvent{
		Type:      domain.EventStopped,
		Timestamp: time.Now().UTC(),
		Message:   "Deployment stop requested via tool execution",
		Details:   result.Error,
		Success:   boolPtr(result.OK),
	})

	return SuccessResult(map[string]interface{}{
		"success":   result.OK,
		"error":     result.Error,
		"timestamp": result.Timestamp,
	}), nil
}

// startDeployment starts/resumes a stopped deployment (async).
// Arguments: deployment (required - name or ID)
func (e *ServerExecutor) startDeployment(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	depID, err := e.resolveDeployment(ctx, args)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	dep, m, err := e.getDeploymentWithManifest(ctx, depID)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	// Validate status
	if dep.Status != domain.StatusStopped && dep.Status != domain.StatusSetupComplete {
		return ErrorResult(
			fmt.Sprintf("cannot start deployment with status '%s'. Must be 'stopped' or 'setup_complete'", dep.Status),
			CodeConflict,
		), nil
	}

	// Generate run ID
	runID := uuid.New().String()

	// Atomic status transition
	if err := e.repo.StartDeploymentStart(ctx, depID, runID); err != nil {
		return ErrorResult("cannot start deployment", CodeConflict), nil
	}

	e.logger("deployment start initiated via tool execution", map[string]interface{}{
		"deployment_id": depID,
		"run_id":        runID,
	})

	// Parse provided secrets
	providedSecrets, _ := args["provided_secrets"].(map[string]interface{})
	secrets := make(map[string]string)
	for k, v := range providedSecrets {
		if s, ok := v.(string); ok {
			secrets[k] = s
		}
	}

	// Start in background
	go e.orchestrator.RunStartPipeline(depID, runID, m, secrets)

	return AsyncResult(map[string]interface{}{
		"deployment_id": depID,
		"message":       "Start initiated. Use check_deployment_status to monitor progress.",
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	}, runID), nil
}

// checkDeploymentStatus returns the current status of a deployment.
// Arguments: deployment (required - name or ID)
func (e *ServerExecutor) checkDeploymentStatus(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	depID, err := e.resolveDeployment(ctx, args)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	dep, err := e.repo.GetDeployment(ctx, depID)
	if err != nil || dep == nil {
		return ErrorResult("deployment not found", CodeNotFound), nil
	}

	return SuccessResult(map[string]interface{}{
		"deployment_id":    dep.ID,
		"name":             dep.Name,
		"status":           dep.Status,
		"scenario_id":      dep.ScenarioID,
		"run_id":           dep.RunID,
		"error_message":    dep.ErrorMessage,
		"error_step":       dep.ErrorStep,
		"progress_step":    dep.ProgressStep,
		"progress_percent": dep.ProgressPercent,
		"created_at":       dep.CreatedAt,
		"updated_at":       dep.UpdatedAt,
		"last_deployed_at": dep.LastDeployedAt,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	}), nil
}

// inspectDeployment fetches status and logs from the deployed VPS.
// Arguments: deployment (required - name or ID)
func (e *ServerExecutor) inspectDeployment(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	depID, err := e.resolveDeployment(ctx, args)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	dep, m, err := e.getDeploymentWithManifest(ctx, depID)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	if m.Target.VPS == nil {
		return ErrorResult("deployment does not have a VPS target", CodeInvalidArgs), nil
	}

	inspectCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	opts := vps.InspectOptions{TailLines: 200}
	result := vps.RunInspect(inspectCtx, m, opts, e.sshRunner)

	e.appendHistoryEvent(ctx, dep.ID, domain.HistoryEvent{
		Type:      domain.EventInspection,
		Timestamp: time.Now().UTC(),
		Message:   "Inspection completed via tool execution",
		Details:   result.Error,
		Success:   boolPtr(result.OK),
	})

	// Store inspect result
	if resultJSON, err := json.Marshal(result); err == nil {
		if err := e.repo.UpdateDeploymentInspectResult(ctx, depID, resultJSON); err != nil {
			e.logger("failed to save inspect result", map[string]interface{}{"error": err.Error()})
		}
	}

	return SuccessResult(map[string]interface{}{
		"result":    result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

// getDeploymentLogs retrieves logs from the VPS.
// Arguments: deployment (required), source (optional), level (optional), tail (optional)
func (e *ServerExecutor) getDeploymentLogs(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	depID, err := e.resolveDeployment(ctx, args)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	_, m, err := e.getDeploymentWithManifest(ctx, depID)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	if m.Target.VPS == nil {
		return ErrorResult("deployment does not have a VPS target", CodeInvalidArgs), nil
	}

	// Parse options
	source, _ := args["source"].(string)
	if source == "" {
		source = "all"
	}
	tail := 200
	if t, ok := args["tail"].(float64); ok && t > 0 && t <= 2000 {
		tail = int(t)
	}

	logsCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cfg := ssh.ConfigFromManifest(m)
	workdir := m.Target.VPS.Workdir
	scenarioID := m.Scenario.ID

	// Simple log fetch - get scenario logs
	cmd := ssh.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario logs %s --tail %d", ssh.QuoteSingle(scenarioID), tail))
	result, err := e.sshRunner.Run(logsCtx, cfg, cmd)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to fetch logs: %v", err), CodeInternalError), nil
	}

	return SuccessResult(map[string]interface{}{
		"logs":      result.Stdout,
		"source":    source,
		"tail":      tail,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

// getLiveState fetches comprehensive live state from the VPS.
// Arguments: deployment (required - name or ID)
func (e *ServerExecutor) getLiveState(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	depID, err := e.resolveDeployment(ctx, args)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	_, m, err := e.getDeploymentWithManifest(ctx, depID)
	if err != nil {
		return ErrorResult(err.Error(), CodeNotFound), nil
	}

	if m.Target.VPS == nil {
		return ErrorResult("deployment does not have a VPS target", CodeInvalidArgs), nil
	}

	stateCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	result := vps.RunLiveStateInspection(stateCtx, m, e.sshRunner)

	return SuccessResult(map[string]interface{}{
		"result":    result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

// validateManifest validates a deployment manifest.
// Arguments: manifest (required)
func (e *ServerExecutor) validateManifest(_ context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	manifestArg, ok := args["manifest"]
	if !ok {
		return ErrorResult("manifest is required", CodeInvalidArgs), nil
	}

	var manifestJSON []byte
	switch m := manifestArg.(type) {
	case string:
		manifestJSON = []byte(m)
	case map[string]interface{}:
		var err error
		manifestJSON, err = json.Marshal(m)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to marshal manifest: %v", err), CodeInvalidArgs), nil
		}
	default:
		return ErrorResult("manifest must be a JSON object or string", CodeInvalidArgs), nil
	}

	var cloudManifest domain.CloudManifest
	if err := json.Unmarshal(manifestJSON, &cloudManifest); err != nil {
		return ErrorResult(fmt.Sprintf("invalid manifest JSON: %v", err), CodeInvalidArgs), nil
	}

	normalized, issues := manifest.ValidateAndNormalize(cloudManifest)

	return SuccessResult(map[string]interface{}{
		"valid":     !manifest.HasBlockingIssues(issues),
		"issues":    issues,
		"manifest":  normalized,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

// runPreflight runs preflight checks for a manifest.
// Arguments: manifest (required)
func (e *ServerExecutor) runPreflight(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	manifestArg, ok := args["manifest"]
	if !ok {
		return ErrorResult("manifest is required", CodeInvalidArgs), nil
	}

	var manifestJSON []byte
	switch m := manifestArg.(type) {
	case string:
		manifestJSON = []byte(m)
	case map[string]interface{}:
		var err error
		manifestJSON, err = json.Marshal(m)
		if err != nil {
			return ErrorResult(fmt.Sprintf("failed to marshal manifest: %v", err), CodeInvalidArgs), nil
		}
	default:
		return ErrorResult("manifest must be a JSON object or string", CodeInvalidArgs), nil
	}

	var cloudManifest domain.CloudManifest
	if err := json.Unmarshal(manifestJSON, &cloudManifest); err != nil {
		return ErrorResult(fmt.Sprintf("invalid manifest JSON: %v", err), CodeInvalidArgs), nil
	}

	normalized, issues := manifest.ValidateAndNormalize(cloudManifest)
	if manifest.HasBlockingIssues(issues) {
		return ErrorResult("manifest has blocking validation issues", CodeValidation), nil
	}

	if normalized.Target.VPS == nil {
		return ErrorResult("manifest does not have a VPS target", CodeInvalidArgs), nil
	}

	preflightCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	result := preflight.Run(preflightCtx, normalized, e.dnsService, e.sshRunner, preflight.RunOptions{})

	return SuccessResult(map[string]interface{}{
		"result":    result,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}), nil
}

// -----------------------------------------------------------------------------
// Helper Methods
// -----------------------------------------------------------------------------

// resolveDeployment resolves a deployment identifier (name or ID) to a UUID.
func (e *ServerExecutor) resolveDeployment(ctx context.Context, args map[string]interface{}) (string, error) {
	// Try different parameter names for flexibility
	var identifier string
	if id, ok := args["deployment"].(string); ok && id != "" {
		identifier = id
	} else if id, ok := args["deployment_id"].(string); ok && id != "" {
		identifier = id
	} else if id, ok := args["id"].(string); ok && id != "" {
		identifier = id
	} else if name, ok := args["name"].(string); ok && name != "" {
		identifier = name
	}

	if identifier == "" {
		return "", fmt.Errorf("deployment identifier is required (use 'deployment', 'deployment_id', 'id', or 'name')")
	}

	return e.resolver.Resolve(ctx, identifier)
}

// getDeploymentWithManifest fetches a deployment and parses its manifest.
func (e *ServerExecutor) getDeploymentWithManifest(ctx context.Context, id string) (*domain.Deployment, domain.CloudManifest, error) {
	dep, err := e.repo.GetDeployment(ctx, id)
	if err != nil {
		return nil, domain.CloudManifest{}, fmt.Errorf("failed to get deployment: %w", err)
	}
	if dep == nil {
		return nil, domain.CloudManifest{}, fmt.Errorf("deployment not found")
	}

	var m domain.CloudManifest
	if err := json.Unmarshal(dep.Manifest, &m); err != nil {
		return nil, domain.CloudManifest{}, fmt.Errorf("failed to parse manifest: %w", err)
	}

	normalized, _ := manifest.ValidateAndNormalize(m)
	return dep, normalized, nil
}

// stopDeploymentOnVPS stops a deployment on the remote VPS.
func (e *ServerExecutor) stopDeploymentOnVPS(ctx context.Context, m domain.CloudManifest) domain.VPSDeployResult {
	cfg := ssh.ConfigFromManifest(m)
	workdir := m.Target.VPS.Workdir

	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := ssh.VrooliCommand(workdir, fmt.Sprintf("vrooli scenario stop %s", ssh.QuoteSingle(m.Scenario.ID)))
	_, err := e.sshRunner.Run(stopCtx, cfg, cmd)
	if err != nil {
		return domain.VPSDeployResult{OK: false, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	return domain.VPSDeployResult{OK: true, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

// appendHistoryEvent persists a history event.
func (e *ServerExecutor) appendHistoryEvent(ctx context.Context, deploymentID string, event domain.HistoryEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if err := e.repo.AppendHistoryEvent(ctx, deploymentID, event); err != nil {
		e.logger("failed to append history event", map[string]interface{}{
			"deployment_id": deploymentID,
			"type":          event.Type,
			"error":         err.Error(),
		})
	}
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}
