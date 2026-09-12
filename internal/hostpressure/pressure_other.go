//go:build !linux && !darwin && !windows

package hostpressure

import (
	"context"
	"time"
)

func collect(context.Context, Options) PressureSnapshot {
	t := time.Now()
	u := func(name string) Reading {
		return NewUnread("unsupported:"+name, "sensor is not implemented on this operating system")
	}
	return PressureSnapshot{CapturedAt: t, CPUPressure: u("cpu-pressure"), Load1: u("load"), MemoryTotal: u("memory-total"), MemoryAvail: u("memory-available"), SwapTotal: u("swap-total"), SwapUsed: u("swap-used"), ProcessCount: u("process-count"), ForkRate: u("fork-rate"), ForkCounter: u("fork-counter")}
}
