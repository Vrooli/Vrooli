package resources

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// The admitted native launcher is Linux and binds 127.0.0.1. procfs
// prints that IPv4 address in little-endian order. Any IPv6 listener at the
// same port is outside this launch contract, including IPv4-mapped addresses.
func openRGBListenerCounts(table io.Reader, port uint16, ipv6 bool) (exact, other int, err error) {
	portHex := fmt.Sprintf("%04X", port)
	scanner := bufio.NewScanner(table)
	if !scanner.Scan() || !strings.Contains(scanner.Text(), "local_address") {
		return 0, 0, errors.New("TCP listener table header is unavailable")
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			return 0, 0, errors.New("invalid TCP listener row")
		}
		address, localPort, ok := strings.Cut(fields[1], ":")
		if !ok {
			return 0, 0, errors.New("invalid TCP local address")
		}
		if fields[3] != "0A" || !strings.EqualFold(localPort, portHex) {
			continue
		}
		if !ipv6 && strings.EqualFold(address, "0100007F") {
			exact++
		} else {
			other++
		}
	}
	return exact, other, scanner.Err()
}
