package metrics

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	devicegraphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/devicegraph/devicegraphconnect"
	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics/metricsconnect"

	"github.com/vrooli/cli-core/cliapp"

	"system-monitor/cli/internal/support"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "metrics",
		Description: "Inspect current, detailed, historical, process, and infrastructure metrics",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{Name: "current", Description: "Get the current metrics snapshot", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "fresh", Description: "Collect a fresh metrics snapshot", Bool: true}}}, RunCtx: h.current},
			{Name: "detailed", Description: "Get detailed system metrics", RunCtx: h.detailed},
			{Name: "processes", Description: "Get process monitoring metrics", RunCtx: h.processes},
			{Name: "process-timeline", Description: "Top process consumers over a window, grouped by source scenario", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Description: "Window duration (e.g. 5m, 1h) or bare seconds", Default: "5m"}, {Name: "owner", Description: "Filter to a single owner/scenario"}, {Name: "top", Description: "Maximum ranked consumers to return", Default: "20"}}}, RunCtx: h.processTimeline},
			{Name: "infrastructure", Description: "Get infrastructure pool and queue metrics", RunCtx: h.infrastructure},
			{Name: "devices", Description: "Show the graded hardware device graph: every enumerated device with its per-rung observability state", RunCtx: h.devices},
			{Name: "timeline", Description: "Get recent metrics history", Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "window", Description: "Timeline window in seconds", Default: "120"}, {Name: "interval", Description: "Sample interval in seconds", Default: "5"}}}, RunCtx: h.timeline},
		},
	}
}

type handlers struct {
	client metricsconnect.MetricsServiceClient
	// deviceGraph is a second client because the device graph is its own
	// service: it is a topology read on a 30s cache, not a metrics sample.
	deviceGraph devicegraphconnect.DeviceGraphServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		client:      metricsconnect.NewMetricsServiceClient(httpClient, baseURL),
		deviceGraph: devicegraphconnect.NewDeviceGraphServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) current(ctx cliapp.RunContext) error {
	resp, err := h.client.GetCurrentMetrics(context.Background(), connect.NewRequest(&metricspb.GetCurrentMetricsRequest{
		Fresh: ctx.BoolFlag("fresh"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get current metrics", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetMetrics() == nil {
		return fmt.Errorf("server returned no current metrics")
	}

	response := resp.Msg.GetMetrics()
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Snapshot time: %s", support.FormatTimestamp(response.GetTimestamp())),
			fmt.Sprintf("CPU usage: %s", support.FormatPercent(response.GetCpuUsage())),
			fmt.Sprintf("Memory usage: %s", support.FormatPercent(response.GetMemoryUsage())),
			fmt.Sprintf("TCP connections: %d", response.GetTcpConnections()),
			fmt.Sprintf("GPU usage: %s", support.FormatMaybePercent(response.GpuUsage)),
			fmt.Sprintf("Swap traffic: %s", metricStateSummary(response.GetSwapTraffic())),
			fmt.Sprintf("Major faults: %s", metricStateSummary(response.GetMajorFaults())),
			fmt.Sprintf("Fragmentation index: %s", metricStateSummary(response.GetFragmentationIndex())),
		},
		ResultsHeading: "Key Signals",
		Results: []string{
			fmt.Sprintf("CPU vs memory delta: %.1f%%", response.GetCpuUsage()-response.GetMemoryUsage()),
			fmt.Sprintf("Fresh collection requested: %s", support.BoolString(ctx.BoolFlag("fresh"), "yes", "no")),
		},
		RetrievalHints: []string{
			"system-monitor metrics detailed",
			"system-monitor metrics timeline --window 300",
			"system-monitor alerts",
		},
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) detailed(ctx cliapp.RunContext) error {
	resp, err := h.client.GetDetailedMetrics(context.Background(), connect.NewRequest(&metricspb.GetDetailedMetricsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get detailed metrics", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetMetrics() == nil {
		return fmt.Errorf("server returned no detailed metrics")
	}

	response := resp.Msg.GetMetrics()
	results := []string{
		fmt.Sprintf("Load average: %s", floatList(response.GetCpuDetails().GetLoadAverage())),
		fmt.Sprintf("Top CPU processes: %s", processNames(response.GetCpuDetails().GetTopProcesses())),
		fmt.Sprintf("Top memory processes: %s", processNames(response.GetMemoryDetails().GetTopProcesses())),
		fmt.Sprintf("Top paging processes: %s", processNames(response.GetMemoryDetails().GetTopPagingProcesses())),
		fmt.Sprintf("Swap traffic: %s", metricStateSummary(response.GetMemoryDetails().GetPaging().GetSwapTrafficPagesPerSecond())),
		fmt.Sprintf("Major faults: %s", metricStateSummary(response.GetMemoryDetails().GetPaging().GetMajorFaultsPerSecond())),
		fmt.Sprintf("Fragmentation max free order: %s", metricStateSummary(response.GetMemoryDetails().GetFragmentation().GetMaxFreeOrder())),
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
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) processes(ctx cliapp.RunContext) error {
	resp, err := h.client.GetProcessMonitor(context.Background(), connect.NewRequest(&metricspb.GetProcessMonitorRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get process metrics", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetData() == nil {
		return fmt.Errorf("server returned no process metrics")
	}

	response := resp.Msg.GetData()
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
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) infrastructure(ctx cliapp.RunContext) error {
	resp, err := h.client.GetInfrastructureMonitor(context.Background(), connect.NewRequest(&metricspb.GetInfrastructureMonitorRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get infrastructure metrics", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetData() == nil {
		return fmt.Errorf("server returned no infrastructure metrics")
	}

	response := resp.Msg.GetData()
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
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) timeline(ctx cliapp.RunContext) error {
	window, err := positiveIntFlag(ctx, "window")
	if err != nil {
		return err
	}
	interval, err := positiveIntFlag(ctx, "interval")
	if err != nil {
		return err
	}

	resp, err := h.client.GetMetricsTimeline(context.Background(), connect.NewRequest(&metricspb.GetMetricsTimelineRequest{
		WindowSeconds:         protoInt32(window),
		SampleIntervalSeconds: protoInt32(interval),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get metrics timeline", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetTimeline() == nil {
		return fmt.Errorf("server returned no metrics timeline")
	}

	response := resp.Msg.GetTimeline()
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
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func (h *handlers) processTimeline(ctx cliapp.RunContext) error {
	windowSeconds, err := windowSecondsFlag(ctx, "window")
	if err != nil {
		return err
	}
	top, err := positiveIntFlag(ctx, "top")
	if err != nil {
		return err
	}
	owner := strings.TrimSpace(ctx.Flag("owner"))

	resp, err := h.client.GetProcessTimeline(context.Background(), connect.NewRequest(&metricspb.GetProcessTimelineRequest{
		WindowSeconds: protoInt32(windowSeconds),
		Owner:         owner,
		Top:           protoInt32(top),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get process timeline", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetTimeline() == nil {
		return fmt.Errorf("server returned no process timeline")
	}

	response := resp.Msg.GetTimeline()
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Window: %ds", response.GetWindowSeconds()),
			fmt.Sprintf("Owner filter: %s", ownerFilterLabel(response.GetOwner())),
			fmt.Sprintf("Ranked consumers: %d", response.GetCount()),
		},
		ResultsHeading: "Top Consumers by Scenario",
		Results:        processTimelineRows(response.GetEntries()),
		RetrievalHints: []string{
			"system-monitor metrics process-timeline --window 1h --json",
			"system-monitor metrics process-timeline --owner security-health",
		},
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, report)
}

func ownerFilterLabel(owner string) string {
	if strings.TrimSpace(owner) == "" {
		return "all owners"
	}
	return owner
}

func processTimelineRows(entries []*metricspb.ProcessTimelineEntry) []string {
	if len(entries) == 0 {
		return []string{"No process samples in the requested window yet."}
	}
	rows := make([]string, 0, len(entries))
	for _, e := range entries {
		if e == nil {
			continue
		}
		pid := "aggregated"
		if !e.GetAggregated() && e.GetPid() > 0 {
			pid = fmt.Sprintf("pid=%d", e.GetPid())
		}
		rows = append(rows, fmt.Sprintf("%s %s cpu=%s rss=%.1fMB samples=%d (%s)",
			e.GetOwner(), e.GetComm(), support.FormatPercent(e.GetCpuPct()), float64(e.GetRssKb())/1024, e.GetSampleCount(), pid))
	}
	return rows
}

func protoInt32(value int) *int32 {
	converted := int32(value)
	return &converted
}

func positiveIntFlag(ctx cliapp.RunContext, name string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(ctx.Flag(name)))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("--%s must be a positive integer", name)
	}
	return value, nil
}

func windowSecondsFlag(ctx cliapp.RunContext, name string) (int, error) {
	raw := strings.TrimSpace(ctx.Flag(name))
	if raw == "" {
		raw = "5m"
	}
	if seconds, err := strconv.Atoi(raw); err == nil {
		if seconds <= 0 {
			return 0, fmt.Errorf("--%s must be greater than 0", name)
		}
		return seconds, nil
	}
	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("--%s must be a positive duration or seconds value", name)
	}
	return int(duration.Seconds()), nil
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
		rows = append(rows, fmt.Sprintf("%s cpu=%s memory=%s tcp=%d gpu=%s swapTraffic=%s majorFaults=%s fragmentation=%s", support.FormatTimestamp(sample.GetTimestamp()), support.FormatPercent(sample.GetCpuUsage()), support.FormatPercent(sample.GetMemoryUsage()), sample.GetTcpConnections(), support.FormatMaybePercent(sample.GpuUsage), metricStateSummary(sample.GetSwapTraffic()), metricStateSummary(sample.GetMajorFaults()), metricStateSummary(sample.GetFragmentationIndex())))
	}
	return rows
}

func metricStateSummary(value *metricspb.MetricValue) string {
	if value == nil {
		return "unavailable"
	}
	switch state := value.GetState().(type) {
	case *metricspb.MetricValue_Measured:
		return fmt.Sprintf("measured %.2f", state.Measured)
	case *metricspb.MetricValue_UnsupportedReason:
		return "unsupported: " + state.UnsupportedReason
	case *metricspb.MetricValue_NotYetSampledReason:
		return "not_yet_sampled: " + state.NotYetSampledReason
	case *metricspb.MetricValue_StaleReason:
		return "stale: " + state.StaleReason
	case *metricspb.MetricValue_FailedError:
		return "failed: " + state.FailedError
	default:
		return "unavailable"
	}
}
