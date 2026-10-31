//go:build linux

package sources

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os"
	"syscall"
)

// The first fidelity profile is intentionally narrow. Reject attributes we
// cannot reproduce instead of silently downgrading the preservation claim.
func checkpointMetadata(path string, info os.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("metadata unavailable")
	}
	if int(st.Uid) != os.Geteuid() || int(st.Gid) != os.Getegid() {
		return fmt.Errorf("checkpoint requires current owner and group: %s", path)
	}
	if info.Mode()&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) != 0 {
		return fmt.Errorf("special mode requires an extended metadata profile: %s", path)
	}
	if info.Mode().IsRegular() && st.Nlink > 1 {
		return fmt.Errorf("hard links require an extended metadata profile: %s", path)
	}
	n, err := unix.Llistxattr(path, nil)
	if err != nil && err != unix.ENOTSUP {
		return fmt.Errorf("cannot inspect extended attributes: %w", err)
	}
	if n > 0 {
		return fmt.Errorf("extended attributes or ACLs require an extended metadata profile: %s", path)
	}
	return nil
}
