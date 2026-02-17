// API Response Types for System Monitor
// Re-exported from proto-generated types (single source of truth)

export type {
  MetricsResponse,
  MetricTimelineSample,
  MetricsTimelineResponse,
  DetailedMetrics,
  CPUMetrics,
  MemoryMetrics,
  NetworkMetrics,
  SystemHealth,
  ProcessInfo,
  TCPConnectionStates,
  ConnectionPool,
  NetworkStatistics,
  ServiceHealth,
  CertificateInfo,
  MemoryGrowth,
  SwapInfo,
  DiskInfo,
  DiskPartitionInfo,
  DiskUsageEntry,
  DiskDetailResponse,
  GPUMetrics,
  GPUSummary,
  GPUDeviceMetrics,
  GPUProcessInfo,
  PortUsageInfo,
  FileDescriptorInfo,
  InotifyWatcherInfo,
  ProcessMonitorData,
  ProcessHealthInfo,
  InfrastructureMonitorData,
  MessageQueueInfo,
  RedisPubSubInfo,
  BackgroundJobsInfo,
  StorageIOInfo,
} from '@vrooli/proto-types/system-monitor/v1/domain/metrics_pb';

export type {
  Investigation,
  InvestigationStep,
  InvestigationFinding,
  RootCause,
  TimelineEvent,
  ImpactAssessment,
  ResolutionPlan,
  ResolutionStep,
  TriggerConfig,
  CooldownStatus,
  Anomaly,
} from '@vrooli/proto-types/system-monitor/v1/domain/investigations_pb';

export {
  InvestigationStatus,
  InvestigationStepStatus,
  FindingType,
  Severity,
  Relevance,
  TriggerCondition,
  RiskLevel,
} from '@vrooli/proto-types/system-monitor/v1/domain/types_pb';

// Script types — now proto-backed
export type {
  InvestigationScript,
  ScriptExecution,
} from '@vrooli/proto-types/system-monitor/v1/domain/scripts_pb';

export {
  ScriptExecutionStatus,
} from '@vrooli/proto-types/system-monitor/v1/domain/scripts_pb';

// Report types — now proto-backed
export type {
  SystemReport,
  ReportSummary,
  ReportMetrics,
  ReportAlert,
  EnhancedSystemReport,
} from '@vrooli/proto-types/system-monitor/v1/domain/reports_pb';

// Settings — now proto-backed
export type {
  SystemSettings,
} from '@vrooli/proto-types/system-monitor/v1/domain/settings_pb';

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

/** Terminal statuses for investigations — lowercase strings matching the API contract. */
export const INVESTIGATION_TERMINAL_STATUSES = new Set([
  'completed',
  'failed',
  'cancelled',
  'stopped',
]);
