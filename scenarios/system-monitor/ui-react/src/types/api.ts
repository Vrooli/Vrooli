// API Response Types for System Monitor
// These interfaces match the Go backend models

export interface MetricsResponse {
  cpu_usage: number;
  memory_usage: number;
  tcp_connections: number;
  timestamp: string;
}

export interface DetailedMetrics {
  cpu_details: CPUMetrics;
  memory_details: MemoryMetrics;
  network_details: NetworkMetrics;
  system_details: SystemHealth;
  timestamp: string;
}

export interface CPUMetrics {
  usage: number;
  top_processes: ProcessInfo[];
  load_average: number[];
  context_switches: number;
  total_goroutines: number;
}

export interface MemoryMetrics {
  usage: number;
  top_processes: ProcessInfo[];
  growth_patterns: MemoryGrowth[];
  swap_usage: SwapInfo;
  disk_usage: DiskInfo;
}

export interface NetworkMetrics {
  tcp_states: TCPConnectionStates;
  port_usage: PortUsageInfo;
  network_stats: NetworkStatistics;
  connection_pools: ConnectionPool[];
}

export interface SystemHealth {
  file_descriptors: FileDescriptorInfo;
  service_dependencies: ServiceHealth[];
  certificates: CertificateInfo[];
}

export interface ProcessInfo {
  pid: number;
  name: string;
  cpu_percent: number;
  memory_mb: number;
  connections: number;
  threads: number;
  file_descriptors: number;
  status: string;
  goroutines?: number;
}

export interface TCPConnectionStates {
  established: number;
  time_wait: number;
  close_wait: number;
  fin_wait1: number;
  fin_wait2: number;
  syn_sent: number;
  syn_recv: number;
  closing: number;
  last_ack: number;
  listen: number;
  total: number;
}

export interface ConnectionPool {
  name: string;
  active: number;
  idle: number;
  max_size: number;
  waiting: number;
  healthy: boolean;
  leak_risk: string;
}

export interface NetworkStatistics {
  bandwidth_in_mbps: number;
  bandwidth_out_mbps: number;
  packet_loss: number;
  dns_success_rate: number;
  dns_latency_ms: number;
}

export interface ServiceHealth {
  name: string;
  status: string;
  latency_ms: number;
  last_check: string;
  endpoint: string;
}

export interface CertificateInfo {
  domain: string;
  days_to_expiry: number;
  status: string;
}

export interface MemoryGrowth {
  process: string;
  growth_mb_per_hour: number;
  risk_level: string;
}

export interface SwapInfo {
  used: number;
  total: number;
  percent: number;
}

export interface DiskInfo {
  used: number;
  total: number;
  percent: number;
}

export interface PortUsageInfo {
  used: number;
  total: number;
}

export interface FileDescriptorInfo {
  used: number;
  max: number;
  percent: number;
}

export interface ProcessMonitorData {
  process_health: ProcessHealthInfo;
  resource_matrix: ProcessInfo[];
  timestamp: string;
}

export interface ProcessHealthInfo {
  total_processes: number;
  zombie_processes: ProcessInfo[];
  high_thread_count: ProcessInfo[];
  leak_candidates: ProcessInfo[];
}

export interface InfrastructureMonitorData {
  database_pools: ConnectionPool[];
  http_client_pools: ConnectionPool[];
  message_queues: MessageQueueInfo;
  storage_io: StorageIOInfo;
  timestamp: string;
}

export interface MessageQueueInfo {
  redis_pubsub: RedisPubSubInfo;
  background_jobs: BackgroundJobsInfo;
}

export interface RedisPubSubInfo {
  subscribers: number;
  channels: number;
}

export interface BackgroundJobsInfo {
  pending: number;
  active: number;
  failed: number;
}

export interface StorageIOInfo {
  disk_queue_depth: number;
  io_wait_percent: number;
  read_mb_per_sec: number;
  write_mb_per_sec: number;
}

// Investigation Types
export interface Investigation {
  id: string;
  status: string;
  timestamp: string;
  trigger_reason: string;
  findings: string;
  confidence_score: number;
  agent_id?: string;
}

export interface InvestigationScript {
  id: string;
  name: string;
  description: string;
  category: string;
  created_at: string;
  updated_at: string;
  author: string;
  enabled: boolean;
}

export interface ScriptExecution {
  script_id: string;
  execution_id: string;
  status: 'running' | 'completed' | 'failed';
  started_at: string;
  completed_at?: string;
  output?: string;
  error?: string;
  exit_code?: number;
}

// Report Types
export interface SystemReport {
  id: string;
  type: 'daily' | 'weekly' | 'monthly';
  generated_at: string;
  period_start: string;
  period_end: string;
  summary: ReportSummary;
  metrics: ReportMetrics;
  alerts: ReportAlert[];
  recommendations: string[];
}

export interface ReportSummary {
  total_alerts: number;
  avg_cpu_usage: number;
  avg_memory_usage: number;
  max_tcp_connections: number;
  uptime_percentage: number;
}

export interface ReportMetrics {
  cpu_trend: number[];
  memory_trend: number[];
  network_trend: number[];
  timestamps: string[];
}

export interface ReportAlert {
  timestamp: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  category: string;
  message: string;
  resolved: boolean;
}

// API Error Response
export interface APIError {
  error: string;
  details?: string;
  timestamp?: string;
}

// Generic API Response Wrapper
export interface APIResponse<T> {
  data?: T;
  error?: APIError;
  status: number;
}