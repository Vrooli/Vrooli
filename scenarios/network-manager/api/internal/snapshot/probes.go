package snapshot

import (
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

type RealProbeRunner struct{}

func (RealProbeRunner) Run(cctx context.Context, _ string) ([]ProbeResult, error) {
	results := []ProbeResult{
		{Name: "host_os", Value: runtime.GOOS, Unit: "label", Status: "healthy"},
		{Name: "host_arch", Value: runtime.GOARCH, Unit: "label", Status: "healthy"},
		hostNameMetric(),
		resolverMetric(),
		dnsLookupMetric(cctx),
		dialMetric(cctx, "wan_https_reachability", "tcp", "example.com:443"),
		dialMetric(cctx, "ipv4_availability", "tcp4", "example.com:443"),
		dialMetric(cctx, "ipv6_availability", "tcp6", "example.com:443"),
		jitterMetric(cctx),
		{Name: "gateway_reachability", Value: "unsupported", Unit: "status", Status: "unsupported", Finding: "Gateway reachability requires a platform adapter; no mutation was attempted."},
		{Name: "throughput_availability", Value: "unavailable", Unit: "status", Status: "unavailable", Finding: "Throughput testing is unavailable until a privacy-reviewed local or approved endpoint is configured."},
	}
	return results, nil
}

func hostNameMetric() ProbeResult {
	name, err := os.Hostname()
	if err != nil || strings.TrimSpace(name) == "" {
		return ProbeResult{Name: "host_name", Value: "unavailable", Unit: "label", Status: "unavailable", Finding: "Host name was unavailable from the operating system."}
	}
	return ProbeResult{Name: "host_name", Value: redactLabel(name), Unit: "redacted_label", Status: "healthy"}
}

func resolverMetric() ProbeResult {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return ProbeResult{Name: "resolver_addresses", Value: "unavailable", Unit: "count", Status: "unavailable", Finding: "Resolver address discovery is unavailable on this platform without a host adapter."}
	}
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" {
			count++
		}
	}
	if count == 0 {
		return ProbeResult{Name: "resolver_addresses", Value: "0", Unit: "count", Status: "degraded", Finding: "No resolver addresses were found in /etc/resolv.conf."}
	}
	return ProbeResult{Name: "resolver_addresses", Value: fmt.Sprintf("%d", count), Unit: "count", Status: "healthy"}
}

func dnsLookupMetric(ctx context.Context) ProbeResult {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	start := time.Now()
	_, err := net.DefaultResolver.LookupHost(ctx, "example.com")
	elapsed := time.Since(start)
	if err != nil {
		return ProbeResult{Name: "dns_lookup_latency", Value: "failed", Unit: "ms", Status: "failed", Finding: "DNS lookup for example.com failed: " + err.Error()}
	}
	status := "healthy"
	if elapsed > 250*time.Millisecond {
		status = "degraded"
	}
	return ProbeResult{Name: "dns_lookup_latency", Value: fmt.Sprintf("%d", elapsed.Milliseconds()), Unit: "ms", Status: status}
}

func dialMetric(ctx context.Context, name, network, address string) ProbeResult {
	elapsed, err := dialOnce(ctx, network, address, 3*time.Second)
	if err != nil {
		status := "failed"
		if network == "tcp6" {
			status = "unavailable"
		}
		return ProbeResult{Name: name, Value: "failed", Unit: "ms", Status: status, Finding: fmt.Sprintf("%s probe failed: %v", name, err)}
	}
	status := "healthy"
	if elapsed > 500*time.Millisecond {
		status = "degraded"
	}
	return ProbeResult{Name: name, Value: fmt.Sprintf("%d", elapsed.Milliseconds()), Unit: "ms", Status: status}
}

func jitterMetric(ctx context.Context) ProbeResult {
	var timings []time.Duration
	failures := 0
	for i := 0; i < 3; i++ {
		elapsed, err := dialOnce(ctx, "tcp", "example.com:443", 2*time.Second)
		if err != nil {
			failures++
			continue
		}
		timings = append(timings, elapsed)
	}
	if len(timings) == 0 {
		return ProbeResult{Name: "packet_loss_jitter_approx", Value: "failed", Unit: "status", Status: "failed", Finding: "TCP reachability approximation failed for every attempt."}
	}
	min, max := timings[0], timings[0]
	for _, t := range timings[1:] {
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
	}
	lossPct := failures * 100 / 3
	jitter := max - min
	status := "healthy"
	if failures > 0 || jitter > 150*time.Millisecond {
		status = "degraded"
	}
	return ProbeResult{Name: "packet_loss_jitter_approx", Value: fmt.Sprintf("loss=%d,jitter=%d", lossPct, jitter.Milliseconds()), Unit: "percent_ms", Status: status}
}

func dialOnce(ctx context.Context, network, address string, timeout time.Duration) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
	if err != nil {
		return 0, err
	}
	_ = conn.Close()
	return time.Since(start), nil
}

func redactLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "redacted"
	}
	if len(value) <= 4 {
		return "redacted"
	}
	return value[:2] + "..." + value[len(value)-2:]
}
