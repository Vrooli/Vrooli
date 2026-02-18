package convert

import (
	"system-monitor-api/internal/models"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func MetricsResponseToProto(m *models.MetricsResponse) *domain.MetricsResponse {
	if m == nil {
		return nil
	}
	pb := &domain.MetricsResponse{
		CpuUsage:       m.CPUUsage,
		MemoryUsage:    m.MemoryUsage,
		TcpConnections: int32(m.TCPConnections),
		GpuUsage:       m.GPUUsage,
		Timestamp:      timestamppb.New(m.Timestamp),
	}
	return pb
}

func MetricsTimelineResponseToProto(m *models.MetricsTimelineResponse) *domain.MetricsTimelineResponse {
	if m == nil {
		return nil
	}
	samples := make([]*domain.MetricTimelineSample, len(m.Samples))
	for i, s := range m.Samples {
		samples[i] = &domain.MetricTimelineSample{
			Timestamp:      timestamppb.New(s.Timestamp),
			CpuUsage:       s.CPUUsage,
			MemoryUsage:    s.MemoryUsage,
			TcpConnections: int32(s.TCPConnections),
			GpuUsage:       s.GPUUsage,
		}
	}
	return &domain.MetricsTimelineResponse{
		WindowSeconds:         int32(m.WindowSeconds),
		SampleIntervalSeconds: int32(m.SampleIntervalSeconds),
		Samples:               samples,
	}
}

func DetailedMetricsToProto(m *models.DetailedMetrics) *domain.DetailedMetrics {
	if m == nil {
		return nil
	}
	pb := &domain.DetailedMetrics{
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

func ProcessMonitorDataToProto(m *models.ProcessMonitorData) *domain.ProcessMonitorData {
	if m == nil {
		return nil
	}
	return &domain.ProcessMonitorData{
		ProcessHealth:  processHealthInfoToProto(m.ProcessHealth),
		ResourceMatrix: processInfoSliceToProto(m.ResourceMatrix),
		Timestamp:      timestamppb.New(m.Timestamp),
	}
}

func InfrastructureMonitorDataToProto(m *models.InfrastructureMonitorData) *domain.InfrastructureMonitorData {
	if m == nil {
		return nil
	}
	return &domain.InfrastructureMonitorData{
		DatabasePools:   connectionPoolSliceToProto(m.DatabasePools),
		HttpClientPools: connectionPoolSliceToProto(m.HTTPClientPools),
		MessageQueues:   messageQueueInfoToProto(m.MessageQueues),
		StorageIo:       storageIOInfoToProto(m.StorageIO),
		Timestamp:       timestamppb.New(m.Timestamp),
	}
}

func cpuMetricsToProto(m models.CPUMetrics) *domain.CPUMetrics {
	return &domain.CPUMetrics{
		Usage:           m.Usage,
		TopProcesses:    processInfoSliceToProto(m.TopProcesses),
		LoadAverage:     m.LoadAverage,
		ContextSwitches: m.ContextSwitches,
		TotalGoroutines: int32(m.Goroutines),
	}
}

func memoryMetricsToProto(m models.MemoryMetrics) *domain.MemoryMetrics {
	pb := &domain.MemoryMetrics{
		Usage:          m.Usage,
		TopProcesses:   processInfoSliceToProto(m.TopProcesses),
		GrowthPatterns: memoryGrowthSliceToProto(m.GrowthPatterns),
		SwapUsage:      swapInfoToProto(m.SwapUsage),
		DiskUsage:      diskInfoToProto(m.DiskUsage),
	}
	return pb
}

func networkMetricsToProto(m models.NetworkMetrics) *domain.NetworkMetrics {
	return &domain.NetworkMetrics{
		TcpStates:       tcpConnectionStatesToProto(m.TCPStates),
		PortUsage:       portUsageInfoToProto(m.PortUsage),
		NetworkStats:    networkStatisticsToProto(m.NetworkStats),
		ConnectionPools: connectionPoolSliceToProto(m.ConnectionPools),
	}
}

func systemHealthToProto(m models.SystemHealth) *domain.SystemHealth {
	pb := &domain.SystemHealth{
		FileDescriptors:     fileDescriptorInfoToProto(m.FileDescriptors),
		ServiceDependencies: serviceHealthSliceToProto(m.ServiceDependencies),
		Certificates:        certificateInfoSliceToProto(m.Certificates),
	}
	if m.InotifyWatchers != nil {
		pb.InotifyWatchers = inotifyWatcherInfoToProto(m.InotifyWatchers)
	}
	return pb
}

func gpuMetricsToProto(m *models.GPUMetrics) *domain.GPUMetrics {
	if m == nil {
		return nil
	}
	return &domain.GPUMetrics{
		Summary:       gpuSummaryToProto(m.Summary),
		Devices:       gpuDeviceMetricsSliceToProto(m.Devices),
		Errors:        m.Errors,
		DriverVersion: m.Driver,
		PrimaryModel:  m.Model,
	}
}

func gpuSummaryToProto(m models.GPUSummary) *domain.GPUSummary {
	return &domain.GPUSummary{
		TotalUtilizationPercent:   m.TotalUtilizationPercent,
		AverageUtilizationPercent: m.AverageUtilizationPercent,
		TotalMemoryMb:             m.TotalMemoryMB,
		UsedMemoryMb:              m.UsedMemoryMB,
		AverageTemperatureC:       m.AverageTemperatureC,
		DeviceCount:               int32(m.DeviceCount),
	}
}

func gpuDeviceMetricsSliceToProto(ms []models.GPUDeviceMetrics) []*domain.GPUDeviceMetrics {
	result := make([]*domain.GPUDeviceMetrics, len(ms))
	for i, m := range ms {
		pb := &domain.GPUDeviceMetrics{
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

func gpuProcessInfoSliceToProto(ms []models.GPUProcessInfo) []*domain.GPUProcessInfo {
	result := make([]*domain.GPUProcessInfo, len(ms))
	for i, m := range ms {
		pb := &domain.GPUProcessInfo{
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

func processInfoSliceToProto(ms []models.ProcessInfo) []*domain.ProcessInfo {
	result := make([]*domain.ProcessInfo, len(ms))
	for i, m := range ms {
		result[i] = &domain.ProcessInfo{
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

func tcpConnectionStatesToProto(m models.TCPConnectionStates) *domain.TCPConnectionStates {
	return &domain.TCPConnectionStates{
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

func connectionPoolSliceToProto(ms []models.ConnectionPool) []*domain.ConnectionPool {
	result := make([]*domain.ConnectionPool, len(ms))
	for i, m := range ms {
		result[i] = &domain.ConnectionPool{
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

func networkStatisticsToProto(m models.NetworkStatistics) *domain.NetworkStatistics {
	return &domain.NetworkStatistics{
		BandwidthInMbps:  m.BandwidthInMbps,
		BandwidthOutMbps: m.BandwidthOutMbps,
		PacketLoss:       m.PacketLoss,
		DnsSuccessRate:   m.DNSSuccessRate,
		DnsLatencyMs:     m.DNSLatencyMs,
	}
}

func portUsageInfoToProto(m models.PortUsageInfo) *domain.PortUsageInfo {
	return &domain.PortUsageInfo{
		Used:  int32(m.Used),
		Total: int32(m.Total),
	}
}

func fileDescriptorInfoToProto(m models.FileDescriptorInfo) *domain.FileDescriptorInfo {
	return &domain.FileDescriptorInfo{
		Used:    int32(m.Used),
		Max:     int32(m.Max),
		Percent: m.Percent,
	}
}

func serviceHealthSliceToProto(ms []models.ServiceHealth) []*domain.ServiceHealth {
	result := make([]*domain.ServiceHealth, len(ms))
	for i, m := range ms {
		result[i] = &domain.ServiceHealth{
			Name:      m.Name,
			Status:    m.Status,
			LatencyMs: m.LatencyMs,
			LastCheck: timestamppb.New(m.LastCheck),
			Endpoint:  m.Endpoint,
		}
	}
	return result
}

func certificateInfoSliceToProto(ms []models.CertificateInfo) []*domain.CertificateInfo {
	result := make([]*domain.CertificateInfo, len(ms))
	for i, m := range ms {
		result[i] = &domain.CertificateInfo{
			Domain:       m.Domain,
			DaysToExpiry: int32(m.DaysToExpiry),
			Status:       m.Status,
		}
	}
	return result
}

func inotifyWatcherInfoToProto(m *models.InotifyWatcherInfo) *domain.InotifyWatcherInfo {
	if m == nil {
		return nil
	}
	return &domain.InotifyWatcherInfo{
		Supported:        m.Supported,
		WatchesUsed:      int32(m.WatchesUsed),
		WatchesMax:       int32(m.WatchesMax),
		WatchesPercent:   m.WatchesPercent,
		InstancesUsed:    int32(m.InstancesUsed),
		InstancesMax:     int32(m.InstancesMax),
		InstancesPercent: m.InstancesPercent,
	}
}

func memoryGrowthSliceToProto(ms []models.MemoryGrowth) []*domain.MemoryGrowth {
	result := make([]*domain.MemoryGrowth, len(ms))
	for i, m := range ms {
		result[i] = &domain.MemoryGrowth{
			Process:         m.Process,
			GrowthMbPerHour: m.GrowthMBPerHour,
			RiskLevel:       m.RiskLevel,
		}
	}
	return result
}

func swapInfoToProto(m models.SwapInfo) *domain.SwapInfo {
	return &domain.SwapInfo{
		Used:    m.Used,
		Total:   m.Total,
		Percent: m.Percent,
	}
}

func diskInfoToProto(m models.DiskInfo) *domain.DiskInfo {
	return &domain.DiskInfo{
		Used:    m.Used,
		Total:   m.Total,
		Percent: m.Percent,
	}
}

func processHealthInfoToProto(m models.ProcessHealthInfo) *domain.ProcessHealthInfo {
	return &domain.ProcessHealthInfo{
		TotalProcesses:  int32(m.TotalProcesses),
		ZombieProcesses: processInfoSliceToProto(m.ZombieProcesses),
		HighThreadCount: processInfoSliceToProto(m.HighThreadCount),
		LeakCandidates:  processInfoSliceToProto(m.LeakCandidates),
	}
}

func messageQueueInfoToProto(m models.MessageQueueInfo) *domain.MessageQueueInfo {
	return &domain.MessageQueueInfo{
		RedisPubsub: &domain.RedisPubSubInfo{
			Subscribers: int32(m.RedisPubSub.Subscribers),
			Channels:    int32(m.RedisPubSub.Channels),
		},
		BackgroundJobs: &domain.BackgroundJobsInfo{
			Pending: int32(m.BackgroundJobs.Pending),
			Active:  int32(m.BackgroundJobs.Active),
			Failed:  int32(m.BackgroundJobs.Failed),
		},
	}
}

func storageIOInfoToProto(m models.StorageIOInfo) *domain.StorageIOInfo {
	return &domain.StorageIOInfo{
		DiskQueueDepth: m.DiskQueueDepth,
		IoWaitPercent:  m.IOWaitPercent,
		ReadMbPerSec:   m.ReadMBPerSec,
		WriteMbPerSec:  m.WriteMBPerSec,
	}
}
