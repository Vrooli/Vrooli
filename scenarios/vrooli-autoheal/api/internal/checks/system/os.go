package system

import "runtime"

// checkOS is a small seam for platform-bound checks. Production uses the
// process host; tests replace it to exercise non-native behavior on Linux.
var checkOS = runtime.GOOS
