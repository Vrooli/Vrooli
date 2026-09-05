package executor

import (
	"fmt"
	"strings"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

var stepTypeCapabilityMatrix = map[string]contracts.CapabilityRequirement{
	"tabswitch":    {NeedsParallelTabs: true},
	"frameswitch":  {NeedsIframes: true},
	"uploadfile":   {NeedsFileUploads: true},
	"upload":       {NeedsFileUploads: true},
	"fileupload":   {NeedsFileUploads: true},
	"downloadfile": {NeedsDownloads: true},
	"download":     {NeedsDownloads: true},
	"networkmock":  {NeedsHAR: true, NeedsTracing: true},
	"har":          {NeedsHAR: true},
	"video":        {NeedsVideo: true},
	"trace":        {NeedsTracing: true},
}

// preflight.go derives capability requirements from plans/instructions so
// engine selection can fail fast before execution starts.

// analyzeRequirements returns both the requirements and the reasons (step types
// or params) that triggered each flag. The reasons map is keyed by requirement
// name (e.g., "har", "downloads", "video").
func analyzeRequirements(plan contracts.ExecutionPlan) (contracts.CapabilityRequirement, map[string][]string) {
	req := contracts.CapabilityRequirement{}
	reasons := make(map[string][]string)
	add := func(key, reason string) {
		if strings.TrimSpace(reason) == "" {
			return
		}
		reasons[key] = append(reasons[key], reason)
	}

	if viewportRaw, ok := plan.Metadata["executionViewport"]; ok {
		if viewport, ok := viewportRaw.(map[string]any); ok {
			if w, ok := viewport["width"].(float64); ok && w > 0 {
				req.MinViewportWidth = int(w)
			}
			if h, ok := viewport["height"].(float64); ok && h > 0 {
				req.MinViewportHeight = int(h)
			}
		}
	}
	req = mergeMetadataCapabilities(req, plan.Metadata, add)

	for _, instr := range plan.Instructions {
		req, reasons = applyInstructionCapabilities(req, reasons, instr, add)
	}

	req, reasons = applyGraphCapabilities(req, reasons, plan.Graph, add)
	return req, reasons
}

func mergeMetadataCapabilities(req contracts.CapabilityRequirement, metadata map[string]any, add func(string, string)) contracts.CapabilityRequirement {
	flag := func(key string) bool {
		raw, ok := metadata[key]
		if !ok {
			return false
		}
		if b, ok := raw.(bool); ok {
			return b
		}
		return false
	}
	if flag("requiresDownloads") {
		req.NeedsDownloads = true
		add("downloads", "metadata.requiresDownloads")
	}
	if flag("requiresFileUploads") {
		req.NeedsFileUploads = true
		add("file_uploads", "metadata.requiresFileUploads")
	}
	if flag("requiresHar") || flag("requiresHAR") {
		req.NeedsHAR = true
		add("har", "metadata.requiresHar")
	}
	if flag("requiresVideo") {
		req.NeedsVideo = true
		add("video", "metadata.requiresVideo")
	}
	if flag("requiresTracing") {
		req.NeedsTracing = true
		add("tracing", "metadata.requiresTracing")
	}
	if flag("requiresPerformanceTrace") {
		req.NeedsPerfTrace = true
		add("perf_trace", "metadata.requiresPerformanceTrace")
	}
	if flag("requiresAccessibility") {
		req.NeedsAccessibility = true
		add("accessibility", "metadata.requiresAccessibility")
	}
	if flag("requiresIframes") {
		req.NeedsIframes = true
		add("iframes", "metadata.requiresIframes")
	}
	if flag("requiresParallelTabs") {
		req.NeedsParallelTabs = true
		add("parallel_tabs", "metadata.requiresParallelTabs")
	}
	return req
}

func applyInstructionCapabilities(req contracts.CapabilityRequirement, reasons map[string][]string, instr contracts.CompiledInstruction, add func(string, string)) (contracts.CapabilityRequirement, map[string][]string) {
	stepType := InstructionStepType(instr)
	normalizedType := normalizeType(stepType)
	if addition, ok := stepTypeCapabilityMatrix[normalizedType]; ok {
		req = mergeRequirements(req, addition)
		if addition.NeedsParallelTabs {
			add("parallel_tabs", fmt.Sprintf("step %d (%s): %s", instr.Index, instr.NodeID, stepType))
		}
		if addition.NeedsHAR {
			add("har", fmt.Sprintf("step %d (%s): %s", instr.Index, instr.NodeID, stepType))
		}
		if addition.NeedsVideo {
			add("video", fmt.Sprintf("step %d (%s): %s", instr.Index, instr.NodeID, stepType))
		}
		if addition.NeedsIframes {
			add("iframes", fmt.Sprintf("step %d (%s): %s", instr.Index, instr.NodeID, stepType))
		}
		if addition.NeedsFileUploads {
			add("file_uploads", fmt.Sprintf("step %d (%s): %s", instr.Index, instr.NodeID, stepType))
		}
		if addition.NeedsDownloads {
			add("downloads", fmt.Sprintf("step %d (%s): %s", instr.Index, instr.NodeID, stepType))
		}
		if addition.NeedsTracing {
			add("tracing", fmt.Sprintf("step %d (%s): %s", instr.Index, instr.NodeID, stepType))
		}
	}

	// These requirements are derived from typed action variants, not field-name
	// heuristics over legacy params maps.
	if IsActionType(instr, basactions.ActionType_ACTION_TYPE_TAB_SWITCH) {
		req.NeedsParallelTabs = true
		add("parallel_tabs", fmt.Sprintf("step %d (%s): tab switch action", instr.Index, instr.NodeID))
	}
	if IsActionType(instr, basactions.ActionType_ACTION_TYPE_FRAME_SWITCH) {
		req.NeedsIframes = true
		add("iframes", fmt.Sprintf("step %d (%s): frame switch action", instr.Index, instr.NodeID))
	}
	if IsActionType(instr, basactions.ActionType_ACTION_TYPE_NETWORK_MOCK) {
		req.NeedsHAR = true
		req.NeedsTracing = true
		add("har", fmt.Sprintf("step %d (%s): network mock", instr.Index, instr.NodeID))
		add("tracing", fmt.Sprintf("step %d (%s): network mock", instr.Index, instr.NodeID))
	}
	if IsActionType(instr, basactions.ActionType_ACTION_TYPE_UPLOAD_FILE) {
		req.NeedsFileUploads = true
		add("file_uploads", fmt.Sprintf("step %d (%s): upload action", instr.Index, instr.NodeID))
	}

	lowerType := strings.ToLower(stepType)
	if strings.Contains(lowerType, "download") {
		req.NeedsDownloads = true
		add("downloads", fmt.Sprintf("step %d (%s): type %s", instr.Index, instr.NodeID, stepType))
	}
	if strings.Contains(lowerType, "har") {
		req.NeedsHAR = true
		add("har", fmt.Sprintf("step %d (%s): type %s", instr.Index, instr.NodeID, stepType))
	}
	if strings.Contains(lowerType, "video") {
		req.NeedsVideo = true
		add("video", fmt.Sprintf("step %d (%s): type %s", instr.Index, instr.NodeID, stepType))
	}
	if strings.Contains(lowerType, "trace") {
		req.NeedsTracing = true
		add("tracing", fmt.Sprintf("step %d (%s): type %s", instr.Index, instr.NodeID, stepType))
	}

	return req, reasons
}

func applyGraphCapabilities(req contracts.CapabilityRequirement, reasons map[string][]string, graph *contracts.PlanGraph, add func(string, string)) (contracts.CapabilityRequirement, map[string][]string) {
	if graph == nil {
		return req, reasons
	}
	for _, step := range graph.Steps {
		// Use planStepToInstruction to preserve Action field
		req, reasons = applyInstructionCapabilities(req, reasons, planStepToInstruction(step), add)
		if step.Loop != nil {
			req, reasons = applyGraphCapabilities(req, reasons, step.Loop, add)
		}
	}
	return req, reasons
}

func mergeRequirements(req, addition contracts.CapabilityRequirement) contracts.CapabilityRequirement {
	req.NeedsParallelTabs = req.NeedsParallelTabs || addition.NeedsParallelTabs
	req.NeedsHAR = req.NeedsHAR || addition.NeedsHAR
	req.NeedsVideo = req.NeedsVideo || addition.NeedsVideo
	req.NeedsIframes = req.NeedsIframes || addition.NeedsIframes
	req.NeedsFileUploads = req.NeedsFileUploads || addition.NeedsFileUploads
	req.NeedsDownloads = req.NeedsDownloads || addition.NeedsDownloads
	req.NeedsTracing = req.NeedsTracing || addition.NeedsTracing
	req.NeedsPerfTrace = req.NeedsPerfTrace || addition.NeedsPerfTrace
	req.NeedsAccessibility = req.NeedsAccessibility || addition.NeedsAccessibility
	if addition.MinViewportWidth > req.MinViewportWidth {
		req.MinViewportWidth = addition.MinViewportWidth
	}
	if addition.MinViewportHeight > req.MinViewportHeight {
		req.MinViewportHeight = addition.MinViewportHeight
	}
	return req
}

func filterReasons(all map[string][]string, keys []string) map[string][]string {
	if len(all) == 0 || len(keys) == 0 {
		return nil
	}
	out := make(map[string][]string)
	for _, key := range keys {
		if vals, ok := all[key]; ok && len(vals) > 0 {
			out[key] = vals
		}
	}
	return out
}

func normalizeType(stepType string) string {
	lower := strings.ToLower(strings.TrimSpace(stepType))
	return strings.ReplaceAll(lower, "_", "")
}
