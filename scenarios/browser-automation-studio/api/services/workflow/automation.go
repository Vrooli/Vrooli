package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// executeWithAutomationEngine is the sole execution path; the legacy
// browserless.Client is quarantined for parity tests and is not invoked here.
func (s *WorkflowService) executeWithAutomationEngine(ctx context.Context, execution *database.Execution, workflow *database.Workflow, selection autoengine.SelectionConfig, eventSink autoevents.Sink) error {
	if s == nil {
		return errors.New("workflow service not configured")
	}

	engineName := selection.Resolve("")

	compiler := s.planCompiler
	if compiler == nil {
		compiler = autoexecutor.PlanCompilerForEngine(engineName)
	}

	// Convert database.Workflow (index) to basapi.WorkflowSummary (proto)
	// The workflow may have FlowDefinition from adhoc store or from disk
	workflowProto, err := s.workflowIndexToProto(ctx, workflow)
	if err != nil {
		return fmt.Errorf("convert workflow to proto: %w", err)
	}

	plan, _, err := autoexecutor.BuildContractsPlanWithCompiler(ctx, execution.ID, workflowProto, compiler)
	if err != nil {
		return err
	}

	if s.executor == nil {
		s.executor = autoexecutor.NewSimpleExecutor(nil)
	}
	if s.engineFactory == nil {
		factory, engErr := autoengine.DefaultFactory(s.log)
		if engErr != nil {
			return engErr
		}
		s.engineFactory = factory
	}
	if s.artifactRecorder == nil {
		s.artifactRecorder = autorecorder.NewDBRecorder(s.repo, nil, s.log)
	}

	if eventSink == nil {
		eventSink = s.newEventSink()
	}

	req := autoexecutor.Request{
		Plan:              plan,
		EngineName:        engineName,
		EngineFactory:     s.engineFactory,
		Recorder:          s.artifactRecorder,
		EventSink:         eventSink,
		HeartbeatInterval: config.Load().Execution.HeartbeatInterval,
		WorkflowResolver:  &executorWorkflowResolver{repo: s.repo, service: s},
		PlanCompiler:      compiler,
		MaxSubflowDepth:   config.Load().Execution.MaxSubflowDepth,
	}

	// Check for namespace-aware parameters stored by ExecuteAdhocWorkflowWithParams.
	// If present, populate the executor request with namespace fields.
	if nsParams := getAndClearExecutionParams(execution.ID); nsParams != nil {
		req.ProjectRoot = nsParams.ProjectRoot
		req.InitialParams = nsParams.InitialParams
		req.InitialStore = nsParams.InitialStore
		req.Env = nsParams.Env
	}

	return s.executor.Execute(ctx, req)
}

// executeResumedWithAutomationEngine executes a workflow from a specific step index.
// This is used when resuming an interrupted execution.
func (s *WorkflowService) executeResumedWithAutomationEngine(ctx context.Context, execution *database.Execution, workflow *database.Workflow, selection autoengine.SelectionConfig, eventSink autoevents.Sink, startFromStepIndex int, initialVars map[string]any, resumedFromID uuid.UUID) error {
	if s == nil {
		return errors.New("workflow service not configured")
	}

	engineName := selection.Resolve("")

	compiler := s.planCompiler
	if compiler == nil {
		compiler = autoexecutor.PlanCompilerForEngine(engineName)
	}

	// Convert database.Workflow (index) to basapi.WorkflowSummary (proto)
	workflowProto, err := s.workflowIndexToProto(ctx, workflow)
	if err != nil {
		return fmt.Errorf("convert workflow to proto: %w", err)
	}

	plan, _, err := autoexecutor.BuildContractsPlanWithCompiler(ctx, execution.ID, workflowProto, compiler)
	if err != nil {
		return err
	}

	if s.executor == nil {
		s.executor = autoexecutor.NewSimpleExecutor(nil)
	}
	if s.engineFactory == nil {
		factory, engErr := autoengine.DefaultFactory(s.log)
		if engErr != nil {
			return engErr
		}
		s.engineFactory = factory
	}
	if s.artifactRecorder == nil {
		s.artifactRecorder = autorecorder.NewDBRecorder(s.repo, nil, s.log)
	}

	if eventSink == nil {
		eventSink = s.newEventSink()
	}

	req := autoexecutor.Request{
		Plan:               plan,
		EngineName:         engineName,
		EngineFactory:      s.engineFactory,
		Recorder:           s.artifactRecorder,
		EventSink:          eventSink,
		HeartbeatInterval:  config.Load().Execution.HeartbeatInterval,
		WorkflowResolver:   &executorWorkflowResolver{repo: s.repo, service: s},
		PlanCompiler:       compiler,
		MaxSubflowDepth:    config.Load().Execution.MaxSubflowDepth,
		StartFromStepIndex: startFromStepIndex,
		InitialVariables:   initialVars,
		ResumedFromID:      &resumedFromID,
	}
	return s.executor.Execute(ctx, req)
}

type executorWorkflowResolver struct {
	repo    database.Repository
	service *WorkflowService
}

func (r *executorWorkflowResolver) GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*basapi.WorkflowSummary, error) {
	workflow, err := r.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	return r.service.workflowIndexToProto(ctx, workflow)
}

func (r *executorWorkflowResolver) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*basapi.WorkflowSummary, error) {
	if version <= 0 {
		return r.GetWorkflow(ctx, workflowID)
	}

	workflow, err := r.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	// For file-based versioning, load the specific version from disk
	fileVersion, err := r.service.getFileVersion(workflow, version)
	if err != nil {
		// Fall back to current version if specific version not found
		return r.GetWorkflow(ctx, workflowID)
	}

	// Build proto from file version
	flowDef, err := flowDefinitionToProto(fileVersion.FlowDefinition)
	if err != nil {
		return nil, fmt.Errorf("convert flow definition to proto: %w", err)
	}

	pb := &basapi.WorkflowSummary{
		Id:             workflow.ID.String(),
		Name:           workflow.Name,
		FolderPath:     workflow.FolderPath,
		Version:        int32(fileVersion.Version),
		CreatedAt:      timestamppb.New(fileVersion.CreatedAt),
		UpdatedAt:      timestamppb.New(workflow.UpdatedAt),
		FlowDefinition: flowDef,
	}
	if workflow.ProjectID != nil {
		pb.ProjectId = workflow.ProjectID.String()
	}
	return pb, nil
}

func (r *executorWorkflowResolver) GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string) (*basapi.WorkflowSummary, error) {
	callingWorkflow, err := r.repo.GetWorkflow(ctx, callingWorkflowID)
	if err != nil {
		return nil, err
	}
	if callingWorkflow.ProjectID == nil {
		return nil, fmt.Errorf("cannot resolve workflowPath=%q: calling workflow has no project_id", workflowPath)
	}

	project, err := r.repo.GetProject(ctx, *callingWorkflow.ProjectID)
	if err != nil {
		return nil, err
	}

	normalized, ok := normalizeWorkflowPath(workflowPath)
	if !ok {
		return nil, fmt.Errorf("invalid workflowPath=%q", workflowPath)
	}

	abs := filepath.Clean(filepath.Join(project.FolderPath, filepath.FromSlash(normalized)))
	projectRoot := filepath.Clean(strings.TrimSpace(project.FolderPath))
	if projectRoot == "" || projectRoot == "." {
		return nil, fmt.Errorf("invalid project root for workflowPath=%q", workflowPath)
	}
	rootWithSep := projectRoot + string(filepath.Separator)
	if abs != projectRoot && !strings.HasPrefix(abs, rootWithSep) {
		return nil, fmt.Errorf("workflowPath=%q escapes project root", workflowPath)
	}

	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, fmt.Errorf("workflowPath=%q is empty", workflowPath)
	}

	// Parse as proto WorkflowDefinitionV2
	var protoDef basworkflows.WorkflowDefinitionV2
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, &protoDef); err != nil {
		return nil, fmt.Errorf("workflowPath=%q does not match proto schema: %w", workflowPath, err)
	}

	id := uuid.New()
	return &basapi.WorkflowSummary{
		Id:             id.String(),
		ProjectId:      callingWorkflow.ProjectID.String(),
		Name:           normalized,
		FolderPath:     "/",
		Version:        1,
		FlowDefinition: &protoDef,
	}, nil
}

func normalizeWorkflowPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, "\\", "/")
	if raw == "" {
		return "", false
	}
	if strings.HasPrefix(raw, "/") {
		return "", false
	}
	parts := strings.Split(raw, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			return "", false
		}
		clean = append(clean, part)
	}
	return strings.Join(clean, "/"), true
}

// unsupportedAutomationNodes returns node types that the new automation
// executor cannot yet orchestrate. Results are used for logging/alerting only;
// execution continues on the automation stack.
func unsupportedAutomationNodes(flowDefinition database.JSONMap) []string {
	if flowDefinition == nil {
		return nil
	}

	nodes, ok := flowDefinition["nodes"].([]any)
	if !ok || len(nodes) == 0 {
		return nil
	}

	supportedLoopTypes := map[string]struct{}{
		"repeat":  {},
		"foreach": {},
		"forEach": {},
		"while":   {},
	}

	unsupported := map[string]struct{}{}
	add := func(kind string) {
		if strings.TrimSpace(kind) == "" {
			return
		}
		unsupported[kind] = struct{}{}
	}

	for _, raw := range nodes {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		nodeType := strings.ToLower(strings.TrimSpace(extractString(node, "type")))
		data, _ := node["data"].(map[string]any)

		switch nodeType {
		case "loop":
			loopType := strings.TrimSpace(extractString(data, "loopType"))
			if loopType == "" {
				add("loop:missing_type")
				continue
			}
			if _, ok := supportedLoopTypes[loopType]; !ok {
				add(fmt.Sprintf("loop:%s", strings.ToLower(loopType)))
			}
		}
	}

	if len(unsupported) == 0 {
		return nil
	}

	out := make([]string, 0, len(unsupported))
	for kind := range unsupported {
		out = append(out, kind)
	}
	sort.Strings(out)
	return out
}

// extractString safely extracts a string value from a map.
func extractString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// flowDefinitionToProto converts a JSON map flow definition to proto.
// This is an inline version to avoid import cycle with protoconv package.
func flowDefinitionToProto(definition database.JSONMap) (*basworkflows.WorkflowDefinitionV2, error) {
	pb := &basworkflows.WorkflowDefinitionV2{}
	if definition == nil {
		return pb, nil
	}
	raw, err := json.Marshal(definition)
	if err != nil {
		return nil, fmt.Errorf("marshal flow_definition: %w", err)
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, pb); err != nil {
		return nil, fmt.Errorf("unmarshal flow_definition: %w", err)
	}
	return pb, nil
}

// workflowIndexToProto converts a database.Workflow (index) to basapi.WorkflowSummary (proto).
// It loads the flow definition from:
// 1. Adhoc flow store (for ephemeral workflows)
// 2. Disk file (for persisted workflows)
func (s *WorkflowService) workflowIndexToProto(ctx context.Context, workflow *database.Workflow) (*basapi.WorkflowSummary, error) {
	if workflow == nil {
		return nil, errors.New("workflow is nil")
	}

	// First, check if this is an adhoc workflow with definition stored in memory
	if adhocDef := getAdhocFlowDefinition(workflow.ID); adhocDef != nil {
		flowDef, err := flowDefinitionToProto(database.JSONMap(adhocDef.FlowDefinition))
		if err != nil {
			return nil, fmt.Errorf("convert adhoc flow definition to proto: %w", err)
		}
		pb := &basapi.WorkflowSummary{
			Id:             workflow.ID.String(),
			Name:           workflow.Name,
			FolderPath:     workflow.FolderPath,
			Version:        int32(workflow.Version),
			CreatedAt:      timestamppb.New(workflow.CreatedAt),
			UpdatedAt:      timestamppb.New(workflow.UpdatedAt),
			FlowDefinition: flowDef,
		}
		if workflow.ProjectID != nil {
			pb.ProjectId = workflow.ProjectID.String()
		}
		return pb, nil
	}

	// Otherwise, load from disk file
	if workflow.FilePath == "" {
		// No file path - return minimal proto with empty definition
		pb := &basapi.WorkflowSummary{
			Id:             workflow.ID.String(),
			Name:           workflow.Name,
			FolderPath:     workflow.FolderPath,
			Version:        int32(workflow.Version),
			CreatedAt:      timestamppb.New(workflow.CreatedAt),
			UpdatedAt:      timestamppb.New(workflow.UpdatedAt),
			FlowDefinition: &basworkflows.WorkflowDefinitionV2{},
		}
		if workflow.ProjectID != nil {
			pb.ProjectId = workflow.ProjectID.String()
		}
		return pb, nil
	}

	// Read workflow file from disk
	raw, err := os.ReadFile(workflow.FilePath)
	if err != nil {
		return nil, fmt.Errorf("read workflow file %s: %w", workflow.FilePath, err)
	}

	// Parse file as JSON to extract all fields
	var fileData map[string]any
	if err := json.Unmarshal(raw, &fileData); err != nil {
		return nil, fmt.Errorf("parse workflow file %s: %w", workflow.FilePath, err)
	}

	// Extract flow definition and convert to proto
	var flowDef *basworkflows.WorkflowDefinitionV2
	if rawDef, ok := fileData["definition_v2"].(map[string]any); ok {
		defBytes, _ := json.Marshal(rawDef)
		flowDef = &basworkflows.WorkflowDefinitionV2{}
		if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(defBytes, flowDef); err != nil {
			return nil, fmt.Errorf("parse definition_v2: %w", err)
		}
	} else if rawDef, ok := fileData["flow_definition"].(map[string]any); ok {
		flowDef, err = flowDefinitionToProto(database.JSONMap(rawDef))
		if err != nil {
			return nil, fmt.Errorf("convert flow_definition: %w", err)
		}
	} else {
		// Try to parse the whole file as flow definition
		flowDef, err = flowDefinitionToProto(database.JSONMap(fileData))
		if err != nil {
			flowDef = &basworkflows.WorkflowDefinitionV2{}
		}
	}

	// Build proto
	pb := &basapi.WorkflowSummary{
		Id:             workflow.ID.String(),
		Name:           workflow.Name,
		FolderPath:     workflow.FolderPath,
		Version:        int32(workflow.Version),
		CreatedAt:      timestamppb.New(workflow.CreatedAt),
		UpdatedAt:      timestamppb.New(workflow.UpdatedAt),
		FlowDefinition: flowDef,
	}
	if workflow.ProjectID != nil {
		pb.ProjectId = workflow.ProjectID.String()
	}

	// Extract optional fields from file
	if desc, ok := fileData["description"].(string); ok {
		pb.Description = desc
	}
	if tags, ok := fileData["tags"].([]any); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok {
				pb.Tags = append(pb.Tags, s)
			}
		}
	}
	if createdBy, ok := fileData["created_by"].(string); ok {
		pb.CreatedBy = createdBy
	}
	if isTemplate, ok := fileData["is_template"].(bool); ok {
		pb.IsTemplate = isTemplate
	}
	if changeDesc, ok := fileData["change_description"].(string); ok {
		pb.LastChangeDescription = changeDesc
	}

	return pb, nil
}
