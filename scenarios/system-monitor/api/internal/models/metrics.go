package models

import (
	"time"
)

// MetricsResponse represents the current system metrics
type MetricsResponse struct {
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	TCPConnections int       `json:"tcp_connections"`
	GPUUsage       *float64  `json:"gpu_usage,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

// DetailedMetrics contains comprehensive system metrics
type DetailedMetrics struct {
	CPUDetails     CPUMetrics     `json:"cpu_details"`
	MemoryDetails  MemoryMetrics  `json:"memory_details"`
	NetworkDetails NetworkMetrics `json:"network_details"`
	GPUDetails     *GPUMetrics    `json:"gpu_details,omitempty"`
	SystemDetails  SystemHealth   `json:"system_details"`
	Timestamp      time.Time      `json:"timestamp"`
}

// GPUMetrics contains GPU-related metrics
type GPUMetrics struct {
	Summary GPUSummary         `json:"summary"`
	Devices []GPUDeviceMetrics `json:"devices"`
	Errors  []string           `json:"errors,omitempty"`
	Driver  string             `json:"driver_version,omitempty"`
	Model   string             `json:"primary_model,omitempty"`
}

// GPUSummary aggregates metrics across all devices
type GPUSummary struct {
	TotalUtilizationPercent   float64 `json:"total_utilization_percent"`
	AverageUtilizationPercent float64 `json:"average_utilization_percent"`
	TotalMemoryMB             float64 `json:"total_memory_mb"`
	UsedMemoryMB              float64 `json:"used_memory_mb"`
	AverageTemperatureC       float64 `json:"average_temperature_c"`
	DeviceCount               int     `json:"device_count"`
}

// GPUDeviceMetrics contains metrics for an individual GPU
type GPUDeviceMetrics struct {
	Index             int              `json:"index"`
	UUID              string           `json:"uuid"`
	Name              string           `json:"name"`
	Utilization       float64          `json:"utilization_percent"`
	MemoryUtilization float64          `json:"memory_utilization_percent"`
	MemoryUsedMB      float64          `json:"memory_used_mb"`
	MemoryTotalMB     float64          `json:"memory_total_mb"`
	TemperatureC      *float64         `json:"temperature_c,omitempty"`
	FanSpeedPercent   *float64         `json:"fan_speed_percent,omitempty"`
	PowerDrawW        *float64         `json:"power_draw_w,omitempty"`
	PowerLimitW       *float64         `json:"power_limit_w,omitempty"`
	SMClockMHz        *float64         `json:"sm_clock_mhz,omitempty"`
	MemoryClockMHz    *float64         `json:"memory_clock_mhz,omitempty"`
	Processes         []GPUProcessInfo `json:"processes,omitempty"`
}

// GPUProcessInfo describes a process consuming GPU resources
type GPUProcessInfo struct {
	PID           int      `json:"pid"`
	ProcessName   string   `json:"process_name"`
	MemoryUsedMB  float64  `json:"memory_used_mb"`
	SMUtilization *float64 `json:"sm_utilization_percent,omitempty"`
	GPUInstanceID string   `json:"gpu_instance_id,omitempty"`
}

// CPUMetrics contains CPU-related metrics
type CPUMetrics struct {
	Usage           float64       `json:"usage"`
	TopProcesses    []ProcessInfo `json:"top_processes"`
	LoadAverage     []float64     `json:"load_average"`
	ContextSwitches int64         `json:"context_switches"`
	Goroutines      int           `json:"total_goroutines"`
}

// MemoryMetrics contains memory-related metrics
type MemoryMetrics struct {
	Usage          float64        `json:"usage"`
	TopProcesses   []ProcessInfo  `json:"top_processes"`
	GrowthPatterns []MemoryGrowth `json:"growth_patterns"`
	SwapUsage      SwapInfo       `json:"swap_usage"`
	DiskUsage      DiskInfo       `json:"disk_usage"`
}

// NetworkMetrics contains network-related metrics
type NetworkMetrics struct {
	TCPStates       TCPConnectionStates `json:"tcp_states"`
	PortUsage       PortUsageInfo       `json:"port_usage"`
	NetworkStats    NetworkStatistics   `json:"network_stats"`
	ConnectionPools []ConnectionPool    `json:"connection_pools"`
}

// SystemHealth contains overall system health information
type SystemHealth struct {
	FileDescriptors     FileDescriptorInfo  `json:"file_descriptors"`
	ServiceDependencies []ServiceHealth     `json:"service_dependencies"`
	Certificates        []CertificateInfo   `json:"certificates"`
	InotifyWatchers     *InotifyWatcherInfo `json:"inotify_watchers,omitempty"`
}

// InotifyWatcherInfo captures inotify watcher and instance utilisation
type InotifyWatcherInfo struct {
	Supported        bool    `json:"supported"`
	WatchesUsed      int     `json:"watches_used"`
	WatchesMax       int     `json:"watches_max"`
	WatchesPercent   float64 `json:"watches_percent"`
	InstancesUsed    int     `json:"instances_used"`
	InstancesMax     int     `json:"instances_max"`
	InstancesPercent float64 `json:"instances_percent"`
}

// DiskPartitionInfo provides usage data for a mounted volume
type DiskPartitionInfo struct {
	Device         string  `json:"device"`
	MountPoint     string  `json:"mount_point"`
	SizeBytes      int64   `json:"size_bytes"`
	SizeHuman      string  `json:"size_human"`
	UsedBytes      int64   `json:"used_bytes"`
	UsedHuman      string  `json:"used_human"`
	AvailableBytes int64   `json:"available_bytes"`
	AvailableHuman string  `json:"available_human"`
	UsePercent     float64 `json:"use_percent"`
}

// DiskUsageEntry represents a directory or file contributing to disk usage
type DiskUsageEntry struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SizeHuman string `json:"size_human"`
	Category  string `json:"category,omitempty"`
}

// DiskDetailResponse aggregates disk usage insights for the UI
type DiskDetailResponse struct {
	Partitions     []DiskPartitionInfo `json:"partitions"`
	ActiveMount    string              `json:"active_mount"`
	Depth          int                 `json:"depth"`
	TopDirectories []DiskUsageEntry    `json:"top_directories"`
	LargestFiles   []DiskUsageEntry    `json:"largest_files"`
	Notes          []string            `json:"notes,omitempty"`
	Timestamp      time.Time           `json:"timestamp"`
}

// ProcessInfo represents information about a system process
type ProcessInfo struct {
	PID         int     `json:"pid"`
	Name        string  `json:"name"`
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryMB    float64 `json:"memory_mb"`
	Connections int     `json:"connections"`
	Threads     int     `json:"threads"`
	FDs         int     `json:"file_descriptors"`
	Status      string  `json:"status"`
	Goroutines  int     `json:"goroutines,omitempty"`
}

// TCPConnectionStates tracks TCP connection states
type TCPConnectionStates struct {
	Established int `json:"established"`
	TimeWait    int `json:"time_wait"`
	CloseWait   int `json:"close_wait"`
	FinWait1    int `json:"fin_wait1"`
	FinWait2    int `json:"fin_wait2"`
	SynSent     int `json:"syn_sent"`
	SynRecv     int `json:"syn_recv"`
	Closing     int `json:"closing"`
	LastAck     int `json:"last_ack"`
	Listen      int `json:"listen"`
	Total       int `json:"total"`
}

// ConnectionPool represents a connection pool's status
type ConnectionPool struct {
	Name     string `json:"name"`
	Active   int    `json:"active"`
	Idle     int    `json:"idle"`
	MaxSize  int    `json:"max_size"`
	Waiting  int    `json:"waiting"`
	Healthy  bool   `json:"healthy"`
	LeakRisk string `json:"leak_risk"`
}

// NetworkStatistics contains network performance metrics
type NetworkStatistics struct {
	BandwidthInMbps  float64 `json:"bandwidth_in_mbps"`
	BandwidthOutMbps float64 `json:"bandwidth_out_mbps"`
	PacketLoss       float64 `json:"packet_loss"`
	DNSSuccessRate   float64 `json:"dns_success_rate"`
	DNSLatencyMs     float64 `json:"dns_latency_ms"`
}

// ServiceHealth represents the health status of a service
type ServiceHealth struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	LatencyMs float64   `json:"latency_ms"`
	LastCheck time.Time `json:"last_check"`
	Endpoint  string    `json:"endpoint"`
}

// CertificateInfo contains SSL certificate information
type CertificateInfo struct {
	Domain       string `json:"domain"`
	DaysToExpiry int    `json:"days_to_expiry"`
	Status       string `json:"status"`
}

// MemoryGrowth tracks memory growth patterns
type MemoryGrowth struct {
	Process         string  `json:"process"`
	GrowthMBPerHour float64 `json:"growth_mb_per_hour"`
	RiskLevel       string  `json:"risk_level"`
}

// SwapInfo contains swap memory information
type SwapInfo struct {
	Used    int64   `json:"used"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
}

// DiskInfo contains disk usage information
type DiskInfo struct {
	Used    int64   `json:"used"`
	Total   int64   `json:"total"`
	Percent float64 `json:"percent"`
}

// PortUsageInfo tracks port usage
type PortUsageInfo struct {
	Used  int `json:"used"`
	Total int `json:"total"`
}

// FileDescriptorInfo tracks file descriptor usage
type FileDescriptorInfo struct {
	Used    int     `json:"used"`
	Max     int     `json:"max"`
	Percent float64 `json:"percent"`
}

// ProcessMonitorData contains process monitoring information
type ProcessMonitorData struct {
	ProcessHealth  ProcessHealthInfo `json:"process_health"`
	ResourceMatrix []ProcessInfo     `json:"resource_matrix"`
	Timestamp      time.Time         `json:"timestamp"`
}

// ProcessHealthInfo summarizes process health
type ProcessHealthInfo struct {
	TotalProcesses  int           `json:"total_processes"`
	ZombieProcesses []ProcessInfo `json:"zombie_processes"`
	HighThreadCount []ProcessInfo `json:"high_thread_count"`
	LeakCandidates  []ProcessInfo `json:"leak_candidates"`
}

// InfrastructureMonitorData contains infrastructure metrics
type InfrastructureMonitorData struct {
	DatabasePools   []ConnectionPool `json:"database_pools"`
	HTTPClientPools []ConnectionPool `json:"http_client_pools"`
	MessageQueues   MessageQueueInfo `json:"message_queues"`
	StorageIO       StorageIOInfo    `json:"storage_io"`
	Timestamp       time.Time        `json:"timestamp"`
}

// MessageQueueInfo contains message queue metrics
type MessageQueueInfo struct {
	RedisPubSub    RedisPubSubInfo    `json:"redis_pubsub"`
	BackgroundJobs BackgroundJobsInfo `json:"background_jobs"`
}

// RedisPubSubInfo contains Redis pub/sub metrics
type RedisPubSubInfo struct {
	Subscribers int `json:"subscribers"`
	Channels    int `json:"channels"`
}

// BackgroundJobsInfo contains background job metrics
type BackgroundJobsInfo struct {
	Pending int `json:"pending"`
	Active  int `json:"active"`
	Failed  int `json:"failed"`
}

// StorageIOInfo contains storage I/O metrics
type StorageIOInfo struct {
	DiskQueueDepth float64 `json:"disk_queue_depth"`
	IOWaitPercent  float64 `json:"io_wait_percent"`
	ReadMBPerSec   float64 `json:"read_mb_per_sec"`
	WriteMBPerSec  float64 `json:"write_mb_per_sec"`
}
