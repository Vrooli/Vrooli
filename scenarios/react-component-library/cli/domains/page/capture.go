package page

// `page capture` turns one served page into one bundle a person, an agent and a
// decision page can all use: pixels for every view, a frozen page per view that
// renders offline and can be edited, and a short outline. BAS owns the browser
// (ux-capture.v1); this command only asks it for each view and assembles the set.
// `page render` turns an edited frozen page back into verified pixels.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/cli-core/cliapp"

	repocontract "github.com/vrooli/repo-contract-go"
)

const captureSetSchema = "ux-capture-set.v1"

const maxFileCaptureBytes = 2 * 1024 * 1024

type view struct {
	Name          string `json:"name"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Theme         string `json:"theme"`
	BundleID      string `json:"bas_bundle_id,omitempty"`
	Pixels        string `json:"pixels,omitempty"`
	Page          string `json:"page,omitempty"`
	PixelsSHA256  string `json:"pixels_sha256,omitempty"`
	Lenses        lenses `json:"lenses,omitempty"`
	ThemeFollowed *bool  `json:"theme_followed,omitempty"`
}

type lens struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}
type lenses map[string]lens

type captureSet struct {
	Schema     string `json:"schema"`
	ID         string `json:"id"`
	CapturedAt string `json:"captured_at"`
	Source     struct {
		Kind     string   `json:"kind"`
		Scenario string   `json:"scenario,omitempty"`
		Route    string   `json:"route,omitempty"`
		URL      string   `json:"url,omitempty"`
		File     string   `json:"file,omitempty"`
		Steps    []string `json:"steps,omitempty"`
	} `json:"source"`
	Build   map[string]string `json:"build,omitempty"`
	Views   []view            `json:"views"`
	Outline string            `json:"outline,omitempty"`
	Failed  []string          `json:"failed,omitempty"`
}

var (
	viewPresets = map[string][2]int{"phone": {390, 844}, "small-phone": {320, 640}, "tablet": {820, 1180}, "desktop": {1280, 800}}
	sizePattern = regexp.MustCompile(`^(\d{3,4})x(\d{3,4})$`)
)

func parseViews(raw string) ([]view, error) {
	var out []view
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if size, ok := viewPresets[name]; ok {
			out = append(out, view{Name: name, Width: size[0], Height: size[1]})
			continue
		}
		match := sizePattern.FindStringSubmatch(name)
		if match == nil {
			return nil, fmt.Errorf("view %q: use phone, small-phone, tablet, desktop or WIDTHxHEIGHT", name)
		}
		width, _ := strconv.Atoi(match[1])
		height, _ := strconv.Atoi(match[2])
		out = append(out, view{Name: name, Width: width, Height: height})
	}
	if len(out) == 0 {
		return nil, errors.New("--views names no view")
	}
	return out, nil
}

func parseThemes(raw string) ([]string, error) {
	var out []string
	for _, theme := range strings.Split(raw, ",") {
		theme = strings.TrimSpace(theme)
		if theme != "light" && theme != "dark" {
			return nil, fmt.Errorf("theme %q: use light, dark or light,dark", theme)
		}
		out = append(out, theme)
	}
	return out, nil
}

// basClient talks to BAS's ux-capture HTTP surface.
type basClient struct {
	base string
	http *http.Client
}

type basResponse struct {
	Manifest struct {
		SchemaVersion string `json:"schema_version"`
		ID            string `json:"id"`
		Source        struct {
			Kind     string `json:"kind"`
			Scenario string `json:"scenario"`
			Route    string `json:"route"`
			URL      string `json:"url"`
			Filename string `json:"filename"`
		} `json:"source"`
		Session struct {
			ID string `json:"id"`
		} `json:"session"`
		Viewport struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"viewport"`
		Theme   string `json:"theme"`
		Privacy struct {
			RedactionStatus string `json:"redaction_status"`
			External        bool   `json:"external"`
			Ephemeral       bool   `json:"ephemeral"`
		} `json:"privacy"`
		Lenses    lenses `json:"lenses"`
		Artifacts []struct {
			Kind   string `json:"kind"`
			Path   string `json:"path"`
			Size   int64  `json:"size_bytes"`
			SHA256 string `json:"sha256"`
		} `json:"artifacts"`
	} `json:"manifest"`
}

func (c basClient) capture(ctx context.Context, kind string, request map[string]any) (basResponse, error) {
	var out basResponse
	body, err := json.Marshal(request)
	if err != nil {
		return out, fmt.Errorf("encode BAS capture request: %w", err)
	}
	if kind != "scenario" && kind != "public" && kind != "file" {
		return out, fmt.Errorf("unsupported BAS capture kind %q", kind)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/v1/ux-capture/"+kind, bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return out, fmt.Errorf("BAS capture: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("BAS capture HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("BAS returned invalid UX bundle JSON: %w", err)
	}
	if err := validateCaptureManifest(out, kind, request); err != nil {
		return out, err
	}
	return out, nil
}

func validateCaptureManifest(out basResponse, kind string, request map[string]any) error {
	m := out.Manifest
	if m.SchemaVersion != "ux-capture.v1" || !canonicalUUID(m.ID) || !canonicalUUID(m.Session.ID) {
		return errors.New("BAS returned no identity-bound ux-capture.v1 bundle")
	}
	if m.Privacy.RedactionStatus != "complete" || m.Privacy.External != (kind == "public") || m.Privacy.Ephemeral != (kind == "public") {
		return errors.New("BAS returned a bundle with mismatched redaction or retention policy")
	}
	if m.Viewport.Width != requestDimension(request, "width") || m.Viewport.Height != requestDimension(request, "height") {
		return errors.New("BAS returned a bundle for a different viewport")
	}
	theme, _ := request["theme"].(string)
	if theme == "" || m.Theme != theme {
		return errors.New("BAS returned a bundle for a different color theme")
	}
	for _, name := range []string{"L0", "L1", "L2", "L3", "L4", "L5", "L6"} {
		lens, ok := m.Lenses[name]
		if !ok || (lens.Status != "present" && lens.Status != "partial" && lens.Status != "absent") {
			return fmt.Errorf("BAS bundle has invalid %s lens status", name)
		}
		if lens.Status != "present" && strings.TrimSpace(lens.Reason) == "" {
			return fmt.Errorf("BAS bundle omits the %s lens reason", name)
		}
	}
	switch kind {
	case "scenario":
		scenario, _ := request["scenario"].(string)
		route, _ := request["route"].(string)
		if route == "" {
			route = "/"
		}
		if m.Source.Kind != "scenario" || m.Source.Scenario != scenario || m.Source.Route != redactRouteQuery(route) {
			return errors.New("BAS returned a bundle for a different scenario route")
		}
	case "public":
		rawURL, _ := request["url"].(string)
		actualURL, actualOK := redactedCaptureURL(m.Source.URL)
		expectedURL, expectedOK := redactedCaptureURL(rawURL)
		if m.Source.Kind != "url" || !actualOK || !expectedOK || actualURL != expectedURL {
			return errors.New("BAS returned a bundle for a different public URL")
		}
	case "file":
		filename, _ := request["filename"].(string)
		if m.Source.Kind != "file" || m.Source.Filename != filepath.Base(filename) {
			return errors.New("BAS returned a bundle for a different file")
		}
	}
	seen := make(map[string]struct{}, len(m.Artifacts))
	for _, artifact := range m.Artifacts {
		if !safeArtifactPath(artifact.Path) || artifact.Size < 0 || artifact.Size > 32<<20 || len(artifact.SHA256) != 64 {
			return errors.New("BAS returned an invalid or unbounded artifact declaration")
		}
		if _, err := hex.DecodeString(artifact.SHA256); err != nil {
			return errors.New("BAS returned an invalid artifact digest")
		}
		if _, exists := seen[artifact.Path]; exists {
			return fmt.Errorf("BAS declared duplicate artifact path %q", artifact.Path)
		}
		seen[artifact.Path] = struct{}{}
	}
	return nil
}

func requestDimension(request map[string]any, key string) int {
	value, ok := request[key].(int)
	if !ok {
		return 0
	}
	return value
}

func canonicalUUID(value string) bool {
	if len(value) != 36 || strings.ToLower(value) != value {
		return false
	}
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func safeArtifactPath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\\x00") || strings.HasPrefix(value, "/") {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

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
	return parsed.Path + "?" + parsed.RawQuery
}

func redactedCaptureURL(raw string) (string, bool) {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return "", false
	}
	query := parsed.Query()
	for key := range query {
		query.Set(key, "[REDACTED]")
	}
	parsed.RawQuery = query.Encode()
	if parsed.Fragment != "" {
		parsed.Fragment = "[REDACTED]"
	}
	parsed.User = nil
	return parsed.String(), true
}

func (c basClient) artifact(ctx context.Context, bundle basResponse, kind string) ([]byte, error) {
	var selected *struct {
		Kind   string `json:"kind"`
		Path   string `json:"path"`
		Size   int64  `json:"size_bytes"`
		SHA256 string `json:"sha256"`
	}
	for i := range bundle.Manifest.Artifacts {
		artifact := &bundle.Manifest.Artifacts[i]
		if artifact.Kind == kind {
			selected = artifact
			break
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("BAS bundle has no %s artifact", kind)
	}
	if !safeArtifactPath(selected.Path) || selected.Size < 0 || selected.Size > 32<<20 {
		return nil, fmt.Errorf("BAS %s artifact is unsafe or exceeds 32 MiB", kind)
	}
	parts := strings.Split(selected.Path, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	artifactURL := c.base + "/api/v1/ux-capture/bundles/" + url.PathEscape(bundle.Manifest.ID) + "/artifacts/" + strings.Join(parts, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("artifact %s: HTTP %d", kind, resp.StatusCode)
	}
	expectedMediaType := map[string]string{"pixels": "image/png", "frozen-page": "text/html", "outline": "application/json"}[kind]
	mediaType, _, mediaErr := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if expectedMediaType == "" || mediaErr != nil || mediaType != expectedMediaType {
		return nil, fmt.Errorf("artifact %s has an unexpected media type", kind)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, (32<<20)+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) != selected.Size || len(content) > 32<<20 {
		return nil, fmt.Errorf("artifact %s size does not match BAS manifest", kind)
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != strings.ToLower(selected.SHA256) {
		return nil, fmt.Errorf("artifact %s digest does not match BAS manifest", kind)
	}
	return content, nil
}

// scenarioPorts asks the control plane, which owns ports, where a scenario serves.
var scenarioPorts = func(ctx context.Context, name string) (map[string]int, string, error) {
	out, err := exec.CommandContext(ctx, "vrooli", "scenario", "status", name, "--json").Output() // #nosec G204 -- fixed control-plane verb
	if err != nil {
		return nil, "", fmt.Errorf("vrooli scenario status %s: %w", name, err)
	}
	var status struct {
		Scenario struct {
			Status    string         `json:"status"`
			Ports     map[string]int `json:"ports"`
			StartedAt string         `json:"started_at"`
		} `json:"scenario"`
	}
	if err := json.Unmarshal(out, &status); err != nil || status.Scenario.Status != "running" {
		return nil, "", fmt.Errorf("scenario %s is not running", name)
	}
	return status.Scenario.Ports, status.Scenario.StartedAt, nil
}

func newBASClient(ctx context.Context) (basClient, error) {
	base := strings.TrimRight(os.Getenv("RCL_BAS_API_BASE"), "/")
	if base == "" {
		ports, _, err := scenarioPorts(ctx, "browser-automation-studio")
		if err != nil {
			return basClient{}, fmt.Errorf("BAS owns the browser for captures: %w", err)
		}
		base = "http://127.0.0.1:" + strconv.Itoa(ports["API_PORT"])
	}
	return basClient{base: base, http: &http.Client{Timeout: 150 * time.Second}}, nil
}

func captureRoot() string {
	if root := os.Getenv("RCL_UX_CAPTURE_ROOT"); root != "" {
		return root
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return ""
	}
	return filepath.Join(root, "data", "react-component-library", "ux-captures")
}

func slug(value string) string {
	value = strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(value, "-"))
	value = strings.Trim(value, "-")
	if len(value) > 32 {
		value = strings.Trim(value[:32], "-")
	}
	if value == "" {
		return "page"
	}
	return value
}

func newCaptureID(subject string, now time.Time) string {
	suffix := make([]byte, 2)
	_, _ = rand.Read(suffix)
	return "uxc-" + slug(subject) + "-" + now.UTC().Format("20060102-150405") + "-" + hex.EncodeToString(suffix)
}

// buildIdentity fingerprints what was served: the root document names its
// content-hashed bundles, so its digest changes exactly when the build does.
func buildIdentity(ctx context.Context, pageURL, startedAt string) map[string]string {
	build := map[string]string{}
	if startedAt != "" {
		build["served_since"] = startedAt
	}
	if pageURL == "" {
		return build
	}
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return build
	}
	resp, err := client.Do(req)
	if err != nil {
		return build
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	sum := sha256.Sum256(body)
	build["document_sha256"] = hex.EncodeToString(sum[:])[:16]
	assets := regexp.MustCompile(`(?:src|href)="(/[^"]*assets/[^"]+\.(?:js|css))"`).FindAllStringSubmatch(string(body), 4)
	var names []string
	for _, match := range assets {
		names = append(names, filepath.Base(match[1]))
	}
	if len(names) > 0 {
		build["assets"] = strings.Join(names, " ")
	}
	return build
}

func (h captureHandlers) capture(rc cliapp.RunContext) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	scenario, rawURL, file := strings.TrimSpace(rc.Flag("scenario")), strings.TrimSpace(rc.Flag("url")), strings.TrimSpace(rc.Flag("file"))
	given := 0
	for _, value := range []string{scenario, rawURL, file} {
		if value != "" {
			given++
		}
	}
	if given != 1 {
		return errors.New("give exactly one of --scenario, --url or --file")
	}
	views, err := parseViews(defaultString(rc.Flag("views"), "phone,desktop"))
	if err != nil {
		return err
	}
	themes, err := parseThemes(defaultString(rc.Flag("themes"), "light,dark"))
	if err != nil {
		return err
	}
	bas, err := h.bas(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	set := captureSet{Schema: captureSetSchema, CapturedAt: now.UTC().Format(time.RFC3339)}
	route := defaultString(rc.Flag("route"), "/")
	// Every branch below assigns both; declaring them avoids an ineffectual
	// initial value while keeping the switch readable.
	var kind string
	var request map[string]any
	var html []byte
	switch {
	case scenario != "":
		ports, startedAt, err := scenarioPorts(ctx, scenario)
		if err != nil {
			return err
		}
		kind = "scenario"
		set.Source.Kind, set.Source.Scenario, set.Source.Route, set.Source.Steps = "scenario", scenario, route, rc.FlagValues("step")
		set.Source.URL = "http://localhost:" + strconv.Itoa(ports["UI_PORT"]) + route
		set.Build = buildIdentity(ctx, set.Source.URL, startedAt)
		request = map[string]any{"scenario": scenario, "route": route, "steps": rc.FlagValues("step")}
		if selector := strings.TrimSpace(rc.Flag("wait-selector")); selector != "" {
			request["wait_selector"] = selector
		}
		set.ID = newCaptureID(scenario, now)
	case rawURL != "":
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
			return fmt.Errorf("--url must be an absolute http(s) URL")
		}
		kind = "public"
		set.Source.Kind, set.Source.URL = "url", rawURL
		set.Build = buildIdentity(ctx, rawURL, "")
		request = map[string]any{"url": rawURL}
		set.ID = newCaptureID(parsed.Hostname(), now)
	default:
		input, openErr := os.Open(file)
		if openErr != nil {
			return openErr
		}
		html, err = io.ReadAll(io.LimitReader(input, maxFileCaptureBytes+1))
		closeErr := input.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		if len(html) == 0 || len(html) > maxFileCaptureBytes {
			return fmt.Errorf("HTML file must contain between 1 byte and 2 MiB")
		}
		kind = "file"
		absolute, _ := filepath.Abs(file)
		sum := sha256.Sum256(html)
		set.Source.Kind, set.Source.File = "file", absolute
		set.Build = map[string]string{"document_sha256": hex.EncodeToString(sum[:])[:16]}
		request = map[string]any{"filename": filepath.Base(file), "html": string(html)}
		set.ID = newCaptureID(strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)), now)
	}
	out := defaultString(rc.Flag("out"), filepath.Join(captureRoot(), set.ID))
	for _, dir := range []string{"pixels", "page", "outline"} {
		if err := os.MkdirAll(filepath.Join(out, dir), 0o755); err != nil {
			return err
		}
	}
	lightPixels := map[string]string{}
	for _, base := range views {
		for _, theme := range themes {
			v := base
			v.Theme = theme
			v.Name = base.Name + "-" + theme
			request["width"], request["height"], request["theme"] = v.Width, v.Height, theme
			response, err := bas.capture(ctx, kind, request)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				set.Failed = append(set.Failed, v.Name+": "+err.Error())
				continue
			}
			v.BundleID, v.Lenses = response.Manifest.ID, response.Manifest.Lenses
			if pixels, err := bas.artifact(ctx, response, "pixels"); err == nil && len(pixels) > 0 {
				v.Pixels = "pixels/" + v.Name + ".png"
				sum := sha256.Sum256(pixels)
				v.PixelsSHA256 = hex.EncodeToString(sum[:])
				if err := os.WriteFile(filepath.Join(out, v.Pixels), pixels, 0o644); err != nil {
					return err
				}
			} else {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				set.Failed = append(set.Failed, v.Name+": pixels: "+err.Error())
			}
			if v.Lenses["L0"].Status != "present" || v.Pixels == "" {
				set.Failed = append(set.Failed, fmt.Sprintf("%s: no usable pixels (L0 %s: %s)", v.Name, v.Lenses["L0"].Status, v.Lenses["L0"].Reason))
			}
			if frozen, err := bas.artifact(ctx, response, "frozen-page"); err == nil && len(frozen) > 0 {
				v.Page = "page/" + v.Name + ".html"
				if err := os.WriteFile(filepath.Join(out, v.Page), frozen, 0o644); err != nil {
					return err
				}
			} else {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				set.Failed = append(set.Failed, v.Name+": frozen-page: "+err.Error())
			}
			if outline, err := bas.artifact(ctx, response, "outline"); err == nil && len(outline) > 0 {
				_ = os.WriteFile(filepath.Join(out, "outline", v.Name+".json"), outline, 0o644)
				if set.Outline == "" {
					set.Outline = "outline.md"
					_ = os.WriteFile(filepath.Join(out, set.Outline), []byte(renderOutline(outline, v)), 0o644)
				}
			} else {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				set.Failed = append(set.Failed, v.Name+": outline: "+err.Error())
			}
			// A page that ignores the system theme paints the same pixels for
			// both; say so instead of presenting a "dark" view that is not one.
			if theme == "light" {
				lightPixels[base.Name] = v.PixelsSHA256
			} else if light := lightPixels[base.Name]; light != "" && v.PixelsSHA256 != "" {
				followed := light != v.PixelsSHA256
				v.ThemeFollowed = &followed
			}
			set.Views = append(set.Views, v)
		}
	}
	manifest, _ := json.MarshalIndent(set, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "capture.json"), append(manifest, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(out, "README.md"), []byte(renderReadme(set, out)), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(rc.Stdout(), "Capture %s → %s\n", set.ID, out)
	for _, v := range set.Views {
		fmt.Fprintf(rc.Stdout(), "  %-16s L0 %-7s L1 %-7s %s\n", v.Name, v.Lenses["L0"].Status, v.Lenses["L1"].Status, v.Pixels)
	}
	if len(set.Failed) > 0 {
		return fmt.Errorf("capture incomplete (%d view(s)); see %s:\n  %s", len(set.Failed), filepath.Join(out, "README.md"), strings.Join(set.Failed, "\n  "))
	}
	return nil
}

func (h captureHandlers) render(rc cliapp.RunContext) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	file := rc.Positional("file")
	html, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	views, err := parseViews(defaultString(rc.Flag("views"), "phone"))
	if err != nil {
		return err
	}
	themes, err := parseThemes(defaultString(rc.Flag("themes"), "light"))
	if err != nil {
		return err
	}
	bas, err := h.bas(ctx)
	if err != nil {
		return err
	}
	prefix := defaultString(rc.Flag("out"), strings.TrimSuffix(file, filepath.Ext(file)))
	var failed []string
	for _, v := range views {
		for _, theme := range themes {
			response, err := bas.capture(ctx, "file", map[string]any{"filename": filepath.Base(file), "html": string(html), "width": v.Width, "height": v.Height, "theme": theme})
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				failed = append(failed, err.Error())
				continue
			}
			l0 := response.Manifest.Lenses["L0"]
			pixels, err := bas.artifact(ctx, response, "pixels")
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				failed = append(failed, err.Error())
				continue
			}
			if l0.Status != "present" || len(pixels) == 0 {
				failed = append(failed, fmt.Sprintf("%s-%s: L0 %s: %s", v.Name, theme, l0.Status, l0.Reason))
				continue
			}
			target := fmt.Sprintf("%s.%s-%s.png", prefix, v.Name, theme)
			if err := os.WriteFile(target, pixels, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(rc.Stdout(), "%s (BAS bundle %s)\n", target, response.Manifest.ID)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("render failed:\n  %s", strings.Join(failed, "\n  "))
	}
	return nil
}

type outlineNode struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Tag        string `json:"tag"`
	Name       string `json:"name"`
	Text       string `json:"text"`
	Level      int    `json:"level"`
	ActionType string `json:"actionType"`
	Disabled   bool   `json:"disabled"`
	Rect       struct {
		X, Y, Width, Height int
	} `json:"rect"`
	Examples []struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"examples"`
}

// renderOutline writes the outline an agent reads first: one line per node,
// in page order, with the id that also marks the element in the frozen page.
func renderOutline(raw []byte, v view) string {
	var outline struct {
		Nodes       []outlineNode `json:"nodes"`
		ActionCount int           `json:"actionCount"`
		Unsupported []struct {
			Surface string `json:"surface"`
			Reason  string `json:"reason"`
		} `json:"unsupported"`
	}
	_ = json.Unmarshal(raw, &outline)
	nodes := outline.Nodes
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Rect.Y != nodes[j].Rect.Y {
			return nodes[i].Rect.Y < nodes[j].Rect.Y
		}
		return nodes[i].Rect.X < nodes[j].Rect.X
	})
	var b strings.Builder
	fmt.Fprintf(&b, "# Outline · %s (%d×%d, %s)\n\n", v.Name, v.Width, v.Height, v.Theme)
	fmt.Fprintf(&b, "%d nodes, %d actions, top to bottom. Each id marks its element in the frozen page: `data-ux-node~=\"n12\"`.\n\n", len(nodes), outline.ActionCount)
	for _, n := range nodes {
		label := n.Name
		if label == "" {
			label = n.Text
		}
		if len(label) > 80 {
			label = label[:77] + "…"
		}
		detail := n.Kind
		switch {
		case n.Kind == "heading" && n.Level > 0:
			detail = fmt.Sprintf("h%d", n.Level)
		case n.Kind == "action":
			detail = "action/" + n.ActionType
			if n.Disabled {
				detail += " (disabled)"
			}
		case n.Kind == "landmark" || n.Kind == "region":
			detail = n.Kind + " <" + n.Tag + ">"
		case n.Kind == "repeated-template":
			detail = fmt.Sprintf("list of %s ×%d", n.Tag, len(n.Examples))
		}
		fmt.Fprintf(&b, "- `%s` %s %q @%d,%d %d×%d\n", n.ID, detail, label, n.Rect.X, n.Rect.Y, n.Rect.Width, n.Rect.Height)
	}
	for _, u := range outline.Unsupported {
		fmt.Fprintf(&b, "\nNot in the outline: %s (%s)", u.Surface, u.Reason)
	}
	return b.String() + "\n"
}

func renderReadme(set captureSet, dir string) string {
	var b strings.Builder
	source := set.Source.URL
	if set.Source.Kind == "file" {
		source = set.Source.File
	}
	fmt.Fprintf(&b, "# UX capture `%s`\n\n- Source (%s): %s\n- Captured: %s\n", set.ID, set.Source.Kind, source, set.CapturedAt)
	if len(set.Source.Steps) > 0 {
		fmt.Fprintf(&b, "- State steps: %s\n", strings.Join(set.Source.Steps, ", "))
	}
	keys := make([]string, 0, len(set.Build))
	for key := range set.Build {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "- Build %s: `%s`\n", key, set.Build[key])
	}
	b.WriteString("\n## Fidelity\n\n| View | Pixels (L0) | Frozen page (L1) | Outline (L2) | Theme | BAS bundle |\n|---|---|---|---|---|---|\n")
	for _, v := range set.Views {
		theme := v.Theme
		if v.ThemeFollowed != nil && !*v.ThemeFollowed {
			theme += " — same pixels as light: the page ignores the system theme"
		}
		fmt.Fprintf(&b, "| %s %d×%d | %s | %s | %s | %s | `%s` |\n", v.Name, v.Width, v.Height, lensCell(v.Lenses["L0"]), lensCell(v.Lenses["L1"]), lensCell(v.Lenses["L2"]), theme, v.BundleID)
	}
	if len(set.Failed) > 0 {
		b.WriteString("\n**Incomplete.** Do not present a missing view as captured:\n\n")
		for _, failure := range set.Failed {
			fmt.Fprintf(&b, "- %s\n", failure)
		}
	}
	// Next steps name a view this bundle actually holds. A frozen page keeps the
	// theme it was captured in, so each theme's page is edited and rendered alone.
	if len(set.Views) == 0 {
		return b.String()
	}
	first := set.Views[0]
	page, preset := first.Name, strings.TrimSuffix(first.Name, "-"+first.Theme)
	fmt.Fprintf(&b, "\n## Next\n\n"+
		"- Read `outline.md` to find a region's id, then edit a copy of the frozen page: `cp page/%[1]s.html proposal-%[3]s.html`. The element is the one with `data-ux-node~=\"<id>\"`. Repeat for each theme's page.\n"+
		"- Render the edit with verified pixels: `react-component-library page render proposal-%[3]s.html --views %[2]s --themes %[3]s`.\n"+
		"- Label images honestly: these pixels are `captured`; an edited frozen page is `composited`.\n"+
		"- Put the before and after on a decision page: skill `ux-capture` in react-component-library.\n\nBundle directory: `%[4]s`\n", page, preset, first.Theme, dir)
	return b.String()
}

func lensCell(l lens) string {
	if l.Status == "" {
		return "—"
	}
	if l.Reason == "" {
		return l.Status
	}
	reason := l.Reason
	if len(reason) > 90 {
		reason = reason[:87] + "…"
	}
	return l.Status + " (" + strings.ReplaceAll(reason, "|", "/") + ")"
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

type captureHandlers struct {
	bas func(context.Context) (basClient, error)
}
