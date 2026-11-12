//go:build !linux

package resources

import (
	platform "github.com/vrooli/platform-go"
	"net/netip"
)

func observeOpenRGBHeadlessListener(endpoint netip.AddrPort, pid int) error {
	listeners, err := platform.TCPListeners(endpoint.Port())
	if err != nil {
		return err
	}
	return verifyOpenRGBNativeListeners(endpoint, pid, listeners)
}
