// Package pageinspect joins a single BAS observation to consumer-resolved source.
package pageinspect

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

//go:embed resolve.mjs
var resolverScript string
var scenarioName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Service struct {
	RepoRoot   string
	ResolveURL func(context.Context, string, string) (string, error)
	HTTPClient *http.Client
}

func (s Service) Inspect(ctx context.Context, req *pagev1.InspectRequest) (*pagev1.InspectResponse, error) {
	route, err := url.Parse(req.Route)
	if !scenarioName.MatchString(req.Scenario) || err != nil || route.IsAbs() || route.Host != "" || !strings.HasPrefix(req.Route, "/") || strings.HasPrefix(req.Route, "//") || strings.ContainsAny(route.Path, "*:\\") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("inspect needs a scenario slug and a concrete same-origin route"))
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	resolve := s.ResolveURL
	if resolve == nil {
		resolve = discovery.ResolveScenarioURL
	}
	ui, err := resolve(ctx, req.Scenario, "UI_PORT")
	if err != nil {
		return nil, fmt.Errorf("resolve running UI: %w", err)
	}
	bas, err := resolve(ctx, "browser-automation-studio", "API_PORT")
	if err != nil {
		return nil, fmt.Errorf("resolve BAS: %w", err)
	}
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 55 * time.Second}
	}
	target := strings.TrimRight(ui, "/") + req.Route
	var waitFor *capturev1.WaitFor
	if req.WaitSelector != "" {
		waitFor = &capturev1.WaitFor{Spec: &capturev1.WaitFor_Selector{Selector: req.WaitSelector}}
	}
	capture, err := captureconnect.NewCaptureServiceClient(client, bas, connect.WithReadMaxBytes(32<<20)).Capture(ctx, connect.NewRequest(&capturev1.CaptureRequest{
		Url: target, Captures: []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT, capturev1.CaptureType_CAPTURE_TYPE_DOM},
		Dimensions: &capturev1.Dimensions{Width: proto.Int32(1280), Height: proto.Int32(800)},
		WaitFor:    waitFor, InlineDomTree: true, Label: "rcl:page-inspect:" + req.Scenario + req.Route,
	}))
	if err != nil {
		return nil, fmt.Errorf("capture page: %w", err)
	}
	response := &pagev1.InspectResponse{Scenario: req.Scenario, Route: req.Route, Url: target, ExecutionId: capture.Msg.ExecutionId}
	for _, artifact := range capture.Msg.Artifacts {
		if artifact.Type == capturev1.CaptureType_CAPTURE_TYPE_SCREENSHOT {
			response.ScreenshotPath = artifact.Path
			if view, parseErr := url.Parse(artifact.Metadata["view_url"]); parseErr == nil && view.String() != "" {
				base, _ := url.Parse(bas)
				response.ScreenshotUrl = base.ResolveReference(view).String()
			}
			if artifact.Primary {
				break
			}
		}
	}
	if response.ScreenshotPath == "" || capture.Msg.DomTreeJson == "" {
		return nil, fmt.Errorf("capture did not return both screenshot and DOM tree")
	}
	var tree map[string]any
	if err := json.Unmarshal([]byte(capture.Msg.DomTreeJson), &tree); err != nil {
		return nil, fmt.Errorf("decode BAS DOM tree: %w", err)
	}
	// BAS providers may wrap the root in a tree envelope.
	if root, ok := tree["tree"].(map[string]any); ok {
		tree = root
	}
	if tag, _ := tree["tagName"].(string); tag == "" {
		return nil, fmt.Errorf("capture returned no DOM root")
	}
	requests := map[string]map[string]string{}
	var gather func(map[string]any)
	gather = func(node map[string]any) {
		data, _ := node["data"].(map[string]any)
		asset, _ := data["rclAsset"].(string)
		version, _ := data["rclVersion"].(string)
		if asset != "" {
			requests[asset+"@"+version] = map[string]string{"asset": asset, "version": version}
		}
		for _, child := range children(node) {
			gather(child)
		}
	}
	gather(tree)
	sources, err := s.resolveSources(ctx, requests)
	if err != nil {
		return nil, err
	}
	var annotate func(map[string]any) (*pagev1.PageNode, error)
	annotate = func(node map[string]any) (*pagev1.PageNode, error) {
		response.NodeCount++
		result := &pagev1.PageNode{Source: &pagev1.AssetSource{Status: "unstamped"}}
		data, _ := node["data"].(map[string]any)
		asset, _ := data["rclAsset"].(string)
		version, _ := data["rclVersion"].(string)
		if asset != "" {
			response.StampedCount++
			result.Source = sources[asset+"@"+version]
			if result.Source == nil {
				return nil, fmt.Errorf("source resolver omitted stamp %s@%s", asset, version)
			}
			if result.Source.Status == "resolved" {
				response.ResolvedCount++
			}
		}
		for _, child := range children(node) {
			annotated, err := annotate(child)
			if err != nil {
				return nil, err
			}
			result.Children = append(result.Children, annotated)
		}
		delete(node, "children")
		result.Observation, err = structpb.NewStruct(node)
		return result, err
	}
	response.Tree, err = annotate(tree)
	if response.StampedCount > response.ResolvedCount {
		response.Warnings = append(response.Warnings, "Some stamped nodes could not resolve; inspect their source reasons.")
	}
	if response.NodeCount >= 4000 {
		response.Warnings = append(response.Warnings, "BAS DOM node limit reached; the tree may be truncated.")
	}
	return response, err
}

func children(node map[string]any) []map[string]any {
	values, _ := node["children"].([]any)
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		if child, ok := value.(map[string]any); ok {
			result = append(result, child)
		}
	}
	return result
}
func (s Service) resolveSources(ctx context.Context, requests map[string]map[string]string) (map[string]*pagev1.AssetSource, error) {
	result := map[string]*pagev1.AssetSource{}
	if len(requests) == 0 {
		return result, nil
	}
	input := make([]map[string]string, 0, len(requests))
	for _, request := range requests {
		input = append(input, request)
	}
	raw, _ := json.Marshal(input)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "--input-type=module", "-e", resolverScript, "--", "page-inspect", s.RepoRoot)

	cmd.Stdin = bytes.NewReader(raw)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("resolve library stamps: %w", err)
	}
	var rows map[string]json.RawMessage
	if err := json.Unmarshal(output, &rows); err != nil {
		return nil, err
	}
	for key, row := range rows {
		source := &pagev1.AssetSource{}
		if err := protojson.Unmarshal(row, source); err != nil {
			return nil, err
		}
		result[key] = source
	}
	return result, nil
}
