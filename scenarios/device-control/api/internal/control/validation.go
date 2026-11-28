package control

import (
	"regexp"

	"device-control/strategy"
)

var androidArtifactRef = regexp.MustCompile(`^android-(debug|release):[0-9a-f]{64}$`)

func capabilityForDerivedStep(kind string) (string, bool) {
	switch kind {
	case "key", "tap", "text", "swipe", "long-press", "double-tap", "drag", "fling":
		return strategy.CapInput, true
	case "wireless-debugging", "wireless-debugging-assert":
		return strategy.CapWirelessDebugging, true
	case "home-set":
		return strategy.CapAppLifecycle, true
	case "install-artifact":
		return strategy.CapAppLifecycle, true
	case "restart-device":
		return strategy.CapDeviceRestart, true
	case "reverse-tcp", "reverse-tcp-remove":
		return strategy.CapReverseTCP, true
	case "property-get", "property-set", "property-assert":
		return strategy.CapProperty, true
	case "sensor-read":
		return strategy.CapSensor, true
	case "media-play", "media-pause", "media-stop", "media-next", "media-previous", "media-volume":
		return strategy.CapMedia, true
	default:
		return "", false
	}
}

func knownStepKind(kind string) bool {
	switch kind {
	case "home-set", "restart-device", "wireless-debugging", "wireless-debugging-assert", "assert-foreground", "reverse-tcp", "reverse-tcp-remove", "tap", "key", "observe", "wait", "assert-frame-different", "semantic-target", "semantic-assert", "swipe", "long-press", "double-tap", "drag", "fling", "pinch", "scroll-to", "text", "device-logs", "framework-logs", "logcat-start", "logcat-stop", "clock-sample", "screenshot", "clipboard-read", "clipboard-write", "screenrecord", "recording-start", "recording-stop", "install", "install-artifact", "launch", "focus", "close", "minimize", "restore", "stop", "uninstall", "clear-data", "grant-permission", "revoke-permission", "package-state", "rotate", "network", "bluetooth", "airplane-mode", "screen", "deep-link", "share", "property-get", "property-set", "property-assert", "sensor-read", "media-play", "media-pause", "media-stop", "media-next", "media-previous", "media-volume":
		return true
	default:
		return false
	}
}

func lifecycleFlowAction(action string) bool {
	switch action {
	case "home-set", "launch", "focus", "close", "minimize", "restore", "stop", "uninstall", "clear-data", "grant-permission", "revoke-permission", "package-state":
		return true
	default:
		return false
	}
}
