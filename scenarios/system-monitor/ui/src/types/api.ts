// API Response Types for System Monitor
// Re-exported from proto-generated types (single source of truth)

export type {
  MetricsResponse,
  MetricTimelineSample,
  MetricsTimelineResponse,
  DetailedMetrics,
  CPUMetrics,
  MetricValue,
  MemoryMetrics,
  NetworkMetrics,
  SystemHealth,
  ProcessInfo,
  TCPConnectionStates,
  ConnectionPool,
  NetworkStatistics,
  ServiceHealth,
  CertificateInfo,
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
} from '@vrooli/proto-types/system-monitor/v1/metrics/metrics_pb';

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
} from '@vrooli/proto-types/system-monitor/v1/investigations/investigations_pb';

export {
  InvestigationStatus,
  InvestigationStepStatus,
  FindingType,
  Severity,
  Relevance,
  TriggerCondition,
  RiskLevel,
} from '@vrooli/proto-types/system-monitor/v1/investigations/investigations_pb';

// Script types — now proto-backed
export type {
  InvestigationScript,
  ScriptExecution,
} from '@vrooli/proto-types/system-monitor/v1/scripts/scripts_pb';

export {
  ScriptExecutionStatus,
} from '@vrooli/proto-types/system-monitor/v1/scripts/scripts_pb';

// Report types — now proto-backed
export type {
  SystemReport,
  ReportSummary,
  ReportMetrics,
  ReportAlert,
  EnhancedSystemReport,
} from '@vrooli/proto-types/system-monitor/v1/reports/reports_pb';

// Settings — now proto-backed
export type {
  SystemSettings,
} from '@vrooli/proto-types/system-monitor/v1/settings/settings_pb';

// Structured error codes returned by the Go backend
export type ErrorCode = 'validation' | 'unauthorized' | 'forbidden' | 'not_found' | 'conflict' | 'cooldown' | 'unavailable' | 'internal' | 'network';

// Recovery hints tell the UI what corrective action is appropriate.
export type RecoveryAction = 'fix_input' | 'authenticate' | 'wait' | 'check_config' | 'contact_admin' | 'none';

export interface ErrorDetail {
  code: ErrorCode;
  message: string;
  retryable: boolean;
  retry_after_seconds?: number;
  field?: string;
  recovery?: RecoveryAction;
  request_id?: string;
}

// API Error Response
export interface APIError {
  error: string;
  detail?: ErrorDetail;
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
