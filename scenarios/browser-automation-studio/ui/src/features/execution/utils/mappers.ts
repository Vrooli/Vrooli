import type { Timestamp } from '@bufbuild/protobuf/wkt';
import {
  ArtifactType,
  ExecutionStatus as ProtoExecutionStatus,
  LogLevel as ProtoLogLevel,
  StepStatus,
  StepType,
} from '@vrooli/proto-types/browser-automation-studio/v1/shared_pb';
export type MappedLogLevel = 'info' | 'warning' | 'error' | 'success';

export type ExecutionStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled';

export const mapExecutionStatus = (
  status?: ProtoExecutionStatus | string | null,
): ExecutionStatus => {
  if (typeof status === 'number') {
    switch (status) {
      case ProtoExecutionStatus.PENDING:
        return 'pending';
      case ProtoExecutionStatus.RUNNING:
        return 'running';
      case ProtoExecutionStatus.COMPLETED:
        return 'completed';
      case ProtoExecutionStatus.FAILED:
        return 'failed';
      case ProtoExecutionStatus.CANCELLED:
        return 'cancelled';
      default:
        return 'pending';
    }
  }
  switch ((status ?? '').toLowerCase()) {
    case 'pending':
    case 'queued':
      return 'pending';
    case 'running':
    case 'in_progress':
      return 'running';
    case 'completed':
    case 'success':
    case 'succeeded':
      return 'completed';
    case 'failed':
    case 'error':
      return 'failed';
    case 'cancelled':
      return 'cancelled';
    default:
      return 'pending';
  }
};

export const mapStepType = (stepType?: StepType | string): string | undefined => {
  if (typeof stepType === 'number') {
    switch (stepType) {
      case StepType.NAVIGATE:
        return 'navigate';
      case StepType.CLICK:
        return 'click';
      case StepType.ASSERT:
        return 'assert';
      case StepType.SUBFLOW:
        return 'subflow';
      case StepType.INPUT:
        return 'input';
      case StepType.CUSTOM:
        return 'custom';
      default:
        return undefined;
    }
  }
  return stepType;
};

export const mapStepStatus = (status?: StepStatus | string): string | undefined => {
  if (typeof status === 'number') {
    switch (status) {
      case StepStatus.PENDING:
        return 'pending';
      case StepStatus.RUNNING:
        return 'running';
      case StepStatus.COMPLETED:
        return 'completed';
      case StepStatus.FAILED:
        return 'failed';
      case StepStatus.CANCELLED:
        return 'cancelled';
      case StepStatus.SKIPPED:
        return 'skipped';
      case StepStatus.RETRYING:
        return 'retrying';
      default:
        return undefined;
    }
  }
  return status;
};

export const mapArtifactType = (type?: ArtifactType | string): string | undefined => {
  if (typeof type === 'number') {
    switch (type) {
      case ArtifactType.TIMELINE_FRAME:
        return 'timeline_frame';
      case ArtifactType.CONSOLE_LOG:
        return 'console_log';
      case ArtifactType.NETWORK_EVENT:
        return 'network_event';
      case ArtifactType.SCREENSHOT:
        return 'screenshot';
      case ArtifactType.DOM_SNAPSHOT:
        return 'dom_snapshot';
      case ArtifactType.TRACE:
        return 'trace';
      case ArtifactType.CUSTOM:
        return 'custom';
      default:
        return undefined;
    }
  }
  return type;
};

export const mapProtoLogLevel = (level?: ProtoLogLevel | string): MappedLogLevel => {
  if (typeof level === 'number') {
    switch (level) {
      case ProtoLogLevel.DEBUG:
        return 'info';
      case ProtoLogLevel.INFO:
        return 'info';
      case ProtoLogLevel.WARN:
        return 'warning';
      case ProtoLogLevel.ERROR:
        return 'error';
      default:
        return 'info';
    }
  }
  switch ((level ?? '').toLowerCase()) {
    case 'debug':
    case 'info':
      return 'info';
    case 'warn':
    case 'warning':
      return 'warning';
    case 'error':
      return 'error';
    default:
      return 'info';
  }
};

export const timestampToDate = (value?: Timestamp | null): Date | undefined => {
  if (!value) return undefined;
  const millis =
    Number(value.seconds ?? 0) * 1000 +
    Math.floor(Number(value.nanos ?? 0) / 1_000_000);
  const result = new Date(millis);
  return Number.isNaN(result.valueOf()) ? undefined : result;
};
