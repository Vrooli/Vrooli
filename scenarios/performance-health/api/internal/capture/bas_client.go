package capture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"

	capturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture"
	captureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/capture/captureconnect"
)

const basScenarioID = "browser-automation-studio"

// componentMark is the prefix BAS's perf trace carries for React component
// commits (Tier 1). Its presence in the trace means ⚛ marks rode along.
const componentMark = "⚛"

// BASConnectClient is the production BASClient: it calls Browser Automation
// Studio's CaptureService with CAPTURE_TYPE_PERFORMANCE to produce a CDP trace +
// web-vitals (Tier 0), then inspects the returned trace for ⚛ component marks
// (Tier 1). BAS stays a dumb mechanism — tier meaning is derived here.
//
// An unavailable artifact retains BAS's reason for the orchestrator to classify.
// Operation errors retain their native cause; only service/discovery
// unavailability is classified as an unavailable capture mechanism.
type BASConnectClient struct {
	// Resolve maps the BAS scenario slug → its API base URL; nil uses the
	// discovery resolver.
	Resolve func(ctx context.Context) (string, error)

	// HTTPClient is the Connect transport; nil uses a default client.
	HTTPClient connect.HTTPClient

	// ReadTrace reads an artifact-owner URL for ⚛ detection; nil performs an
	// HTTP GET using the capture caller's context.
	ReadTrace func(path string) ([]byte, error)
	// ArtifactStore retains owner-downloaded files for the existing analyzer,
	// API, and CLI path contract. Production wiring uses SCENARIO_DATA_DIR.
	ArtifactStore ArtifactStore
}

// ArtifactStore writes durable BAS-owned bytes into Performance Health's own
// retained artifact space and returns a local path for existing consumers.
type ArtifactStore interface {
	Store(context.Context, string, string, []byte) (string, error)
}

// FileArtifactStore stores capture files below the scenario's declared data
// root, partitioned by exact BAS execution identity.
type FileArtifactStore struct{ Root string }

// NewBASConnectClient wires retained artifact staging to the standard scenario
// data directory supplied by the lifecycle manager.
func NewBASConnectClient() *BASConnectClient {
	return &BASConnectClient{ArtifactStore: FileArtifactStore{Root: strings.TrimSpace(os.Getenv("SCENARIO_DATA_DIR"))}}
}

func (s FileArtifactStore) Store(ctx context.Context, executionID, name string, raw []byte) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if strings.TrimSpace(s.Root) == "" {
		return "", errors.New("scenario data root is required")
	}
	if executionID == "" || filepath.Base(name) != name {
		return "", errors.New("invalid artifact identity or name")
	}
	dir := filepath.Join(s.Root, "performance-health", "captures", executionID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

var _ BASClient = (*BASConnectClient)(nil)

// BASRequestError retains a capture operation failure returned by BAS.
// An operation failure does not establish a missing capture mechanism.
type BASRequestError struct {
	Code connect.Code
	Err  error
}

func (e BASRequestError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("browser-automation-studio capture operation failed: %s", e.Code)
	}
	return fmt.Sprintf("browser-automation-studio capture operation failed: %s: %v", e.Code, e.Err)
}

func (e BASRequestError) Unwrap() error {
	return e.Err
}

// CapturePerf drives a BAS perf capture for url. interactionFlowJSON is a raw
// bas/flows-shape JSON body (a WorkflowDefinitionV2 protojson) that BAS splices
// after the navigate, inside the perf-trace window; empty uses BAS's default
// navigate+settle interaction.
func (c *BASConnectClient) CapturePerf(ctx context.Context, url, interactionFlowJSON string) (Artifacts, error) {
	if strings.TrimSpace(url) == "" {
		return Artifacts{}, errors.New("capture: url is required")
	}

	baseURL, err := c.resolve(ctx)
	if err != nil {
		// Discovery could not locate the capture service. Retain that cause;
		// this alone does not establish that the browser is absent.
		return Artifacts{}, fmt.Errorf("%w: resolve browser-automation-studio: %w", ErrCaptureUnavailable, err)
	}

	client := captureconnect.NewCaptureServiceClient(c.httpClient(), baseURL)
	resp, err := client.Capture(ctx, connect.NewRequest(&capturev1.CaptureRequest{
		Url:                 url,
		Captures:            []capturev1.CaptureType{capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE},
		Label:               "performance-health audit",
		InteractionFlowJson: interactionFlowJSON,
	}))
	if err != nil {
		code := connect.CodeOf(err)
		if code == connect.CodeUnavailable {
			return Artifacts{}, fmt.Errorf("%w: %w", ErrCaptureUnavailable, err)
		}
		return Artifacts{}, BASRequestError{Code: code, Err: err}
	}
	if resp == nil || resp.Msg == nil {
		return Artifacts{}, errors.New("browser-automation-studio returned an empty capture response")
	}

	return c.artifactsFromResponse(ctx, baseURL, resp.Msg)
}

// artifactsFromResponse resolves only owner-published artifact URLs. The
// BAS-local Path is deliberately ignored: it is meaningful only on the BAS
// host. Downloaded bytes are copied into PH-owned files because analysis and
// CLI consumers use the established filesystem artifact contract.
func (c *BASConnectClient) artifactsFromResponse(ctx context.Context, baseURL string, msg *capturev1.CaptureResponse) (Artifacts, error) {
	var art Artifacts
	type ownedArtifact struct {
		kind string
		url  string
		ref  string
		size int64
	}
	var trace, vitals ownedArtifact
	for _, a := range msg.GetArtifacts() {
		if a.GetType() != capturev1.CaptureType_CAPTURE_TYPE_PERFORMANCE {
			continue
		}
		if a.GetMetadata()["unavailable"] == "true" {
			// BAS was reached but could not produce a trace this run. Capture
			// the reason verbatim (no browser vs a transient capture failure)
			// so the orchestrator reports honestly instead of guessing.
			art.Unavailable = true
			if r := a.GetMetadata()["reason"]; r != "" {
				art.UnavailableReason = r
			}
			continue
		}
		switch a.GetMetadata()["artifact"] {
		case "cdp-trace":
			trace = ownedArtifact{kind: "cdp-trace", url: a.GetMetadata()["view_url"], ref: a.GetReference(), size: a.GetSizeBytes()}
		case "web-vitals":
			vitals = ownedArtifact{kind: "web-vitals", url: a.GetMetadata()["view_url"], ref: a.GetReference(), size: a.GetSizeBytes()}
		default:
			// Older shape: first non-vitals perf artifact is the trace.
			if trace.kind == "" {
				trace = ownedArtifact{kind: "cdp-trace", url: a.GetMetadata()["view_url"], ref: a.GetReference(), size: a.GetSizeBytes()}
			}
		}
	}
	if trace.kind == "" {
		return art, nil
	}
	traceURL, executionID, err := artifactOwnerURL(baseURL, trace.ref, trace.kind, trace.url)
	if err != nil {
		return art, fmt.Errorf("resolve BAS trace artifact URL: %w", err)
	}
	read := c.ReadTrace
	if read == nil {
		read = func(rawURL string) ([]byte, error) { return c.readOwnerArtifact(ctx, rawURL) }
	}
	traceBytes, err := read(traceURL)
	if err != nil {
		return art, fmt.Errorf("read BAS trace artifact: %w", err)
	}
	if trace.size > 0 && int64(len(traceBytes)) != trace.size {
		return art, fmt.Errorf("BAS trace size mismatch: declared %d, read %d", trace.size, len(traceBytes))
	}
	art.HasComponentMarks = traceHasComponentMarks(traceBytes)
	if c.ArtifactStore == nil {
		return Artifacts{}, errors.New("Performance Health artifact store is not configured")
	}
	tracePath, err := c.ArtifactStore.Store(ctx, executionID, "performance.json", traceBytes)
	if err != nil {
		return Artifacts{}, fmt.Errorf("persist PH trace artifact: %w", err)
	}
	art.TraceArtifact = tracePath
	art.TraceReference = trace.ref
	if vitals.kind != "" {
		vitalsURL, vitalsID, err := artifactOwnerURL(baseURL, vitals.ref, vitals.kind, vitals.url)
		if err != nil || vitalsID != executionID {
			if err == nil {
				err = errors.New("web-vitals execution identity differs from trace")
			}
			return Artifacts{}, fmt.Errorf("resolve BAS web-vitals artifact URL: %w", err)
		}
		vitalsBytes, err := read(vitalsURL)
		if err != nil {
			return Artifacts{}, fmt.Errorf("read BAS web-vitals artifact: %w", err)
		}
		if vitals.size > 0 && int64(len(vitalsBytes)) != vitals.size {
			return Artifacts{}, fmt.Errorf("BAS web-vitals size mismatch: declared %d, read %d", vitals.size, len(vitalsBytes))
		}
		vitalsPath, err := c.ArtifactStore.Store(ctx, executionID, "performance.web-vitals.json", vitalsBytes)
		if err != nil {
			return Artifacts{}, fmt.Errorf("persist PH web-vitals artifact: %w", err)
		}
		art.WebVitalsArtifact = vitalsPath
		art.WebVitalsReference = vitals.ref
	}
	return art, nil
}

// traceHasComponentMarks reports whether the trace contains React ⚛ component
// marks (Tier 1). On any read failure it returns false — capture never fails for
// lack of instrumentation; the result simply downgrades to Tier 0.
func traceHasComponentMarks(raw []byte) bool {
	// CDP commonly escapes the atom as \u269b. Inspect decoded event names,
	// not raw bytes or React scheduler track labels inside event arguments.
	type event struct {
		Name string `json:"name"`
	}
	var trace struct {
		Name   string  `json:"name"`
		Events []event `json:"traceEvents"`
	}
	var events []event
	if json.Unmarshal(raw, &trace) == nil {
		events = append(trace.Events, event{Name: trace.Name})
	} else if json.Unmarshal(raw, &events) != nil {
		return false
	}
	for _, entry := range events {
		if strings.HasPrefix(strings.TrimSpace(entry.Name), componentMark+" ") {
			return true
		}
	}
	return false
}

func (c *BASConnectClient) readOwnerArtifact(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("artifact owner returned HTTP %d", resp.StatusCode)
	}
	const maxArtifactBytes = 64 << 20
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxArtifactBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxArtifactBytes {
		return nil, fmt.Errorf("artifact exceeds %d-byte limit", maxArtifactBytes)
	}
	return raw, nil
}

func artifactOwnerURL(baseURL, reference, kind, rawURL string) (string, string, error) {
	ref, err := url.Parse(reference)
	if err != nil || ref.Scheme != "bas-capture" || !isExecutionID(ref.Host) || ref.RawQuery != "" || ref.Fragment != "" || strings.Trim(ref.Path, "/") != kind {
		return "", "", errors.New("missing or invalid BAS capture artifact reference")
	}
	owner, err := url.Parse(rawURL)
	if err != nil {
		return "", "", errors.New("invalid BAS artifact URL")
	}
	parts := strings.Split(strings.Trim(owner.Path, "/"), "/")
	if owner.IsAbs() || owner.Host != "" || owner.RawQuery != "" || owner.Fragment != "" || len(parts) != 6 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "artifacts" || parts[3] != "artifacts" || parts[4] == "" || parts[5] == "" || parts[5] == "." || parts[5] == ".." {
		return "", "", errors.New("BAS artifact URL is not a relative URL bound to the referenced execution")
	}
	// Storage object names encode the exact BAS execution UUID after the
	// duplicated route/object prefix; reject URL paths that select another run.
	if parts[4] != ref.Host {
		return "", "", errors.New("BAS artifact URL execution identity does not match its reference")
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", "", errors.New("invalid BAS service URL")
	}
	base.Path = strings.TrimRight(base.Path, "/") + owner.Path
	base.RawQuery, base.Fragment = "", ""
	return base.String(), ref.Host, nil
}

func isExecutionID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func (c *BASConnectClient) resolve(ctx context.Context) (string, error) {
	if c.Resolve != nil {
		return c.Resolve(ctx)
	}
	return discovery.NewResolver(discovery.ResolverConfig{}).ResolveScenarioURLDefault(ctx, basScenarioID)
}

func (c *BASConnectClient) httpClient() connect.HTTPClient {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 120 * time.Second}
}
