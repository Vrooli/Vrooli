// Package envx is the CLI environment seam.
package envx

import "os"

// Get reads an environment variable through the production seam. Tests can
// provide a substitute reader to command registration without touching the
// process environment.
func Get(key string) string { return os.Getenv(key) }
