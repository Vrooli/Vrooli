package portspec

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	ephemeralParameterA = 2
)

// ianaDynamicStart and ianaDynamicEnd are the RFC 6335 dynamic/private range.
// Used as a safe fallback when the host OS range cannot be detected, so the
// validator still has a deterministic answer.
const (
	ianaDynamicStart = 49152
	ianaDynamicEnd   = 65535
)

// EphemeralRange describes the OS ephemeral port allocation window at the
// moment it was queried. Min and Max are inclusive.
type EphemeralRange struct {
	Min       int
	Max       int
	Source    string // "linux-proc", "darwin-sysctl", "windows-netsh", "fallback-iana"
	OS        string // runtime.GOOS at detection time
	Fallback  bool   // true when detection failed and the IANA default was substituted
	DetectErr error  // underlying detection error (populated when Fallback is true)
}

// Contains reports whether the given port sits inside the ephemeral window.
func (e EphemeralRange) Contains(port int) bool {
	return port >= e.Min && port <= e.Max
}

// Overlaps reports whether the inclusive range [start,end] intersects the
// ephemeral window. Callers should pass start<=end.
func (e EphemeralRange) Overlaps(start, end int) bool {
	if end < start {
		start, end = end, start
	}
	return start <= e.Max && end >= e.Min
}

// ephemeralReader is the seam tests inject to fake each OS probe.
type ephemeralReader interface {
	Read(ctx context.Context) (EphemeralRange, error)
}

// readEphemeral is the package-level reader. Tests swap it out via
// SetEphemeralReader.
var readEphemeral ephemeralReader = defaultReader{}

// SetEphemeralReader replaces the package-level reader. The returned restore
// function reinstates the prior reader; call it with defer in tests.
func SetEphemeralReader(r ephemeralReader) (restore func()) {
	prev := readEphemeral
	readEphemeral = r
	return func() { readEphemeral = prev }
}

// OSEphemeralRange returns the OS's current ephemeral port window.
//
// Detection strategy:
//   - Linux:   /proc/sys/net/ipv4/ip_local_port_range
//   - macOS:   sysctl -n net.inet.ip.portrange.first net.inet.ip.portrange.last
//   - Windows: netsh int ipv4 show dynamicport tcp
//
// If detection fails for any reason, the IANA-recommended 49152-65535 is
// returned with Fallback=true and DetectErr populated. Callers can treat the
// result as authoritative because every code path yields a well-formed range.
func OSEphemeralRange(ctx context.Context) EphemeralRange {
	if ctx == nil {
		ctx = context.Background()
	}
	r, err := readEphemeral.Read(ctx)
	if err == nil && r.Min > 0 && r.Max >= r.Min {
		return r
	}
	return EphemeralRange{
		Min:       ianaDynamicStart,
		Max:       ianaDynamicEnd,
		Source:    "fallback-iana",
		OS:        runtime.GOOS,
		Fallback:  true,
		DetectErr: err,
	}
}

// defaultReader picks a reader based on runtime.GOOS.
type defaultReader struct{}

func (defaultReader) Read(ctx context.Context) (EphemeralRange, error) {
	switch runtime.GOOS {
	case "linux":
		return readLinuxEphemeral(ctx)
	case "darwin":
		return readDarwinEphemeral(ctx)
	case "windows":
		return readWindowsEphemeral(ctx)
	default:
		return EphemeralRange{}, fmt.Errorf("ports: unsupported OS %q for ephemeral probe", runtime.GOOS)
	}
}

func readLinuxEphemeral(_ context.Context) (EphemeralRange, error) {
	const path = "/proc/sys/net/ipv4/ip_local_port_range"
	data, err := os.ReadFile(path)
	if err != nil {
		return EphemeralRange{}, fmt.Errorf("read %s: %w", path, err)
	}
	return parseLinuxEphemeral(string(data))
}

func parseLinuxEphemeral(raw string) (EphemeralRange, error) {
	fields := strings.Fields(strings.TrimSpace(raw))
	if len(fields) < ephemeralParameterA {
		return EphemeralRange{}, fmt.Errorf("unexpected ip_local_port_range output: %q", raw)
	}
	lo, err := strconv.Atoi(fields[0])
	if err != nil {
		return EphemeralRange{}, fmt.Errorf("parse low %q: %w", fields[0], err)
	}
	hi, err := strconv.Atoi(fields[1])
	if err != nil {
		return EphemeralRange{}, fmt.Errorf("parse high %q: %w", fields[1], err)
	}
	if lo <= 0 || hi < lo {
		return EphemeralRange{}, fmt.Errorf("implausible ephemeral window %d..%d", lo, hi)
	}
	return EphemeralRange{Min: lo, Max: hi, Source: "linux-proc", OS: "linux"}, nil
}

func readDarwinEphemeral(ctx context.Context) (EphemeralRange, error) {
	cctx, cancel := context.WithTimeout(ctx, tuning.ShortOperationDeadline)
	defer cancel()
	out, err := exec.CommandContext(cctx, "sysctl", "-n",
		"net.inet.ip.portrange.first", "net.inet.ip.portrange.last").Output()
	if err != nil {
		return EphemeralRange{}, fmt.Errorf("sysctl portrange: %w", err)
	}
	return parseDarwinEphemeral(string(out))
}

func parseDarwinEphemeral(raw string) (EphemeralRange, error) {
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) < ephemeralParameterA {
		return EphemeralRange{}, fmt.Errorf("unexpected sysctl output: %q", raw)
	}
	lo, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return EphemeralRange{}, fmt.Errorf("parse first %q: %w", lines[0], err)
	}
	hi, err := strconv.Atoi(strings.TrimSpace(lines[1]))
	if err != nil {
		return EphemeralRange{}, fmt.Errorf("parse last %q: %w", lines[1], err)
	}
	if lo <= 0 || hi < lo {
		return EphemeralRange{}, fmt.Errorf("implausible ephemeral window %d..%d", lo, hi)
	}
	return EphemeralRange{Min: lo, Max: hi, Source: "darwin-sysctl", OS: "darwin"}, nil
}

func readWindowsEphemeral(ctx context.Context) (EphemeralRange, error) {
	cctx, cancel := context.WithTimeout(ctx, tuning.HealthCheckTimeout)
	defer cancel()
	out, err := exec.CommandContext(cctx, "netsh", "int", "ipv4", "show", "dynamicport", "tcp").Output()
	if err != nil {
		return EphemeralRange{}, fmt.Errorf("netsh dynamicport: %w", err)
	}
	return parseWindowsEphemeral(string(out))
}

// parseWindowsEphemeral handles the localization-tolerant netsh output, which
// looks roughly like:
//
//	Protocol tcp Dynamic Port Range
//	---------------------------------
//	Start Port      : 49152
//	Number of Ports : 16384
//
// We accept any line that contains ":" and where the left side ends in
// "port" (case-insensitive) for start, and ("number" or "count") for the
// size. This is intentionally permissive to tolerate translated builds.
func parseWindowsEphemeral(raw string) (EphemeralRange, error) {
	var (
		start = -1
		count = -1
	)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:colon]))
		val := strings.TrimSpace(line[colon+1:])
		num, err := strconv.Atoi(val)
		if err != nil {
			continue
		}
		switch {
		case strings.Contains(key, "start") && strings.Contains(key, "port"):
			start = num
		case strings.Contains(key, "number") && strings.Contains(key, "port"),
			strings.Contains(key, "count"):
			count = num
		}
	}
	if start <= 0 || count <= 0 {
		return EphemeralRange{}, fmt.Errorf("unable to parse netsh dynamicport output: %q", raw)
	}
	end := start + count - 1
	if end < start {
		return EphemeralRange{}, fmt.Errorf("implausible netsh window %d..%d", start, end)
	}
	return EphemeralRange{Min: start, Max: end, Source: "windows-netsh", OS: "windows"}, nil
}
