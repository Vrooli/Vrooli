package playbooks

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"test-genie/internal/evidence"
	"test-genie/internal/playbooks/execution"

	"github.com/vrooli/api-core/discovery"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

type visualHealthAnalyzer interface {
	AnalyzeArtifacts(context.Context, *visualpb.AnalyzeArtifactsRequest) (*visualpb.AnalyzeArtifactsResponse, error)
}

type defaultVisualHealthAnalyzer struct{}

func (defaultVisualHealthAnalyzer) AnalyzeArtifacts(ctx context.Context, req *visualpb.AnalyzeArtifactsRequest) (*visualpb.AnalyzeArtifactsResponse, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "ui-health")
	if err != nil {
		return nil, err
	}
	client := visualconnect.NewVisualHealthServiceClient(&http.Client{Timeout: 30 * time.Second}, baseURL)
	resp, err := client.AnalyzeArtifacts(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// evidenceObservations runs the shared browser-evidence analyzer over a
// playbooks workflow timeline and renders its findings as additive
// observations.
//
// The findings are additive: a workflow's own assertion pass/fail remains the
// primary verdict, so these observations never flip a passing workflow to
// failed. They exist to surface browser-level problems the workflow assertions
// did not assert on — console errors, failed network requests, uncaught page
// exceptions, and ui-health generic visual findings — using the shared browser
// evidence analyzer plus ui-health's visual-health authority:
//
//   - failed network requests / uncaught page exceptions → error observations;
//   - console errors → an error observation (counted, surfaced, non-fatal);
//   - console warnings → a warning observation;
//   - ui-health visual-health errors/warnings/info → matching observations.
//
// A clean timeline (no console errors/warnings, no network failures, no page
// errors) yields no observations.
func evidenceObservations(ctx context.Context, analyzer visualHealthAnalyzer, workflowFile string, parsed *execution.ParsedTimeline) []Observation {
	ev := execution.ToEvidence(parsed, execution.ToEvidenceOptions{Label: workflowFile})
	verdict := evidence.Analyze(ev)

	var obs []Observation

	if verdict.NetworkFailureCount > 0 {
		obs = append(obs, NewErrorObservation(fmt.Sprintf(
			"%s: %d failed network request(s) observed: %s",
			workflowFile, verdict.NetworkFailureCount, summarizeNetwork(ev.Network))))
	}
	for _, pe := range ev.PageErrors {
		obs = append(obs, NewErrorObservation(fmt.Sprintf(
			"%s: page error: %s", workflowFile, pe.Message)))
	}
	if verdict.ConsoleErrorCount > 0 {
		obs = append(obs, NewErrorObservation(fmt.Sprintf(
			"%s: %d console error(s) observed", workflowFile, verdict.ConsoleErrorCount)))
	}
	if verdict.ConsoleWarningCount > 0 {
		obs = append(obs, NewWarningObservation(fmt.Sprintf(
			"%s: %d console warning(s) observed", workflowFile, verdict.ConsoleWarningCount)))
	}
	obs = append(obs, visualHealthObservations(ctx, analyzer, workflowFile, parsed)...)

	return obs
}

func visualHealthObservations(ctx context.Context, analyzer visualHealthAnalyzer, workflowFile string, parsed *execution.ParsedTimeline) []Observation {
	if analyzer == nil {
		return nil
	}
	step := execution.ToVisualStepArtifact(parsed, execution.ToEvidenceOptions{Label: workflowFile})
	if step == nil {
		return nil
	}
	resp, err := analyzer.AnalyzeArtifacts(ctx, &visualpb.AnalyzeArtifactsRequest{Steps: []*visualpb.VisualStepArtifact{step}})
	if err != nil {
		return []Observation{NewWarningObservation(fmt.Sprintf(
			"%s: ui-health visual analysis skipped: %v", workflowFile, err))}
	}
	var obs []Observation
	for _, finding := range resp.GetFindings() {
		msg := fmt.Sprintf("%s: ui-health %s: %s", workflowFile, finding.GetCode(), finding.GetMessage())
		switch finding.GetSeverity() {
		case visualpb.VisualSeverity_VISUAL_SEVERITY_ERROR:
			obs = append(obs, NewErrorObservation(msg))
		case visualpb.VisualSeverity_VISUAL_SEVERITY_WARNING:
			obs = append(obs, NewWarningObservation(msg))
		default:
			obs = append(obs, NewInfoObservation(msg))
		}
	}
	return obs
}

// summarizeNetwork renders a short, capped description of failed requests.
func summarizeNetwork(failures []evidence.NetworkEntry) string {
	const cap = 3
	out := ""
	for i, f := range failures {
		if i >= cap {
			out += fmt.Sprintf(", ... and %d more", len(failures)-cap)
			break
		}
		if i > 0 {
			out += ", "
		}
		switch {
		case f.Status != nil:
			out += fmt.Sprintf("HTTP %d %s", *f.Status, f.URL)
		case f.ErrorText != "":
			out += fmt.Sprintf("%s %s", f.ErrorText, f.URL)
		default:
			out += f.URL
		}
	}
	return out
}
