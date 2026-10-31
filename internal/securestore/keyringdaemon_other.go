//go:build !linux

package securestore

import (
	"os"
	"time"
)

var keyringDaemonStartTime = func() (time.Time, bool) { return time.Time{}, false }

func addStaleDaemonReport(report *KeyringReport, _ os.FileInfo) {
	report.StaleDaemonCheck = "not-run"
}
