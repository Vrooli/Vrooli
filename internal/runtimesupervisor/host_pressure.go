package runtimesupervisor

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HostPressureProvider reads the kernel's bounded memory-pressure evidence.
// It deliberately reports unknown on unsupported or degraded hosts; recovery
// must never treat a missing /proc source as a pressure-clear signal.
type HostPressureProvider struct {
	someAvg10Threshold float64
	now                func() time.Time
	readFile           func(string) ([]byte, error)
	goos               string

	mu           sync.Mutex
	lastOOMKills int64
	hasBaseline  bool
}

func NewHostPressureProvider(someAvg10Threshold float64) *HostPressureProvider {
	if someAvg10Threshold <= 0 {
		someAvg10Threshold = DefaultPressureSomeAvg10
	}
	return &HostPressureProvider{
		someAvg10Threshold: someAvg10Threshold,
		now:                time.Now,
		readFile:           os.ReadFile,
		goos:               runtime.GOOS,
	}
}

func (p *HostPressureProvider) Snapshot(context.Context) PressureState {
	if p == nil || p.goos != "linux" {
		return PressureState{Source: "host-psi", Reason: "memory PSI is unavailable on this host"}
	}
	psiRaw, err := p.readFile("/proc/pressure/memory")
	if err != nil {
		return PressureState{Source: "host-psi", Reason: fmt.Sprintf("read memory PSI: %v", err)}
	}
	vmstatRaw, err := p.readFile("/proc/vmstat")
	if err != nil {
		return PressureState{Source: "host-psi", Reason: fmt.Sprintf("read vmstat: %v", err)}
	}
	someAvg10, ok := psiSomeAvg10(string(psiRaw))
	if !ok {
		return PressureState{Source: "host-psi", Reason: "memory PSI some.avg10 is malformed"}
	}
	oomKills, ok := vmStatCounter(string(vmstatRaw), "oom_kill")
	if !ok {
		return PressureState{Source: "host-psi", Reason: "oom_kill counter is unavailable"}
	}

	p.mu.Lock()
	oomAdvanced := p.hasBaseline && oomKills > p.lastOOMKills
	p.lastOOMKills = oomKills
	p.hasBaseline = true
	p.mu.Unlock()

	underPressure := someAvg10 >= p.someAvg10Threshold || oomAdvanced
	reason := fmt.Sprintf("memory PSI some.avg10=%.2f threshold=%.2f oom_kill=%d", someAvg10, p.someAvg10Threshold, oomKills)
	if oomAdvanced {
		reason += " (oom_kill advanced)"
	}
	return PressureState{Known: true, UnderPressure: underPressure, ObservedAt: p.now(), Source: "host-psi", Reason: reason}
}

func psiSomeAvg10(raw string) (float64, bool) {
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "some" {
			continue
		}
		for _, field := range fields[1:] {
			parts := strings.SplitN(field, "=", 2)
			if len(parts) != 2 || parts[0] != "avg10" {
				continue
			}
			value, err := strconv.ParseFloat(parts[1], 64)
			return value, err == nil
		}
	}
	return 0, false
}

func vmStatCounter(raw, name string) (int64, bool) {
	prefix := name + " "
	for _, line := range strings.Split(raw, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, prefix)), 10, 64)
		return value, err == nil
	}
	return 0, false
}
