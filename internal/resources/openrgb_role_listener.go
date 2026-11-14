package resources

import (
	"net/netip"
)

func observeManagedUpdateRoleListener(resource string, port, pid int) error {
	if resource == openRGBResourceName {
		return observeOpenRGBHeadlessListener(netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(port)), pid)
	}
	return managedUpdateOwnedListener(port, pid)
}
