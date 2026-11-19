//go:build !unix

package sysmounts

// DeviceKey is unavailable on this platform; callers fall back to persisted
// device identity and stay conservative when none exists.
func DeviceKey(string) string { return "" }
