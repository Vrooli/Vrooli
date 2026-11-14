package tuning

import "os"

// File modes are named by the kind of filesystem object they protect. Keep
// Dir and Executable separate even though both currently use 0755.
const (
	PermDir               os.FileMode = 0o755
	PermPrivateDir        os.FileMode = 0o700
	PermFile              os.FileMode = 0o644
	PermSecret            os.FileMode = 0o600
	PermGroupDir          os.FileMode = 0o750
	PermExecutable        os.FileMode = 0o755
	PermReadOnlyGroup     os.FileMode = 0o640
	PermSudoers           os.FileMode = 0o440
	PermSocket            os.FileMode = 0o660
	PermLock              os.FileMode = 0o666
	PermExecuteMask       os.FileMode = 0o111
	PermGroupAndOtherMask os.FileMode = 0o077
	PermGroupReadWrite    os.FileMode = 0o060
	PermOwnerWrite        os.FileMode = 0o200
	PermNone              os.FileMode = 0o000
)
