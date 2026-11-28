//go:build darwin && !cgo

package hostreqkit

import (
	"fmt"
	platform "github.com/vrooli/platform-go"
	osuser "os/user"
)

func lookupDarwinInvokingUser(string) (*osuser.User, error) {
	return nil, fmt.Errorf("macOS cross-user OpenDirectory home resolution requires the native CGO build: %w", platform.ErrUnsupported)
}
