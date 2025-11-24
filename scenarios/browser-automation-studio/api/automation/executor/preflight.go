package executor

import (
	"fmt"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

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
		params := instr.Params
		if params == nil {
			continue
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
		if requiresFileUploads(params) {
			req.NeedsFileUploads = true
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
