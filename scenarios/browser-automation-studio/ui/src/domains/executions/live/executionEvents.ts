import type { Timestamp } from '@bufbuild/protobuf/wkt';
import {
  TimelineStreamMessageSchema,
  type TimelineStreamMessage,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import {
  ExecutionStatus as ProtoExecutionStatus,
  StepStatus,
} from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
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

const streamMessageTimestampToIso = (
  message: TimelineStreamMessage | null | undefined,
): string | undefined => {
  if (!message?.payload?.case) return undefined;
  switch (message.payload.case) {
    case 'entry':
      return timestampToIso(message.payload.value.timestamp);
    case 'log':
      return timestampToIso(message.payload.value.timestamp);
    case 'heartbeat':
      return timestampToIso(message.payload.value.timestamp);
    default:
      return undefined;
  }
};

/**
 * Parse a TimelineStreamMessage from raw JSON.
 * This is the new unified streaming format that replaces ExecutionEventEnvelope.
 */
export const parseStreamMessage = (value: unknown): TimelineStreamMessage | null => {
  try {
    return parseProtoStrict(TimelineStreamMessageSchema, value);
  } catch {
    return null;
  }
};

// Backwards compatibility alias
export const parseEventEnvelope = parseStreamMessage;

/**
 * Convert a TimelineStreamMessage to an ExecutionEventMessage.
 * This bridges the new unified format to the existing event handling code.
 */
export const streamMessageToExecutionEvent = (
  message: TimelineStreamMessage | null | undefined,
): ExecutionEventMessage | null => {
  if (!message) return null;

  const base = {
    execution_id: '',
    workflow_id: '',
    timestamp: undefined as string | undefined,
  };

  switch (message.payload?.case) {
    case 'entry': {
      const entry = message.payload.value;
      const context = entry.context;
      const aggregates = entry.aggregates;

      // Extract execution_id from context
      if (context?.origin?.case === 'executionId') {
        base.execution_id = context.origin.value;
      }
      base.timestamp = entry.timestamp ? timestampToIso(entry.timestamp) : undefined;

      const status = aggregates ? mapStepStatus(aggregates.status) : undefined;
      const type: ExecutionEventType = (() => {
        if (!aggregates) return 'execution.progress';
        switch (aggregates.status) {
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
        step_index: entry.stepIndex,
        step_node_id: entry.nodeId,
        step_type: entry.action?.type ? mapStepType(entry.action.type) : undefined,
        status,
        progress: aggregates?.progress,
        payload: {
          assertion: context?.assertion ?? undefined,
          retry_attempt: context?.retryStatus?.currentAttempt ?? undefined,
          retry_max_attempts: context?.retryStatus?.maxAttempts ?? undefined,
          retry_delay_ms: context?.retryStatus?.delayMs ?? undefined,
          dom_snapshot_preview: aggregates?.domSnapshotPreview ?? undefined,
        },
      };
    }
    case 'status': {
      const update = message.payload.value;
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
        execution_id: update.id,
        type,
        status: mappedStatus,
        progress: update.progress,
        message: update.error ?? undefined,
      };
    }
    case 'log': {
      const log = message.payload.value;
      return {
        ...base,
        type: 'step.log',
        message: log.message,
        timestamp: log.timestamp ? timestampToIso(log.timestamp) : undefined,
        payload: {
          level: mapProtoLogLevel(log.level),
          step_name: log.stepName ?? undefined,
        },
      };
    }
    case 'heartbeat': {
      const heartbeat = message.payload.value;
      return {
        ...base,
        type: 'step.heartbeat',
        timestamp: heartbeat.timestamp ? timestampToIso(heartbeat.timestamp) : undefined,
        payload: {
          session_id: heartbeat.sessionId ?? undefined,
        },
      };
    }
    default:
      return null;
  }
};

// Backwards compatibility alias
export const envelopeToExecutionEvent = streamMessageToExecutionEvent;

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
              fallbackTimestamp: streamMessageTimestampToIso(envelope),
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
                fallbackTimestamp: streamMessageTimestampToIso(nestedEnvelope),
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
