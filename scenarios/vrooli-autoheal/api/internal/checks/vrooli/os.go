package vrooli

import "runtime"

// checkOS is injectable so platform-bound checks can prove their boundary on
// the Linux development host.
var checkOS = runtime.GOOS
