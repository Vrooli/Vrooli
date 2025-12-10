package workflow

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	autoevents "github.com/vrooli/browser-automation-studio/automation/events"
	autoexecutor "github.com/vrooli/browser-automation-studio/automation/executor"
	autorecorder "github.com/vrooli/browser-automation-studio/automation/recorder"
	"github.com/vrooli/browser-automation-studio/database"
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
		HeartbeatInterval: 2 * time.Second,
		WorkflowResolver:  s.repo,
		PlanCompiler:      compiler,
		MaxSubflowDepth:   5,
	}
	return s.executor.Execute(ctx, req)
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
