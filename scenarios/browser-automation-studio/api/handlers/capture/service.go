package capture

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/vrooli/browser-automation-studio/services/workflow"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	workflowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
)

// service implements captureconnect.CaptureServiceHandler.
type service struct {
	deps Deps
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

	outDir := strings.TrimSpace(msg.GetOutDir())
	if outDir == "" {
		outDir = filepath.Join("/tmp/bas-capture", uuid.NewString())
	}

	if isDryRun(req.Header().Get("X-Dry-Run")) {
		execID := "dry-run-" + uuid.NewString()
		return connect.NewResponse(&capturev1.CaptureResponse{
			ExecutionId: execID,
			OutDir:      outDir,
			Artifacts:   synthesizeArtifacts(outDir, captures),
			DurationMs:  0,
			DryRun:      true,
		}), nil
	}

	adhocReq := buildAdhocRequest(resolvedURL, msg, width, height)
	opts := &workflow.ExecuteOptions{}
	for _, ct := range captures {
		if ct == capturev1.CaptureType_CAPTURE_TYPE_VIDEO {
			opts.RequiresVideo = true
			break
		}
	}

	resp, err := s.deps.Executor.ExecuteAdhocWorkflowAPIWithOptions(ctx, adhocReq, opts)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("execute adhoc: %w", err))
	}

	execID := resp.GetExecutionId()
	executionOutDir := filepath.Join(outDir, execID)
	return connect.NewResponse(&capturev1.CaptureResponse{
		ExecutionId: execID,
		OutDir:      executionOutDir,
		Artifacts:   synthesizeArtifacts(executionOutDir, captures),
		DurationMs:  s.deps.Now().Sub(start).Milliseconds(),
		DryRun:      false,
	}), nil
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
	base, err := s.deps.Resolver.ResolveScenarioURLDefault(ctx, slug)
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
		return 1440, 900, nil
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
		return 390, 844, nil
	case capturev1.DimensionsPreset_DIMENSIONS_PRESET_TABLET:
		return 768, 1024, nil
	case capturev1.DimensionsPreset_DIMENSIONS_PRESET_DESKTOP,
		capturev1.DimensionsPreset_DIMENSIONS_PRESET_UNSPECIFIED:
		return 1440, 900, nil
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
// Greenfield: this is the only translation. No fallback path, no compat
// shim with the REST ExecuteAdhocWorkflow body shape. Capture-type
// fan-out into per-artifact steps is the executor's responsibility in a
// future PR; Phase 2 establishes only the contract surface.
func buildAdhocRequest(
	resolvedURL string,
	msg *capturev1.CaptureRequest,
	width, height int32,
) *basexecution.ExecuteAdhocRequest {
	navigateNode := &workflowsv1.WorkflowNodeV2{
		Id: uuid.NewString(),
		Action: &actionsv1.ActionDefinition{
			Type: actionsv1.ActionType_ACTION_TYPE_NAVIGATE,
			Params: &actionsv1.ActionDefinition_Navigate{
				Navigate: navigateParamsFor(resolvedURL, msg.GetWaitFor()),
			},
		},
	}

	flowName := "capture"
	flowDesc := "capture @ " + resolvedURL
	flow := &workflowsv1.WorkflowDefinitionV2{
		Metadata: &workflowsv1.WorkflowMetadataV2{
			Name:        &flowName,
			Description: &flowDesc,
		},
		Nodes: []*workflowsv1.WorkflowNodeV2{navigateNode},
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
	}
}

func navigateParamsFor(url string, waitFor *capturev1.WaitFor) *actionsv1.NavigateParams {
	p := &actionsv1.NavigateParams{Url: url}
	if waitFor == nil {
		return p
	}
	switch spec := waitFor.GetSpec().(type) {
	case *capturev1.WaitFor_Selector:
		s := spec.Selector
		p.WaitForSelector = &s
	case *capturev1.WaitFor_TimeoutMs:
		t := spec.TimeoutMs
		p.TimeoutMs = &t
	case *capturev1.WaitFor_Networkidle:
		if spec.Networkidle {
			ev := actionsv1.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_NETWORKIDLE
			p.WaitUntil = &ev
		}
	}
	return p
}

// synthesizeArtifacts produces one CaptureArtifact per requested type
// with a deterministic placeholder path. The executor populates real
// paths in a future PR; for dry-run and for the Phase-2 wire response
// these placeholders let callers exercise their bundle-handling code.
func synthesizeArtifacts(outDir string, captures []capturev1.CaptureType) []*capturev1.CaptureArtifact {
	out := make([]*capturev1.CaptureArtifact, 0, len(captures))
	for _, c := range captures {
		out = append(out, &capturev1.CaptureArtifact{
			Type: c,
			Path: filepath.Join(outDir, captureTypeShortName(c)+captureTypeExt(c)),
		})
	}
	return out
}

func captureTypeShortName(c capturev1.CaptureType) string {
	switch c {
	case capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT:
		return "screenshot"
	case capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS:
		return "console-logs"
	case capturev1.CaptureType_CAPTURE_TYPE_NETWORK:
		return "network"
	case capturev1.CaptureType_CAPTURE_TYPE_VIDEO:
		return "video"
	case capturev1.CaptureType_CAPTURE_TYPE_DOM:
		return "dom"
	case capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE:
		return "performance"
	default:
		return "unknown"
	}
}

func captureTypeExt(c capturev1.CaptureType) string {
	switch c {
	case capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT:
		return ".png"
	case capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS:
		return ".md"
	case capturev1.CaptureType_CAPTURE_TYPE_NETWORK, capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE:
		return ".json"
	case capturev1.CaptureType_CAPTURE_TYPE_VIDEO:
		return ".webm"
	case capturev1.CaptureType_CAPTURE_TYPE_DOM:
		return ".html"
	default:
		return ""
	}
}
