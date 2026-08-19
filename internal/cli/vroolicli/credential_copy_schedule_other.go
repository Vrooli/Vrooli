//go:build !linux && !darwin && !windows

package vroolicli

import (
	"github.com/vrooli/vrooli/internal/resources/securestore"
	"time"
)

func installCredentialCopySchedule(executable string, interval time.Duration, enabled bool) error {
	_, err := securestore.InstallCopySchedule(executable, interval, enabled)
	return err
}
