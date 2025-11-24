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

func deriveRequirements(plan contracts.ExecutionPlan) contracts.CapabilityRequirement {
	req := contracts.CapabilityRequirement{}
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
	req = mergeMetadataCapabilities(req, plan.Metadata)

	for _, instr := range plan.Instructions {
		req = applyInstructionCapabilities(req, instr.Type, instr.Params)
	}

	req = applyGraphCapabilities(req, plan.Graph)
	return req
}

func mergeMetadataCapabilities(req contracts.CapabilityRequirement, metadata map[string]any) contracts.CapabilityRequirement {
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
	}
	if flag("requiresFileUploads") {
		req.NeedsFileUploads = true
	}
	if flag("requiresHar") || flag("requiresHAR") {
		req.NeedsHAR = true
	}
	if flag("requiresVideo") {
		req.NeedsVideo = true
	}
	if flag("requiresTracing") {
		req.NeedsTracing = true
	}
	if flag("requiresIframes") {
		req.NeedsIframes = true
	}
	if flag("requiresParallelTabs") {
		req.NeedsParallelTabs = true
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

func applyInstructionCapabilities(req contracts.CapabilityRequirement, stepType string, params map[string]any) contracts.CapabilityRequirement {
	normalizedType := normalizeType(stepType)
	if addition, ok := stepTypeCapabilityMatrix[normalizedType]; ok {
		req = mergeRequirements(req, addition)
	}

	// Multi-tab support required when any tab switch directive is present.
	if requiresParallelTabs(params) {
		req.NeedsParallelTabs = true
	}
	// Iframe support required when frame navigation occurs.
	if requiresIframes(params) {
		req.NeedsIframes = true
	}
	// Network interception/mocking implies HAR/tracing capability.
	if requiresNetworkMock(params) {
		req.NeedsHAR = true
		req.NeedsTracing = true
	}
	// File upload support required when uploads are configured.
	if requiresFileUploads(params) || strings.EqualFold(stepType, "upload") {
		req.NeedsFileUploads = true
	}

	lowerType := strings.ToLower(stepType)
	if strings.Contains(lowerType, "download") {
		req.NeedsDownloads = true
	}
	if strings.Contains(lowerType, "har") {
		req.NeedsHAR = true
	}
	if strings.Contains(lowerType, "video") {
		req.NeedsVideo = true
	}
	if strings.Contains(lowerType, "trace") {
		req.NeedsTracing = true
	}

	// Param-derived signals for HAR/video/tracing/downloads.
	if params != nil {
		for key := range params {
			lower := strings.ToLower(key)
			switch {
			case strings.Contains(lower, "download"):
				req.NeedsDownloads = true
			case strings.Contains(lower, "har"):
				req.NeedsHAR = true
			case strings.Contains(lower, "video"):
				req.NeedsVideo = true
			case strings.Contains(lower, "trace"):
				req.NeedsTracing = true
			}
		}
		// Instruction-level viewport hints should not shrink global minima.
		if w, ok := numericParam(params, "viewportWidth"); ok && w > req.MinViewportWidth {
			req.MinViewportWidth = w
		}
		if h, ok := numericParam(params, "viewportHeight"); ok && h > req.MinViewportHeight {
			req.MinViewportHeight = h
		}
	}
	return req
}

func applyGraphCapabilities(req contracts.CapabilityRequirement, graph *contracts.PlanGraph) contracts.CapabilityRequirement {
	if graph == nil {
		return req
	}
	for _, step := range graph.Steps {
		req = applyInstructionCapabilities(req, step.Type, step.Params)
		if step.Loop != nil {
			req = applyGraphCapabilities(req, step.Loop)
		}
	}
	return req
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

func normalizeType(stepType string) string {
	lower := strings.ToLower(strings.TrimSpace(stepType))
	return strings.ReplaceAll(lower, "_", "")
}
