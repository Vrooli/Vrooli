package metrics

import (
	"fmt"
	"os"
	"sort"
	"strings"

	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	"system-monitor/cli/internal/support"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{
		Name:        "metrics",
		Description: "Inspect current, detailed, historical, process, and infrastructure metrics",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "current", Description: "Get the current metrics snapshot", Run: func(args []string) error { return runCurrent(core, args) }},
			{Name: "detailed", Description: "Get detailed system metrics", Run: func(args []string) error { return runDetailed(core, args) }},
			{Name: "processes", Description: "Get process monitoring metrics", Run: func(args []string) error { return runProcesses(core, args) }},
			{Name: "infrastructure", Description: "Get infrastructure pool and queue metrics", Run: func(args []string) error { return runInfrastructure(core, args) }},
			{Name: "timeline", Description: "Get recent metrics history", Run: func(args []string) error { return runTimeline(core, args) }},
		},
	}
}

func runCurrent(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("metrics current")
	fresh := fs.Bool("fresh", false, "Collect a fresh metrics snapshot")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	var body []byte
	var err error
	if *fresh {
		body, err = core.Get("/metrics/current", map[string][]string{"fresh": {"true"}})
	} else {
		body, err = core.Get("/metrics/current", nil)
	}
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response metricspb.MetricsResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Snapshot time: %s", support.FormatTimestamp(response.GetTimestamp())),
			fmt.Sprintf("CPU usage: %s", support.FormatPercent(response.GetCpuUsage())),
			fmt.Sprintf("Memory usage: %s", support.FormatPercent(response.GetMemoryUsage())),
			fmt.Sprintf("TCP connections: %d", response.GetTcpConnections()),
			fmt.Sprintf("GPU usage: %s", support.FormatMaybePercent(response.GpuUsage)),
		},
		ResultsHeading: "Key Signals",
		Results: []string{
			fmt.Sprintf("CPU vs memory delta: %.1f%%", response.GetCpuUsage()-response.GetMemoryUsage()),
			fmt.Sprintf("Fresh collection requested: %s", support.BoolString(*fresh, "yes", "no")),
		},
		RetrievalHints: []string{
			"system-monitor metrics detailed",
			"system-monitor metrics timeline --window 300",
			"system-monitor alerts",
		},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runDetailed(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("metrics detailed")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/metrics/detailed", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response metricspb.DetailedMetrics
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}

	results := []string{
		fmt.Sprintf("Load average: %s", floatList(response.GetCpuDetails().GetLoadAverage())),
		fmt.Sprintf("Top CPU processes: %s", processNames(response.GetCpuDetails().GetTopProcesses())),
		fmt.Sprintf("Top memory processes: %s", processNames(response.GetMemoryDetails().GetTopProcesses())),
		fmt.Sprintf("Network bandwidth: in %.2f Mbps / out %.2f Mbps", response.GetNetworkDetails().GetNetworkStats().GetBandwidthInMbps(), response.GetNetworkDetails().GetNetworkStats().GetBandwidthOutMbps()),
		fmt.Sprintf("File descriptors: %d / %d (%.1f%%)", response.GetSystemDetails().GetFileDescriptors().GetUsed(), response.GetSystemDetails().GetFileDescriptors().GetMax(), response.GetSystemDetails().GetFileDescriptors().GetPercent()),
	}
	if gpu := response.GetGpuDetails(); gpu != nil {
		results = append(results,
			fmt.Sprintf("GPU devices: %d", gpu.GetSummary().GetDeviceCount()),
			fmt.Sprintf("GPU memory used: %.1f MB / %.1f MB", gpu.GetSummary().GetUsedMemoryMb(), gpu.GetSummary().GetTotalMemoryMb()),
		)
	}
	if deps := response.GetSystemDetails().GetServiceDependencies(); len(deps) > 0 {
		results = append(results, "Dependencies: "+serviceHealthSummary(deps))
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Detailed metrics collected at %s", support.FormatTimestamp(response.GetTimestamp())),
			fmt.Sprintf("CPU usage: %s", support.FormatPercent(response.GetCpuDetails().GetUsage())),
			fmt.Sprintf("Memory usage: %s", support.FormatPercent(response.GetMemoryDetails().GetUsage())),
		},
		ResultsHeading: "System Detail",
		Results:        results,
		RetrievalHints: []string{"system-monitor metrics processes", "system-monitor metrics infrastructure", "system-monitor metrics timeline --window 300"},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runProcesses(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("metrics processes")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/metrics/processes", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response metricspb.ProcessMonitorData
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Snapshot time: %s", support.FormatTimestamp(response.GetTimestamp())),
			fmt.Sprintf("Zombie processes: %d", len(response.GetProcessHealth().GetZombieProcesses())),
			fmt.Sprintf("High-thread processes: %d", len(response.GetProcessHealth().GetHighThreadCount())),
			fmt.Sprintf("Leak candidates: %d", len(response.GetProcessHealth().GetLeakCandidates())),
		},
		ResultsHeading: "Process Matrix",
		Results:        processRows(response.GetResourceMatrix()),
		RetrievalHints: []string{"system-monitor metrics current", "system-monitor investigations trigger --note \"review abnormal processes\""},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runInfrastructure(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("metrics infrastructure")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}

	body, err := core.Get("/metrics/infrastructure", nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response metricspb.InfrastructureMonitorData
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}

	results := append(poolRows("DB", response.GetDatabasePools()), poolRows("HTTP", response.GetHttpClientPools())...)
	results = append(results, fmt.Sprintf("Storage IO: read %.2f MB/s, write %.2f MB/s, disk queue depth %.2f, io wait %.2f%%", response.GetStorageIo().GetReadMbPerSec(), response.GetStorageIo().GetWriteMbPerSec(), response.GetStorageIo().GetDiskQueueDepth(), response.GetStorageIo().GetIoWaitPercent()))

	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Snapshot time: %s", support.FormatTimestamp(response.GetTimestamp())),
			fmt.Sprintf("Database pools: %d", len(response.GetDatabasePools())),
			fmt.Sprintf("HTTP client pools: %d", len(response.GetHttpClientPools())),
			fmt.Sprintf("Redis subscribers: %d", response.GetMessageQueues().GetRedisPubsub().GetSubscribers()),
			fmt.Sprintf("Background jobs pending: %d", response.GetMessageQueues().GetBackgroundJobs().GetPending()),
		},
		ResultsHeading: "Infrastructure State",
		Results:        results,
		RetrievalHints: []string{"system-monitor status", "system-monitor metrics detailed"},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func runTimeline(core *cliapp.ScenarioApp, args []string) error {
	fs := support.NewFlagSet("metrics timeline")
	window := fs.Int("window", 120, "Timeline window in seconds")
	interval := fs.Int("interval", 5, "Sample interval in seconds")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if *window <= 0 || *interval <= 0 {
		return fmt.Errorf("--window and --interval must be greater than 0")
	}

	body, err := core.Get("/metrics/timeline", map[string][]string{
		"window":   {fmt.Sprintf("%d", *window)},
		"interval": {fmt.Sprintf("%d", *interval)},
	})
	if err != nil {
		return err
	}
	if *jsonOutput {
		return support.PrettyPrintJSON(body)
	}

	var response metricspb.MetricsTimelineResponse
	if err := support.DecodeProto(body, &response); err != nil {
		return err
	}
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Window: %ds", response.GetWindowSeconds()),
			fmt.Sprintf("Sample interval: %ds", response.GetSampleIntervalSeconds()),
			fmt.Sprintf("Samples: %d", len(response.GetSamples())),
		},
		ResultsHeading: "Recent Samples",
		Results:        timelineRows(response.GetSamples()),
		RetrievalHints: []string{"system-monitor watch", "system-monitor metrics current --fresh"},
	}
	return cliapp.RenderListReport(os.Stdout, report)
}

func floatList(values []float64) string {
	if len(values) == 0 {
		return "n/a"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%.2f", value))
	}
	return strings.Join(parts, ", ")
}

func processNames(processes []*metricspb.ProcessInfo) string {
	if len(processes) == 0 {
		return "none"
	}
	limit := len(processes)
	if limit > 5 {
		limit = 5
	}
	names := make([]string, 0, limit)
	for _, process := range processes[:limit] {
		names = append(names, fmt.Sprintf("%s(pid=%d)", process.GetName(), process.GetPid()))
	}
	return strings.Join(names, ", ")
}

func serviceHealthSummary(services []*metricspb.ServiceHealth) string {
	if len(services) == 0 {
		return "none"
	}
	items := make([]string, 0, len(services))
	for _, service := range services {
		items = append(items, fmt.Sprintf("%s=%s", service.GetName(), service.GetStatus()))
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}

func processRows(processes []*metricspb.ProcessInfo) []string {
	if len(processes) == 0 {
		return []string{"No process entries were returned."}
	}
	limit := len(processes)
	if limit > 8 {
		limit = 8
	}
	rows := make([]string, 0, limit)
	for _, process := range processes[:limit] {
		rows = append(rows, fmt.Sprintf("%s (pid=%d) cpu=%s memory=%.1fMB threads=%d connections=%d status=%s", process.GetName(), process.GetPid(), support.FormatPercent(process.GetCpuPercent()), process.GetMemoryMb(), process.GetThreads(), process.GetConnections(), process.GetStatus()))
	}
	return rows
}

func poolRows(prefix string, pools []*metricspb.ConnectionPool) []string {
	if len(pools) == 0 {
		return []string{fmt.Sprintf("%s pools: none", prefix)}
	}
	rows := make([]string, 0, len(pools))
	for _, pool := range pools {
		rows = append(rows, fmt.Sprintf("%s pool %s active=%d idle=%d max=%d waiting=%d healthy=%t leakRisk=%s", prefix, pool.GetName(), pool.GetActive(), pool.GetIdle(), pool.GetMaxSize(), pool.GetWaiting(), pool.GetHealthy(), pool.GetLeakRisk()))
	}
	return rows
}

func timelineRows(samples []*metricspb.MetricTimelineSample) []string {
	if len(samples) == 0 {
		return []string{"No timeline samples were returned."}
	}
	limit := len(samples)
	if limit > 10 {
		limit = 10
	}
	start := len(samples) - limit
	rows := make([]string, 0, limit)
	for _, sample := range samples[start:] {
		rows = append(rows, fmt.Sprintf("%s cpu=%s memory=%s tcp=%d gpu=%s", support.FormatTimestamp(sample.GetTimestamp()), support.FormatPercent(sample.GetCpuUsage()), support.FormatPercent(sample.GetMemoryUsage()), sample.GetTcpConnections(), support.FormatMaybePercent(sample.GpuUsage)))
	}
	return rows
}
