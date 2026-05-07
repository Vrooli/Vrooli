package netconsole

import "os"

// defaultGetenv is split into its own file so the os.Getenv import doesn't
// pull into handler.go (which has its own GetenvFn seam). Production builds
// resolve to os.Getenv; tests substitute GetenvFn directly.
func defaultGetenv(key string) string {
	return os.Getenv(key)
}
