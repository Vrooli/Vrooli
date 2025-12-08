import type { Timestamp } from '@bufbuild/protobuf/wkt';
import { ExecutionEventEnvelopeSchema, type ExecutionEventEnvelope } from '@vrooli/proto-types/browser-automation-studio/v1/execution_pb';
import {
  EventKind,
  ExecutionStatus as ProtoExecutionStatus,
  StepStatus,
} from '@vrooli/proto-types/browser-automation-studio/v1/shared_pb';
import { parseProtoStrict } from '../../../utils/proto';
import type { ExecutionStatus } from '../utils/mappers';
import {
  mapExecutionStatus,
  mapProtoLogLevel,
  mapStepStatus,
  mapStepType,
  timestampToDate,
} from '../utils/mappers';

export type ExecutionEventType =
  | 'execution.started'
  | 'execution.progress'
  | 'execution.completed'
  | 'execution.failed'
  | 'execution.cancelled'
  | 'step.started'
  | 'step.completed'
  | 'step.failed'
  | 'step.screenshot'
  | 'step.heartbeat'
  | 'step.log'
  | 'step.telemetry';

export interface ExecutionEventMessage {
  type: ExecutionEventType;
  execution_id: string;
  workflow_id: string;
  step_index?: number;
  step_node_id?: string;
  step_type?: string;
  status?: string;
  progress?: number;
  message?: string;
  payload?: Record<string, unknown> | null;
  timestamp?: string;
}

export interface ExecutionUpdateMessage {
  type: string;
  execution_id?: string;
  status?: string;
  progress?: number;
  current_step?: string;
  message?: string;
  data?: ExecutionEventMessage | null;
  timestamp?: string;
}

const timestampToIso = (value?: Timestamp | null): string | undefined =>
  timestampToDate(value)?.toISOString();

export const parseEventEnvelope = (value: unknown): ExecutionEventEnvelope | null => {
  try {
    return parseProtoStrict(ExecutionEventEnvelopeSchema, value);
  } catch {
    return null;
  }
};

export const envelopeToExecutionEvent = (
  envelope: ExecutionEventEnvelope | null | undefined,
): ExecutionEventMessage | null => {
  if (!envelope) return null;

  const executionId = typeof envelope.executionId === 'string' ? envelope.executionId : '';
  const workflowId = typeof envelope.workflowId === 'string' ? envelope.workflowId : '';
  const timestampIso = timestampToIso(envelope.timestamp);

  const base = {
    execution_id: executionId,
    workflow_id: workflowId,
    step_index: envelope.stepIndex,
    timestamp: timestampIso,
  };

  const kind = typeof envelope.kind === 'number' ? envelope.kind : EventKind.UNSPECIFIED;

  switch (envelope.payload?.case) {
    case 'statusUpdate': {
      const update = envelope.payload.value;
      const mappedStatus: ExecutionStatus = mapExecutionStatus(update.status);
      const type: ExecutionEventType = (() => {
        switch (update.status) {
          case ProtoExecutionStatus.RUNNING:
            return 'execution.started';
          case ProtoExecutionStatus.COMPLETED:
            return 'execution.completed';
          case ProtoExecutionStatus.FAILED:
            return 'execution.failed';
          case ProtoExecutionStatus.CANCELLED:
            return 'execution.cancelled';
          default:
            return 'execution.progress';
        }
      })();

      return {
        ...base,
        type,
        status: mappedStatus,
        progress: update.progress,
        message: update.error ?? undefined,
      };
    }
    case 'timelineFrame': {
      const frame = envelope.payload.value.frame;
      if (!frame) return null;
      const status = mapStepStatus(frame.status);
      const type: ExecutionEventType = (() => {
        switch (frame.status) {
          case StepStatus.RUNNING:
            return 'step.started';
          case StepStatus.COMPLETED:
            return 'step.completed';
          case StepStatus.FAILED:
          case StepStatus.CANCELLED:
            return 'step.failed';
          default:
            return 'execution.progress';
        }
      })();

      return {
        ...base,
        type,
        step_index: frame.stepIndex,
        step_node_id: frame.nodeId,
        step_type: mapStepType(frame.stepType),
        status,
        progress: frame.progress,
        payload: {
          assertion: frame.assertion ?? undefined,
          retry_attempt: frame.retryAttempt ?? undefined,
          retry_max_attempts: frame.retryMaxAttempts ?? undefined,
          retry_delay_ms: frame.retryDelayMs ?? undefined,
          dom_snapshot_preview: frame.domSnapshotPreview ?? undefined,
        },
      };
    }
    case 'log': {
      const log = envelope.payload.value;
      return {
        ...base,
        type: 'step.log',
        message: log.message,
        payload: {
          level: mapProtoLogLevel(log.level),
          step_index: envelope.stepIndex ?? undefined,
        },
      };
    }
    case 'heartbeat': {
      const heartbeat = envelope.payload.value;
      return {
        ...base,
        type: 'step.heartbeat',
        progress: heartbeat.progress,
        payload: {
          metrics: heartbeat.metrics,
          received_at: heartbeat.receivedAt ? timestampToIso(heartbeat.receivedAt) : undefined,
        },
      };
    }
    case 'telemetry': {
      const telemetry = envelope.payload.value;
      return {
        ...base,
        type: 'step.telemetry',
        payload: telemetry.metrics as Record<string, unknown> | null | undefined,
      };
    }
    default: {
      if (kind === EventKind.STATUS_UPDATE) {
        return { ...base, type: 'execution.progress' };
      }
      return null;
    }
  }
};

export const parseLegacyUpdate = (value: unknown): ExecutionUpdateMessage | null => {
  if (!value || typeof value !== 'object') {
    return null;
  }
  const candidate = value as Record<string, unknown>;
  if (typeof candidate.type !== 'string') {
    return null;
  }
  return {
    type: candidate.type,
    execution_id: typeof candidate.execution_id === 'string' ? candidate.execution_id : undefined,
    status: typeof candidate.status === 'string' ? candidate.status : undefined,
    progress: typeof candidate.progress === 'number' ? candidate.progress : undefined,
    current_step: typeof candidate.current_step === 'string' ? candidate.current_step : undefined,
    message: typeof candidate.message === 'string' ? candidate.message : undefined,
    data: candidate.data as ExecutionEventMessage | null | undefined,
    timestamp: typeof candidate.timestamp === 'string' ? candidate.timestamp : undefined,
  };
};

type EventContext = {
  fallbackTimestamp?: string;
  fallbackProgress?: number;
};

type LoggerLike = {
  error: (message: string, meta?: Record<string, unknown>, err?: unknown) => void;
  warn?: (message: string, meta?: Record<string, unknown>, err?: unknown) => void;
};

export class ExecutionEventsClient {
  private readonly onEvent: (event: ExecutionEventMessage, ctx: EventContext) => void;
  private readonly onLegacy?: (legacy: ExecutionUpdateMessage) => void;
  private readonly logger?: LoggerLike;

  constructor(options: {
    onEvent: (event: ExecutionEventMessage, ctx: EventContext) => void;
    onLegacy?: (legacy: ExecutionUpdateMessage) => void;
    logger?: LoggerLike;
  }) {
    this.onEvent = options.onEvent;
    this.onLegacy = options.onLegacy;
    this.logger = options.logger;
  }

  createMessageListener(): EventListener {
    return (event) => {
      const messageEvent = event as MessageEvent;
      let payload: unknown = messageEvent.data;

      if (typeof payload === 'string') {
        try {
          payload = JSON.parse(payload) as unknown;
        } catch (err) {
          this.logger?.error?.('Failed to parse WebSocket payload as JSON', {}, err);
          return;
        }
      }

      try {
        const envelope = parseEventEnvelope(payload);
        if (envelope) {
          const parsedEvent = envelopeToExecutionEvent(envelope);
          if (parsedEvent) {
            if (!parsedEvent.payload && payload && typeof payload === 'object') {
              const legacyPayload = (payload as Record<string, unknown>).legacy_payload;
              if (legacyPayload !== undefined) {
                parsedEvent.payload = legacyPayload as Record<string, unknown>;
              }
            }
            this.onEvent(parsedEvent, {
              fallbackTimestamp: timestampToIso(envelope.timestamp),
              fallbackProgress: parsedEvent.progress,
            });
          }
          return;
        }

        if (payload && typeof payload === 'object') {
          const nestedEnvelope = parseEventEnvelope(
            (payload as Record<string, unknown>).data,
          );
          if (nestedEnvelope) {
            const parsedEvent = envelopeToExecutionEvent(nestedEnvelope);
            if (parsedEvent) {
              if (!parsedEvent.payload) {
                const legacyPayload = (payload as Record<string, unknown>).legacy_payload;
                if (legacyPayload !== undefined) {
                  parsedEvent.payload = legacyPayload as Record<string, unknown>;
                }
              }
              this.onEvent(parsedEvent, {
                fallbackTimestamp: timestampToIso(nestedEnvelope.timestamp),
                fallbackProgress: parsedEvent.progress,
              });
            }
            return;
          }
        }

        const legacy = parseLegacyUpdate(payload);
        if (legacy) {
          this.onLegacy?.(legacy);
        }
      } catch (err) {
        this.logger?.error?.('Failed to process execution event payload', {}, err);
      }
    };
  }
}
