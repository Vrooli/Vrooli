//go:build linux

package resources

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
)

// The existing ownership probe attributes PIDs by port. Check the actual
// kernel bindings as well, since a wildcard or unattributed second socket can
// otherwise be invisible to a readiness probe against literal loopback.
func observeOpenRGBHeadlessListener(endpoint netip.AddrPort, pid int) error {
	var exact, other int
	for _, table := range []struct {
		path string
		ipv6 bool
	}{{"/proc/net/tcp", false}, {"/proc/net/tcp6", true}} {
		file, err := os.Open(table.path)
		if table.ipv6 && errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("observe OpenRGB listener addresses: %w", err)
		}
		local, unqualified, err := openRGBListenerCounts(file, endpoint.Port(), table.ipv6)
		_ = file.Close()
		if err != nil {
			return fmt.Errorf("observe OpenRGB listener addresses in %s: %w", table.path, err)
		}
		exact += local
		other += unqualified
	}
	if other != 0 || exact > 1 {
		return fmt.Errorf("OpenRGB port %d has unexpected or ambiguous listener addresses", endpoint.Port())
	}
	if exact == 0 {
		return fmt.Errorf("%w: no listener at %s", errOpenRGBNotReady, endpoint)
	}
	return managedUpdateOwnedListener(int(endpoint.Port()), pid)
}
