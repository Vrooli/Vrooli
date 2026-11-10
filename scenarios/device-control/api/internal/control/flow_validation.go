package control

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"device-control/strategy"
)

func (s *Service) Validate(ctx context.Context, flow Flow, strategyID string) GapReport {
	// A device id may be supplied for preflight so validation sees the same
	// promoted, endpoint-scoped adapter that execution will use. This matters
	// for wireless devices: the registry's generic android-adb strategy has no
	// live capabilities until it is bound to the device's restored transport.
	if strat, ok := s.strategyForFlow(strategyID, flow.Transport); ok {
		report, transportUnavailable := validateAgainstDeclarationDetailed(ctx, flow, strat)
		_, fallbackReport, fallback := s.preflightUSBForImplicitWireless(ctx, strategyID, flow, report)
		if fallback {
			return fallbackReport
		}
		if transportUnavailable && !report.Runnable && strings.TrimSpace(flow.Transport) == "" && s.recoverWirelessBeforeUse(ctx, strategyID) {
			if recovered, ok := s.strategyForFlow(strategyID, "wireless"); ok {
				recoveredReport, _ := validateAgainstDeclarationDetailed(ctx, flow, recovered)
				return recoveredReport
			}
		}
		return report
	}
	d, ok := s.strategyForDevice(strategyID)
	if !ok {
		return GapReport{Gaps: []string{"unknown strategy " + strategyID}}
	}
	return validateAgainstDeclaration(ctx, flow, d)
}

// preflightUSBForImplicitWireless only selects the serial-bound USB strategy
// before execution starts. A requested transport is never silently replaced.
func (s *Service) preflightUSBForImplicitWireless(ctx context.Context, deviceID string, flow Flow, report GapReport) (strategy.Strategy, GapReport, bool) {
	if report.Runnable || strings.TrimSpace(flow.Transport) != "" {
		return nil, report, false
	}
	candidate, ok := s.knownSerialAndroidUSB(deviceID)
	if !ok {
		return nil, report, false
	}
	usbFlow := flow
	usbFlow.Transport = "usb"
	if usbReport := validateAgainstDeclaration(ctx, usbFlow, candidate); usbReport.Runnable {
		return candidate, usbReport, true
	}
	return nil, report, false
}

func (s *Service) knownSerialAndroidUSB(deviceID string) (strategy.Strategy, bool) {
	record, found := s.devices.Get(deviceID)
	if !found || !strings.EqualFold(record.Transport, "wireless") || record.StrategyID != "android-adb" || record.Serial == "" {
		return nil, false
	}
	base, baseOK := s.registry.Get(record.StrategyID)
	if _, scoped := base.(strategy.DeviceScoped); !baseOK || !scoped {
		return nil, false
	}
	for _, profile := range record.Transports {
		if profile.Name == "usb" && profile.StrategyID == record.StrategyID {
			return s.strategyForFlow(deviceID, "usb")
		}
	}
	return nil, false
}

// recoverWirelessBeforeUse reconnects only an existing Android wireless
// identity. The reconnect owner verifies its serial before changing the saved
// endpoint; callers must invoke this before any device effect.
func (s *Service) recoverWirelessBeforeUse(ctx context.Context, deviceID string) bool {
	record, ok := s.devices.Get(deviceID)
	if !ok || record.StrategyID != "android-adb" || !strings.EqualFold(record.Transport, "wireless") {
		return false
	}
	reconnectCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	_, err := s.ReconnectWireless(reconnectCtx, deviceID)
	return err == nil
}

func validateAgainstDeclaration(ctx context.Context, flow Flow, d strategy.Strategy) GapReport {
	report, _ := validateAgainstDeclarationDetailed(ctx, flow, d)
	return report
}

// validateAgainstDeclarationDetailed separates a transport that cannot currently
// be observed from a healthy transport that lacks a requested capability. Only
// the former is eligible for automatic reconnect; reconnecting cannot add support
// for a capability the adapter does not provide.
func validateAgainstDeclarationDetailed(ctx context.Context, flow Flow, d strategy.Strategy) (GapReport, bool) {
	var decl strategy.Declaration
	if prober, ok := d.(strategy.CaptureDescriber); ok && simpleCaptureKind(flow) != "" {
		decl, _ = prober.DescribeCapture(ctx, simpleCaptureKind(flow))
	} else {
		decl, _ = d.Describe(ctx)
	}
	g := GapReport{Runnable: true, Gaps: []string{}, Warnings: []string{}}
	if err := validateFlowContract(flow); err != nil {
		g.Runnable = false
		g.Gaps = append(g.Gaps, err.Error())
	}
	for _, step := range flow.Steps {
		if (step.Kind == "wireless-debugging" || step.Kind == "wireless-debugging-assert") && step.Target != "enable" && step.Target != "disable" {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires target enable or disable", step.ID))
		}
		if value, present := step.Arguments["require_audio"]; present {
			if _, ok := value.(bool); !ok || step.Kind != "recording-start" {
				g.Runnable = false
				g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires a boolean require_audio on recording-start", step.ID))
			}
		}
		if !knownStepKind(step.Kind) {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s uses unsupported kind %q", step.ID, step.Kind))
		}
		if step.Kind == "semantic-target" && decl.Capabilities[strategy.CapSemanticTree].Status != strategy.StatusAvailable {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s", step.ID, strategy.CapSemanticTree))
		}
		if step.Kind == "observe" && decl.Capabilities[strategy.CapScreenshot].Status != strategy.StatusAvailable {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s", step.ID, strategy.CapScreenshot))
		}
		if (step.Kind == "recording-start" || step.Kind == "recording-stop") && decl.Capabilities[strategy.CapScreenRecording].Status != strategy.StatusAvailable && decl.Capabilities[strategy.CapNativeRecording].Status != strategy.StatusAvailable {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s", step.ID, strategy.CapScreenRecording))
		}
		if required, _ := step.Arguments["require_audio"].(bool); step.Kind == "recording-start" && required {
			if _, ok := d.(strategy.AudioSessionRecorder); !ok || decl.Capabilities[strategy.CapRecordingAudio].Status != strategy.StatusAvailable {
				g.Runnable = false
				g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires recording with audio", step.ID))
			}
		}
		if step.Kind == "pinch" && decl.Capabilities[strategy.CapMultiTouch].Status != strategy.StatusAvailable {
			g.Runnable = false
			g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s", step.ID, strategy.CapMultiTouch))
		}
		for _, cap := range step.RequiredCapabilities {
			if decl.Capabilities[cap].Status != strategy.StatusAvailable {
				g.Runnable = false
				g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s (%s)", step.ID, cap, decl.Capabilities[cap].NextAction))
			}
		}
		if capability, ok := capabilityForDerivedStep(step.Kind); ok && decl.Capabilities[capability].Status != strategy.StatusAvailable {
			g.Runnable = false
			detail := decl.Capabilities[capability].Reason
			if detail == "" {
				detail = decl.Capabilities[capability].NextAction
			}
			if detail == "" {
				g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s", step.ID, capability))
			} else {
				g.Gaps = append(g.Gaps, fmt.Sprintf("step %s requires %s (%s)", step.ID, capability, detail))
			}
		}
	}
	return g, decl.Status != strategy.StatusAvailable
}

// Only the exact simple capture shapes use targeted preflight. Complex flows
// retain the full declaration, including every explicitly required capability.
func simpleCaptureKind(flow Flow) string {
	if len(flow.Steps) == 1 {
		step := flow.Steps[0]
		kind, cap := "", ""
		switch step.Kind {
		case "observe":
			kind, cap = "screenshot", strategy.CapScreenshot
		case "device-logs":
			kind, cap = "logs", strategy.CapDeviceLogs
		}
		if kind == "" {
			return ""
		}
		for _, required := range step.RequiredCapabilities {
			if required != cap {
				return ""
			}
		}
		return kind
	}
	if len(flow.Steps) == 3 && flow.Steps[0].Kind == "recording-start" && flow.Steps[1].Kind == "wait" && flow.Steps[2].Kind == "recording-stop" {
		for _, step := range flow.Steps {
			for _, required := range step.RequiredCapabilities {
				if required != strategy.CapScreenRecording && required != strategy.CapNativeRecording {
					return ""
				}
			}
		}
		return "recording"
	}
	return ""
}

const (
	maxFlowSteps      = 64
	maxFlowDurationMS = int64(5 * time.Minute / time.Millisecond)
	maxStepTimeoutMS  = int64(2 * time.Minute / time.Millisecond)
	maxRetryBudget    = 3
)

// FlowHTTPWriteTimeout leaves time to serialize evidence after the bounded
// flow execution window. The API server must not close a response while a
// valid maximum-duration flow is still producing its result.
func FlowHTTPWriteTimeout() time.Duration {
	return time.Duration(maxFlowDurationMS)*time.Millisecond + time.Minute
}

func validateFlowContract(flow Flow) error {
	if len(flow.Steps) > maxFlowSteps {
		return fmt.Errorf("flow exceeds %d steps", maxFlowSteps)
	}
	if flow.MaxDurationMS < 0 || flow.MaxDurationMS > maxFlowDurationMS {
		return fmt.Errorf("flow max_duration_ms must be between 0 and %d", maxFlowDurationMS)
	}
	if flow.RetryBudget < 0 || flow.RetryBudget > maxRetryBudget {
		return fmt.Errorf("flow retry_budget must be between 0 and %d", maxRetryBudget)
	}
	ids := map[string]struct{}{}
	reversePorts := map[int]bool{}
	for _, step := range flow.Steps {
		if step.Kind == "install-artifact" {
			artifactRef, _ := step.Arguments["artifact_ref"].(string)
			packageName, _ := step.Arguments["package"].(string)
			versionName, hasVersion := step.Arguments["version_name"].(string)
			if step.Target != "" || !androidArtifactRef.MatchString(artifactRef) || !androidPackageName.MatchString(packageName) || len(step.Arguments) < 2 || len(step.Arguments) > 3 || (hasVersion && (strings.TrimSpace(versionName) != versionName || len(versionName) > 128)) || (!hasVersion && step.Arguments["version_name"] != nil) {
				return fmt.Errorf("step %q install-artifact requires an immutable Android ref, fully-qualified package, and optional version_name", step.ID)
			}
			if step.RetryBudget != 0 || flow.RetryBudget != 0 {
				return fmt.Errorf("step %q install-artifact cannot be retried automatically", step.ID)
			}
		}
		if step.Kind == "home-set" && (!flow.RequireUnlocked || step.RetryBudget != 0 || flow.RetryBudget != 0) {
			return fmt.Errorf("home-set requires an unlocked flow without retries")
		}
		if step.Kind == "home-set" {
			prior, _ := step.Arguments["value"].(string)
			if !androidPackageName.MatchString(step.Target) || len(step.Arguments) != 1 || prior == "" || len(prior) > 512 {
				return fmt.Errorf("home-set requires one package target and expected prior Home value")
			}
		}
		if step.Kind == "restart-device" && (len(flow.Steps) != 1 || !flow.RequireUnlocked || step.Target != "" || len(step.Arguments) != 0 || step.RetryBudget != 0 || flow.RetryBudget != 0) {
			return fmt.Errorf("restart-device requires one unlocked, argument-free step without retries")
		}
		if step.Kind == "assert-foreground" && (strings.TrimSpace(step.Target) == "" || len(step.Target) > 256) {
			return fmt.Errorf("step %q requires a bounded foreground application identity", step.ID)
		}
		if step.Kind == "reverse-tcp" || step.Kind == "reverse-tcp-remove" {
			devicePort, _, err := reverseTCPPorts(step)
			if err != nil {
				return fmt.Errorf("step %q: %w", step.ID, err)
			}
			if reversePorts[devicePort] || len(reversePorts) >= 4 {
				return fmt.Errorf("reverse-tcp requires at most four distinct device ports")
			}
			reversePorts[devicePort] = true
		}
		if strings.TrimSpace(step.ID) == "" {
			return fmt.Errorf("flow step id is required")
		}
		if _, exists := ids[step.ID]; exists {
			return fmt.Errorf("flow step id %q is duplicated", step.ID)
		}
		ids[step.ID] = struct{}{}
		if step.TimeoutMS < 0 || step.TimeoutMS > maxStepTimeoutMS {
			return fmt.Errorf("step %q timeout_ms must be between 0 and %d", step.ID, maxStepTimeoutMS)
		}
		if step.RetryBudget < 0 || step.RetryBudget > maxRetryBudget {
			return fmt.Errorf("step %q retry_budget must be between 0 and %d", step.ID, maxRetryBudget)
		}
		if key := strings.TrimSpace(step.IdempotencyKey); key != "" {
			if len(key) > 128 {
				return fmt.Errorf("step %q idempotency_key exceeds 128 bytes", step.ID)
			}
		}
		for _, condition := range append(append([]Condition{}, step.Preconditions...), step.Postconditions...) {
			if err := validateCondition(condition); err != nil {
				return fmt.Errorf("step %q: %w", step.ID, err)
			}
		}
	}
	return nil
}

// Numeric TCP endpoints only; no shell, privileged ports, or socket paths.
func reverseTCPPorts(step Step) (int, int, error) {
	devicePort, err := strconv.Atoi(step.Target)
	raw, ok := step.Arguments["host_port"].(float64)
	hostPort := int(raw)
	if err != nil || !ok || raw != float64(hostPort) || devicePort < 1024 || devicePort > 65535 || hostPort < 1024 || hostPort > 65535 {
		return 0, 0, fmt.Errorf("reverse-tcp requires integer device/host TCP ports from 1024 through 65535")
	}
	return devicePort, hostPort, nil
}

func validateCondition(condition Condition) error {
	kind := strings.ToLower(strings.TrimSpace(condition.Kind))
	switch kind {
	case "device-unlocked", "screen-on", "screen-off", "frame-different":
		return nil
	case "capability", "property-equals":
		if strings.TrimSpace(condition.Target) == "" {
			return fmt.Errorf("condition %q requires a target", condition.Kind)
		}
		if kind == "property-equals" && condition.Expected == nil {
			return fmt.Errorf("property-equals condition requires expected")
		}
		return nil
	default:
		return fmt.Errorf("unsupported condition kind %q", condition.Kind)
	}
}

func claimClassForStep(step Step) string {
	if value, ok := step.Arguments["claim_class"].(string); ok {
		switch value {
		case string(strategy.ClaimStatic), string(strategy.ClaimTransition), string(strategy.ClaimAnimation):
			return value
		}
	}
	return string(strategy.ClaimTransition)
}
