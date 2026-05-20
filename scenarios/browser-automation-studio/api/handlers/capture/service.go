package capture

import (
	"context"
	"errors"
	"fmt"
	"os"
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

	executionUUID, err := uuid.Parse(execID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid execution id %q from executor: %w", execID, err))
	}
	if err := s.deps.Executor.ExportToFolder(ctx, executionUUID, executionOutDir, s.deps.Storage); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("export artifacts: %w", err))
	}

	artifacts, err := harvestArtifacts(executionOutDir, captures)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("harvest artifacts: %w", err))
	}

	return connect.NewResponse(&capturev1.CaptureResponse{
		ExecutionId: execID,
		OutDir:      executionOutDir,
		Artifacts:   artifacts,
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
// with a deterministic placeholder path. Used only for dry-run, where
// the executor is intentionally not called and there is no real bundle
// on disk; callers can still exercise their response-handling code.
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

// harvestArtifacts walks the executor's output directory and assembles
// one CaptureArtifact per requested CaptureType. ExportToFolder owns the
// write layout (`screenshots/step-NN-*.png`, `console-logs.md`,
// `network-activity.md`, …); harvest is the read-side counterpart.
//
// For SCREENSHOT, every file under screenshots/ is exposed as its own
// artifact — single-location captures usually produce one PNG, but
// nothing in the contract forbids multi-step (a future wait-for selector
// step, e.g.).
//
// CaptureTypes the executor's folder export does not produce today
// (VIDEO, DOM, PERFORMANCE) return an artifact with the canonical
// per-type filename and metadata.unavailable=true so callers see the
// gap explicitly rather than a silent omission.
func harvestArtifacts(outDir string, captures []capturev1.CaptureType) ([]*capturev1.CaptureArtifact, error) {
	out := make([]*capturev1.CaptureArtifact, 0, len(captures))
	for _, c := range captures {
		arts, err := harvestOne(outDir, c)
		if err != nil {
			return nil, err
		}
		out = append(out, arts...)
	}
	return out, nil
}

func harvestOne(outDir string, c capturev1.CaptureType) ([]*capturev1.CaptureArtifact, error) {
	switch c {
	case capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT:
		shotsDir := filepath.Join(outDir, "screenshots")
		entries, err := os.ReadDir(shotsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return []*capturev1.CaptureArtifact{unavailableArtifact(c, filepath.Join(shotsDir, "screenshot.png"))}, nil
			}
			return nil, fmt.Errorf("read screenshots dir: %w", err)
		}
		out := make([]*capturev1.CaptureArtifact, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			full := filepath.Join(shotsDir, e.Name())
			info, err := e.Info()
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", full, err)
			}
			out = append(out, &capturev1.CaptureArtifact{
				Type:      c,
				Path:      full,
				SizeBytes: info.Size(),
				Metadata:  map[string]string{"filename": e.Name()},
			})
		}
		if len(out) == 0 {
			return []*capturev1.CaptureArtifact{unavailableArtifact(c, filepath.Join(shotsDir, "screenshot.png"))}, nil
		}
		return out, nil
	case capturev1.CaptureType_CAPTURE_TYPE_CONSOLE_LOGS:
		return []*capturev1.CaptureArtifact{artifactFromFile(c, filepath.Join(outDir, "console-logs.md"))}, nil
	case capturev1.CaptureType_CAPTURE_TYPE_NETWORK:
		return []*capturev1.CaptureArtifact{artifactFromFile(c, filepath.Join(outDir, "network-activity.md"))}, nil
	case capturev1.CaptureType_CAPTURE_TYPE_VIDEO,
		capturev1.CaptureType_CAPTURE_TYPE_DOM,
		capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE:
		return []*capturev1.CaptureArtifact{unavailableArtifact(c, filepath.Join(outDir, captureTypeShortName(c)+captureTypeExt(c)))}, nil
	}
	return nil, fmt.Errorf("unsupported capture type: %v", c)
}

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

func unavailableArtifact(c capturev1.CaptureType, path string) *capturev1.CaptureArtifact {
	return &capturev1.CaptureArtifact{
		Type: c,
		Path: path,
		Metadata: map[string]string{
			"unavailable": "true",
			"reason":      "executor folder export does not produce this artifact type yet",
		},
	}
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
