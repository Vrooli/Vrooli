package control

func knownStepKind(kind string) bool {
	switch kind {
	case "tap", "key", "observe", "wait", "assert-frame-different", "semantic-target", "semantic-assert", "swipe", "text", "device-logs", "screenrecord", "install", "launch", "stop", "uninstall", "clear-data", "grant-permission", "revoke-permission", "package-state", "rotate", "network", "deep-link":
		return true
	default:
		return false
	}
}
