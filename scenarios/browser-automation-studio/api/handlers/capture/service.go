package capture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"buf.build/go/protovalidate"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/browser-automation-studio/automation/compiler"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/internal/compat"
	"github.com/vrooli/browser-automation-studio/services/retention"
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
	deps      Deps
	validator protovalidate.Validator
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
	plan, err := s.prepareCapture(ctx, msg)
	if err != nil {
		return nil, err
	}

	if isDryRun(req.Header().Get("X-Dry-Run")) {
		return dryRunResponse(msg, plan)
	}

	releaseEvidence := retention.BeginEvidenceActivity(filepath.Clean(plan.outDir))
	defer releaseEvidence()
	adhocReq, domNodeIDs, err := buildAdhocRequest(plan.resolvedURL, msg, plan.width, plan.height, s.deps.InlineDom.Expression)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	readiness := s.resolveCaptureReadiness(ctx, msg, adhocReq)
	opts := captureExecutionOptions(plan.captures, msg)
	execution, err := s.executeCapture(ctx, adhocReq, opts, plan.outDir)
	if err != nil {
		return nil, err
	}
	artifacts, inline, err := s.materializeCaptureArtifacts(ctx, msg, plan.captures, execution, domNodeIDs)
	if err != nil {
		return nil, err
	}
	inline.accessibilityJSON = s.readInlineAccessibility(msg, execution.outDir)
	duration := s.deps.Now().Sub(start).Milliseconds()
	timing := s.captureReadinessTiming(execution.outDir, readiness.selected)
	return connect.NewResponse(&capturev1.CaptureResponse{
		ExecutionId:       execution.id,
		OutDir:            execution.outDir,
		Artifacts:         artifacts,
		DurationMs:        duration,
		DomHtml:           inline.domHTMLForResponse(msg),
		AccessibilityJson: inline.accessibilityJSON,
		DomTreeJson:       inline.domTreeForResponse(msg),
		Readiness: captureReadinessDiagnosticsWithTiming(msg.GetWaitFor(), readiness.selected,
			readiness.outcome(timing), duration, readiness.fallbackReason, readiness.declaredResolution, timing),
	}), nil
}

type capturePlan struct {
	resolvedURL string
	captures    []capturev1.CaptureType
	width       int32
	height      int32
	outDir      string
}

type captureReadiness struct {
	selected           string
	fallbackReason     string
	declaredResolution ReadinessResolution
}

func (r captureReadiness) outcome(timing readinessTimelineTiming) string {
	if r.selected == "declared-surface" && timing.outcome != "" {
		return timing.outcome
	}
	return "ready"
}

type captureExecution struct {
	id     string
	uuid   uuid.UUID
	outDir string
}

type captureInlineResults struct {
	domHTML           string
	domTreeJSON       string
	accessibilityJSON string
	domHTMLTruncated  bool
	domTreeTruncated  bool
}

func (s *service) prepareCapture(ctx context.Context, msg *capturev1.CaptureRequest) (capturePlan, error) {
	if err := s.validator.Validate(msg); err != nil {
		var invalid *protovalidate.ValidationError
		if errors.As(err, &invalid) {
			return capturePlan{}, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return capturePlan{}, connect.NewError(connect.CodeInternal, fmt.Errorf("validate capture request: %w", err))
	}
	resolvedURL, err := s.resolveURL(ctx, msg.GetUrl())
	if err != nil {
		return capturePlan{}, err
	}
	captures, err := normalizeCaptures(msg.GetCaptures())
	if err != nil {
		return capturePlan{}, err
	}
	width, height, err := resolveDimensions(msg.GetDimensions())
	if err != nil {
		return capturePlan{}, err
	}
	outDir, err := resolveCaptureOutDir(s.deps.CapturesRoot, msg.GetOutDir())
	if err != nil {
		return capturePlan{}, err
	}
	return capturePlan{resolvedURL: resolvedURL, captures: captures, width: width, height: height, outDir: outDir}, nil
}

func resolveCaptureOutDir(capturesRoot, requested string) (string, error) {
	capturesRoot = strings.TrimSpace(capturesRoot)
	if capturesRoot == "" {
		capturesRoot = filepath.Join(os.TempDir(), "bas-capture")
	}
	outDir := strings.TrimSpace(requested)
	switch {
	case outDir == "":
		return filepath.Join(capturesRoot, uuid.NewString()), nil
	case filepath.IsAbs(outDir):
		return outDir, nil
	}
	// Relative out dirs anchor to the captures root, not the API process's
	// working directory. Reject traversal that would escape it.
	joined := filepath.Join(capturesRoot, outDir)
	rel, err := filepath.Rel(capturesRoot, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("out dir %q escapes the captures root", requested))
	}
	return joined, nil
}

func dryRunResponse(msg *capturev1.CaptureRequest, plan capturePlan) (*connect.Response[capturev1.CaptureResponse], error) {
	execID := "dry-run-" + uuid.NewString()
	artifacts := synthesizeArtifacts(plan.outDir, plan.captures)
	attachArtifactReferences(execID, artifacts)
	return connect.NewResponse(&capturev1.CaptureResponse{
		ExecutionId: execID,
		OutDir:      plan.outDir,
		Artifacts:   artifacts,
		DurationMs:  0,
		DryRun:      true,
		Readiness:   captureReadinessDiagnostics(msg.GetWaitFor(), "generic-navigation", "dry-run", 0, "dry run does not navigate"),
	}), nil
}

func (s *service) resolveCaptureReadiness(ctx context.Context, msg *capturev1.CaptureRequest, adhocReq *basexecution.ExecuteAdhocRequest) captureReadiness {
	readiness := captureReadiness{selected: "generic-navigation"}
	if msg.GetWaitFor() != nil {
		readiness.selected = requestedReadinessStrategy(msg.GetWaitFor())
		return readiness
	}
	if s.deps.ReadinessResolver == nil {
		return readiness
	}
	scenario, route, ok := scenarioTarget(msg.GetUrl())
	if !ok {
		return readiness
	}
	resolution, err := s.deps.ReadinessResolver.ResolveReadinessWaits(ctx, scenario, route)
	if err != nil {
		readiness.fallbackReason = "declared readiness profile unavailable: " + err.Error()
		return readiness
	}
	readiness.declaredResolution = resolution
	if len(resolution.Waits) > 0 {
		appendPostNavigationWaits(adhocReq, resolution.Waits)
		readiness.selected = "declared-surface"
		return readiness
	}
	switch {
	case resolution.ProfileVersion == "":
		readiness.fallbackReason = "declared readiness profile returned no version"
	case !resolution.RouteMatched:
		readiness.fallbackReason = "declared readiness profile does not include the requested route"
	default:
		readiness.fallbackReason = "declared readiness route has no bound required surfaces"
	}
	return readiness
}

func captureExecutionOptions(captures []capturev1.CaptureType, msg *capturev1.CaptureRequest) *workflow.ExecuteOptions {
	opts := &workflow.ExecuteOptions{}
	for _, captureType := range captures {
		switch captureType {
		case capturev1.CaptureType_CAPTURE_TYPE_VIDEO:
			opts.RequiresVideo = true
		case capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE:
			opts.RequiresPerfTrace = true
		case capturev1.CaptureType_CAPTURE_TYPE_ACCESSIBILITY:
			opts.RequiresAccessibility = true
		}
	}
	// Inline accessibility independently drives the AX capture.
	if msg.GetInlineAccessibility() || msg.GetInlineComputedStyle() {
		opts.RequiresAccessibility = true
	}
	return opts
}

func (s *service) executeCapture(ctx context.Context, req *basexecution.ExecuteAdhocRequest, opts *workflow.ExecuteOptions, outDir string) (captureExecution, error) {
	response, err := s.deps.Executor.ExecuteAdhocWorkflowAPIWithOptions(ctx, req, opts)
	if err != nil {
		return captureExecution{}, connect.NewError(connect.CodeInternal, fmt.Errorf("execute adhoc: %w", err))
	}
	execID := response.GetExecutionId()
	executionUUID, err := uuid.Parse(execID)
	if err != nil {
		return captureExecution{}, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid execution id %q from executor: %w", execID, err))
	}
	executionOutDir := filepath.Join(outDir, execID)
	if err := s.deps.Executor.ExportToFolder(ctx, executionUUID, executionOutDir, s.deps.Storage); err != nil {
		return captureExecution{}, connect.NewError(connect.CodeInternal, fmt.Errorf("export artifacts: %w", err))
	}
	if response.GetStatus() != basebase.ExecutionStatus_EXECUTION_STATUS_COMPLETED {
		status := strings.ToLower(strings.TrimPrefix(response.GetStatus().String(), "EXECUTION_STATUS_"))
		failure := firstNonEmpty(strings.TrimSpace(response.GetError()), strings.TrimSpace(response.GetMessage()), status)
		return captureExecution{}, connect.NewError(connect.CodeInternal, fmt.Errorf("capture execution %s finished %s: %s", execID, status, failure))
	}
	return captureExecution{
		id: execID, uuid: executionUUID, outDir: executionOutDir,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (s *service) materializeCaptureArtifacts(ctx context.Context, msg *capturev1.CaptureRequest, captures []capturev1.CaptureType, execution captureExecution, domNodeIDs inlineDomNodeIDs) ([]*capturev1.CaptureArtifact, captureInlineResults, error) {
	artifacts, err := s.deps.Producers.ProduceAll(captures, execution.outDir)
	if err != nil {
		return nil, captureInlineResults{}, connect.NewError(connect.CodeInternal, fmt.Errorf("harvest artifacts: %w", err))
	}
	inline := s.readInlineCaptureResults(msg, execution.outDir, domNodeIDs)
	if slices.Contains(captures, capturev1.CaptureType_CAPTURE_TYPE_DOM) && inline.domHTML != "" {
		if err := publishInlineArtifact(execution.outDir, artifacts, capturev1.CaptureType_CAPTURE_TYPE_DOM, inline.domHTML, inline.domHTMLTruncated); err != nil {
			return nil, captureInlineResults{}, connect.NewError(connect.CodeInternal, fmt.Errorf("write DOM artifact: %w", err))
		}
	}
	if (slices.Contains(captures, capturev1.CaptureType_CAPTURE_TYPE_DOM_TREE) || msg.GetInlineDomTree()) && inline.domTreeJSON != "" {
		if err := publishInlineArtifact(execution.outDir, artifacts, capturev1.CaptureType_CAPTURE_TYPE_DOM_TREE, inline.domTreeJSON, inline.domTreeTruncated); err != nil {
			return nil, captureInlineResults{}, connect.NewError(connect.CodeInternal, fmt.Errorf("write DOM-tree artifact: %w", err))
		}
	}
	attachArtifactReferences(execution.id, artifacts)
	if err := s.publishCaptureArtifacts(ctx, execution, artifacts); err != nil {
		return nil, captureInlineResults{}, err
	}
	return artifacts, inline, nil
}

func (s *service) readInlineCaptureResults(msg *capturev1.CaptureRequest, outDir string, ids inlineDomNodeIDs) captureInlineResults {
	var inline captureInlineResults
	if ids.html != "" {
		var err error
		inline.domHTML, inline.domHTMLTruncated, err = s.deps.InlineDom.readInlineDom(outDir, ids.html)
		if err != nil && s.deps.Logger != nil {
			s.deps.Logger.WithError(err).Warn("capture: inline DOM read failed")
		}
	}
	if ids.tree != "" {
		treeReader := s.deps.InlineDom
		treeReader.Expression = defaultInlineDomTreeExpression
		treeReader.MaxBytes = 16 << 20
		var err error
		inline.domTreeJSON, inline.domTreeTruncated, err = treeReader.readInlineDom(outDir, ids.tree)
		if err != nil && s.deps.Logger != nil {
			s.deps.Logger.WithError(err).Warn("capture: inline DOM-tree read failed")
		}
	}
	return inline
}

func (s *service) readInlineAccessibility(msg *capturev1.CaptureRequest, outDir string) string {
	if !msg.GetInlineAccessibility() && !msg.GetInlineComputedStyle() {
		return ""
	}
	accessibilityJSON, err := s.deps.InlineAccessibility.readInlineAccessibility(outDir)
	if err != nil && s.deps.Logger != nil {
		s.deps.Logger.WithError(err).Warn("capture: inline accessibility read failed")
	}
	return accessibilityJSON
}

func (s *service) publishCaptureArtifacts(ctx context.Context, execution captureExecution, artifacts []*capturev1.CaptureArtifact) error {
	for _, artifact := range artifacts {
		if s.deps.Storage == nil {
			break
		}
		if artifact == nil || artifact.GetPath() == "" || artifact.GetMetadata()["unavailable"] == "true" {
			continue
		}
		stored, err := s.deps.Storage.StoreArtifactFromFile(ctx, execution.uuid,
			"capture/"+strings.ToLower(strings.TrimPrefix(artifact.GetType().String(), "CAPTURE_TYPE_")),
			artifact.GetPath(), mime.TypeByExtension(filepath.Ext(artifact.GetPath())))
		if err != nil {
			if s.deps.Logger != nil {
				s.deps.Logger.WithError(err).WithField("artifact", artifact.GetPath()).Warn("capture artifact URL publication failed")
			}
			continue
		}
		if artifact.Metadata == nil {
			artifact.Metadata = map[string]string{}
		}
		artifact.Metadata["view_url"] = stored.URL
		artifact.Metadata["content_type"] = stored.ContentType
	}
	if err := writeCaptureArtifactSummary(execution.outDir, artifacts); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("write capture artifact summary: %w", err))
	}
	return nil
}

func (s *service) captureReadinessTiming(outDir, selected string) readinessTimelineTiming {
	timing, err := readinessTiming(outDir)
	if err != nil {
		if s.deps.Logger != nil {
			s.deps.Logger.WithError(err).Warn("capture: readiness timing unavailable")
		}
		return readinessTimelineTiming{}
	}
	if selected == "declared-surface" && timing.outcome == "" && s.deps.Logger != nil {
		s.deps.Logger.Warn("capture: declared readiness outcome unavailable")
	}
	return timing
}

func (r captureInlineResults) domHTMLForResponse(msg *capturev1.CaptureRequest) string {
	if msg.GetInlineDom() {
		return r.domHTML
	}
	return ""
}

func (r captureInlineResults) domTreeForResponse(msg *capturev1.CaptureRequest) string {
	if msg.GetInlineDomTree() {
		return r.domTreeJSON
	}
	return ""
}

func publishInlineArtifact(outDir string, artifacts []*capturev1.CaptureArtifact, captureType capturev1.CaptureType, contents string, truncated bool) error {
	path := filepath.Join(outDir, canonicalFileName(captureType))
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		return err
	}
	for _, artifact := range artifacts {
		if artifact == nil || artifact.GetType() != captureType {
			continue
		}
		artifact.Path = path
		artifact.SizeBytes = int64(len(contents))
		artifact.Metadata = map[string]string{
			"filename":     filepath.Base(path),
			"content_type": mime.TypeByExtension(filepath.Ext(path)),
		}
		if truncated {
			artifact.Metadata["truncated"] = "true"
		}
	}
	return nil
}

func attachArtifactReferences(executionID string, artifacts []*capturev1.CaptureArtifact) {
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		artifact.Reference = fmt.Sprintf("bas-capture://%s/%s", executionID, strings.ToLower(strings.TrimPrefix(artifact.GetType().String(), "CAPTURE_TYPE_")))
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
		if _, known := capturev1.CaptureType_name[int32(c)]; !known || c == capturev1.CaptureType_CAPTURE_TYPE_UNSPECIFIED {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("capture type %v is not supported", c))
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
// This translation preserves the interaction graph and adds only the requested
// readiness, snapshot and final-image actions through the ordinary executor.
type inlineDomNodeIDs struct {
	html string
	tree string
}

func buildAdhocRequest(
	resolvedURL string,
	msg *capturev1.CaptureRequest,
	width, height int32,
	inlineDomExpression string,
) (*basexecution.ExecuteAdhocRequest, inlineDomNodeIDs, error) {
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
	predecessorIDs := []string{navigateNode.Id}
	appendNode := func(node *workflowsv1.WorkflowNodeV2) {
		nodes = append(nodes, node)
		for _, predecessorID := range predecessorIDs {
			edges = append(edges, &workflowsv1.WorkflowEdgeV2{Id: uuid.NewString(), Source: predecessorID, Target: node.Id})
		}
		predecessorIDs = []string{node.Id}
	}
	if waitNode := postNavigationWaitNode(msg.GetWaitFor()); waitNode != nil {
		appendNode(waitNode)
	}
	if direction := strings.ToLower(strings.TrimSpace(msg.GetDirection())); direction != "" {
		if direction != "ltr" && direction != "rtl" {
			return nil, inlineDomNodeIDs{}, fmt.Errorf("direction must be ltr or rtl")
		}
		directionNode := &workflowsv1.WorkflowNodeV2{
			Id: uuid.NewString(),
			Action: &actionsv1.ActionDefinition{
				Type: actionsv1.ActionType_ACTION_TYPE_EVALUATE,
				Params: &actionsv1.ActionDefinition_Evaluate{
					Evaluate: &actionsv1.EvaluateParams{Expression: "document.documentElement.dir = " + strconv.Quote(direction) + "; document.body.dir = " + strconv.Quote(direction) + "; true"},
				},
			},
		}
		appendNode(directionNode)
	}

	// Compose the interaction at the compiler's outer boundaries. Its original
	// branch and loop edges decide the path; every outer terminal reaches the
	// requested postlude, independently of declaration order.
	if raw := strings.TrimSpace(msg.GetInteractionFlowJson()); raw != "" {
		spliced, err := spliceInteractionFlow(predecessorIDs[0], raw)
		if err != nil {
			return nil, inlineDomNodeIDs{}, err
		}
		nodes = append(nodes, spliced.nodes...)
		edges = append(edges, spliced.edges...)
		predecessorIDs = spliced.terminals
	}

	domNodeIDs := appendCapturePostlude(msg, inlineDomExpression, appendNode)

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
	browserProfile := captureBrowserProfile(msg)
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
			BrowserProfile: browserProfile,
			ArtifactConfig: &basexecution.ArtifactCollectionConfig{Profile: proto.String(config.ProfileCapture)},
		},
		WaitForCompletion: true,
	}, domNodeIDs, nil
}

// appendCapturePostlude attaches only the requested DOM reads and final image
// after every interaction terminal, preserving the original graph branching.
func appendCapturePostlude(msg *capturev1.CaptureRequest, inlineDomExpression string, appendNode func(*workflowsv1.WorkflowNodeV2)) inlineDomNodeIDs {
	var ids inlineDomNodeIDs
	appendDomRead := func(expression string) string {
		node := &workflowsv1.WorkflowNodeV2{
			Id: uuid.NewString(),
			Action: &actionsv1.ActionDefinition{
				Type: actionsv1.ActionType_ACTION_TYPE_EVALUATE,
				Params: &actionsv1.ActionDefinition_Evaluate{
					Evaluate: &actionsv1.EvaluateParams{Expression: expression},
				},
			},
		}
		appendNode(node)
		return node.Id
	}
	if msg.GetInlineDom() || slices.Contains(msg.GetCaptures(), capturev1.CaptureType_CAPTURE_TYPE_DOM) {
		ids.html = appendDomRead(inlineDomExpression)
	}
	if msg.GetInlineDomTree() || slices.Contains(msg.GetCaptures(), capturev1.CaptureType_CAPTURE_TYPE_DOM_TREE) {
		ids.tree = appendDomRead(defaultInlineDomTreeExpression)
	}
	if captureRequestsScreenshot(msg) {
		selector := strings.TrimSpace(msg.GetScreenshotSelector())
		appendNode(&workflowsv1.WorkflowNodeV2{
			Id: uuid.NewString(),
			Action: &actionsv1.ActionDefinition{
				Type: actionsv1.ActionType_ACTION_TYPE_SCREENSHOT,
				Params: &actionsv1.ActionDefinition_Screenshot{
					Screenshot: &actionsv1.ScreenshotParams{Selector: &selector, FullPage: proto.Bool(false)},
				},
			},
		})
	}
	return ids
}

func captureRequestsScreenshot(msg *capturev1.CaptureRequest) bool {
	return strings.TrimSpace(msg.GetScreenshotSelector()) != "" || len(msg.GetCaptures()) == 0 ||
		slices.Contains(msg.GetCaptures(), capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT)
}

// captureBrowserProfile applies the per-capture scale override without mutating
// the caller's reusable profile or replacing unrelated fingerprint settings.
func captureBrowserProfile(msg *capturev1.CaptureRequest) *basebase.BrowserProfile {
	profile := msg.GetBrowserProfile()
	dimensions := msg.GetDimensions()
	if dimensions == nil || dimensions.DeviceScaleFactor == nil {
		return profile
	}
	if profile == nil {
		profile = &basebase.BrowserProfile{}
	} else {
		profile = proto.Clone(profile).(*basebase.BrowserProfile)
	}
	if profile.Fingerprint == nil {
		profile.Fingerprint = &basebase.FingerprintSettings{}
	}
	profile.Fingerprint.DeviceScaleFactor = proto.Float64(dimensions.GetDeviceScaleFactor())
	return profile
}

// splicedFlow holds the nodes/edges contributed by an interaction flow,
// already wired to follow the navigate node.
type splicedFlow struct {
	nodes     []*workflowsv1.WorkflowNodeV2
	edges     []*workflowsv1.WorkflowEdgeV2
	terminals []string
}

// spliceInteractionFlow parses a raw bas/flows-shape JSON body (a
// WorkflowDefinitionV2 protojson) and returns its nodes/edges plus a single
// edge linking the supplied predecessor to the flow's actual entry. The
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
	entry, terminals, err := compiler.WorkflowBoundaries(&def)
	if err != nil {
		return splicedFlow{}, fmt.Errorf("interaction_flow_json: %w", err)
	}
	out := splicedFlow{
		nodes:     def.GetNodes(),
		terminals: terminals,
		edges:     append([]*workflowsv1.WorkflowEdgeV2{}, def.GetEdges()...),
	}
	out.edges = append(out.edges, &workflowsv1.WorkflowEdgeV2{
		Id:     uuid.NewString(),
		Source: navigateNodeID,
		Target: entry,
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
