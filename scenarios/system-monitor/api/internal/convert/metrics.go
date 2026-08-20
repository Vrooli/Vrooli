package convert

import (
	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func metricValue(s models.MetricState, fallback float64) *metricspb.MetricValue {
	if s.Status == "" {
		s.Status = "measured"
		s.Value = fallback
	}
	value := &metricspb.MetricValue{
		Provenance: s.Provenance,
	}
	if !s.ObservedAt.IsZero() {
		value.ObservedAt = timestamppb.New(s.ObservedAt)
	}
	switch s.Status {
	case "measured":
		value.State = &metricspb.MetricValue_Measured{Measured: s.Value}
	case "unsupported":
		if s.Reason == "" {
			s.Reason = "metric is not supported on this host"
		}
		value.State = &metricspb.MetricValue_UnsupportedReason{UnsupportedReason: s.Reason}
	default:
		if s.Reason == "" {
			s.Reason = "metric collection failed"
		}
		value.State = &metricspb.MetricValue_FailedError{FailedError: s.Reason}
	}
	return value
}

func MetricsResponseToProto(m *models.MetricsResponse) *metricspb.MetricsResponse {
	if m == nil {
		return nil
	}
	gpuState := m.GPUState
	if gpuState.Status == "" && m.GPUUsage == nil {
		gpuState = models.MetricState{Status: "unsupported", Reason: "GPU collector unavailable"}
	}
	pb := &metricspb.MetricsResponse{
		CpuUsage:       m.CPUUsage,
		MemoryUsage:    m.MemoryUsage,
		TcpConnections: int32(m.TCPConnections),
		GpuUsage:       m.GPUUsage,
		Timestamp:      timestamppb.New(m.Timestamp),
		Cpu:            metricValue(m.CPUState, m.CPUUsage),
		Memory:         metricValue(m.MemoryState, m.MemoryUsage),
		Connections:    metricValue(m.ConnectionsState, float64(m.TCPConnections)),
		Gpu:            metricValue(gpuState, dereferenceFloat(m.GPUUsage)),
		Disk:           metricValue(m.DiskState, m.DiskUsage),
	}
	pb.Cpu = metricValue(m.CPUState, m.CPUUsage)
	pb.Memory = metricValue(m.MemoryState, m.MemoryUsage)
	pb.Connections = metricValue(m.ConnectionsState, float64(m.TCPConnections))
	pb.Gpu = metricValue(gpuState, dereferenceFloat(m.GPUUsage))
	pb.Disk = metricValue(m.DiskState, m.DiskUsage)
	return pb
}

func dereferenceFloat(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func MetricsTimelineResponseToProto(m *models.MetricsTimelineResponse) *metricspb.MetricsTimelineResponse {
	if m == nil {
		return nil
	}
	samples := make([]*metricspb.MetricTimelineSample, len(m.Samples))
	for i, s := range m.Samples {
		samples[i] = &metricspb.MetricTimelineSample{
			Timestamp:      timestamppb.New(s.Timestamp),
			CpuUsage:       s.CPUUsage,
			MemoryUsage:    s.MemoryUsage,
			TcpConnections: int32(s.TCPConnections),
			GpuUsage:       s.GPUUsage,
			Cpu:            metricValue(s.CPUState, s.CPUUsage),
			Memory:         metricValue(s.MemoryState, s.MemoryUsage),
			Connections:    metricValue(s.ConnectionsState, float64(s.TCPConnections)),
			Gpu:            metricValue(s.GPUState, dereferenceFloat(s.GPUUsage)),
		}
	}
	return &metricspb.MetricsTimelineResponse{
		WindowSeconds:         int32(m.WindowSeconds),
		SampleIntervalSeconds: int32(m.SampleIntervalSeconds),
		Samples:               samples,
	}
}

func DetailedMetricsToProto(m *models.DetailedMetrics) *metricspb.DetailedMetrics {
	if m == nil {
		return nil
	}
	pb := &metricspb.DetailedMetrics{
		CpuDetails:     cpuMetricsToProto(m.CPUDetails),
		MemoryDetails:  memoryMetricsToProto(m.MemoryDetails),
		NetworkDetails: networkMetricsToProto(m.NetworkDetails),
		SystemDetails:  systemHealthToProto(m.SystemDetails),
		Timestamp:      timestamppb.New(m.Timestamp),
	}
	if m.GPUDetails != nil {
		pb.GpuDetails = gpuMetricsToProto(m.GPUDetails)
	}
	return pb
}

func ProcessMonitorDataToProto(m *models.ProcessMonitorData) *metricspb.ProcessMonitorData {
	if m == nil {
		return nil
	}
	return &metricspb.ProcessMonitorData{
		ProcessHealth:  processHealthInfoToProto(m.ProcessHealth),
		ResourceMatrix: processInfoSliceToProto(m.ResourceMatrix),
		Timestamp:      timestamppb.New(m.Timestamp),
	}
}

func InfrastructureMonitorDataToProto(m *models.InfrastructureMonitorData) *metricspb.InfrastructureMonitorData {
	if m == nil {
		return nil
	}
	return &metricspb.InfrastructureMonitorData{
		DatabasePools:   connectionPoolSliceToProto(m.DatabasePools),
		HttpClientPools: connectionPoolSliceToProto(m.HTTPClientPools),
		MessageQueues:   messageQueueInfoToProto(m.MessageQueues),
		StorageIo:       storageIOInfoToProto(m.StorageIO),
		Timestamp:       timestamppb.New(m.Timestamp),
	}
}

func DiskDetailResponseToProto(m *models.DiskDetailResponse) *metricspb.DiskDetailResponse {
	if m == nil {
		return nil
	}
	return &metricspb.DiskDetailResponse{
		Partitions:     diskPartitionInfoSliceToProto(m.Partitions),
		ActiveMount:    m.ActiveMount,
		Depth:          int32(m.Depth),
		TopDirectories: diskUsageEntrySliceToProto(m.TopDirectories),
		LargestFiles:   diskUsageEntrySliceToProto(m.LargestFiles),
		Notes:          m.Notes,
		Timestamp:      timestamppb.New(m.Timestamp),
	}
}

func ProcessTimelineResponseToProto(windowSeconds int, owner string, top int, entries []repository.ProcessTimelineEntry) *metricspb.ProcessTimelineResponse {
	pbEntries := make([]*metricspb.ProcessTimelineEntry, 0, len(entries))
	for _, e := range entries {
		row := &metricspb.ProcessTimelineEntry{
			Owner:       e.Owner,
			Comm:        e.Comm,
			Pid:         int32(e.PID),
			Aggregated:  e.Aggregated,
			CpuPct:      e.CPUPct,
			RssKb:       e.RSSKB,
			SampleCount: e.SampleCount,
		}
		if !e.FirstSeen.IsZero() {
			row.FirstSeen = timestamppb.New(e.FirstSeen)
		}
		if !e.LastSeen.IsZero() {
			row.LastSeen = timestamppb.New(e.LastSeen)
		}
		pbEntries = append(pbEntries, row)
	}
	return &metricspb.ProcessTimelineResponse{
		WindowSeconds: int32(windowSeconds),
		Owner:         owner,
		Top:           int32(top),
		Count:         int32(len(pbEntries)),
		Entries:       pbEntries,
	}
}

func cpuMetricsToProto(m models.CPUMetrics) *metricspb.CPUMetrics {
	return &metricspb.CPUMetrics{
		Usage:           m.Usage,
		TopProcesses:    processInfoSliceToProto(m.TopProcesses),
		LoadAverage:     m.LoadAverage,
		ContextSwitches: m.ContextSwitches,
		TotalGoroutines: int32(m.Goroutines),
	}
}

func memoryMetricsToProto(m models.MemoryMetrics) *metricspb.MemoryMetrics {
	pb := &metricspb.MemoryMetrics{
		Usage:          m.Usage,
		TopProcesses:   processInfoSliceToProto(m.TopProcesses),
		GrowthPatterns: memoryGrowthSliceToProto(m.GrowthPatterns),
		SwapUsage:      swapInfoToProto(m.SwapUsage),
		DiskUsage:      diskInfoToProto(m.DiskUsage),
	}
	return pb
}

func networkMetricsToProto(m models.NetworkMetrics) *metricspb.NetworkMetrics {
	return &metricspb.NetworkMetrics{
		TcpStates:       tcpConnectionStatesToProto(m.TCPStates),
		PortUsage:       portUsageInfoToProto(m.PortUsage),
		NetworkStats:    networkStatisticsToProto(m.NetworkStats),
		ConnectionPools: connectionPoolSliceToProto(m.ConnectionPools),
	}
}

func systemHealthToProto(m models.SystemHealth) *metricspb.SystemHealth {
	pb := &metricspb.SystemHealth{
		FileDescriptors:     fileDescriptorInfoToProto(m.FileDescriptors),
		ServiceDependencies: serviceHealthSliceToProto(m.ServiceDependencies),
		Certificates:        certificateInfoSliceToProto(m.Certificates),
	}
	if m.InotifyWatchers != nil {
		pb.InotifyWatchers = inotifyWatcherInfoToProto(m.InotifyWatchers)
	}
	return pb
}

func gpuMetricsToProto(m *models.GPUMetrics) *metricspb.GPUMetrics {
	if m == nil {
		return nil
	}
	return &metricspb.GPUMetrics{
		Summary:       gpuSummaryToProto(m.Summary),
		Devices:       gpuDeviceMetricsSliceToProto(m.Devices),
		Errors:        m.Errors,
		DriverVersion: m.Driver,
		PrimaryModel:  m.Model,
	}
}

func gpuSummaryToProto(m models.GPUSummary) *metricspb.GPUSummary {
	return &metricspb.GPUSummary{
		TotalUtilizationPercent:   m.TotalUtilizationPercent,
		AverageUtilizationPercent: m.AverageUtilizationPercent,
		TotalMemoryMb:             m.TotalMemoryMB,
		UsedMemoryMb:              m.UsedMemoryMB,
		AverageTemperatureC:       m.AverageTemperatureC,
		DeviceCount:               int32(m.DeviceCount),
	}
}

func gpuDeviceMetricsSliceToProto(ms []models.GPUDeviceMetrics) []*metricspb.GPUDeviceMetrics {
	result := make([]*metricspb.GPUDeviceMetrics, len(ms))
	for i, m := range ms {
		pb := &metricspb.GPUDeviceMetrics{
			Index:                    int32(m.Index),
			Uuid:                     m.UUID,
			Name:                     m.Name,
			UtilizationPercent:       m.Utilization,
			MemoryUtilizationPercent: m.MemoryUtilization,
			MemoryUsedMb:             m.MemoryUsedMB,
			MemoryTotalMb:            m.MemoryTotalMB,
			Processes:                gpuProcessInfoSliceToProto(m.Processes),
		}
		if m.TemperatureC != nil {
			pb.TemperatureC = m.TemperatureC
		}
		if m.FanSpeedPercent != nil {
			pb.FanSpeedPercent = m.FanSpeedPercent
		}
		if m.PowerDrawW != nil {
			pb.PowerDrawW = m.PowerDrawW
		}
		if m.PowerLimitW != nil {
			pb.PowerLimitW = m.PowerLimitW
		}
		if m.SMClockMHz != nil {
			pb.SmClockMhz = m.SMClockMHz
		}
		if m.MemoryClockMHz != nil {
			pb.MemoryClockMhz = m.MemoryClockMHz
		}
		result[i] = pb
	}
	return result
}

func gpuProcessInfoSliceToProto(ms []models.GPUProcessInfo) []*metricspb.GPUProcessInfo {
	result := make([]*metricspb.GPUProcessInfo, len(ms))
	for i, m := range ms {
		pb := &metricspb.GPUProcessInfo{
			Pid:           int32(m.PID),
			ProcessName:   m.ProcessName,
			MemoryUsedMb:  m.MemoryUsedMB,
			GpuInstanceId: m.GPUInstanceID,
		}
		if m.SMUtilization != nil {
			pb.SmUtilizationPercent = m.SMUtilization
		}
		result[i] = pb
	}
	return result
}

func diskPartitionInfoSliceToProto(ms []models.DiskPartitionInfo) []*metricspb.DiskPartitionInfo {
	result := make([]*metricspb.DiskPartitionInfo, len(ms))
	for i, m := range ms {
		result[i] = &metricspb.DiskPartitionInfo{
			Device:         m.Device,
			MountPoint:     m.MountPoint,
			SizeBytes:      m.SizeBytes,
			SizeHuman:      m.SizeHuman,
			UsedBytes:      m.UsedBytes,
			UsedHuman:      m.UsedHuman,
			AvailableBytes: m.AvailableBytes,
			AvailableHuman: m.AvailableHuman,
			UsePercent:     m.UsePercent,
		}
	}
	return result
}

func diskUsageEntrySliceToProto(ms []models.DiskUsageEntry) []*metricspb.DiskUsageEntry {
	result := make([]*metricspb.DiskUsageEntry, len(ms))
	for i, m := range ms {
		result[i] = &metricspb.DiskUsageEntry{
			Path:      m.Path,
			SizeBytes: m.SizeBytes,
			SizeHuman: m.SizeHuman,
			Category:  m.Category,
		}
	}
	return result
}

func processInfoSliceToProto(ms []models.ProcessInfo) []*metricspb.ProcessInfo {
	result := make([]*metricspb.ProcessInfo, len(ms))
	for i, m := range ms {
		result[i] = &metricspb.ProcessInfo{
			Pid:             int32(m.PID),
			Name:            m.Name,
			CpuPercent:      m.CPUPercent,
			MemoryMb:        m.MemoryMB,
			Connections:     int32(m.Connections),
			Threads:         int32(m.Threads),
			FileDescriptors: int32(m.FDs),
			Status:          m.Status,
			Goroutines:      int32(m.Goroutines),
		}
	}
	return result
}

func tcpConnectionStatesToProto(m models.TCPConnectionStates) *metricspb.TCPConnectionStates {
	return &metricspb.TCPConnectionStates{
		Established: int32(m.Established),
		TimeWait:    int32(m.TimeWait),
		CloseWait:   int32(m.CloseWait),
		FinWait1:    int32(m.FinWait1),
		FinWait2:    int32(m.FinWait2),
		SynSent:     int32(m.SynSent),
		SynRecv:     int32(m.SynRecv),
		Closing:     int32(m.Closing),
		LastAck:     int32(m.LastAck),
		Listen:      int32(m.Listen),
		Total:       int32(m.Total),
	}
}

func connectionPoolSliceToProto(ms []models.ConnectionPool) []*metricspb.ConnectionPool {
	result := make([]*metricspb.ConnectionPool, len(ms))
	for i, m := range ms {
		result[i] = &metricspb.ConnectionPool{
			Name:     m.Name,
			Active:   int32(m.Active),
			Idle:     int32(m.Idle),
			MaxSize:  int32(m.MaxSize),
			Waiting:  int32(m.Waiting),
			Healthy:  m.Healthy,
			LeakRisk: m.LeakRisk,
		}
	}
	return result
}

func networkStatisticsToProto(m models.NetworkStatistics) *metricspb.NetworkStatistics {
	return &metricspb.NetworkStatistics{
		BandwidthInMbps:  m.BandwidthInMbps,
		BandwidthOutMbps: m.BandwidthOutMbps,
		PacketLoss:       m.PacketLoss,
		DnsSuccessRate:   m.DNSSuccessRate,
		DnsLatencyMs:     m.DNSLatencyMs,
	}
}

func portUsageInfoToProto(m models.PortUsageInfo) *metricspb.PortUsageInfo {
	return &metricspb.PortUsageInfo{
		Used:  int32(m.Used),
		Total: int32(m.Total),
	}
}

func fileDescriptorInfoToProto(m models.FileDescriptorInfo) *metricspb.FileDescriptorInfo {
	return &metricspb.FileDescriptorInfo{
		Used:    int32(m.Used),
		Max:     int32(m.Max),
		Percent: m.Percent,
	}
}

func serviceHealthSliceToProto(ms []models.ServiceHealth) []*metricspb.ServiceHealth {
	result := make([]*metricspb.ServiceHealth, len(ms))
	for i, m := range ms {
		result[i] = &metricspb.ServiceHealth{
			Name:      m.Name,
			Status:    m.Status,
			LatencyMs: m.LatencyMs,
			LastCheck: timestamppb.New(m.LastCheck),
			Endpoint:  m.Endpoint,
		}
	}
	return result
}

func certificateInfoSliceToProto(ms []models.CertificateInfo) []*metricspb.CertificateInfo {
	result := make([]*metricspb.CertificateInfo, len(ms))
	for i, m := range ms {
		result[i] = &metricspb.CertificateInfo{
			Domain:       m.Domain,
			DaysToExpiry: int32(m.DaysToExpiry),
			Status:       m.Status,
		}
	}
	return result
}

func inotifyWatcherInfoToProto(m *models.InotifyWatcherInfo) *metricspb.InotifyWatcherInfo {
	if m == nil {
		return nil
	}
	return &metricspb.InotifyWatcherInfo{
		Supported:        m.Supported,
		WatchesUsed:      int32(m.WatchesUsed),
		WatchesMax:       int32(m.WatchesMax),
		WatchesPercent:   m.WatchesPercent,
		InstancesUsed:    int32(m.InstancesUsed),
		InstancesMax:     int32(m.InstancesMax),
		InstancesPercent: m.InstancesPercent,
	}
}

func memoryGrowthSliceToProto(ms []models.MemoryGrowth) []*metricspb.MemoryGrowth {
	result := make([]*metricspb.MemoryGrowth, len(ms))
	for i, m := range ms {
		result[i] = &metricspb.MemoryGrowth{
			Process:         m.Process,
			GrowthMbPerHour: m.GrowthMBPerHour,
			RiskLevel:       m.RiskLevel,
		}
	}
	return result
}

func swapInfoToProto(m models.SwapInfo) *metricspb.SwapInfo {
	return &metricspb.SwapInfo{
		Used:    m.Used,
		Total:   m.Total,
		Percent: m.Percent,
	}
}

func diskInfoToProto(m models.DiskInfo) *metricspb.DiskInfo {
	return &metricspb.DiskInfo{
		Used:    m.Used,
		Total:   m.Total,
		Percent: m.Percent,
	}
}

func processHealthInfoToProto(m models.ProcessHealthInfo) *metricspb.ProcessHealthInfo {
	return &metricspb.ProcessHealthInfo{
		TotalProcesses:  int32(m.TotalProcesses),
		ZombieProcesses: processInfoSliceToProto(m.ZombieProcesses),
		HighThreadCount: processInfoSliceToProto(m.HighThreadCount),
		LeakCandidates:  processInfoSliceToProto(m.LeakCandidates),
	}
}

func messageQueueInfoToProto(m models.MessageQueueInfo) *metricspb.MessageQueueInfo {
	return &metricspb.MessageQueueInfo{
		RedisPubsub: &metricspb.RedisPubSubInfo{
			Subscribers: int32(m.RedisPubSub.Subscribers),
			Channels:    int32(m.RedisPubSub.Channels),
		},
		BackgroundJobs: &metricspb.BackgroundJobsInfo{
			Pending: int32(m.BackgroundJobs.Pending),
			Active:  int32(m.BackgroundJobs.Active),
			Failed:  int32(m.BackgroundJobs.Failed),
		},
	}
}

func storageIOInfoToProto(m models.StorageIOInfo) *metricspb.StorageIOInfo {
	return &metricspb.StorageIOInfo{
		DiskQueueDepth: m.DiskQueueDepth,
		IoWaitPercent:  m.IOWaitPercent,
		ReadMbPerSec:   m.ReadMBPerSec,
		WriteMbPerSec:  m.WriteMBPerSec,
	}
}
