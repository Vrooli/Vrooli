package control

func knownStepKind(kind string) bool {
	switch kind {
	case "tap", "key", "observe", "wait", "assert-frame-different", "semantic-target", "semantic-assert", "swipe", "long-press", "double-tap", "drag", "fling", "pinch", "scroll-to", "text", "device-logs", "logcat-start", "logcat-stop", "clock-sample", "screenshot", "clipboard-read", "clipboard-write", "screenrecord", "recording-start", "recording-stop", "install", "launch", "stop", "uninstall", "clear-data", "grant-permission", "revoke-permission", "package-state", "rotate", "network", "bluetooth", "airplane-mode", "screen", "deep-link", "share":
		return true
	default:
		return false
	}
}
