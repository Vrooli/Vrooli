package platform

var verifyDarwinNoChildren = func(int) error { return ErrUnsupported }

// VerifyDarwinNoChildrenProcess checks the kernel-inherited fork/spawn denial required by Darwin's ProcessOnly service body.
// The domain owner applies these in its one transport profile before witnessing
// selected-device access. This readonly seam never installs a second sandbox.
// The caller separately proves boot, unique birth, image and manager ownership.
func VerifyDarwinNoChildrenProcess(pid int) error {
	if pid <= 0 {
		return ErrOperationIdentity
	}
	return verifyDarwinNoChildren(pid)
}
