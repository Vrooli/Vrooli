// Package pageinspect joins a canonical BAS UX capture to consumer-resolved source.
package pageinspect

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	pagev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/page"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

//go:embed resolve.mjs
var resolverScript string

var scenarioName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var captureBundleID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func redactRouteQuery(route string) string {
	parsed, err := url.ParseRequestURI(route)
	if err != nil || parsed.RawQuery == "" {
		return route
	}
	query := parsed.Query()
	for key := range query {
		query.Set(key, "[REDACTED]")
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

type Service struct {
	RepoRoot   string
	ResolveURL func(context.Context, string, string) (string, error)
	HTTPClient *http.Client
}

type captureResponse struct {
	Manifest struct {
		SchemaVersion string `json:"schema_version"`
		ID            string `json:"id"`
		Source        struct {
			Kind     string `json:"kind"`
			Scenario string `json:"scenario"`
			Route    string `json:"route"`
		} `json:"source"`
		Lenses map[string]struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"lenses"`
		Artifacts []struct {
			Kind      string `json:"kind"`
			Path      string `json:"path"`
			SizeBytes int64  `json:"size_bytes"`
			SHA256    string `json:"sha256"`
		} `json:"artifacts"`
	} `json:"manifest"`
}

type outline struct {
	Schema      string           `json:"schema"`
	Nodes       []outlineNode    `json:"nodes"`
	Unsupported []outlineSurface `json:"unsupported"`
}
type outlineNode struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Tag      string `json:"tag"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Text     string `json:"text"`
	Level    int    `json:"level"`
	Selector string `json:"selector"`
	Rect     any    `json:"rect"`
	Disabled bool   `json:"disabled"`
}
type outlineSurface struct {
	Surface string `json:"surface"`
	Reason  string `json:"reason"`
}
type componentIdentity struct {
	Selector   string `json:"selector"`
	Asset      string `json:"asset"`
	Version    string `json:"version"`
	Stamp      string `json:"stamp"`
	SourceSlot string `json:"source_slot,omitempty"`
}
type identityArtifact struct {
	Schema       string              `json:"schema"`
	ElementCount uint32              `json:"element_count"`
	StampedCount uint32              `json:"stamped_count"`
	Components   []componentIdentity `json:"components"`
}

func (s Service) Inspect(ctx context.Context, req *pagev1.InspectRequest) (*pagev1.InspectResponse, error) {
	route, err := url.Parse(req.Route)
	if !scenarioName.MatchString(req.Scenario) || err != nil || route.IsAbs() || route.Host != "" || !strings.HasPrefix(req.Route, "/") || strings.HasPrefix(req.Route, "//") || strings.ContainsAny(route.Path, "*:\\") {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("inspect needs a scenario slug and a concrete same-origin route"))
	}
	if len(req.WaitSelector) > 512 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("wait selector must be at most 512 characters"))
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
	body, err := json.Marshal(map[string]any{"scenario": req.Scenario, "route": req.Route, "wait_selector": req.WaitSelector, "theme": "light", "width": 1280, "height": 800})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(bas, "/")+"/api/v1/ux-capture/scenario", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("capture page through BAS UX capture: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("BAS UX capture failed (HTTP %d): %s", response.StatusCode, strings.TrimSpace(string(message)))
	}
	var captured captureResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&captured); err != nil {
		return nil, fmt.Errorf("decode BAS UX capture response: %w", err)
	}
	if captured.Manifest.SchemaVersion != "ux-capture.v1" || !captureBundleID.MatchString(captured.Manifest.ID) || captured.Manifest.Source.Kind != "scenario" || captured.Manifest.Source.Scenario != req.Scenario || captured.Manifest.Source.Route != redactRouteQuery(req.Route) {
		return nil, fmt.Errorf("BAS returned a mismatched or invalid ux-capture.v1 bundle")
	}
	artifacts := map[string]struct {
		Path      string
		SizeBytes int64
		SHA256    string
	}{}
	for _, artifact := range captured.Manifest.Artifacts {
		artifacts[artifact.Kind] = struct {
			Path      string
			SizeBytes int64
			SHA256    string
		}{artifact.Path, artifact.SizeBytes, artifact.SHA256}
	}
	readArtifact := func(kind string) ([]byte, error) {
		artifact := artifacts[kind]
		rel := filepath.Clean(filepath.FromSlash(artifact.Path))
		if artifact.Path == "" || strings.ContainsAny(artifact.Path, "\\\x00") || filepath.IsAbs(rel) || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.ToSlash(rel) != artifact.Path || artifact.SizeBytes < 0 {
			return nil, fmt.Errorf("BAS bundle has no safe %s artifact", kind)
		}
		if artifact.SizeBytes > 5<<20 {
			return nil, fmt.Errorf("BAS %s artifact exceeds the 5 MiB limit", kind)
		}
		endpoint, err := captureArtifactURL(bas, captured.Manifest.ID, artifact.Path)
		if err != nil {
			return nil, err
		}
		artifactRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		artifactResponse, err := client.Do(artifactRequest)
		if err != nil {
			return nil, fmt.Errorf("fetch BAS %s artifact: %w", kind, err)
		}
		defer artifactResponse.Body.Close()
		if artifactResponse.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("BAS %s artifact returned HTTP %d", kind, artifactResponse.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(artifactResponse.Body, (5<<20)+1))
		if err != nil {
			return nil, fmt.Errorf("read BAS %s artifact: %w", kind, err)
		}
		if len(data) > 5<<20 || int64(len(data)) != artifact.SizeBytes {
			return nil, fmt.Errorf("BAS %s artifact size does not match its manifest", kind)
		}
		digest := sha256.Sum256(data)
		if !strings.EqualFold(hex.EncodeToString(digest[:]), artifact.SHA256) {
			return nil, fmt.Errorf("BAS %s artifact digest does not match its manifest", kind)
		}
		return data, nil
	}
	outlineBytes, err := readArtifact("outline")
	if err != nil {
		return nil, err
	}
	var pageOutline outline
	if err := json.Unmarshal(outlineBytes, &pageOutline); err != nil || pageOutline.Schema != "bas-agent-outline/v1" {
		return nil, fmt.Errorf("BAS outline artifact is invalid")
	}
	var identities identityArtifact
	if identityPath := artifacts["component-identity"]; identityPath.Path != "" {
		identityBytes, err := readArtifact("component-identity")
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(identityBytes, &identities); err != nil || identities.Schema != "bas-component-identity/v1" {
			return nil, fmt.Errorf("BAS component identity artifact is invalid")
		}
	} else if captured.Manifest.Lenses["L4"].Status == "present" {
		return nil, fmt.Errorf("BAS marked L4 present but omitted its component identity artifact")
	}
	requests := map[string]map[string]string{}
	for _, identity := range identities.Components {
		if identity.Asset != "" {
			requests[identity.Asset+"@"+identity.Version] = map[string]string{"asset": identity.Asset, "version": identity.Version}
		}
	}
	sources, err := s.resolveSources(ctx, requests)
	if err != nil {
		return nil, err
	}
	safeTarget := strings.TrimRight(ui, "/") + redactRouteQuery(req.Route)
	result := &pagev1.InspectResponse{Scenario: req.Scenario, Route: redactRouteQuery(req.Route), Url: safeTarget, ExecutionId: captured.Manifest.ID}
	if artifact := artifacts["pixels"]; artifact.Path != "" {
		result.ScreenshotUrl, err = captureArtifactURL(bas, captured.Manifest.ID, artifact.Path)
		if err != nil {
			return nil, err
		}
	}
	identityBySelector := map[string]componentIdentity{}
	for _, identity := range identities.Components {
		identityBySelector[identity.Selector] = identity
	}
	root := &pagev1.PageNode{Source: &pagev1.AssetSource{Status: "unstamped"}}
	root.Observation, _ = structpb.NewStruct(map[string]any{"kind": "outline", "schema": pageOutline.Schema})
	for _, node := range pageOutline.Nodes {
		result.NodeCount++
		observation := map[string]any{"id": node.ID, "kind": node.Kind, "tag": node.Tag, "role": node.Role, "name": node.Name, "text": node.Text, "selector": node.Selector, "rect": node.Rect, "disabled": node.Disabled}
		if node.Level != 0 {
			observation["level"] = node.Level
		}
		child := &pagev1.PageNode{Source: &pagev1.AssetSource{Status: "unstamped"}}
		if identity, ok := identityBySelector[node.Selector]; ok {
			result.StampedCount++
			observation["data"] = map[string]any{"rclAsset": identity.Asset, "rclVersion": identity.Version, "rclStamp": identity.Stamp, "rclSourceSlot": identity.SourceSlot}
			child.Source = sources[identity.Asset+"@"+identity.Version]
			if child.Source == nil {
				return nil, fmt.Errorf("source resolver omitted stamp %s@%s", identity.Asset, identity.Version)
			}
			if child.Source.Status == "resolved" {
				result.ResolvedCount++
			}
		}
		child.Observation, err = structpb.NewStruct(observation)
		if err != nil {
			return nil, err
		}
		root.Children = append(root.Children, child)
	}
	result.Tree = root
	if result.StampedCount > result.ResolvedCount {
		result.Warnings = append(result.Warnings, "Some stamped nodes could not resolve; inspect their source reasons.")
	}
	for lens, value := range captured.Manifest.Lenses {
		if value.Status != "present" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("BAS %s lens %s: %s", lens, value.Status, value.Reason))
		}
	}
	if len(pageOutline.Unsupported) > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("BAS outline reports %d unsupported page surface(s).", len(pageOutline.Unsupported)))
	}
	return result, nil
}

func captureArtifactURL(base, bundleID, artifactPath string) (string, error) {
	if !captureBundleID.MatchString(bundleID) {
		return "", fmt.Errorf("BAS returned an invalid UX bundle identity")
	}
	rel := filepath.Clean(filepath.FromSlash(artifactPath))
	if artifactPath == "" || strings.ContainsAny(artifactPath, "\\\x00") || filepath.IsAbs(rel) || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.ToSlash(rel) != artifactPath {
		return "", fmt.Errorf("BAS returned an unsafe UX artifact path")
	}
	segments := strings.Split(filepath.ToSlash(rel), "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	return strings.TrimRight(base, "/") + "/api/v1/ux-capture/bundles/" + url.PathEscape(bundleID) + "/artifacts/" + strings.Join(segments, "/"), nil
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
