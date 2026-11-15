package capture

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/repo-contract-go"
	visualv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

type uxArtifact struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Size   int64  `json:"size_bytes"`
	SHA256 string `json:"sha256"`
}
type uxManifest struct {
	SchemaVersion string `json:"schema_version"`
	ID            string `json:"id"`
	Source        struct {
		Scenario string `json:"scenario"`
		Route    string `json:"route"`
	} `json:"source"`
	Viewport struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport"`
	Lenses map[string]struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"lenses"`
	Privacy struct {
		RedactionStatus string `json:"redaction_status"`
	} `json:"privacy"`
	Artifacts []uxArtifact `json:"artifacts"`
}
type uxResponse struct {
	BundleDir string     `json:"bundle_dir"`
	Manifest  uxManifest `json:"manifest"`
}

// captureUXBundle calls the canonical one-session BAS entry and only reads files
// declared in its manifest. Each file is confined to the configured BAS root and
// checked against both manifest size and SHA-256 before it reaches analysis.
func (h *handlers) captureUXBundle(ctx context.Context, basURL, scenario, route, theme string, viewport string, strictTokens bool) (uxResponse, map[string][]byte, error) {
	width, height := dimensionsForBundle(viewport)
	body, _ := json.Marshal(map[string]any{"scenario": scenario, "route": route, "steps": []string{}, "width": width, "height": height, "theme": theme, "strict_tokens": strictTokens})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(basURL, "/")+"/api/v1/ux-capture/scenario", bytes.NewReader(body))
	if err != nil {
		return uxResponse{}, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.http.Do(req)
	if err != nil {
		return uxResponse{}, nil, fmt.Errorf("BAS ux capture: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return uxResponse{}, nil, fmt.Errorf("BAS ux capture returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var result uxResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, nil, fmt.Errorf("decode BAS UX bundle: %w", err)
	}
	if result.Manifest.SchemaVersion != "ux-capture.v1" || result.Manifest.Source.Scenario != scenario {
		return result, nil, errors.New("BAS returned an incompatible or mismatched UX bundle")
	}
	if len(result.Manifest.Lenses) != 7 {
		return result, nil, errors.New("BAS UX manifest must report all L0-L6 lens states")
	}
	for _, name := range []string{"L0", "L1", "L2", "L3", "L4", "L5", "L6"} {
		lens, ok := result.Manifest.Lenses[name]
		if !ok || (lens.Status != "present" && lens.Status != "partial" && lens.Status != "absent") {
			return result, nil, fmt.Errorf("BAS UX manifest has invalid %s lens status", name)
		}
		if lens.Status != "present" && strings.TrimSpace(lens.Reason) == "" {
			return result, nil, fmt.Errorf("BAS UX manifest omits the %s lens reason", name)
		}
	}
	if result.Manifest.Privacy.RedactionStatus != "complete" {
		return result, nil, fmt.Errorf("BAS UX bundle redaction is %q; refusing to analyze it", result.Manifest.Privacy.RedactionStatus)
	}
	root, err := captureRoot()
	if err != nil {
		return result, nil, err
	}
	dir, err := confinedBundleDir(root, result.BundleDir)
	if err != nil {
		return result, nil, err
	}
	files := map[string][]byte{}
	for _, a := range result.Manifest.Artifacts {
		p, err := confinedArtifact(dir, a.Path)
		if err != nil {
			return result, nil, err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return result, nil, fmt.Errorf("read UX artifact %s: %w", a.Path, err)
		}
		sum := sha256.Sum256(b)
		if int64(len(b)) != a.Size || !strings.EqualFold(hex.EncodeToString(sum[:]), a.SHA256) {
			return result, nil, fmt.Errorf("UX artifact %s failed manifest integrity check", a.Path)
		}
		files[filepath.ToSlash(a.Path)] = b
	}
	return result, files, nil
}

func captureRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv("BAS_CAPTURES_ROOT"))
	if root == "" {
		repo, err := repocontract.ResolveRepoRoot()
		if err != nil {
			return "", err
		}
		root = filepath.Join(repo, "scenarios", "browser-automation-studio", "data", "captures", "ux")
	} else {
		root = filepath.Join(root, "ux")
	}
	return filepath.Abs(filepath.Clean(root))
}

func confinedBundleDir(root, bundle string) (string, error) {
	if !filepath.IsAbs(bundle) {
		return "", errors.New("BAS bundle_dir must be absolute")
	}
	clean := filepath.Clean(bundle)
	rel, err := filepath.Rel(root, clean)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("BAS bundle_dir escapes capture root")
	}
	if err := rejectSymlinkPath(root, clean); err != nil {
		return "", err
	}
	st, err := os.Stat(clean)
	if err != nil || !st.IsDir() {
		return "", errors.New("BAS bundle_dir is not a directory")
	}
	return clean, nil
}

func confinedArtifact(dir, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || strings.Contains(relative, `\`) {
		return "", errors.New("unsafe UX artifact path")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("UX artifact path escapes bundle")
	}
	p := filepath.Join(dir, clean)
	rel, err := filepath.Rel(dir, p)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", errors.New("UX artifact path escapes bundle")
	}
	if err := rejectSymlinkPath(dir, p); err != nil {
		return "", err
	}
	return p, nil
}

func rejectSymlinkPath(root, target string) error {
	root = filepath.Clean(root)
	volume := filepath.VolumeName(root)
	current := volume + string(filepath.Separator)
	for _, part := range strings.Split(strings.TrimPrefix(root, current), string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		st, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in UX capture root %s", current)
		}
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	current = root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		st, e := os.Lstat(current)
		if e != nil {
			return e
		}
		if st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in UX bundle path %s", current)
		}
	}
	return nil
}

func dimensionsForBundle(viewport string) (int, int) {
	switch viewport {
	case "mobile":
		return 390, 844
	case "tablet":
		return 820, 1180
	default:
		return 1280, 800
	}
}

// ex9Layout adapts the privacy-safe outline and style census to the subset of
// UI Health's historic layout contract used by its visual analyzer.
func ex9Layout(man uxManifest, outlineRaw, styleRaw []byte) string {
	var outline struct {
		Nodes []struct {
			Kind, Tag, Name, Text, Selector string
			Role                            *string
			Rect                            struct{ X, Y, Width, Height float64 }
			Disabled                        bool   `json:"disabled"`
			ActionType                      string `json:"actionType"`
		} `json:"nodes"`
	}
	_ = json.Unmarshal(outlineRaw, &outline)
	var style struct {
		Clusters []struct {
			Category, Property, Value string
			Sources                   []struct{ Selector, Tag string }
		} `json:"clusters"`
	}
	_ = json.Unmarshal(styleRaw, &style)
	elements := make([]map[string]any, 0, len(outline.Nodes))
	for _, n := range outline.Nodes {
		role := ""
		if n.Role != nil {
			role = *n.Role
		}
		text := n.Text
		if text == "" {
			text = n.Name
		}
		nodeType := "element"
		if n.Kind == "action" {
			nodeType = "action"
		}
		elements = append(elements, map[string]any{"selector": n.Selector, "tag": n.Tag, "role": role, "text": text, "type": n.ActionType, "x": n.Rect.X, "y": n.Rect.Y, "width": n.Rect.Width, "height": n.Rect.Height, "disabled": n.Disabled, "interactive": n.Kind == "action", "nodeType": nodeType})
	}
	tokens := make([]string, 0)
	for _, c := range style.Clusters {
		tokens = append(tokens, c.Property+":"+c.Value)
	}
	data, _ := json.Marshal(map[string]any{"viewport": map[string]any{"width": man.Viewport.Width, "height": man.Viewport.Height}, "document": map[string]any{"width": man.Viewport.Width, "height": man.Viewport.Height}, "capture": map[string]any{"kind": "browser_viewport"}, "elements": elements, "tokens": tokens})
	return string(data)
}

func analyzeUXBundle(ctx context.Context, visual visualconnect.VisualHealthServiceClient, scenario, route, viewport, theme string, man uxManifest, files map[string][]byte) ([]finding, error) {
	var pixels []byte
	for k, b := range files {
		if strings.HasPrefix(k, "pixels/") && strings.Contains(k, theme+"-") {
			pixels = b
			break
		}
	}
	layout := ex9Layout(man, files["outline/outline.json"], files["style/census.json"])
	resp, err := visual.AnalyzeArtifacts(ctx, connect.NewRequest(&visualv1.AnalyzeArtifactsRequest{Scenario: "ui-health", RunId: man.ID, Steps: []*visualv1.VisualStepArtifact{{StepId: "capture", Label: viewport + " " + theme, Url: route, Viewport: viewportFor(viewport), LayoutJson: layout, ScreenshotPng: pixels, DomHtml: string(files["page/frozen.html"])}}}))
	if err != nil {
		return nil, err
	}
	out := []finding{}
	for _, s := range resp.Msg.GetSteps() {
		for _, f := range s.GetFindings() {
			out = append(out, finding{Rule: f.GetCode(), Severity: f.GetSeverity().String(), Selector: f.GetLocation(), Message: f.GetMessage(), Viewport: viewport, Theme: theme})
		}
	}
	return out, nil
}
