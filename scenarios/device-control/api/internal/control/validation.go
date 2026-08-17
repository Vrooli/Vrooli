package control

import "device-control/strategy"

func capabilityForDerivedStep(kind string) (string, bool) {
	switch kind {
	case "property-get", "property-set":
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
	case "tap", "key", "observe", "wait", "assert-frame-different", "semantic-target", "semantic-assert", "swipe", "long-press", "double-tap", "drag", "fling", "pinch", "scroll-to", "text", "device-logs", "logcat-start", "logcat-stop", "clock-sample", "screenshot", "clipboard-read", "clipboard-write", "screenrecord", "recording-start", "recording-stop", "install", "launch", "stop", "uninstall", "clear-data", "grant-permission", "revoke-permission", "package-state", "rotate", "network", "bluetooth", "airplane-mode", "screen", "deep-link", "share", "property-get", "property-set", "sensor-read", "media-play", "media-pause", "media-stop", "media-next", "media-previous", "media-volume":
		return true
	default:
		return false
	}
}
