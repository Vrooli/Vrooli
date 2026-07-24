package execution

import (
	"strings"

	"test-genie/internal/evidence"

	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

// This file is the single home for mapping a BAS [ParsedTimeline] onto the
// engine-agnostic [evidence.Evidence] model. The playbooks phase consumes it
// (via [ToEvidence]). Keeping console/network extraction here — beside the
// ParsedTimeline it reads — means the timeline→evidence rules have exactly one
// implementation and cannot drift between consumers.

// ConsoleEntries extracts console messages from the container-level timeline
// logs, normalizing each level so the evidence analyzer can classify errors vs
// warnings. Returns nil when the timeline carries no logs.
func ConsoleEntries(tl *ParsedTimeline) []evidence.ConsoleEntry {
	if tl == nil || len(tl.Logs) == 0 {
		return nil
	}
	out := make([]evidence.ConsoleEntry, 0, len(tl.Logs))
	for _, l := range tl.Logs {
		out = append(out, evidence.ConsoleEntry{
			Level:   normalizeConsoleLevel(l.Level),
			Message: l.Message,
		})
	}
	return out
}

// NetworkFailures extracts failed network requests from the per-entry telemetry
// network-event artifacts surfaced on the timeline aggregates. A request counts
// as a failure when it carries a transport failure or a >=400 status. Returns
// nil when the timeline has no failing requests.
func NetworkFailures(tl *ParsedTimeline) []evidence.NetworkEntry {
	if tl == nil || tl.Proto == nil {
		return nil
	}
	var out []evidence.NetworkEntry
	for _, entry := range tl.Proto.GetEntries() {
		agg := entry.GetAggregates()
		if agg == nil {
			continue
		}
		for _, art := range agg.GetArtifacts() {
			if art.GetType() != basbase.ArtifactType_ARTIFACT_TYPE_NETWORK_EVENT {
				continue
			}
			if ne, ok := networkFailureFromArtifact(art); ok {
				out = append(out, ne)
			}
		}
	}
	return out
}

// ToEvidenceOptions configures how a playbooks timeline is folded into a single
// [evidence.Evidence] value.
type ToEvidenceOptions struct {
	// URL labels the surface under test (the workflow's target). Optional.
	URL string
	// Label names the workflow/run for multi-surface reporting. Optional.
	Label string
}

// ToEvidence folds a whole playbooks workflow timeline into one
// [evidence.Evidence] value suitable for [evidence.Analyze].
//
// A playbooks workflow runs many steps across many frames, so the evidence is
// aggregated over the entire run: console and network observations are the union
// across all frames and the screenshot reference is taken from the final frame
// (the run's end state). Playbooks does not gate on the iframe-bridge handshake
// because it drives authored workflows against the scenario UI top-level, so
// Handshake is reported as signaled (the workflow's own assertions are the
// primary contract; this evidence is additive). Loaded is false only when BAS
// produced no usable timeline at all, which the analyzer treats as a load
// failure.
func ToEvidence(tl *ParsedTimeline, opts ToEvidenceOptions) evidence.Evidence {
	ev := evidence.Evidence{
		URL:    opts.URL,
		Label:  opts.Label,
		Loaded: tl != nil && tl.Proto != nil,
		// Playbooks does not embed the host-iframe bridge. Report signaled so
		// the analyzer does not raise a handshake failure for a phase that never
		// performs the handshake.
		Handshake: evidence.Handshake{Signaled: true},
	}
	if !ev.Loaded {
		ev.LoadError = "BAS produced no timeline for the workflow"
		return ev
	}

	ev.URL = firstNonEmptyString(opts.URL, finalFrameURL(tl))
	ev.Console = ConsoleEntries(tl)
	ev.Network = NetworkFailures(tl)
	ev.ScreenshotRef = finalScreenshotRef(tl)
	return ev
}

// ToVisualStepArtifact maps the same timeline into ui-health's generic visual
// artifact contract. Playbooks owns workflow execution; ui-health owns generic
// DOM/pixel/browser-artifact judgment.
func ToVisualStepArtifact(tl *ParsedTimeline, opts ToEvidenceOptions) *visualpb.VisualStepArtifact {
	if tl == nil || tl.Proto == nil {
		return nil
	}
	step := &visualpb.VisualStepArtifact{
		StepId:  opts.Label,
		Label:   opts.Label,
		Url:     firstNonEmptyString(opts.URL, finalFrameURL(tl)),
		DomHtml: firstNonEmptyString(tl.FinalDOM, tl.FinalDOMPreview),
	}
	if ref := finalScreenshotRef(tl); ref != "" {
		step.ScreenshotRef = &visualpb.ArtifactRef{Uri: ref}
	}
	for _, c := range ConsoleEntries(tl) {
		step.Console = append(step.Console, &visualpb.ConsoleEntry{Level: c.Level, Message: c.Message})
	}
	for _, n := range NetworkFailures(tl) {
		entry := &visualpb.NetworkEntry{
			Url:          n.URL,
			Method:       n.Method,
			ResourceType: n.ResourceType,
			ErrorText:    n.ErrorText,
		}
		if n.Status != nil {
			entry.Status = int32(*n.Status)
		}
		step.Network = append(step.Network, entry)
	}
	return step
}

// finalFrameURL returns the URL of the last frame that recorded one.
func finalFrameURL(tl *ParsedTimeline) string {
	for i := len(tl.Frames) - 1; i >= 0; i-- {
		if u := strings.TrimSpace(tl.Frames[i].FinalURL); u != "" {
			return u
		}
	}
	return ""
}

// finalScreenshotRef returns the screenshot reference from the last frame that
// captured one (the run's end-state screenshot).
func finalScreenshotRef(tl *ParsedTimeline) string {
	for i := len(tl.Frames) - 1; i >= 0; i-- {
		ss := tl.Frames[i].Screenshot
		if ss == nil {
			continue
		}
		if ref := firstNonEmptyString(ss.URL, ss.ArtifactID); ref != "" {
			return ref
		}
	}
	return ""
}

// networkFailureFromArtifact interprets one network-event artifact payload,
// returning a NetworkEntry and true when it represents a failure.
func networkFailureFromArtifact(art *bastimeline.TimelineArtifact) (evidence.NetworkEntry, bool) {
	payload := art.GetPayload()
	if payload == nil {
		return evidence.NetworkEntry{}, false
	}
	failure := payloadStringValue(payload, "failure")
	status := payloadIntValue(payload, "status")
	if failure == "" && (status == nil || *status < 400) {
		return evidence.NetworkEntry{}, false
	}
	return evidence.NetworkEntry{
		URL:          payloadStringValue(payload, "url"),
		Method:       payloadStringValue(payload, "method"),
		ResourceType: payloadStringValue(payload, "resourceType"),
		Status:       status,
		ErrorText:    failure,
	}, true
}

// normalizeConsoleLevel maps a BAS log level to the console levels the evidence
// analyzer recognizes ("error", "warn"/"warning", ...).
func normalizeConsoleLevel(level string) string {
	return strings.ToLower(strings.TrimSpace(level))
}

func payloadStringValue(payload map[string]*commonv1.JsonValue, key string) string {
	v, ok := payload[key]
	if !ok || v == nil {
		return ""
	}
	if s, isString := v.GetKind().(*commonv1.JsonValue_StringValue); isString {
		return s.StringValue
	}
	return ""
}

func payloadIntValue(payload map[string]*commonv1.JsonValue, key string) *int {
	v, ok := payload[key]
	if !ok || v == nil {
		return nil
	}
	switch k := v.GetKind().(type) {
	case *commonv1.JsonValue_IntValue:
		n := int(k.IntValue)
		return &n
	case *commonv1.JsonValue_DoubleValue:
		n := int(k.DoubleValue)
		return &n
	default:
		return nil
	}
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
