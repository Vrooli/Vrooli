//go:build !linux && !darwin && !windows

package vroolicli

import (
	"errors"
	"time"
)

func installCredentialCopySchedule(string, time.Duration, bool) error {
	return errors.New("credential-store copy scheduling is unsupported on this operating system")
}
