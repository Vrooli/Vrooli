//go:build !unix

package runs

// runtimeDeviceKey is unavailable on this platform, so the degraded-write
// redundancy check falls back to persisted identity only and stays
// conservative when a destination has none.
func runtimeDeviceKey(string) string { return "" }
