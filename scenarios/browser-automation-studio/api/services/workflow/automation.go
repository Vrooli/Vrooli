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
	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
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

	plan, _, err := autoexecutor.BuildContractsPlanWithCompiler(ctx, execution.ID, workflow, compiler)
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
		WorkflowResolver:  &executorWorkflowResolver{repo: s.repo},
		PlanCompiler:      compiler,
		MaxSubflowDepth:   config.Load().Execution.MaxSubflowDepth,
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

	plan, _, err := autoexecutor.BuildContractsPlanWithCompiler(ctx, execution.ID, workflow, compiler)
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
		WorkflowResolver:   &executorWorkflowResolver{repo: s.repo},
		PlanCompiler:       compiler,
		MaxSubflowDepth:    config.Load().Execution.MaxSubflowDepth,
		StartFromStepIndex: startFromStepIndex,
		InitialVariables:   initialVars,
		ResumedFromID:      &resumedFromID,
	}
	return s.executor.Execute(ctx, req)
}

type executorWorkflowResolver struct {
	repo database.Repository
}

func (r *executorWorkflowResolver) GetWorkflow(ctx context.Context, workflowID uuid.UUID) (*database.Workflow, error) {
	return r.repo.GetWorkflow(ctx, workflowID)
}

func (r *executorWorkflowResolver) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*database.Workflow, error) {
	if version <= 0 {
		return r.repo.GetWorkflow(ctx, workflowID)
	}

	base, err := r.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	v, err := r.repo.GetWorkflowVersion(ctx, workflowID, version)
	if err != nil {
		return nil, err
	}

	// Rehydrate a Workflow struct with the versioned definition.
	// Only fields required by the executor/compiler are populated.
	return &database.Workflow{
		ID:             base.ID,
		ProjectID:      base.ProjectID,
		Name:           base.Name,
		FolderPath:     base.FolderPath,
		WorkflowType:   base.WorkflowType,
		FlowDefinition: v.FlowDefinition,
		Version:        v.Version,
	}, nil
}

func (r *executorWorkflowResolver) GetWorkflowByProjectPath(ctx context.Context, callingWorkflowID uuid.UUID, workflowPath string) (*database.Workflow, error) {
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

	var def map[string]any
	if err := json.Unmarshal(raw, &def); err != nil {
		return nil, fmt.Errorf("workflowPath=%q is not valid JSON: %w", workflowPath, err)
	}

	// Fail-fast: ensure the definition matches the canonical proto shape.
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, &basv1.WorkflowDefinitionV2{}); err != nil {
		return nil, fmt.Errorf("workflowPath=%q does not match proto schema: %w", workflowPath, err)
	}

	id := uuid.New()
	return &database.Workflow{
		ID:             id,
		ProjectID:      callingWorkflow.ProjectID,
		Name:           normalized,
		FolderPath:     "/",
		WorkflowType:   callingWorkflow.WorkflowType,
		FlowDefinition: database.JSONMap(def),
		Version:        1,
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
