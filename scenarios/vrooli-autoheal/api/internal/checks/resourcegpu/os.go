package resourcegpu

import "runtime"

// checkOS is injectable so the Linux-only boundary is testable on Linux.
var checkOS = runtime.GOOS
