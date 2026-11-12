//go:build windows

package resources

import (
	platform "github.com/vrooli/platform-go"
	"os"
)

func openRGBNativeOperatorPrincipal() (string, error) {
	security, err := platform.ProcessSecurity(os.Getpid())
	if err != nil {
		return "", err
	}
	return "sid:" + security.Principal, nil
}
