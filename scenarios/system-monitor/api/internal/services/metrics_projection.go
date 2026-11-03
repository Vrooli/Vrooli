package services

import (
	"runtime"
	"strings"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

func populateCPUDetails(detailed *models.DetailedMetrics, cpuData, processData *collectors.MetricData, topCPUProcs, topCPUSecondsProcs []map[string]interface{}) {
	if cpuData == nil {
		return
	}

	detailed.CPUDetails = models.CPUMetrics{
		Usage:                    getFloat64Value(cpuData.Values, "usage_percent"),
		LoadAverage:              getFloat64Slice(cpuData.Values, "load_average"),
		UsageState:               metricState(cpuData, "usage_percent", "CPU has not been measured"),
		ContextSwitchesPerSecond: metricState(cpuData, "context_switches_per_second", "context-switch counter rate has not been sampled"),
		InterruptsPerSecond:      metricState(cpuData, "interrupts_per_second", "interrupt counter rate has not been sampled"),
		NormalizedLoad1:          metricState(cpuData, "normalized_load_1", "normalized load is unavailable"),
		NormalizedLoad5:          metricState(cpuData, "normalized_load_5", "normalized load is unavailable"),
		RunQueueDepth:            metricState(cpuData, "run_queue_depth", "run-queue depth is unavailable"),
		ModeBreakdown:            metricStateMap(cpuData, "mode_breakdown", "CPU mode accounting is unavailable"),
		PerCoreUtilization:       metricStateMap(cpuData, "per_core_utilization", "per-core utilization is unavailable"),
		CoreImbalanceIndex:       metricState(cpuData, "core_imbalance_index", "core imbalance is unavailable"),
		QuotaThrottling:          metricState(cpuData, "quota_throttling", "CPU quota throttling is unavailable"),
		FrequencyDerateRatio:     metricState(cpuData, "frequency_derate_ratio", "CPU frequency backend is unavailable"),
		ThermalThrottleEvidence:  metricState(cpuData, "thermal_throttle_evidence", "thermal attribution is unavailable"),
		ThermalTripPointCelsius:  metricState(cpuData, "thermal_trip_point_celsius", "thermal trip point is unavailable"),
		ForkRate:                 metricState(processData, "fork_rate", "fork-rate counter is unavailable"),
	}
	detailed.CPUDetails.LoadAverageState = cpuLoadState(cpuData)

	for _, proc := range topCPUProcs {
		detailed.CPUDetails.TopProcesses = append(detailed.CPUDetails.TopProcesses, convertToProcessInfo(proc))
	}
	for _, proc := range topCPUSecondsProcs {
		detailed.CPUDetails.TopCPUSecondsProcesses = append(detailed.CPUDetails.TopCPUSecondsProcesses, convertToProcessInfo(proc))
	}
}

func metricStateMap(data *collectors.MetricData, key, reason string) map[string]models.MetricState {
	result := map[string]models.MetricState{}
	if data == nil {
		return result
	}
	if raw, ok := data.Values[key].(map[string]float64); ok {
		for name, value := range raw {
			result[name] = models.MetricState{Status: "measured", Value: value, Units: "percent", Provenance: data.Tags["source"], ObservedAt: data.Timestamp}
		}
	}
	if len(result) == 0 {
		result["_state"] = metricState(data, key, reason)
	}
	return result
}

func cpuLoadState(data *collectors.MetricData) models.MetricState {
	state := metricState(data, "load_average", "load average is unavailable")
	if data == nil {
		return state
	}
	if status, ok := data.Values["load_average_status"].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := data.Values["load_average_reason"].(string); ok {
		state.Reason = reason
	}
	if provenance, ok := data.Values["load_average_provenance"].(string); ok && provenance != "" {
		state.Provenance = provenance
	}
	if loads, ok := data.Values["load_average"].([]float64); ok && len(loads) > 0 && state.Status == "measured" {
		state.Value = loads[0]
		state.Units = "load"
		state.Reason = ""
	}
	if state.Status != "measured" {
		state.Value = 0
	}
	return state
}

// populateMemoryDetails fills the memory section of detailed (including disk
// usage, which is stored under MemoryMetrics in proto).
func populateMemoryDetails(detailed *models.DetailedMetrics, memData, diskData, pressureData *collectors.MetricData, topMemProcs, topPagingProcs []map[string]interface{}) {
	if memData == nil {
		return
	}

	detailed.MemoryDetails = models.MemoryMetrics{
		Usage:        getFloat64Value(memData.Values, "usage_percent"),
		KernelMemory: repository.KernelMemoryFromValues(memData.Values, "", memData.Timestamp),
	}

	if swapInfo, ok := memData.Values["swap"].(map[string]interface{}); ok {
		detailed.MemoryDetails.SwapUsage = models.SwapInfo{
			Used:    getInt64Value(swapInfo, "used"),
			Total:   getInt64Value(swapInfo, "total"),
			Percent: getFloat64Value(swapInfo, "percent"),
		}
	}

	for _, proc := range topMemProcs {
		detailed.MemoryDetails.TopProcesses = append(detailed.MemoryDetails.TopProcesses, convertToProcessInfo(proc))
	}
	for _, proc := range topPagingProcs {
		detailed.MemoryDetails.TopPagingProcesses = append(detailed.MemoryDetails.TopPagingProcesses, convertToProcessInfo(proc))
	}
	populatePressureDetails(&detailed.MemoryDetails, pressureData)

	if diskData == nil {
		return
	}
	if diskUsage, ok := diskData.Values["usage"].(map[string]interface{}); ok {
		detailed.MemoryDetails.DiskUsage = models.DiskInfo{
			Used:    getInt64Value(diskUsage, "used"),
			Total:   getInt64Value(diskUsage, "total"),
			Percent: getFloat64Value(diskUsage, "percent"),
		}
	}
}

func populatePressureDetails(memory *models.MemoryMetrics, data *collectors.MetricData) {
	if data == nil {
		return
	}
	// Cross-platform aliases published by the paging seam: Linux maps
	// pswpin/pswpout/pgmajfault/pgfault, macOS maps host_statistics64
	// swapins/swapouts/faults, and an alias a platform cannot measure
	// carries an explicit unsupported status.
	memory.Paging = models.PagingMetrics{
		SwapInPerSecond:           pressureState(data, "swap_in_per_second", "", ""),
		SwapOutPerSecond:          pressureState(data, "swap_out_per_second", "", ""),
		SwapTrafficPagesPerSecond: pressureState(data, "swap_traffic_pages_per_second", "swap_traffic_rate_status", "swap_traffic_rate_reason"),
		MajorFaultsPerSecond:      pressureState(data, "major_faults_per_second", "", ""),
		PageFaultsPerSecond:       pressureState(data, "page_faults_per_second", "", ""),
	}
	fragmentation := models.FragmentationMetrics{
		MaxFreeOrder:           pressureState(data, "fragmentation_max_free_order", "fragmentation_status", "fragmentation_reason"),
		LowOrderShare:          pressureState(data, "fragmentation_low_order_share", "fragmentation_status", "fragmentation_reason"),
		CompactionFailureRatio: pressureState(data, "compaction_failure_ratio", "compaction_failure_ratio_status", "compaction_failure_ratio_reason"),
		CompactionRates:        make(map[string]models.MetricState),
	}
	if histogram, ok := data.Values["buddyinfo"].(map[string]string); ok {
		fragmentation.Buddyinfo = histogram
	}
	for key := range data.Values {
		if strings.HasSuffix(key, "_per_second") && strings.HasPrefix(key, "compact_") {
			fragmentation.CompactionRates[strings.TrimSuffix(key, "_per_second")] = pressureState(data, key, strings.TrimSuffix(key, "_per_second")+"_rate_status", strings.TrimSuffix(key, "_per_second")+"_rate_reason")
		}
	}
	memory.Fragmentation = fragmentation
}

func pressureState(data *collectors.MetricData, valueKey, statusKey, reasonKey string) models.MetricState {
	units := "per second"
	if strings.Contains(valueKey, "psi_") {
		units = "percent"
	}
	state := models.MetricState{Status: "not_yet_sampled", Reason: "rate has not been sampled", Provenance: "system-monitor/pressure", Units: units, ObservedAt: data.Timestamp}
	if _, hasSignalStatus := data.Values[valueKey+"_status"]; hasSignalStatus {
		statusKey = valueKey + "_status"
		reasonKey = valueKey + "_reason"
	}
	if status, ok := data.Values[statusKey].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := data.Values[reasonKey].(string); ok && reason != "" {
		state.Reason = reason
	}
	if value, ok := data.Values[valueKey].(float64); ok {
		state.Status, state.Value, state.Reason = "measured", value, ""
	}
	return state
}

// populateNetworkDetails fills the network section of detailed from the network collector data.
func populateNetworkDetails(detailed *models.DetailedMetrics, netData *collectors.MetricData) {
	if netData == nil {
		detailed.NetworkDetails.Verdict = models.NetworkVerdict{State: "unavailable", Summary: "Network evidence is unavailable", Reasons: []string{"network collector returned no data"}}
		return
	}

	detailed.NetworkDetails.Capabilities = models.NetworkCapabilities{
		TCPStates:         models.MetricState{Status: "measured", Value: 1, Reason: "native TCP state inventory", Provenance: "procfs"},
		InterfaceCounters: models.MetricState{Status: "measured", Value: 1, Reason: "native interface counters", Provenance: "procfs"},
		TransportCounters: models.MetricState{Status: "unsupported", Reason: "host does not expose cumulative transport open/close counters", Provenance: "native collector"},
		Ownership:         models.MetricState{Status: "not_yet_sampled", Reason: "bounded attribution runs only for an on-demand diagnostic", Provenance: "diagnostic RPC"},
		Endpoints:         models.MetricState{Status: "not_yet_sampled", Reason: "bounded endpoint inventory runs only for an on-demand diagnostic", Provenance: "diagnostic RPC"},
	}

	if states, ok := netData.Values["tcp_states"].(map[string]int); ok {
		detailed.NetworkDetails.TCPStates = models.TCPConnectionStates{
			Established: states["established"],
			TimeWait:    states["time_wait"],
			CloseWait:   states["close_wait"],
			Listen:      states["listen"],
			Total:       states["total"],
		}
	}
	if _, ok := netData.Values["tcp_states"].(map[string]int); !ok {
		detailed.NetworkDetails.Capabilities.TCPStates = models.MetricState{Status: "unavailable", Reason: "native TCP state inventory failed", Provenance: "procfs"}
	}

	if portUsage, ok := netData.Values["port_usage"].(map[string]int); ok {
		detailed.NetworkDetails.PortUsage = models.PortUsageInfo{
			Used:  portUsage["used"],
			Total: portUsage["total"],
		}
	}

	if bw, ok := netData.Values["bandwidth"].(map[string]interface{}); ok {
		detailed.NetworkDetails.NetworkStats = models.NetworkStatistics{
			BandwidthInMbps:  getFloat64Value(bw, "in_mbps"),
			BandwidthOutMbps: getFloat64Value(bw, "out_mbps"),
		}
	}

	// Attribution is present only when the collector judged the connection count
	// alarming enough to walk /proc, so its absence is expected and not an error.
	if attribution, ok := netData.Values["socket_owners"].(collectors.SocketAttribution); ok {
		owners := make([]models.SocketOwnerInfo, 0, len(attribution.Owners))
		for _, owner := range attribution.Owners {
			owners = append(owners, models.SocketOwnerInfo{PID: owner.PID, Name: owner.Comm, Connections: owner.Count})
		}
		detailed.NetworkDetails.SocketOwners = &models.SocketOwnership{
			Owners:     owners,
			Attributed: attribution.Attributed,
			Total:      attribution.Total,
			Supported:  attribution.Supported,
			Reason:     attribution.Reason,
		}
		detailed.NetworkDetails.Ownership = &models.NetworkOwnership{
			TotalConnections: attribution.Total, AttributedConnections: attribution.Attributed,
			AttributionCoveragePercent: coveragePercent(attribution.Attributed, attribution.Total),
			Truncated:                  attribution.Truncated, Reason: attribution.Reason, Provenance: "bounded /proc diagnostic",
		}
		detailed.NetworkDetails.Capabilities.Ownership = models.MetricState{Status: "measured", Value: 1, Reason: "bounded attribution evidence available", Provenance: "diagnostic RPC"}
	}

	detailed.NetworkDetails.EstablishedRate = networkRateState(netData.Values, "established_rate_per_second")
	detailed.NetworkDetails.TimeWaitRate = networkRateState(netData.Values, "time_wait_rate_per_second")
	detailed.NetworkDetails.CloseWaitRate = networkRateState(netData.Values, "close_wait_rate_per_second")
	detailed.NetworkDetails.ConnectionsOpenedRate = models.MetricState{Status: "unsupported", Reason: "no native cumulative open counter", Units: "connections/second"}
	detailed.NetworkDetails.ConnectionsClosedRate = models.MetricState{Status: "unsupported", Reason: "no native cumulative close counter", Units: "connections/second"}
	if rows, ok := netData.Values["interfaces"].([]map[string]interface{}); ok {
		for _, row := range rows {
			detailed.NetworkDetails.Interfaces = append(detailed.NetworkDetails.Interfaces, models.NetworkInterface{
				Name: getStringValue(row, "name"), Up: getBoolValue(row, "up"),
				ReceivedBytes: networkCounterState(row, "received_bytes", "bytes"), TransmittedBytes: networkCounterState(row, "transmitted_bytes", "bytes"),
				ReceivedPackets: networkCounterState(row, "received_packets", "packets"), TransmittedPackets: networkCounterState(row, "transmitted_packets", "packets"),
				ReceiveErrors: networkCounterState(row, "receive_errors", "errors"), TransmitErrors: networkCounterState(row, "transmit_errors", "errors"),
				ReceiveDrops: networkCounterState(row, "receive_drops", "drops"), TransmitDrops: networkCounterState(row, "transmit_drops", "drops"),
				ReceiveBytesPerSecond: networkRateState(row, "received_bytes_per_second"), TransmitBytesPerSecond: networkRateState(row, "transmitted_bytes_per_second"),
			})
		}
	} else {
		detailed.NetworkDetails.Capabilities.InterfaceCounters = models.MetricState{Status: "unavailable", Reason: "native interface counters failed", Provenance: "procfs"}
	}

	detailed.NetworkDetails.Verdict = models.NetworkVerdict{State: "warning", Summary: "Network evidence is partial", Reasons: []string{"transport open/close counters are unsupported on this host"}}
	if detailed.NetworkDetails.Capabilities.TCPStates.Status == "unavailable" && detailed.NetworkDetails.Capabilities.InterfaceCounters.Status == "unavailable" {
		detailed.NetworkDetails.Verdict = models.NetworkVerdict{State: "unavailable", Summary: "Network evidence is unavailable", Reasons: []string{"native TCP and interface collectors failed"}}
	}
}

func coveragePercent(attributed, total int) float64 {
	if total <= 0 {
		return 0
	}
	coverage := float64(attributed) * 100 / float64(total)
	if coverage > 100 {
		return 100
	}
	return coverage
}

func networkRateState(values map[string]interface{}, key string) models.MetricState {
	state := models.MetricState{Status: "not_yet_sampled", Reason: "requires two consecutive samples", Units: "per second"}
	if value, ok := values[key].(float64); ok {
		state.Status, state.Value = "measured", value
	} else if value, ok := values[key].(int64); ok {
		state.Status, state.Value = "measured", float64(value)
	}
	return state
}

func networkCounterState(values map[string]interface{}, key, units string) models.MetricState {
	state := networkRateState(values, key)
	state.Units = units
	if state.Status == "not_yet_sampled" {
		state.Status = "measured"
	}
	return state
}

// populateSystemDetails fills the system section of detailed (file descriptors
// and inotify watchers) from the disk collector data.
func populateSystemDetails(detailed *models.DetailedMetrics, diskData *collectors.MetricData) {
	detailed.SystemDetails.APIProcessGoroutines = runtime.NumGoroutine()
	if diskData == nil {
		return
	}

	if fdInfo, ok := diskData.Values["file_descriptors"].(map[string]interface{}); ok {
		detailed.SystemDetails.FileDescriptors = models.FileDescriptorInfo{
			Used:    getIntValue(fdInfo, "used"),
			Max:     getIntValue(fdInfo, "max"),
			Percent: getFloat64Value(fdInfo, "percent"),
		}
	}

	if inotifyInfo, ok := diskData.Values["inotify_watchers"].(map[string]interface{}); ok {
		info := models.InotifyWatcherInfo{
			Supported:        getBoolValue(inotifyInfo, "supported"),
			WatchesUsed:      getIntValue(inotifyInfo, "watches_used"),
			WatchesMax:       getIntValue(inotifyInfo, "watches_max"),
			WatchesPercent:   getFloat64Value(inotifyInfo, "watches_percent"),
			InstancesUsed:    getIntValue(inotifyInfo, "instances_used"),
			InstancesMax:     getIntValue(inotifyInfo, "instances_max"),
			InstancesPercent: getFloat64Value(inotifyInfo, "instances_percent"),
		}
		detailed.SystemDetails.InotifyWatchers = &info
	}
}

// populateGPUDetails fills the GPU section of detailed from the gpu collector data.
func populateGPUDetails(detailed *models.DetailedMetrics, gpuData *collectors.MetricData) {
	if gpuData == nil {
		return
	}

	metrics := models.GPUMetrics{}
	if summary, ok := gpuData.Values["summary"].(models.GPUSummary); ok {
		metrics.Summary = summary
	}
	if devices, ok := gpuData.Values["devices"].([]models.GPUDeviceMetrics); ok {
		metrics.Devices = devices
	}
	if driver, ok := gpuData.Values["driver_version"].(string); ok {
		metrics.Driver = driver
	}
	if model, ok := gpuData.Values["primary_model"].(string); ok {
		metrics.Model = model
	}
	if warnings, ok := gpuData.Values["warnings"].([]string); ok {
		metrics.Errors = warnings
	}
	detailed.GPUDetails = &metrics
}
