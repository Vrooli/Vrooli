package resources

import (
	"errors"
	"fmt"
	platform "github.com/vrooli/platform-go"
	"net/netip"
)

// Verify every listener at this port, including native IPv6/wildcard and
// foreign owners. Looking only at this process's sockets cannot exclude an
// additional unauthenticated listener bound by a different process.
func verifyOpenRGBNativeListeners(endpoint netip.AddrPort, pid int, listeners []platform.ProcessTCPListener) error {
	if !endpoint.IsValid() || endpoint.Addr().String() != "127.0.0.1" || endpoint.Port() == 0 || pid <= 0 {
		return errors.New("OpenRGB requires an exact native process and literal IPv4 loopback endpoint")
	}
	if len(listeners) == 0 {
		return fmt.Errorf("%w: native OpenRGB listener absent", errOpenRGBNotReady)
	}
	if len(listeners) != 1 {
		return errors.New("OpenRGB SDK port has ambiguous native listeners")
	}
	listener := listeners[0]
	address, err := netip.ParseAddr(listener.Address)
	if err != nil || address != endpoint.Addr() || listener.Port != endpoint.Port() || listener.PID != pid {
		return errors.New("OpenRGB native SDK listener address or process ownership mismatch")
	}
	return nil
}
