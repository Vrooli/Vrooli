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

const (
	hostPressureParameterA = 2
	hostPressureLinux      = "linux"
)

// HostPressureProvider reads the kernel's bounded pressure evidence for both
// memory and CPU. It deliberately reports unknown on unsupported or degraded
// hosts; recovery must never treat a missing /proc source as a pressure-clear
// signal.
//
// Memory and CPU are separate signals with separate thresholds because they
// mean different things. Memory stall is nearly always pathological. CPU stall
// is normal on a busy build host, so its threshold is much higher — it is meant
// to catch the state where the run queue is so deep that restarting a service
// makes things worse, not ordinary contention.
//
// CPU was added after 2026-08-19, when this host reached a load average of 110
// on 32 CPUs with a 132-deep run queue while memory sat at 40% used. Memory PSI
// alone never crossed its threshold, so nothing gated the restarts that were
// being attempted into a machine that could not schedule them.
type HostPressureProvider struct {
	someAvg10Threshold    float64
	cpuSomeAvg10Threshold float64
	now                   func() time.Time
	readFile              func(string) ([]byte, error)
	goos                  string

	mu           sync.Mutex
	lastOOMKills int64
	hasBaseline  bool
}

func NewHostPressureProvider(someAvg10Threshold float64) *HostPressureProvider {
	return NewHostPressureProviderWithCPU(someAvg10Threshold, DefaultPressureCPUSomeAvg10)
}

// NewHostPressureProviderWithCPU builds a provider with an explicit CPU stall
// threshold. A non-positive value falls back to the default.
func NewHostPressureProviderWithCPU(someAvg10Threshold, cpuSomeAvg10Threshold float64) *HostPressureProvider {
	if someAvg10Threshold <= 0 {
		someAvg10Threshold = DefaultPressureSomeAvg10
	}
	if cpuSomeAvg10Threshold <= 0 {
		cpuSomeAvg10Threshold = DefaultPressureCPUSomeAvg10
	}
	return &HostPressureProvider{
		someAvg10Threshold:    someAvg10Threshold,
		cpuSomeAvg10Threshold: cpuSomeAvg10Threshold,
		now:                   time.Now,
		readFile:              os.ReadFile,
		goos:                  runtime.GOOS,
	}
}

func (p *HostPressureProvider) Snapshot(context.Context) PressureState {
	if p == nil || p.goos != hostPressureLinux {
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

	// CPU stall is a second, independent way for the host to be unable to act
	// on a restart. An unreadable CPU PSI file degrades that signal to absent
	// rather than failing the whole snapshot: memory evidence is still worth
	// acting on, and reporting unknown here would disable gating entirely.
	cpuSomeAvg10, cpuKnown := 0.0, false
	if cpuRaw, cpuErr := p.readFile("/proc/pressure/cpu"); cpuErr == nil {
		cpuSomeAvg10, cpuKnown = psiSomeAvg10(string(cpuRaw))
	}

	p.mu.Lock()
	oomAdvanced := p.hasBaseline && oomKills > p.lastOOMKills
	p.lastOOMKills = oomKills
	p.hasBaseline = true
	p.mu.Unlock()

	cpuSaturated := cpuKnown && cpuSomeAvg10 >= p.cpuSomeAvg10Threshold
	underPressure := someAvg10 >= p.someAvg10Threshold || oomAdvanced || cpuSaturated
	reason := fmt.Sprintf("memory PSI some.avg10=%.2f threshold=%.2f oom_kill=%d", someAvg10, p.someAvg10Threshold, oomKills)
	if cpuKnown {
		reason += fmt.Sprintf(" cpu PSI some.avg10=%.2f threshold=%.2f", cpuSomeAvg10, p.cpuSomeAvg10Threshold)
	} else {
		reason += " cpu PSI unavailable"
	}
	if oomAdvanced {
		reason += " (oom_kill advanced)"
	}
	if cpuSaturated {
		reason += " (cpu saturated)"
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
			parts := strings.SplitN(field, "=", hostPressureParameterA)
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
