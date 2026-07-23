package capture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vrooli/browser-automation-studio/internal/compat"
	"github.com/vrooli/browser-automation-studio/services/workflow"
	"github.com/vrooli/browser-automation-studio/viewport"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basebase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// service implements captureconnect.CaptureServiceHandler.
type service struct {
	deps Deps
}

// writeCaptureArtifactSummary makes the response artifact contract durable in
// the exported result.json, including the canonical screenshot selection.
func writeCaptureArtifactSummary(outDir string, artifacts []*capturev1.CaptureArtifact) error {
	path := filepath.Join(outDir, "result.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read result.json: %w", err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("parse result.json: %w", err)
	}
	summaries := make([]map[string]any, 0, len(artifacts))
	for _, artifact := range artifacts {
		summary := map[string]any{"type": artifact.GetType().String(), "path": artifact.GetPath(), "size_bytes": artifact.GetSizeBytes(), "metadata": artifact.GetMetadata(), "primary": artifact.GetPrimary()}
		summaries = append(summaries, summary)
		if artifact.GetPrimary() {
			result["primary_artifact_path"] = artifact.GetPath()
		}
	}
	result["capture_artifacts"] = summaries
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("encode result.json: %w", err)
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}

var (
	scenarioSlugRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
	truthyValues   = map[string]struct{}{"true": {}, "1": {}, "yes": {}}
)

// Capture loads a URL once and produces every requested artifact from
// that one session. Contract: plan
// bas-phase-2-connect-rpc-captureservice-handler-side-by-side-chi-mount §8.
func (s *service) Capture(
	ctx context.Context,
	req *connect.Request[capturev1.CaptureRequest],
) (*connect.Response[capturev1.CaptureResponse], error) {
	start := s.deps.Now()
	msg := req.Msg

	resolvedURL, err := s.resolveURL(ctx, msg.GetUrl())
	if err != nil {
		return nil, err
	}

	captures, err := normalizeCaptures(msg.GetCaptures())
	if err != nil {
		return nil, err
	}

	width, height, err := resolveDimensions(msg.GetDimensions())
	if err != nil {
		return nil, err
	}

	capturesRoot := strings.TrimSpace(s.deps.CapturesRoot)
	if capturesRoot == "" {
		capturesRoot = filepath.Join(os.TempDir(), "bas-capture")
	}
	outDir := strings.TrimSpace(msg.GetOutDir())
	switch {
	case outDir == "":
		outDir = filepath.Join(capturesRoot, uuid.NewString())
	case !filepath.IsAbs(outDir):
		// Relative out dirs anchor to the captures root, not the API
		// process's working directory. Reject traversal that would escape it.
		joined := filepath.Join(capturesRoot, outDir)
		if rel, relErr := filepath.Rel(capturesRoot, joined); relErr != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("out dir %q escapes the captures root", msg.GetOutDir()))
		}
		outDir = joined
	}

	if isDryRun(req.Header().Get("X-Dry-Run")) {
		execID := "dry-run-" + uuid.NewString()
		return connect.NewResponse(&capturev1.CaptureResponse{
			ExecutionId: execID,
			OutDir:      outDir,
			Artifacts:   synthesizeArtifacts(outDir, captures),
			DurationMs:  0,
			DryRun:      true,
			Readiness:   captureReadinessDiagnostics(msg.GetWaitFor(), "generic-navigation", "dry-run", 0, "dry run does not navigate"),
		}), nil
	}

	adhocReq, domNodeID, err := buildAdhocRequest(resolvedURL, msg, width, height, s.deps.InlineDom.Expression)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// Explicit caller waits are authoritative. For a known local scenario with
	// no explicit wait, ask Experience Manager for its compiled profile and add
	// the selected required-surface binding as a post-navigation Wait action.
	// Missing/unavailable profiles intentionally fall back to generic capture.
	selectedReadiness := "generic-navigation"
	var declaredResolution ReadinessResolution
	fallbackReason := ""
	if msg.GetWaitFor() != nil {
		selectedReadiness = requestedReadinessStrategy(msg.GetWaitFor())
	}
	if msg.GetWaitFor() == nil && s.deps.ReadinessResolver != nil {
		if scenario, route, ok := scenarioTarget(msg.GetUrl()); ok {
			if resolution, resolveErr := s.deps.ReadinessResolver.ResolveReadinessWaits(ctx, scenario, route); resolveErr == nil {
				declaredResolution = resolution
				if len(resolution.Waits) > 0 {
					appendPostNavigationWaits(adhocReq, resolution.Waits)
					selectedReadiness = "declared-surface"
				} else if resolution.ProfileVersion == "" {
					fallbackReason = "declared readiness profile returned no version"
				} else if !resolution.RouteMatched {
					fallbackReason = "declared readiness profile does not include the requested route"
				} else {
					fallbackReason = "declared readiness route has no bound required surfaces"
				}
			} else {
				fallbackReason = "declared readiness profile unavailable: " + resolveErr.Error()
			}
		}
	}
	opts := &workflow.ExecuteOptions{}
	for _, ct := range captures {
		switch ct {
		case capturev1.CaptureType_CAPTURE_TYPE_VIDEO:
			opts.RequiresVideo = true
		case capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE:
			opts.RequiresPerfTrace = true
		case capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY:
			opts.RequiresAccessibility = true
		}
	}
	// inline_accessibility independently drives the AX capture (mirrors how
	// inline_dom injects its own read regardless of the captures list), so a
	// caller can request the inline snapshot without also listing ACCESSIBILITY.
	if msg.GetInlineAccessibility() {
		opts.RequiresAccessibility = true
	}

	resp, err := s.deps.Executor.ExecuteAdhocWorkflowAPIWithOptions(ctx, adhocReq, opts)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("execute adhoc: %w", err))
	}

	execID := resp.GetExecutionId()
	readinessOutcome := readinessOutcomeForExecutionStatus(resp.GetStatus())
	executionOutDir := filepath.Join(outDir, execID)

	executionUUID, err := uuid.Parse(execID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid execution id %q from executor: %w", execID, err))
	}
	if err := s.deps.Executor.ExportToFolder(ctx, executionUUID, executionOutDir, s.deps.Storage); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("export artifacts: %w", err))
	}
	if resp.GetStatus() != basebase.ExecutionStatus_EXECUTION_STATUS_COMPLETED {
		failure := strings.TrimSpace(resp.GetError())
		if failure == "" {
			failure = strings.TrimSpace(resp.GetMessage())
		}
		if failure == "" {
			failure = strings.ToLower(strings.TrimPrefix(resp.GetStatus().String(), "EXECUTION_STATUS_"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("capture execution %s finished %s: %s", execID, strings.ToLower(strings.TrimPrefix(resp.GetStatus().String(), "EXECUTION_STATUS_")), failure))
	}

	artifacts, err := s.deps.Producers.ProduceAll(captures, executionOutDir)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("harvest artifacts: %w", err))
	}
	if err := writeCaptureArtifactSummary(executionOutDir, artifacts); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("write capture artifact summary: %w", err))
	}

	// Inline DOM is best-effort: a failed in-page read degrades to an empty
	// dom_html (documented on the proto field) rather than failing a capture
	// whose other artifacts are already on disk.
	domHTML := ""
	if domNodeID != "" {
		domHTML, err = s.deps.InlineDom.readInlineDom(executionOutDir, domNodeID)
		if err != nil && s.deps.Logger != nil {
			s.deps.Logger.WithError(err).Warn("capture: inline DOM read failed")
		}
	}

	// Inline accessibility is best-effort: a missing/failed AX capture
	// degrades to an empty accessibility_json (documented on the proto field)
	// rather than failing a capture whose other artifacts are on disk. The
	// snapshot the driver produced is written by ExportToFolder as
	// accessibility.json in the execution out dir, so we read it back here.
	accessibilityJSON := ""
	if msg.GetInlineAccessibility() {
		accessibilityJSON, err = s.deps.InlineAccessibility.readInlineAccessibility(executionOutDir)
		if err != nil && s.deps.Logger != nil {
			s.deps.Logger.WithError(err).Warn("capture: inline accessibility read failed")
		}
	}

	duration := s.deps.Now().Sub(start).Milliseconds()
	timing := readinessTimelineTiming{}
	timingAvailable := false
	if observedTiming, timingErr := readinessTiming(executionOutDir); timingErr != nil {
		if s.deps.Logger != nil {
			s.deps.Logger.WithError(timingErr).Warn("capture: readiness timing unavailable")
		}
	} else {
		timing = observedTiming
		timingAvailable = true
	}
	if selectedReadiness == "declared-surface" {
		if timing.outcome != "" {
			readinessOutcome = timing.outcome
		} else if !timingAvailable && s.deps.Logger != nil {
			s.deps.Logger.Warn("capture: declared readiness outcome unavailable")
		}
	}
	return connect.NewResponse(&capturev1.CaptureResponse{
		ExecutionId:       execID,
		OutDir:            executionOutDir,
		Artifacts:         artifacts,
		DurationMs:        duration,
		DryRun:            false,
		DomHtml:           domHTML,
		AccessibilityJson: accessibilityJSON,
		Readiness:         captureReadinessDiagnosticsWithTiming(msg.GetWaitFor(), selectedReadiness, readinessOutcome, duration, fallbackReason, declaredResolution, timing),
	}), nil
}

// readinessOutcomeForExecutionStatus preserves the readiness contract's
// user-facing success value while deriving it from the executor's terminal
// state rather than assuming every completed RPC is ready.
func readinessOutcomeForExecutionStatus(status basebase.ExecutionStatus) string {
	switch status {
	case basebase.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
		return "ready"
	case basebase.ExecutionStatus_EXECUTION_STATUS_FAILED:
		return "failed"
	case basebase.ExecutionStatus_EXECUTION_STATUS_CANCELLED:
		return "cancelled"
	case basebase.ExecutionStatus_EXECUTION_STATUS_RUNNING:
		return "running"
	case basebase.ExecutionStatus_EXECUTION_STATUS_PENDING:
		return "pending"
	default:
		return "unknown"
	}
}

func requestedReadinessStrategy(wait *capturev1.WaitFor) string {
	if wait == nil {
		return "generic-navigation"
	}
	switch wait.GetSpec().(type) {
	case *capturev1.WaitFor_Selector:
		return "explicit-selector"
	case *capturev1.WaitFor_TimeoutMs:
		return "explicit-delay"
	case *capturev1.WaitFor_Networkidle:
		return "explicit-networkidle"
	default:
		return "generic-navigation"
	}
}

func captureReadinessDiagnostics(wait *capturev1.WaitFor, selected, outcome string, duration int64, fallback string) *capturev1.CaptureReadinessDiagnostics {
	return captureReadinessDiagnosticsWithResolution(wait, selected, outcome, duration, fallback, ReadinessResolution{})
}

func captureReadinessDiagnosticsWithResolution(wait *capturev1.WaitFor, selected, outcome string, duration int64, fallback string, resolution ReadinessResolution) *capturev1.CaptureReadinessDiagnostics {
	return captureReadinessDiagnosticsWithTiming(wait, selected, outcome, duration, fallback, resolution, readinessTimelineTiming{})
}

func captureReadinessDiagnosticsWithTiming(wait *capturev1.WaitFor, selected, outcome string, duration int64, fallback string, resolution ReadinessResolution, timing readinessTimelineTiming) *capturev1.CaptureReadinessDiagnostics {
	return &capturev1.CaptureReadinessDiagnostics{RequestedStrategy: requestedReadinessStrategy(wait), SelectedStrategy: selected, Outcome: outcome, DurationMs: duration, FallbackReason: fallback, ProfileVersion: resolution.ProfileVersion, Route: resolution.Route, RequiredSurfaceIds: resolution.RequiredSurfaceIDs, NavigationDurationMs: timing.navigationMS, ReadinessWaitDurationMs: timing.readinessWaitMS}
}

func scenarioTarget(raw string) (scenario, route string, ok bool) {
	if !strings.HasPrefix(strings.TrimSpace(raw), "scenario=") {
		return "", "", false
	}
	scenario, route, err := parseScenarioShorthand(raw)
	if err != nil {
		return "", "", false
	}
	return scenario, route, true
}

// appendPostNavigationWait inserts a resolved profile wait after Navigate and
// before any inline DOM or interaction nodes. It is intentionally graph-level
// rather than a NavigateParams timeout so the diagnostic workflow preserves
// the distinction between navigation and functional readiness.
func appendPostNavigationWaits(req *basexecution.ExecuteAdhocRequest, waits []*actionsv1.WaitParams) {
	if req == nil || req.GetFlowDefinition() == nil || len(waits) == 0 {
		return
	}
	flow := req.GetFlowDefinition()
	var navID string
	for _, node := range flow.GetNodes() {
		if node.GetAction().GetNavigate() != nil {
			navID = node.GetId()
			break
		}
	}
	if navID == "" {
		return
	}
	var waitIDs []string
	for _, wait := range waits {
		if wait == nil {
			continue
		}
		waitID := uuid.NewString()
		waitIDs = append(waitIDs, waitID)
		flow.Nodes = append(flow.Nodes, &workflowsv1.WorkflowNodeV2{Id: waitID, Action: &actionsv1.ActionDefinition{Type: actionsv1.ActionType_ACTION_TYPE_WAIT, Params: &actionsv1.ActionDefinition_Wait{Wait: wait}}})
	}
	if len(waitIDs) == 0 {
		return
	}
	var edges []*workflowsv1.WorkflowEdgeV2
	for _, edge := range flow.GetEdges() {
		if edge.GetSource() == navID {
			edges = append(edges, &workflowsv1.WorkflowEdgeV2{Id: edge.GetId(), Source: waitIDs[len(waitIDs)-1], Target: edge.GetTarget()})
			continue
		}
		edges = append(edges, edge)
	}
	edges = append(edges, &workflowsv1.WorkflowEdgeV2{Id: uuid.NewString(), Source: navID, Target: waitIDs[0]})
	for index := 1; index < len(waitIDs); index++ {
		edges = append(edges, &workflowsv1.WorkflowEdgeV2{Id: uuid.NewString(), Source: waitIDs[index-1], Target: waitIDs[index]})
	}
	flow.Edges = edges
}

func appendPostNavigationWait(req *basexecution.ExecuteAdhocRequest, wait *actionsv1.WaitParams) {
	appendPostNavigationWaits(req, []*actionsv1.WaitParams{wait})
}

// resolveURL accepts either a fully-qualified http(s) URL or the
// `scenario=<slug>[,path=<path>]` shorthand documented in §8.2.
func (s *service) resolveURL(ctx context.Context, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("url is required"))
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw, nil
	}
	if !strings.HasPrefix(raw, "scenario=") {
		return "", connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("url must be http(s) or scenario=<slug>[,path=<path>] shorthand; got %q", raw))
	}
	slug, path, err := parseScenarioShorthand(raw)
	if err != nil {
		return "", connect.NewError(connect.CodeInvalidArgument, err)
	}
	if s.deps.Resolver == nil {
		return "", connect.NewError(connect.CodeInvalidArgument,
			errors.New("scenario= shorthand requires a URL resolver; none configured"))
	}
	base := ""
	if resolver, ok := s.deps.Resolver.(uiURLResolver); ok {
		base, err = resolver.ResolveScenarioURL(ctx, slug, "UI_PORT")
	}
	if base == "" || err != nil {
		base, err = s.deps.Resolver.ResolveScenarioURLDefault(ctx, slug)
	}
	if err != nil {
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("resolve scenario %q: %w", slug, err))
	}
	return strings.TrimRight(base, "/") + path, nil
}

// parseScenarioShorthand splits `scenario=<slug>[,path=<path>]`. Order-
// independent; missing `path=` defaults to `/`.
func parseScenarioShorthand(raw string) (slug, path string, err error) {
	path = "/"
	for _, segment := range strings.Split(raw, ",") {
		kv := strings.SplitN(strings.TrimSpace(segment), "=", 2)
		if len(kv) != 2 {
			return "", "", fmt.Errorf("malformed shorthand segment %q", segment)
		}
		switch kv[0] {
		case "scenario":
			slug = kv[1]
		case "path":
			path = kv[1]
		default:
			return "", "", fmt.Errorf("unknown shorthand key %q", kv[0])
		}
	}
	if !scenarioSlugRE.MatchString(slug) {
		return "", "", fmt.Errorf("scenario slug %q must match %s", slug, scenarioSlugRE)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return slug, path, nil
}

func normalizeCaptures(in []capturev1.CaptureType) ([]capturev1.CaptureType, error) {
	if len(in) == 0 {
		return []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT}, nil
	}
	for _, c := range in {
		if c == capturev1.CaptureType_CAPTURE_TYPE_UNSPECIFIED {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				errors.New("capture type CAPTURE_TYPE_UNSPECIFIED is not allowed"))
		}
	}
	return in, nil
}

// resolveDimensions applies the preset table from §8.1. Explicit
// width AND height win when both are set. Setting only one is invalid.
func resolveDimensions(d *capturev1.Dimensions) (int32, int32, error) {
	if d == nil {
		preset := viewport.Default()
		return preset.Width, preset.Height, nil
	}
	hasW, hasH := d.Width != nil, d.Height != nil
	if hasW != hasH {
		return 0, 0, connect.NewError(connect.CodeInvalidArgument,
			errors.New("dimensions.width and dimensions.height must be set together"))
	}
	if hasW {
		return d.GetWidth(), d.GetHeight(), nil
	}
	switch d.GetPreset() {
	case capturev1.DimensionsPreset_DIMENSIONS_PRESET_MOBILE:
		preset, err := viewport.Resolve("mobile")
		return preset.Width, preset.Height, err
	case capturev1.DimensionsPreset_DIMENSIONS_PRESET_TABLET:
		preset, err := viewport.Resolve("tablet")
		return preset.Width, preset.Height, err
	case capturev1.DimensionsPreset_DIMENSIONS_PRESET_DESKTOP,
		capturev1.DimensionsPreset_DIMENSIONS_PRESET_UNSPECIFIED:
		preset := viewport.Default()
		return preset.Width, preset.Height, nil
	default:
		return 0, 0, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("unknown dimensions preset: %v", d.GetPreset()))
	}
}

func isDryRun(header string) bool {
	_, ok := truthyValues[strings.ToLower(strings.TrimSpace(header))]
	return ok
}

// buildAdhocRequest constructs the minimal adhoc-workflow payload the
// existing engine accepts: a single navigate node carrying the resolved
// URL, with dimensions surfaced via ExecutionParameters.ViewportWidth/
// Height (engine-native fields, not synthetic env).
//
// When msg.InteractionFlowJson is non-empty it carries a raw
// `bas/flows`-shape JSON (a WorkflowDefinitionV2 protojson body); its
// nodes/edges are spliced after the navigate node so a perf trace spans the
// interaction. Malformed JSON yields a typed error (the handler maps it to
// InvalidArgument). Empty = the default navigate+settle capture.
//
// Greenfield: this is the only translation. No fallback path, no compat
// shim with the REST ExecuteAdhocWorkflow body shape. Capture-type
// fan-out into per-artifact steps is the executor's responsibility in a
// future PR; Phase 2 establishes only the contract surface.
func buildAdhocRequest(
	resolvedURL string,
	msg *capturev1.CaptureRequest,
	width, height int32,
	inlineDomExpression string,
) (*basexecution.ExecuteAdhocRequest, string, error) {
	navigateNode := &workflowsv1.WorkflowNodeV2{
		Id: uuid.NewString(),
		Action: &actionsv1.ActionDefinition{
			Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &actionsv1.ActionDefinition_Navigate{
				Navigate: navigateParamsFor(resolvedURL, msg.GetWaitFor()),
			},
		},
	}

	nodes := []*workflowsv1.WorkflowNodeV2{navigateNode}
	var edges []*workflowsv1.WorkflowEdgeV2
	// Readiness is deliberately a distinct node. A capture's caller-supplied
	// wait must happen after navigation, not silently become page.goto's
	// deadline. Network-idle remains a NavigateParams wait_until because the
	// driver performs that signal after goto completes.
	predecessorID := navigateNode.Id
	if waitNode := postNavigationWaitNode(msg.GetWaitFor()); waitNode != nil {
		nodes = append(nodes, waitNode)
		edges = append(edges, &workflowsv1.WorkflowEdgeV2{Id: uuid.NewString(), Source: navigateNode.Id, Target: waitNode.Id})
		predecessorID = waitNode.Id
	}

	// Splice an interaction flow after the navigate node, inside the same
	// perf-trace window. The compiler orders nodes topologically (roots =
	// no incoming edge, tie-broken by array index), so the explicit
	// navigate→entry edge guarantees the navigate runs first and the
	// interaction's own internal edges sequence the rest.
	if raw := strings.TrimSpace(msg.GetInteractionFlowJson()); raw != "" {
		spliced, err := spliceInteractionFlow(predecessorID, raw)
		if err != nil {
			return nil, "", err
		}
		nodes = append(nodes, spliced.nodes...)
		edges = append(edges, spliced.edges...)
	}

	domNodeID := ""
	if msg.GetInlineDom() {
		domNode := &workflowsv1.WorkflowNodeV2{
			Id: uuid.NewString(),
			Action: &actionsv1.ActionDefinition{
				Type: actionsv1.ActionType_ACTION_TYPE_EVALUATE,
				Params: &actionsv1.ActionDefinition_Evaluate{
					Evaluate: &actionsv1.EvaluateParams{Expression: inlineDomExpression},
				},
			},
		}
		domNodeID = domNode.Id
		nodes = append(nodes, domNode)
		edges = append(edges, &workflowsv1.WorkflowEdgeV2{
			Id:     uuid.NewString(),
			Source: predecessorID,
			Target: domNode.Id,
		})
	}

	flowName := "capture"
	flowDesc := "capture @ " + resolvedURL
	flow := &workflowsv1.WorkflowDefinitionV2{
		Metadata: &workflowsv1.WorkflowMetadataV2{
			Name:        &flowName,
			Description: &flowDesc,
		},
		Settings: &workflowsv1.WorkflowSettingsV2{
			ViewportWidth:  &width,
			ViewportHeight: &height,
		},
		Nodes: nodes,
		Edges: edges,
	}

	startURL := resolvedURL
	w := width
	h := height
	return &basexecution.ExecuteAdhocRequest{
		FlowDefinition: flow,
		Metadata: &basexecution.ExecutionMetadata{
			Name:        "capture",
			Description: strings.TrimSpace(msg.GetLabel()),
		},
		Parameters: &basexecution.ExecutionParameters{
			StartUrl:       &startURL,
			ViewportWidth:  &w,
			ViewportHeight: &h,
		},
		WaitForCompletion: true,
	}, domNodeID, nil
}

// splicedFlow holds the nodes/edges contributed by an interaction flow,
// already wired to follow the navigate node.
type splicedFlow struct {
	nodes []*workflowsv1.WorkflowNodeV2
	edges []*workflowsv1.WorkflowEdgeV2
}

// spliceInteractionFlow parses a raw bas/flows-shape JSON body (a
// WorkflowDefinitionV2 protojson) and returns its nodes/edges plus a single
// edge linking the supplied navigate node to the flow's first node. The
// flow's own internal edges sequence the rest. Malformed JSON or an empty
// node set is a typed error.
func spliceInteractionFlow(navigateNodeID, raw string) (splicedFlow, error) {
	// Apply the same compat normalization the `execute-adhoc --flow-file` path
	// uses so a raw bas/flows body (short-form execution_mode, viewport
	// settings, V1 node shape) parses identically here.
	normalized, err := compat.NormalizeWorkflowDefinitionV2Bytes([]byte(raw))
	if err != nil {
		return splicedFlow{}, fmt.Errorf("interaction_flow_json is not valid JSON: %w", err)
	}
	var def workflowsv1.WorkflowDefinitionV2
	if err := protojson.Unmarshal(normalized, &def); err != nil {
		return splicedFlow{}, fmt.Errorf("interaction_flow_json is not a valid WorkflowDefinitionV2: %w", err)
	}
	if len(def.GetNodes()) == 0 {
		return splicedFlow{}, errors.New("interaction_flow_json has no nodes")
	}
	out := splicedFlow{
		nodes: def.GetNodes(),
		edges: append([]*workflowsv1.WorkflowEdgeV2{}, def.GetEdges()...),
	}
	out.edges = append(out.edges, &workflowsv1.WorkflowEdgeV2{
		Id:     uuid.NewString(),
		Source: navigateNodeID,
		Target: def.GetNodes()[0].GetId(),
	})
	return out, nil
}

func navigateParamsFor(url string, waitFor *capturev1.WaitFor) *actionsv1.NavigateParams {
	p := &actionsv1.NavigateParams{Url: url}
	if waitFor == nil {
		return p
	}
	switch spec := waitFor.GetSpec().(type) {
	case *capturev1.WaitFor_Networkidle:
		if spec.Networkidle {
			ev := actionsv1.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_NETWORKIDLE
			p.WaitUntil = &ev
		}
	}
	return p
}

// postNavigationWaitNode maps capture-specific selector and duration waits to
// the workflow's first-class Wait action. Keeping this separate from Navigate
// preserves the navigation timeout as a bounded transport operation.
func postNavigationWaitNode(waitFor *capturev1.WaitFor) *workflowsv1.WorkflowNodeV2 {
	if waitFor == nil {
		return nil
	}
	wait := &actionsv1.WaitParams{}
	switch spec := waitFor.GetSpec().(type) {
	case *capturev1.WaitFor_Selector:
		wait.WaitFor = &actionsv1.WaitParams_Selector{Selector: spec.Selector}
	case *capturev1.WaitFor_TimeoutMs:
		wait.WaitFor = &actionsv1.WaitParams_DurationMs{DurationMs: spec.TimeoutMs}
	default:
		return nil
	}
	return &workflowsv1.WorkflowNodeV2{
		Id: uuid.NewString(),
		Action: &actionsv1.ActionDefinition{
			Type:   actionsv1.ActionType_ACTION_TYPE_WAIT,
			Params: &actionsv1.ActionDefinition_Wait{Wait: wait},
		},
	}
}

// synthesizeArtifacts produces one CaptureArtifact per requested type
// with a deterministic placeholder path. Used only for dry-run, where
// the executor is intentionally not called and there is no real bundle
// on disk; callers can still exercise their response-handling code.
func synthesizeArtifacts(outDir string, captures []capturev1.CaptureType) []*capturev1.CaptureArtifact {
	out := make([]*capturev1.CaptureArtifact, 0, len(captures))
	for _, c := range captures {
		out = append(out, &capturev1.CaptureArtifact{
			Type: c,
			Path: filepath.Join(outDir, canonicalFileName(c)),
		})
	}
	return out
}

// artifactFromFile builds an artifact for a single named export file,
// degrading to an unavailable artifact when the file is absent. Shared
// with producer.go's fileProducer.
func artifactFromFile(c capturev1.CaptureType, path string) *capturev1.CaptureArtifact {
	info, err := os.Stat(path)
	if err != nil {
		return unavailableArtifact(c, path)
	}
	return &capturev1.CaptureArtifact{
		Type:      c,
		Path:      path,
		SizeBytes: info.Size(),
		Metadata:  map[string]string{"filename": filepath.Base(path)},
	}
}

// unavailableArtifact builds the placeholder artifact returned for a
// capture type the executor's folder export cannot produce. The reason
// is sourced from the captureTypeMetadata table so it stays in lockstep
// with availability.
func unavailableArtifact(c capturev1.CaptureType, path string) *capturev1.CaptureArtifact {
	reason := metaFor(c).availableReason
	if reason == "" {
		reason = unavailableExportReason
	}
	return unavailableArtifactWithReason(c, path, reason)
}

// perfTraceMissingReason is the accurate reason a performance capture that
// otherwise executed has no trace file: the browser session ran but did not
// finalize performance.json — a capture failure that is often a transient
// casualty of concurrent capture load and clears on retry. It is deliberately
// NOT the generic export reason, and deliberately does NOT assert "no browser":
// a genuinely browser-less environment fails session start (surfaced as an RPC
// error upstream), not as a completed-but-traceless run.
const perfTraceMissingReason = "the browser session did not finalize a performance trace this run (capture failed — often transient under concurrent capture load; retry)"

// unavailableArtifactWithReason builds an unavailable artifact carrying an
// explicit reason in metadata, so the omission is surfaced honestly rather
// than silently or with a misleading default.
func unavailableArtifactWithReason(c capturev1.CaptureType, path, reason string) *capturev1.CaptureArtifact {
	return &capturev1.CaptureArtifact{
		Type: c,
		Path: path,
		Metadata: map[string]string{
			"unavailable": "true",
			"reason":      reason,
		},
	}
}
