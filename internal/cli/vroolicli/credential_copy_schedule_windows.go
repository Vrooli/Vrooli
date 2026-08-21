//go:build windows

package vroolicli

import (
	"time"

	"github.com/vrooli/vrooli/internal/resources/securestore"
)

func installCredentialCopySchedule(executable string, interval time.Duration, enabled bool) error {
	_, err := securestore.InstallCopySchedule(executable, interval, enabled)
	return err
}
