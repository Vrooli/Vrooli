//go:build linux

package hostpressure

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	pressureLinuxParameterA = 2
)

const forkRateProvenance = "system-monitor:platform_forkrate_linux:/proc/stat"

func collect(ctx context.Context, opts Options) PressureSnapshot {
	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}
	s := PressureSnapshot{CapturedAt: now()}
	root := opts.ProcRoot
	if root == "" {
		root = "/proc"
	}
	read := func(name string) ([]byte, error) { return readProcFile(root, name) }

	s.CPUPressure = readPSI(read, "pressure/cpu")
	if b, err := read("loadavg"); err == nil {
		fields := strings.Fields(string(b))
		if len(fields) > 0 {
			v, err := strconv.ParseFloat(fields[0], 64)
			if err == nil {
				s.Load1 = NewRead(v, "linux:/proc/loadavg")
			}
		}
		if s.Load1.State != Read {
			s.Load1 = NewUnread("linux:/proc/loadavg", "first load field is unread")
		}
	} else {
		s.Load1 = NewUnread("linux:/proc/loadavg", err.Error())
	}
	mem := parseMeminfo(read)
	s.MemoryTotal, s.MemoryAvail = mem["MemTotal"], mem["MemAvailable"]
	s.SwapTotal = mem["SwapTotal"]
	if total, ok := s.SwapTotal.Number(); ok {
		if free, ok := mem["SwapFree"].Number(); ok {
			s.SwapUsed = NewRead(total-free, "linux:/proc/meminfo")
		}
	}
	if s.SwapTotal.State == Unread {
		s.SwapUsed = NewUnread("linux:/proc/meminfo", "SwapTotal is unread")
	}
	if b, err := read("stat"); err == nil {
		if n, ok := statCounter(string(b), "processes"); ok {
			s.ForkCounter = NewRead(float64(n), forkRateProvenance)
			if opts.Previous != nil && opts.Previous.ForkCounter.State == Read {
				elapsed := s.CapturedAt.Sub(opts.Previous.CapturedAt).Seconds()
				if elapsed > 0 {
					s.ForkRate = NewRead((float64(n)-opts.Previous.ForkCounter.Value)/elapsed, forkRateProvenance)
				}
			}
			if s.ForkRate.State != Read {
				s.ForkRate = NewUnread(forkRateProvenance, "not primed")
			}
		} else {
			s.ForkCounter = NewUnread(forkRateProvenance, "processes counter is absent")
		}
	} else {
		s.ForkCounter = NewUnread(forkRateProvenance, err.Error())
	}
	s.ProcessCount, s.Processes = linuxProcesses(ctx, root)
	return s
}

// readProcFile accepts the real /proc layout and the flat captured layout
// used by host-pressure fixtures. The fallback is limited to named fixture
// files, so a live Linux read remains an ordinary /proc read.
func readProcFile(root, name string) ([]byte, error) {
	path := filepath.Join(root, name)
	b, err := os.ReadFile(path)
	if err == nil {
		return b, nil
	}
	flat := map[string]string{
		"pressure/cpu":    "proc-pressure-cpu",
		"pressure/io":     "proc-pressure-io",
		"pressure/memory": "proc-pressure-memory",
		"loadavg":         "proc-loadavg",
		"meminfo":         "proc-meminfo",
		"stat":            "proc-stat-t1",
	}
	if fallback, ok := flat[name]; ok {
		if fixture, fixtureErr := os.ReadFile(filepath.Join(root, fallback)); fixtureErr == nil {
			return fixture, nil
		}
	}
	return nil, err
}

func readPSI(read func(string) ([]byte, error), name string) Reading {
	b, err := read(name)
	if err != nil {
		return NewUnread("linux:/proc/"+name, err.Error())
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "some ") {
			for _, field := range strings.Fields(line)[1:] {
				if strings.HasPrefix(field, "avg10=") {
					v, e := strconv.ParseFloat(strings.TrimPrefix(field, "avg10="), 64)
					if e == nil {
						return NewRead(v, "linux:/proc/"+name)
					}
				}
			}
		}
	}
	return NewUnread("linux:/proc/"+name, "avg10 is absent")
}

func parseMeminfo(read func(string) ([]byte, error)) map[string]Reading {
	result := map[string]Reading{}
	b, err := read("meminfo")
	if err != nil {
		for _, k := range []string{"MemTotal", "MemAvailable", "SwapTotal", "SwapFree"} {
			result[k] = NewUnread("linux:/proc/meminfo", err.Error())
		}
		return result
	}
	for _, line := range strings.Split(string(b), "\n") {
		parts := strings.Fields(line)
		if len(parts) < pressureLinuxParameterA {
			continue
		}
		v, e := strconv.ParseUint(parts[1], 10, 64)
		if e != nil {
			continue
		}
		if len(parts) > 2 && parts[2] == "kB" {
			v *= 1024
		}
		result[strings.TrimSuffix(parts[0], ":")] = NewRead(float64(v), "linux:/proc/meminfo")
	}
	for _, k := range []string{"MemTotal", "MemAvailable", "SwapTotal", "SwapFree"} {
		if _, ok := result[k]; !ok {
			result[k] = NewUnread("linux:/proc/meminfo", k+" is absent")
		}
	}
	return result
}

func linuxProcesses(ctx context.Context, root string) (Reading, []Process) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return NewUnread("linux:/proc", err.Error()), nil
	}
	var processes []Process
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return NewUnread("linux:/proc", ctx.Err().Error()), processes
		default:
		}
		pid, err := strconv.ParseInt(entry.Name(), 10, 64)
		if err != nil {
			continue
		}
		p := Process{PID: pid}
		b, err := os.ReadFile(filepath.Join(root, entry.Name(), "status"))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(b), "\n") {
			f := strings.Fields(line)
			if len(f) < pressureLinuxParameterA {
				continue
			}
			switch f[0] {
			case "Name:":
				p.Name = f[1]
			case "PPid:":
				p.PPID, _ = strconv.ParseInt(f[1], 10, 64)
			case "VmRSS:":
				p.Resident = parseBytes(f[1:])
			case "VmSwap:":
				p.Swapped = parseBytes(f[1:])
			}
		}
		processes = append(processes, p)
	}
	return NewRead(float64(len(processes)), "linux:/proc process directories"), processes
}

func parseBytes(fields []string) uint64 {
	if len(fields) == 0 {
		return 0
	}
	n, e := strconv.ParseUint(fields[0], 10, 64)
	if e != nil {
		return 0
	}
	if len(fields) > 1 && fields[1] == "kB" {
		n *= 1024
	}
	return n
}
