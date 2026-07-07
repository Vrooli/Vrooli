package uiruntime

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"

	"ui-health/internal/evidence"
	"ui-health/internal/services/manifestvalidation"
	"ui-health/internal/visualhealth"
)

const basScenarioID = "browser-automation-studio"

type viewportProfile struct {
	id     string
	width  int
	height int
}

var runtimeViewportProfiles = []viewportProfile{
	{id: "desktop", width: defaultViewportWidth, height: defaultViewportHeight},
	{id: "mobile", width: 390, height: 844},
}

// Runner is the production runtime/render Checker. It resolves the target UI,
// drives the iframe-bridge handshake workflow on BAS, and maps the verdict to
// findings. Missing infrastructure degrades to skipped findings.
type Runner struct {
	bas basRunner
	// resolveUI returns the target scenario's running UI base URL (or an error /
	// empty when it is not up).
	resolveUI func(ctx context.Context, scenario string) (string, error)
	log       *log.Logger
}

// New constructs the production runtime checker over BAS + discovery.
func New(logger *log.Logger) *Runner {
	if logger == nil {
		logger = log.Default()
	}
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	return &Runner{
		bas: &connectRunner{
			resolveBAS: func(ctx context.Context) (string, error) {
				return resolver.ResolveScenarioURLDefault(ctx, basScenarioID)
			},
			httpClient:   &http.Client{Timeout: 120 * time.Second},
			pollInterval: time.Second,
			pollTimeout:  90 * time.Second,
		},
		resolveUI: func(ctx context.Context, scenario string) (string, error) {
			return resolver.ResolveScenarioURL(ctx, scenario, "UI_PORT")
		},
		log: logger,
	}
}

// Check runs the runtime/render group. It never returns an error; infra absence
// becomes a skipped finding so the single report still carries the static groups.
func (r *Runner) Check(ctx context.Context, in Input) []manifestvalidation.Finding {
	url, err := r.resolveUI(ctx, in.Scenario)
	if err != nil || strings.TrimSpace(url) == "" {
		return []manifestvalidation.Finding{skip(
			"runtime_skipped_ui_unavailable",
			in.ScenarioDir,
			fmt.Sprintf("runtime render skipped: target UI for %q is not running/resolvable (%s)", in.Scenario, errText(err)),
		)}
	}

	var all []manifestvalidation.Finding
	for _, profile := range runtimeViewportProfiles {
		def := buildHandshakeWorkflow(url, nil, 0, profile.width, profile.height)
		res, err := r.bas.Run(ctx, def)
		if err != nil {
			return []manifestvalidation.Finding{skip(
				"runtime_skipped_bas_unavailable",
				url,
				"runtime render skipped: browser-automation-studio is unavailable; static checks still ran",
			)}
		}
		ev := res.evidenceFor(url)
		visualFinds := applyVisualHealth(&ev, res.visualStep(url, profile.id))
		finds := findingsFromEvidence(ev, profile.id)
		all = append(all, finds...)
		all = append(all, visualFinds...)
	}
	return all
}

func (r *runResult) visualStep(url, profileID string) *visualpb.VisualStepArtifact {
	stepID := "runtime-render"
	if profileID != "" {
		stepID += "-" + profileID
	}
	if r == nil {
		return &visualpb.VisualStepArtifact{StepId: stepID, Url: url}
	}
	step := &visualpb.VisualStepArtifact{
		StepId:        stepID,
		Url:           url,
		ScreenshotPng: r.screenshotPNG,
		DomHtml:       r.domHTML,
		LayoutJson:    r.layoutJSON,
		ScreenshotRef: &visualpb.ArtifactRef{Uri: r.screenshotRef},
	}
	if r.viewportWidth > 0 || r.viewportHeight > 0 {
		step.Viewport = &visualpb.Viewport{Width: r.viewportWidth, Height: r.viewportHeight}
	}
	for _, entry := range r.network {
		status := int32(0)
		if entry.Status != nil {
			status = int32(*entry.Status)
		}
		step.Network = append(step.Network, &visualpb.NetworkEntry{
			Url:          entry.URL,
			Method:       entry.Method,
			ResourceType: entry.ResourceType,
			Status:       status,
			ErrorText:    entry.ErrorText,
		})
	}
	return step
}

func applyVisualHealth(ev *evidence.Evidence, step *visualpb.VisualStepArtifact) []manifestvalidation.Finding {
	if ev == nil || step == nil {
		return nil
	}
	resp := visualhealth.DefaultAnalyzer().Analyze(&visualpb.AnalyzeArtifactsRequest{
		Steps: []*visualpb.VisualStepArtifact{step},
	})
	var details []manifestvalidation.Finding
	for _, finding := range resp.GetFindings() {
		if finding.GetSeverity() == visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR && !ev.RenderBroken {
			ev.RenderBroken = true
			ev.RenderBrokenReason = firstNonEmpty(finding.GetEvidence(), finding.GetMessage())
		}
		details = append(details, visualFindingToManifest(finding))
	}
	return details
}

func visualFindingToManifest(finding *visualpb.VisualFinding) manifestvalidation.Finding {
	if finding == nil {
		return manifestvalidation.Finding{}
	}
	severity := manifestvalidation.SeverityInfo
	switch finding.GetSeverity() {
	case visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR:
		severity = manifestvalidation.SeverityError
	case visualpb.VisualSeverity_VISUAL_SEVERITY_WARNING:
		severity = manifestvalidation.SeverityWarning
	}
	return manifestvalidation.Finding{
		Severity:   severity,
		Code:       finding.GetCode(),
		Location:   finding.GetLocation(),
		Message:    finding.GetMessage(),
		Suggestion: finding.GetRemediation(),
	}
}

// findingsFromEvidence runs the shared verdict and maps it to ui-health findings.
func findingsFromEvidence(ev evidence.Evidence, profileID string) []manifestvalidation.Finding {
	v := evidence.Analyze(ev)
	profileSuffix := ""
	if profileID != "" {
		profileSuffix = " [" + profileID + "]"
	}
	var finds []manifestvalidation.Finding
	if v.Passed() {
		finds = append(finds, manifestvalidation.Finding{
			Severity: manifestvalidation.SeverityInfo,
			Code:     "runtime_render_ok",
			Location: ev.URL,
			Message:  fmt.Sprintf("UI rendered and the iframe-bridge handshake succeeded%s (%s)", profileSuffix, ev.URL),
		})
	} else {
		finds = append(finds, manifestvalidation.Finding{
			Severity:   manifestvalidation.SeverityError,
			Code:       codeForFailure(ev),
			Location:   ev.URL,
			Message:    strings.TrimSpace(v.Message + profileSuffix),
			Suggestion: remediationFor(ev),
		})
	}
	if v.ConsoleErrorCount > 0 {
		finds = append(finds, manifestvalidation.Finding{
			Severity: manifestvalidation.SeverityWarning,
			Code:     "runtime_console_errors",
			Location: ev.URL,
			Message:  fmt.Sprintf("%d console error(s) logged during render (non-fatal)", v.ConsoleErrorCount),
		})
	}
	return finds
}

// codeForFailure picks the finding code matching the verdict's failure cause,
// following the same precedence as evidence.Analyze.
func codeForFailure(ev evidence.Evidence) string {
	switch {
	case !ev.Loaded:
		return "runtime_load_failed"
	case !ev.Handshake.Signaled:
		return "runtime_handshake_failed"
	case len(ev.Network) > 0:
		return "runtime_network_failure"
	case ev.RenderBroken:
		return "runtime_render_broken"
	case len(ev.PageErrors) > 0:
		return "runtime_page_error"
	default:
		return "runtime_render_failed"
	}
}

func remediationFor(ev evidence.Evidence) string {
	switch {
	case !ev.Loaded:
		return "The browser session failed to load the UI. Check that the scenario UI starts and serves its index."
	case !ev.Handshake.Signaled:
		return "The UI must call @vrooli/iframe-bridge initIframeBridgeChild() on boot so it signals READY when embedded. Verify the bridge dependency and init call."
	case len(ev.Network) > 0:
		return "Resolve the failing network requests (assets/APIs) the embedded UI issued on load."
	case ev.RenderBroken:
		return "The frame rendered blank/solid — verify the UI mounts content and styles load."
	case len(ev.PageErrors) > 0:
		return "Fix the uncaught exception thrown while rendering the UI."
	default:
		return ""
	}
}

func skip(code, location, message string) manifestvalidation.Finding {
	return manifestvalidation.Finding{
		Severity: manifestvalidation.SeverityInfo,
		Code:     code,
		Location: location,
		Message:  message,
	}
}

func errText(err error) string {
	if err == nil {
		return "not resolvable"
	}
	return err.Error()
}
