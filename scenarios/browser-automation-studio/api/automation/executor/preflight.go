package executor

import (
	"fmt"
	"strings"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
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

// deriveRequirements preserves the historical API, returning only the
// aggregated requirement flags.
func deriveRequirements(plan contracts.ExecutionPlan) contracts.CapabilityRequirement {
	req, _ := analyzeRequirements(plan)
	return req
}

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

func requiresParallelTabs(params map[string]any) bool {
	for _, key := range []string{
		"tabSwitchBy", "tabIndex", "tabTitleMatch", "tabUrlMatch", "tabWaitForNew", "tabCloseOld",
	} {
		if _, ok := params[key]; ok {
			return true
		}
	}
	return false
}

func requiresIframes(params map[string]any) bool {
	for _, key := range []string{
		"frameSwitchBy", "frameIndex", "frameName", "frameSelector", "frameUrlMatch",
	} {
		if _, ok := params[key]; ok {
			return true
		}
	}
	return false
}

func requiresFileUploads(params map[string]any) bool {
	if v, ok := params["filePath"]; ok && v != nil && fmt.Sprint(v) != "" {
		return true
	}
	if v, ok := params["filePaths"]; ok {
		if arr, ok := v.([]any); ok && len(arr) > 0 {
			return true
		}
	}
	return false
}

func requiresNetworkMock(params map[string]any) bool {
	for _, key := range []string{
		"networkMockType", "networkUrlPattern", "networkMethod", "networkStatusCode", "networkHeaders", "networkBody", "networkDelayMs", "networkAbortReason",
	} {
		if _, ok := params[key]; ok {
			return true
		}
	}
	return false
}

func numericParam(params map[string]any, key string) (int, bool) {
	if raw, ok := params[key]; ok {
		switch v := raw.(type) {
		case float64:
			return int(v), true
		case int:
			return v, true
		}
	}
	return 0, false
}

func boolParamTrue(params map[string]any, keys ...string) bool {
	for _, key := range keys {
		raw, ok := params[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case bool:
			if v {
				return true
			}
		case string:
			val := strings.ToLower(strings.TrimSpace(v))
			if val == "true" || val == "1" || val == "yes" || val == "on" {
				return true
			}
		}
	}
	return false
}

func applyInstructionCapabilities(req contracts.CapabilityRequirement, reasons map[string][]string, instr contracts.CompiledInstruction, add func(string, string)) (contracts.CapabilityRequirement, map[string][]string) {
	stepType := InstructionStepType(instr)
	params := InstructionParams(instr)
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

	// Multi-tab support required when any tab switch directive is present.
	if requiresParallelTabs(params) {
		req.NeedsParallelTabs = true
		add("parallel_tabs", fmt.Sprintf("step %d (%s): tab switch params", instr.Index, instr.NodeID))
	}
	// Iframe support required when frame navigation occurs.
	if requiresIframes(params) {
		req.NeedsIframes = true
		add("iframes", fmt.Sprintf("step %d (%s): frame switch params", instr.Index, instr.NodeID))
	}
	// Network interception/mocking implies HAR/tracing capability.
	if requiresNetworkMock(params) {
		req.NeedsHAR = true
		req.NeedsTracing = true
		add("har", fmt.Sprintf("step %d (%s): network mock", instr.Index, instr.NodeID))
		add("tracing", fmt.Sprintf("step %d (%s): network mock", instr.Index, instr.NodeID))
	}
	// File upload support required when uploads are configured.
	if requiresFileUploads(params) || strings.EqualFold(stepType, "upload") {
		req.NeedsFileUploads = true
		add("file_uploads", fmt.Sprintf("step %d (%s): upload params", instr.Index, instr.NodeID))
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

	// Param-derived signals for HAR/video/tracing/downloads.
	if params != nil {
		for key := range params {
			lower := strings.ToLower(key)
			switch {
			case strings.Contains(lower, "download"):
				req.NeedsDownloads = true
				add("downloads", fmt.Sprintf("step %d (%s): param %s", instr.Index, instr.NodeID, key))
			case strings.Contains(lower, "har"):
				req.NeedsHAR = true
				add("har", fmt.Sprintf("step %d (%s): param %s", instr.Index, instr.NodeID, key))
			case strings.Contains(lower, "video"):
				req.NeedsVideo = true
				add("video", fmt.Sprintf("step %d (%s): param %s", instr.Index, instr.NodeID, key))
			case strings.Contains(lower, "trace"):
				req.NeedsTracing = true
				add("tracing", fmt.Sprintf("step %d (%s): param %s", instr.Index, instr.NodeID, key))
			case strings.Contains(lower, "recordnetwork"):
				req.NeedsHAR = true
				add("har", fmt.Sprintf("step %d (%s): param %s", instr.Index, instr.NodeID, key))
			case strings.Contains(lower, "recordtrace"):
				req.NeedsTracing = true
				add("tracing", fmt.Sprintf("step %d (%s): param %s", instr.Index, instr.NodeID, key))
			case strings.Contains(lower, "recordvideo"):
				req.NeedsVideo = true
				add("video", fmt.Sprintf("step %d (%s): param %s", instr.Index, instr.NodeID, key))
			}
		}
		if boolParamTrue(params, "recordNetwork", "captureNetwork", "networkRecording") {
			req.NeedsHAR = true
			add("har", fmt.Sprintf("step %d (%s): recordNetwork", instr.Index, instr.NodeID))
		}
		if boolParamTrue(params, "recordHar", "captureHar") {
			req.NeedsHAR = true
			add("har", fmt.Sprintf("step %d (%s): recordHar", instr.Index, instr.NodeID))
		}
		if boolParamTrue(params, "recordTrace", "captureTrace", "traceEnabled") {
			req.NeedsTracing = true
			add("tracing", fmt.Sprintf("step %d (%s): recordTrace", instr.Index, instr.NodeID))
		}
		if boolParamTrue(params, "recordVideo", "captureVideo", "videoRecording") {
			req.NeedsVideo = true
			add("video", fmt.Sprintf("step %d (%s): recordVideo", instr.Index, instr.NodeID))
		}
		// Instruction-level viewport hints should not shrink global minima.
		if w, ok := numericParam(params, "viewportWidth"); ok && w > req.MinViewportWidth {
			req.MinViewportWidth = w
		}
		if h, ok := numericParam(params, "viewportHeight"); ok && h > req.MinViewportHeight {
			req.MinViewportHeight = h
		}
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
