//go:build !linux

package agentsession

import (
	"time"

	platformgo "github.com/vrooli/platform-go"
)

// The process facts owner inference and liveness need (parent, start stamp,
// controlling terminal, every process's environment) are read from /proc on
// Linux. No other platform's reader is written or host-verified yet, so each
// reports unsupported and the affected owners stay unknown, which is never
// reaped.

func processFact(int) (ProcessFact, error) { return ProcessFact{}, platformgo.ErrUnsupported }

func processStartTime(int) (string, error) { return "", platformgo.ErrUnsupported }

func processStartedAt(int) (time.Time, error) { return time.Time{}, platformgo.ErrUnsupported }

func ttyActivity(string) (time.Time, error) { return time.Time{}, platformgo.ErrUnsupported }

func desktopAppProcesses(string) ([]int, error) { return nil, platformgo.ErrUnsupported }

func processMemory(int) (int64, int64, error) { return 0, 0, platformgo.ErrUnsupported }
