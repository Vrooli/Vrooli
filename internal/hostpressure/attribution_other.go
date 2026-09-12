//go:build !linux

package hostpressure

// parentScope has no cgroup to read off Linux; attribution is unread there
// anyway (see attributionFor).
func parentScope(int64, string) string { return "" }
