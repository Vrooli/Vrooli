import {
  ArtifactType,
  ExecutionStatus as ProtoExecutionStatus,
  LogLevel as ProtoLogLevel,
  StepStatus,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ActionType } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import {
  ExecutionStatusName,
  ActionTypeName,
  StepStatusName,
  ArtifactTypeName,
  LogLevelName,
} from '@vrooli/proto-types/enum-names';

export type MappedLogLevel = 'info' | 'warning' | 'error' | 'success';

export type ExecutionStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled';

/**
 * Maps ExecutionStatus proto enum to UI status string.
 * Uses generated enum names for numeric values, with fallback aliases for string input.
 */
export const mapExecutionStatus = (
  status?: ProtoExecutionStatus | string | null,
): ExecutionStatus => {
  if (typeof status === 'number') {
    const name = ExecutionStatusName[status as ProtoExecutionStatus];
    // Map to valid ExecutionStatus (exclude 'unspecified')
    if (name === 'pending' || name === 'running' || name === 'completed' || name === 'failed' || name === 'cancelled') {
      return name;
    }
    return 'pending'; // default for unspecified
  }
  // String fallback with aliases for legacy compatibility
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

/**
 * Maps ActionType enum to display string.
 * Uses generated enum names, returns undefined for unspecified.
 */
export const mapStepType = (actionType?: ActionType | string): string | undefined => {
  if (typeof actionType === 'number') {
    const name = ActionTypeName[actionType as ActionType];
    return name === 'unspecified' ? undefined : name;
  }
  return actionType;
};

/**
 * Maps StepStatus enum to display string.
 * Uses generated enum names, returns undefined for unspecified.
 */
export const mapStepStatus = (status?: StepStatus | string): string | undefined => {
  if (typeof status === 'number') {
    const name = StepStatusName[status as StepStatus];
    return name === 'unspecified' ? undefined : name;
  }
  return status;
};

/**
 * Maps ArtifactType enum to display string.
 * Uses generated enum names, returns undefined for unspecified.
 */
export const mapArtifactType = (type?: ArtifactType | string): string | undefined => {
  if (typeof type === 'number') {
    const name = ArtifactTypeName[type as ArtifactType];
    return name === 'unspecified' ? undefined : name;
  }
  return type;
};

/**
 * Maps LogLevel enum to UI-specific log level.
 * Note: DEBUG and INFO both map to 'info', WARN maps to 'warning'.
 * The 'success' level is UI-only (not in proto).
 */
export const mapProtoLogLevel = (level?: ProtoLogLevel | string): MappedLogLevel => {
  if (typeof level === 'number') {
    const name = LogLevelName[level as ProtoLogLevel];
    switch (name) {
      case 'debug':
      case 'info':
        return 'info';
      case 'warn':
        return 'warning';
      case 'error':
        return 'error';
      default:
        return 'info';
    }
  }
  // String fallback
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

// Re-export centralized timestamp utility for backwards compatibility
export { protoTimestampToDate as timestampToDate } from '../../../utils/timestamps';
