import { fromJson } from '@bufbuild/protobuf';
import type { Timestamp } from '@bufbuild/protobuf/wkt';
import {
  ExecutionSchema,
  GetScreenshotsResponseSchema,
  type GetScreenshotsResponse as ProtoGetScreenshotsResponse,
  ExecutionEventEnvelopeSchema,
  type ExecutionEventEnvelope,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution_pb';
import {
  ExecutionTimelineSchema,
  type ExecutionTimeline as ProtoExecutionTimeline,
  type TimelineFrame as ProtoTimelineFrame,
  type TimelineLog as ProtoTimelineLog,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline_pb';
import {
  ArtifactType,
  EventKind,
  ExecutionStatus as ProtoExecutionStatus,
  LogLevel as ProtoLogLevel,
  StepStatus,
  StepType,
} from '@vrooli/proto-types/browser-automation-studio/v1/shared_pb';
import { create } from 'zustand';
import { getConfig } from '../config';
import { logger } from '../utils/logger';
import {
  processExecutionEvent,
  createId,
  parseTimestamp,
  type ExecutionEventMessage,
  type ExecutionEventHandlers,
  type ExecutionEventType,
  type Screenshot,
  type LogEntry,
} from './executionEventProcessor';

const WS_RETRY_LIMIT = 5;
const WS_RETRY_BASE_DELAY_MS = 1500;

const timestampToDate = (value?: Timestamp | null): Date | undefined => {
  if (!value) return undefined;
  const millis = Number(value.seconds ?? 0) * 1000 + Math.floor(Number(value.nanos ?? 0) / 1_000_000);
  const result = new Date(millis);
  return Number.isNaN(result.valueOf()) ? undefined : result;
};

const coerceDate = (value: unknown): Date | undefined => {
  if (value instanceof Date) return value;
  if (typeof value === 'string' || typeof value === 'number') {
    const d = new Date(value);
    return Number.isNaN(d.valueOf()) ? undefined : d;
  }
  return undefined;
};

const toNumber = (value?: number | bigint | null): number | undefined => {
  if (value == null) return undefined;
  return typeof value === 'bigint' ? Number(value) : value;
};

const protoTimestampToDate = (ts?: Timestamp | null): Date | undefined => timestampToDate(ts);

/**
 * Converts a snake_case string to camelCase.
 * E.g., "workflow_id" -> "workflowId", "started_at" -> "startedAt"
 */
const snakeToCamel = (str: string): string =>
  str.replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase());

/**
 * Recursively normalizes JSON object keys from snake_case to camelCase.
 * This ensures compatibility with protobuf JSON parsing which expects camelCase field names.
 */
const normalizeJsonKeys = (value: unknown): unknown => {
  if (value === null || value === undefined) return value;
  if (Array.isArray(value)) return value.map(normalizeJsonKeys);
  if (typeof value === 'object') {
    const result: Record<string, unknown> = {};
    for (const [key, val] of Object.entries(value as Record<string, unknown>)) {
      result[snakeToCamel(key)] = normalizeJsonKeys(val);
    }
    return result;
  }
  return value;
};

interface ExecutionUpdateMessage {
  type: string;
  execution_id?: string;
  status?: string;
  progress?: number;
  current_step?: string;
  message?: string;
  data?: ExecutionEventMessage | null;
  timestamp?: string;
}

export interface TimelineBoundingBox {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
}

export interface TimelineRegion {
  selector?: string;
  bounding_box?: TimelineBoundingBox;
  padding?: number;
  color?: string;
  opacity?: number;
}

export interface TimelineRetryHistoryEntry {
  attempt?: number;
  success?: boolean;
  duration_ms?: number;
  call_duration_ms?: number;
  error?: string;
}

export interface TimelineScreenshot {
  artifact_id: string;
  url?: string;
  thumbnail_url?: string;
  width?: number;
  height?: number;
  content_type?: string;
  size_bytes?: number;
}

export interface TimelineAssertion {
  mode?: string;
  selector?: string;
  expected?: unknown;
  actual?: unknown;
  success?: boolean;
  message?: string;
  negated?: boolean;
  caseSensitive?: boolean;
}

export interface TimelineArtifact {
  id: string;
  type: string;
  label?: string;
  storage_url?: string;
  thumbnail_url?: string;
  content_type?: string;
  size_bytes?: number;
  step_index?: number;
  payload?: Record<string, unknown> | null;
}

export interface TimelineLog {
  id?: string;
  level?: string;
  message?: string;
  step_name?: string;
  stepName?: string;
  timestamp?: string;
}

export interface TimelineFrame {
  step_index: number;
  stepIndex?: number;
  node_id?: string;
  nodeId?: string;
  step_type?: StepType | string;
  stepType?: StepType | string;
  status?: StepStatus | string;
  success: boolean;
  duration_ms?: number;
  durationMs?: number;
  total_duration_ms?: number;
  totalDurationMs?: number;
  progress?: number;
  started_at?: string;
  startedAt?: string;
  completed_at?: string;
  completedAt?: string;
  final_url?: string;
  finalUrl?: string;
  error?: string;
  console_log_count?: number;
  consoleLogCount?: number;
  network_event_count?: number;
  networkEventCount?: number;
  extracted_data_preview?: unknown;
  extractedDataPreview?: unknown;
  highlight_regions?: TimelineRegion[];
  highlightRegions?: TimelineRegion[];
  mask_regions?: TimelineRegion[];
  maskRegions?: TimelineRegion[];
  focused_element?: { selector?: string; bounding_box?: TimelineBoundingBox };
  focusedElement?: { selector?: string; boundingBox?: TimelineBoundingBox };
  element_bounding_box?: TimelineBoundingBox;
  elementBoundingBox?: TimelineBoundingBox;
  click_position?: { x?: number; y?: number };
  clickPosition?: { x?: number; y?: number };
  cursor_trail?: Array<{ x?: number; y?: number }>;
  cursorTrail?: Array<{ x?: number; y?: number }>;
  zoom_factor?: number;
  zoomFactor?: number;
  screenshot?: TimelineScreenshot | null;
  artifacts?: TimelineArtifact[];
  assertion?: TimelineAssertion | null;
  retry_attempt?: number;
  retryAttempt?: number;
  retry_max_attempts?: number;
  retryMaxAttempts?: number;
  retry_configured?: number;
  retryConfigured?: number;
  retry_delay_ms?: number;
  retryDelayMs?: number;
  retry_backoff_factor?: number;
  retryBackoffFactor?: number;
  retry_history?: TimelineRetryHistoryEntry[];
  retryHistory?: TimelineRetryHistoryEntry[];
  dom_snapshot_preview?: string;
  domSnapshotPreview?: string;
  dom_snapshot_artifact_id?: string;
  domSnapshotArtifactId?: string;
  dom_snapshot?: TimelineArtifact | null;
  timeline_artifact_id?: string;
  timelineArtifactId?: string;
}

export interface Execution {
  id: string;
  workflowId: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  startedAt: Date;
  completedAt?: Date;
  screenshots: Screenshot[];
  timeline: TimelineFrame[];
  logs: LogEntry[];
  currentStep?: string;
  progress: number;
  error?: string;
  lastHeartbeat?: {
    step?: string;
    elapsedMs?: number;
    timestamp: Date;
  };
}

interface ExecutionStore {
  executions: Execution[];
  currentExecution: Execution | null;
  viewerWorkflowId: string | null;
  socket: WebSocket | null;
  socketListeners: Map<string, EventListener>;
  websocketStatus: 'idle' | 'connecting' | 'connected' | 'error';
  websocketAttempts: number;

  openViewer: (workflowId: string) => void;
  closeViewer: () => void;
  startExecution: (workflowId: string, saveWorkflowFn?: () => Promise<void>) => Promise<void>;
  stopExecution: (executionId: string) => Promise<void>;
  loadExecutions: (workflowId?: string) => Promise<void>;
  loadExecution: (executionId: string) => Promise<void>;
  refreshTimeline: (executionId: string) => Promise<void>;
  connectWebSocket: (executionId: string, attempt?: number) => Promise<void>;
  disconnectWebSocket: () => void;
  addScreenshot: (screenshot: Screenshot) => void;
  addLog: (log: LogEntry) => void;
  updateExecutionStatus: (status: Execution['status'], error?: string) => void;
  updateProgress: (progress: number, currentStep?: string) => void;
  recordHeartbeat: (step?: string, elapsedMs?: number) => void;
  clearCurrentExecution: () => void;
}

const mapExecutionStatus = (status?: ProtoExecutionStatus | string | null): Execution['status'] => {
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

const mapStepType = (stepType?: StepType | string): string | undefined => {
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

const mapStepStatus = (status?: StepStatus | string): string | undefined => {
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

const mapArtifactType = (type?: ArtifactType | string): string | undefined => {
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

const mapProtoLogLevel = (level?: ProtoLogLevel | string): LogEntry['level'] => {
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

const parseExecutionProto = (raw: unknown): Execution => {
  // Normalize snake_case keys to camelCase for proto compatibility
  const normalized = normalizeJsonKeys(raw) as Record<string, unknown>;

  // Attempt proto parsing with normalized data, fall back to empty object on failure
  let proto: ReturnType<typeof fromJson<typeof ExecutionSchema>>;
  try {
    proto = fromJson(ExecutionSchema, normalized as any, { ignoreUnknownFields: true });
  } catch {
    // Proto parsing failed - use empty proto, fallbacks will handle the data
    proto = {} as ReturnType<typeof fromJson<typeof ExecutionSchema>>;
  }

  // Extract dates with fallbacks (normalized uses camelCase)
  const startedAt =
    protoTimestampToDate(proto.startedAt) ??
    coerceDate(normalized?.startedAt) ??
    new Date();
  const completedAt =
    protoTimestampToDate(proto.completedAt) ??
    coerceDate(normalized?.completedAt);
  const lastHeartbeat = proto.lastHeartbeat
    ? protoTimestampToDate(proto.lastHeartbeat)
    : coerceDate(normalized?.lastHeartbeat);

  // Extract fields with comprehensive fallbacks
  // Note: normalized already has camelCase keys (executionId, workflowId, etc.)
  return {
    id: proto.id || String(normalized?.id ?? normalized?.executionId ?? ''),
    workflowId: proto.workflowId || String(normalized?.workflowId ?? ''),
    status: mapExecutionStatus(proto.status || (normalized?.status as string | undefined)),
    startedAt,
    completedAt: completedAt || undefined,
    screenshots: [],
    timeline: [],
    logs: [],
    currentStep: proto.currentStep || (normalized?.currentStep ? String(normalized.currentStep) : undefined),
    progress: typeof proto.progress === 'number' && proto.progress > 0
      ? proto.progress
      : (typeof normalized?.progress === 'number' ? normalized.progress as number : 0),
    error: proto.error ?? (normalized?.error ? String(normalized.error) : undefined),
    lastHeartbeat: lastHeartbeat
      ? {
          timestamp: lastHeartbeat,
        }
      : undefined,
  };
};

const mapBoundingBoxFromProto = (bbox?: { x?: number; y?: number; width?: number; height?: number } | null) => {
  if (!bbox) return undefined;
  const { x, y, width, height } = bbox;
  if ([x, y, width, height].every((v) => v == null)) return undefined;
  return { x, y, width, height };
};

const mapTimelineArtifactFromProto = (artifact?: ProtoTimelineFrame['artifacts'][number]): TimelineArtifact | undefined => {
  if (!artifact) return undefined;
  const stepIndex = artifact.stepIndex ?? undefined;
  return {
    id: artifact.id,
    type: mapArtifactType(artifact.type) ?? 'unknown',
    label: artifact.label || undefined,
    storage_url: artifact.storageUrl || undefined,
    thumbnail_url: artifact.thumbnailUrl || undefined,
    content_type: artifact.contentType || undefined,
    size_bytes: toNumber(artifact.sizeBytes),
    step_index: stepIndex,
    payload: artifact.payload ?? undefined,
  };
};

const mapTimelineFrameFromProto = (frame: ProtoTimelineFrame): TimelineFrame => {
  const screenshot = frame.screenshot
    ? {
        artifact_id: frame.screenshot.artifactId,
        url: frame.screenshot.url,
        thumbnail_url: frame.screenshot.thumbnailUrl,
        width: frame.screenshot.width,
        height: frame.screenshot.height,
        content_type: frame.screenshot.contentType,
        size_bytes: toNumber(frame.screenshot.sizeBytes),
      }
    : undefined;

  const focusBox = frame.focusedElement?.boundingBox ? mapBoundingBoxFromProto(frame.focusedElement.boundingBox) : undefined;
  const elementBox = mapBoundingBoxFromProto(frame.elementBoundingBox);

  const domSnapshot = frame.domSnapshot ? mapTimelineArtifactFromProto(frame.domSnapshot) : undefined;

  return {
    step_index: frame.stepIndex,
    stepIndex: frame.stepIndex,
    node_id: frame.nodeId,
    nodeId: frame.nodeId,
    step_type: mapStepType(frame.stepType),
    stepType: mapStepType(frame.stepType),
    status: mapStepStatus(frame.status),
    success: frame.success,
    duration_ms: frame.durationMs,
    durationMs: frame.durationMs,
    total_duration_ms: frame.totalDurationMs,
    totalDurationMs: frame.totalDurationMs,
    progress: frame.progress,
    started_at: timestampToDate(frame.startedAt)?.toISOString(),
    startedAt: timestampToDate(frame.startedAt)?.toISOString(),
    completed_at: timestampToDate(frame.completedAt)?.toISOString(),
    completedAt: timestampToDate(frame.completedAt)?.toISOString(),
    final_url: frame.finalUrl || undefined,
    finalUrl: frame.finalUrl || undefined,
    error: frame.error || undefined,
    console_log_count: frame.consoleLogCount,
    consoleLogCount: frame.consoleLogCount,
    network_event_count: frame.networkEventCount,
    networkEventCount: frame.networkEventCount,
    extracted_data_preview: frame.extractedDataPreview,
    extractedDataPreview: frame.extractedDataPreview,
    highlight_regions: frame.highlightRegions?.map((region) => ({
      selector: region.selector,
      bounding_box: mapBoundingBoxFromProto(region.boundingBox),
      padding: region.padding,
      color: region.color,
    })),
    highlightRegions: frame.highlightRegions?.map((region) => ({
      selector: region.selector,
      boundingBox: mapBoundingBoxFromProto(region.boundingBox),
      padding: region.padding,
      color: region.color,
    })),
    mask_regions: frame.maskRegions?.map((region) => ({
      selector: region.selector,
      bounding_box: mapBoundingBoxFromProto(region.boundingBox),
      opacity: region.opacity,
    })),
    maskRegions: frame.maskRegions?.map((region) => ({
      selector: region.selector,
      boundingBox: mapBoundingBoxFromProto(region.boundingBox),
      opacity: region.opacity,
    })),
    focused_element: frame.focusedElement
      ? { selector: frame.focusedElement.selector, bounding_box: focusBox }
      : undefined,
    focusedElement: frame.focusedElement
      ? { selector: frame.focusedElement.selector, boundingBox: focusBox }
      : undefined,
    element_bounding_box: elementBox,
    elementBoundingBox: elementBox,
    click_position: frame.clickPosition ? { x: frame.clickPosition.x, y: frame.clickPosition.y } : undefined,
    clickPosition: frame.clickPosition ? { x: frame.clickPosition.x, y: frame.clickPosition.y } : undefined,
    cursor_trail: frame.cursorTrail?.map((pt) => ({ x: pt.x, y: pt.y })),
    cursorTrail: frame.cursorTrail?.map((pt) => ({ x: pt.x, y: pt.y })),
    zoom_factor: frame.zoomFactor,
    zoomFactor: frame.zoomFactor,
    screenshot: screenshot ?? null,
    artifacts: frame.artifacts?.map((artifact) => mapTimelineArtifactFromProto(artifact)!).filter(Boolean) ?? [],
    assertion: frame.assertion
      ? {
          mode: frame.assertion.mode,
          selector: frame.assertion.selector,
          expected: frame.assertion.expected,
          actual: frame.assertion.actual,
          success: frame.assertion.success,
          message: frame.assertion.message || undefined,
          negated: frame.assertion.negated,
          caseSensitive: frame.assertion.caseSensitive,
        }
      : null,
    retry_attempt: frame.retryAttempt,
    retryAttempt: frame.retryAttempt,
    retry_max_attempts: frame.retryMaxAttempts,
    retryMaxAttempts: frame.retryMaxAttempts,
    retry_configured: frame.retryConfigured,
    retryConfigured: frame.retryConfigured,
    retry_delay_ms: frame.retryDelayMs,
    retryDelayMs: frame.retryDelayMs,
    retry_backoff_factor: frame.retryBackoffFactor,
    retryBackoffFactor: frame.retryBackoffFactor,
    retry_history: frame.retryHistory?.map((entry) => ({
      attempt: entry.attempt,
      success: entry.success,
      duration_ms: entry.durationMs,
      call_duration_ms: entry.callDurationMs,
      error: entry.error || undefined,
    })) ?? [],
    retryHistory: frame.retryHistory?.map((entry) => ({
      attempt: entry.attempt,
      success: entry.success,
      duration_ms: entry.durationMs,
      call_duration_ms: entry.callDurationMs,
      error: entry.error || undefined,
    })) ?? [],
    dom_snapshot_preview: frame.domSnapshotPreview || undefined,
    domSnapshotPreview: frame.domSnapshotPreview || undefined,
    dom_snapshot_artifact_id: domSnapshot?.id,
    domSnapshotArtifactId: domSnapshot?.id,
    dom_snapshot: domSnapshot ?? null,
    timeline_artifact_id: undefined,
    timelineArtifactId: undefined,
  };
};

const mapTimelineLogFromProto = (log: ProtoTimelineLog): LogEntry | null => {
  const baseMessage = typeof log.message === 'string' ? log.message.trim() : '';
  const stepName = typeof log.stepName === 'string' ? log.stepName : '';
  const composedMessage = stepName && baseMessage ? `${stepName}: ${baseMessage}` : baseMessage || stepName;
  if (!composedMessage) {
    return null;
  }

  const rawTimestamp = timestampToDate(log.timestamp) ?? coerceDate(log.timestamp ?? undefined);
  const id = log.id && log.id.trim().length > 0 ? log.id.trim() : `${stepName}|${composedMessage}|${rawTimestamp?.toISOString() ?? ''}`;

  return {
    id,
    level: mapProtoLogLevel(log.level),
    message: composedMessage,
    timestamp: rawTimestamp ?? new Date(),
  };
};

/**
 * Fallback parser for timeline frames from raw (normalized) JSON.
 * Used when proto parsing fails or returns incomplete data.
 */
const mapTimelineFrameFromRaw = (raw: Record<string, unknown>): TimelineFrame => {
  const stepIndex = typeof raw.stepIndex === 'number' ? raw.stepIndex : 0;
  const nodeId = raw.nodeId ? String(raw.nodeId) : undefined;
  const stepType = raw.stepType ? String(raw.stepType) : undefined;
  const status = raw.status ? String(raw.status) : undefined;
  const success = raw.success === true;
  const durationMs = typeof raw.durationMs === 'number' ? raw.durationMs : undefined;

  return {
    step_index: stepIndex,
    stepIndex,
    node_id: nodeId,
    nodeId,
    step_type: stepType,
    stepType,
    status,
    success,
    duration_ms: durationMs,
    durationMs,
    total_duration_ms: typeof raw.totalDurationMs === 'number' ? raw.totalDurationMs : undefined,
    totalDurationMs: typeof raw.totalDurationMs === 'number' ? raw.totalDurationMs : undefined,
    progress: typeof raw.progress === 'number' ? raw.progress : undefined,
    started_at: raw.startedAt ? String(raw.startedAt) : undefined,
    startedAt: raw.startedAt ? String(raw.startedAt) : undefined,
    completed_at: raw.completedAt ? String(raw.completedAt) : undefined,
    completedAt: raw.completedAt ? String(raw.completedAt) : undefined,
    final_url: raw.finalUrl ? String(raw.finalUrl) : undefined,
    finalUrl: raw.finalUrl ? String(raw.finalUrl) : undefined,
    error: raw.error ? String(raw.error) : undefined,
    screenshot: null,
    artifacts: [],
    assertion: null,
    retry_history: [],
    retryHistory: [],
  };
};

/**
 * Fallback parser for timeline logs from raw (normalized) JSON.
 * Used when proto parsing fails or returns incomplete data.
 */
const mapTimelineLogFromRaw = (raw: Record<string, unknown>): LogEntry | null => {
  const baseMessage = typeof raw.message === 'string' ? raw.message.trim() : '';
  const stepName = typeof raw.stepName === 'string' ? raw.stepName : '';
  const composedMessage = stepName && baseMessage ? `${stepName}: ${baseMessage}` : baseMessage || stepName;
  if (!composedMessage) {
    return null;
  }

  const rawTimestamp = coerceDate(raw.timestamp);
  const id = raw.id && typeof raw.id === 'string' && raw.id.trim().length > 0
    ? raw.id.trim()
    : `${stepName}|${composedMessage}|${rawTimestamp?.toISOString() ?? ''}`;

  const level = typeof raw.level === 'string' ? raw.level.toLowerCase() : 'info';
  const mappedLevel: LogEntry['level'] =
    level === 'error' ? 'error' :
    level === 'warn' || level === 'warning' ? 'warning' : 'info';

  return {
    id,
    level: mappedLevel,
    message: composedMessage,
    timestamp: rawTimestamp ?? new Date(),
  };
};

const mapScreenshotsFromProto = (raw: unknown): Screenshot[] => {
  const proto = fromJson(GetScreenshotsResponseSchema, raw as any, { ignoreUnknownFields: true }) as ProtoGetScreenshotsResponse;
  if (!proto.screenshots || proto.screenshots.length === 0) {
    return [];
  }
  return proto.screenshots
    .map((shot) => {
      const ts = protoTimestampToDate(shot.timestamp) ?? coerceDate(shot.timestamp) ?? parseTimestamp(shot.timestamp as any);
      const url = shot.storageUrl || shot.thumbnailUrl || '';
      if (!url) {
        return null;
      }
      return {
        id: shot.id || createId(),
        timestamp: ts,
        url,
        stepName: shot.stepName || `Step ${shot.stepIndex}`,
      } satisfies Screenshot;
    })
    .filter((screenshot): screenshot is Screenshot => screenshot !== null);
};

const parseEventEnvelope = (value: unknown): ExecutionEventEnvelope | null => {
  try {
    return fromJson(ExecutionEventEnvelopeSchema, value as any, { ignoreUnknownFields: true });
  } catch {
    return null;
  }
};

const envelopeToExecutionEvent = (envelope: ExecutionEventEnvelope | null | undefined): ExecutionEventMessage | null => {
  if (!envelope) return null;

  const executionId = typeof envelope.executionId === 'string' ? envelope.executionId : '';
  const workflowId = typeof envelope.workflowId === 'string' ? envelope.workflowId : '';
  const timestampIso = protoTimestampToDate(envelope.timestamp)?.toISOString();

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
      const mappedStatus = mapExecutionStatus(update.status);
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
          received_at: heartbeat.receivedAt ? protoTimestampToDate(heartbeat.receivedAt)?.toISOString() : undefined,
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

export const useExecutionStore = create<ExecutionStore>((set, get) => ({
  executions: [],
  currentExecution: null,
  viewerWorkflowId: null,
  socket: null,
  socketListeners: new Map(),
  websocketStatus: 'idle',
  websocketAttempts: 0,

  openViewer: (workflowId: string) => {
    const state = get();
    if (state.currentExecution && state.currentExecution.workflowId !== workflowId) {
      state.disconnectWebSocket();
      set({ currentExecution: null });
    }
    set({ viewerWorkflowId: workflowId });
  },

  closeViewer: () => {
    get().disconnectWebSocket();
    set({ currentExecution: null, viewerWorkflowId: null });
  },

  startExecution: async (workflowId: string, saveWorkflowFn?: () => Promise<void>) => {
    try {
      // Save workflow first if save function provided (to ensure latest changes are used)
      if (saveWorkflowFn) {
        await saveWorkflowFn();
      }

      const state = get();
      if (state.viewerWorkflowId && state.viewerWorkflowId !== workflowId) {
        state.disconnectWebSocket();
      }
      set({ viewerWorkflowId: workflowId });

      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/workflows/${workflowId}/execute`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          wait_for_completion: false,
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to start execution: ${response.status}`);
      }

      const data = await response.json();

      const execution: Execution = {
        id: data.execution_id,
        workflowId,
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 0,
        lastHeartbeat: undefined,
      };

      set({ currentExecution: execution });
      await get().connectWebSocket(execution.id);
      void get().refreshTimeline(execution.id);
    } catch (error) {
      logger.error('Failed to start execution', { component: 'ExecutionStore', action: 'startExecution', workflowId }, error);
      throw error;
    }
  },

  stopExecution: async (executionId: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/executions/${executionId}/stop`, {
        method: 'POST',
      });

      if (!response.ok) {
        throw new Error(`Failed to stop execution: ${response.status}`);
      }

      get().disconnectWebSocket();
      set((state) => ({
        currentExecution:
          state.currentExecution?.id === executionId
            ? {
                ...state.currentExecution,
                status: 'cancelled' as const,
                error: 'Execution cancelled by user',
                currentStep: 'Stopped by user',
                completedAt: new Date(),
              }
            : state.currentExecution,
      }));
    } catch (error) {
      logger.error('Failed to stop execution', { component: 'ExecutionStore', action: 'stopExecution', executionId }, error);
      throw error;
    }
  },

  loadExecutions: async (workflowId?: string) => {
    try {
      const config = await getConfig();
      const url = workflowId
        ? `${config.API_URL}/executions?workflow_id=${encodeURIComponent(workflowId)}`
        : `${config.API_URL}/executions`;
      const response = await fetch(url);

      if (response.status === 404) {
        set({ executions: [] });
        return;
      }

      if (!response.ok) {
        throw new Error(`Failed to load executions: ${response.status}`);
      }

      const data = await response.json();
      const executions = Array.isArray(data?.executions) ? data.executions : [];
      const normalizedExecutions = executions
        .map((item: unknown) => {
          try {
            return parseExecutionProto(item);
          } catch (err) {
            logger.error('Failed to parse execution proto', { component: 'ExecutionStore', action: 'loadExecutions' }, err);
            return null;
          }
        })
        .filter((entry: Execution | null): entry is Execution => entry !== null);
      set({ executions: normalizedExecutions });
    } catch (error) {
      logger.error('Failed to load executions', { component: 'ExecutionStore', action: 'loadExecutions', workflowId }, error);
    }
  },

  loadExecution: async (executionId: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/executions/${executionId}`);

      if (!response.ok) {
        throw new Error(`Failed to load execution: ${response.status}`);
      }

      const data = await response.json();
      const execution = parseExecutionProto(data);
      let screenshots: Screenshot[] = [];
      try {
        const shotsResponse = await fetch(`${config.API_URL}/executions/${executionId}/screenshots`);
        if (shotsResponse.ok) {
          const shotsJson = await shotsResponse.json();
          screenshots = mapScreenshotsFromProto(shotsJson);
        }
      } catch (shotsErr) {
        logger.error('Failed to load execution screenshots', { component: 'ExecutionStore', action: 'loadExecution', executionId }, shotsErr);
      }

      set({ currentExecution: { ...execution, screenshots }, viewerWorkflowId: execution.workflowId });
      void get().refreshTimeline(executionId);
    } catch (error) {
      logger.error('Failed to load execution', { component: 'ExecutionStore', action: 'loadExecution', executionId }, error);
    }
  },

  refreshTimeline: async (executionId: string) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/executions/${executionId}/timeline`);

      if (!response.ok) {
        throw new Error(`Failed to load execution timeline: ${response.status}`);
      }

      const data = await response.json();

      // Normalize snake_case keys to camelCase for proto compatibility
      const normalizedData = normalizeJsonKeys(data) as Record<string, unknown>;

      // Attempt proto parsing with normalized data, fall back gracefully on failure
      let protoTimeline: ProtoExecutionTimeline;
      let protoParsingSucceeded = true;
      try {
        protoTimeline = fromJson(ExecutionTimelineSchema, normalizedData as any, { ignoreUnknownFields: true });
      } catch (schemaError) {
        logger.error(
          'Timeline response failed proto validation, using fallback parsing',
          { component: 'ExecutionStore', action: 'refreshTimeline', executionId },
          schemaError,
        );
        // Create empty proto object - we'll use fallbacks
        protoTimeline = {} as ProtoExecutionTimeline;
        protoParsingSucceeded = false;
      }

      // Parse frames: use proto if available, otherwise use raw fallback parser
      let frames: TimelineFrame[];
      if (protoParsingSucceeded && protoTimeline.frames && protoTimeline.frames.length > 0) {
        frames = protoTimeline.frames.map((frame) => mapTimelineFrameFromProto(frame));
      } else {
        // Fallback: parse frames from normalized raw data
        const rawFrames = Array.isArray(normalizedData.frames) ? normalizedData.frames : [];
        frames = rawFrames.map((frame) => mapTimelineFrameFromRaw(frame as Record<string, unknown>));
      }

      // Parse logs: use proto if available, otherwise use raw fallback parser
      let normalizedLogs: LogEntry[];
      if (protoParsingSucceeded && protoTimeline.logs && protoTimeline.logs.length > 0) {
        normalizedLogs = protoTimeline.logs
          .map((log) => mapTimelineLogFromProto(log))
          .filter((entry): entry is LogEntry => Boolean(entry));
      } else {
        // Fallback: parse logs from normalized raw data
        const rawLogs = Array.isArray(normalizedData.logs) ? normalizedData.logs : [];
        normalizedLogs = rawLogs
          .map((log) => mapTimelineLogFromRaw(log as Record<string, unknown>))
          .filter((entry): entry is LogEntry => Boolean(entry));
      }

      // Extract status with fallback to raw data
      const mappedStatus = mapExecutionStatus(
        protoTimeline.status || (normalizedData.status as string | undefined)
      );

      // Extract progress with fallback to raw data
      const progressValue = typeof protoTimeline.progress === 'number'
        ? protoTimeline.progress
        : (typeof normalizedData.progress === 'number' ? normalizedData.progress as number : undefined);

      // Extract completedAt with fallback to raw data
      const completedAt = timestampToDate(protoTimeline.completedAt) ?? coerceDate(normalizedData.completedAt);

      // Extract error from raw data (ExecutionTimeline proto doesn't have error field)
      const errorValue = normalizedData.error ? String(normalizedData.error) : undefined;

      set((state) => {
        if (!state.currentExecution || state.currentExecution.id !== executionId) {
          return {};
        }

        const updated: Execution = {
          ...state.currentExecution,
          timeline: frames,
          logs: [...(state.currentExecution.logs ?? [])],
        };

        if (normalizedLogs.length > 0) {
          const merged = new Map<string, LogEntry>();
          for (const existingLog of updated.logs) {
            merged.set(existingLog.id, existingLog);
          }
          for (const newLog of normalizedLogs) {
            merged.set(newLog.id, newLog);
          }
          updated.logs = Array.from(merged.values()).sort(
            (a, b) => a.timestamp.getTime() - b.timestamp.getTime(),
          );
        }

        if (mappedStatus && updated.status !== mappedStatus) {
          updated.status = mappedStatus;
        }

        if (progressValue != null) {
          updated.progress = progressValue;
        }

        if (completedAt && (updated.status === 'completed' || updated.status === 'failed')) {
          updated.completedAt = completedAt;
        }

        if (errorValue && updated.status === 'failed') {
          updated.error = errorValue;
        }

        if (frames.length > 0) {
          const lastFrame = frames[frames.length - 1];
          const label = lastFrame?.node_id ?? lastFrame?.step_type;
          if (typeof label === 'string' && label.trim().length > 0) {
            updated.currentStep = label.trim();
          }
        }

        return { currentExecution: updated };
      });
    } catch (error) {
      logger.error('Failed to load execution timeline', { component: 'ExecutionStore', action: 'refreshTimeline', executionId }, error);
    }
  },

  connectWebSocket: async (executionId: string, attempt = 0) => {
    get().disconnectWebSocket();

    const config = await getConfig();
    if (!config.WS_URL) {
      logger.error('WebSocket base URL not configured', { component: 'ExecutionStore', action: 'connectWebSocket', executionId });
      set({ websocketStatus: 'error', websocketAttempts: attempt });
      return;
    }

    const wsBase = config.WS_URL.endsWith('/') ? config.WS_URL : `${config.WS_URL}/`;

    const toAbsoluteWsUrl = (value: string): string => {
      if (/^wss?:\/\//i.test(value)) {
        return value;
      }
      if (typeof window !== 'undefined' && window.location) {
        if (value.startsWith('/')) {
          return `${window.location.origin}${value}`;
        }
        try {
          return new URL(value, window.location.origin).toString();
        } catch {
          const normalized = value.startsWith('/') ? value : `/${value}`;
          return `${window.location.origin}${normalized}`;
        }
      }
      return value;
    };

    const absoluteWsBase = toAbsoluteWsUrl(wsBase);
    let wsUrl: URL;
    try {
      wsUrl = new URL(absoluteWsBase);
    } catch (err) {
      logger.error('Failed to construct WebSocket URL', { component: 'ExecutionStore', action: 'connectWebSocket', executionId, wsBase: absoluteWsBase }, err);
      set({ websocketStatus: 'error', websocketAttempts: attempt });
      return;
    }
    wsUrl.searchParams.set('execution_id', executionId);

    set({ websocketStatus: 'connecting', websocketAttempts: attempt });

    const socket = new WebSocket(wsUrl.toString());
    const listeners = new Map<string, EventListener>();
    let reconnectScheduled = false;
    let interruptionLogged = false;

    const announceInterruption = (message: string) => {
      if (interruptionLogged) {
        return;
      }
      const current = get().currentExecution;
      if (current && current.id === executionId) {
        get().addLog({
          id: createId(),
          level: 'warning',
          message,
          timestamp: new Date(),
        });
      }
      interruptionLogged = true;
    };

    const scheduleReconnect = () => {
      if (reconnectScheduled || typeof window === 'undefined') {
        return;
      }

      const current = get().currentExecution;
      if (!current || current.id !== executionId) {
        set({ websocketStatus: 'idle', websocketAttempts: attempt });
        return;
      }

      if (current.status !== 'running') {
        set({ websocketStatus: 'idle', websocketAttempts: attempt });
        return;
      }

      if (attempt >= WS_RETRY_LIMIT) {
        announceInterruption('Live execution stream disconnected. Max reconnect attempts reached.');
        logger.error('Max WebSocket reconnect attempts reached', { component: 'ExecutionStore', action: 'connectWebSocket', executionId, attempt });
        set({ websocketStatus: 'error', websocketAttempts: attempt });
        return;
      }

      const nextAttempt = attempt + 1;
      const delay = WS_RETRY_BASE_DELAY_MS * Math.min(nextAttempt, 5);
      reconnectScheduled = true;
      announceInterruption('Live execution stream interrupted. Retrying connection…');
      set({ websocketStatus: 'connecting', websocketAttempts: nextAttempt });

      window.setTimeout(() => {
        const currentExecution = get().currentExecution;
        if (!currentExecution || currentExecution.id !== executionId || currentExecution.status !== 'running') {
          return;
        }
        void get().connectWebSocket(executionId, nextAttempt);
      }, delay);
    };

    const handleEvent = (event: ExecutionEventMessage, fallbackTimestamp?: string, fallbackProgress?: number) => {
      const handlers: ExecutionEventHandlers = {
        updateExecutionStatus: get().updateExecutionStatus,
        updateProgress: get().updateProgress,
        addLog: get().addLog,
        addScreenshot: get().addScreenshot,
        recordHeartbeat: get().recordHeartbeat,
      };

      processExecutionEvent(handlers, event, {
        fallbackTimestamp,
        fallbackProgress,
      });
    };

    const handleEnvelopeMessage = (envelope: ExecutionEventEnvelope) => {
      const event = envelopeToExecutionEvent(envelope);
      if (!event) return;
      const fallbackTimestamp = protoTimestampToDate(envelope.timestamp)?.toISOString();
      handleEvent(event, fallbackTimestamp, event.progress);
    };

    const handleLegacyUpdate = (raw: ExecutionUpdateMessage) => {
      const progressValue = typeof raw.progress === 'number' ? raw.progress : undefined;
      const currentStep = raw.current_step;

      switch (raw.type) {
        case 'connected':
          return;
        case 'progress':
          if (typeof progressValue === 'number') {
            get().updateProgress(progressValue, currentStep);
          }
          return;
        case 'log': {
          const message = raw.message ?? 'Execution log entry';
          get().addLog({
            id: createId(),
            level: 'info',
            message,
            timestamp: parseTimestamp(raw.timestamp),
          });
          return;
        }
        case 'failed':
          get().updateExecutionStatus('failed', raw.message);
          return;
        case 'completed':
          get().updateExecutionStatus('completed');
          return;
        case 'cancelled':
          get().updateExecutionStatus('cancelled', raw.message ?? 'Execution cancelled');
          return;
        case 'event':
          if (raw.data) {
            handleEvent(raw.data, raw.timestamp, progressValue);
          }
          return;
        default:
          return;
      }
    };

    const messageListener: EventListener = (event) => {
      try {
    const data = JSON.parse((event as MessageEvent).data) as unknown;

    const envelope = parseEventEnvelope(data);
    if (envelope) {
      handleEnvelopeMessage(envelope);
      return;
    }

    if (data && typeof data === 'object') {
      const nestedEnvelope = parseEventEnvelope((data as Record<string, unknown>).data);
      if (nestedEnvelope) {
        handleEnvelopeMessage(nestedEnvelope);
        return;
      }
    }

    handleLegacyUpdate(data as unknown as ExecutionUpdateMessage);
      } catch (err) {
        logger.error('Failed to parse execution update', { component: 'ExecutionStore', action: 'handleWebSocketMessage', executionId }, err);
      }
    };

    const openListener: EventListener = () => {
      reconnectScheduled = false;
      interruptionLogged = false;
      set({ websocketStatus: 'connected', websocketAttempts: attempt });
    };

    const errorListener: EventListener = (event) => {
      logger.error('WebSocket error', { component: 'ExecutionStore', action: 'handleWebSocketError', executionId }, event);
      scheduleReconnect();
    };

    const closeListener: EventListener = (event) => {
      listeners.forEach((listener, eventType) => {
        socket.removeEventListener(eventType, listener);
      });
      listeners.clear();
      set({ socket: null, socketListeners: new Map() });

      const closeEvent = event as CloseEvent;
      const current = get().currentExecution;
      const isRunning = current && current.id === executionId && current.status === 'running';

      if (!reconnectScheduled && isRunning && !closeEvent.wasClean) {
        scheduleReconnect();
        return;
      }

      if (!reconnectScheduled) {
        set({
          websocketStatus: isRunning ? 'error' : 'idle',
          websocketAttempts: attempt,
        });
      }
    };

    // Track listeners for cleanup
    listeners.set('open', openListener);
    listeners.set('message', messageListener);
    listeners.set('error', errorListener);
    listeners.set('close', closeListener);

    // Add event listeners
    socket.addEventListener('open', openListener);
    socket.addEventListener('message', messageListener);
    socket.addEventListener('error', errorListener);
    socket.addEventListener('close', closeListener);

    set({ socket, socketListeners: listeners });
  },

  disconnectWebSocket: () => {
    const { socket, socketListeners } = get();
    if (socket) {
      // Remove all event listeners before closing to prevent memory leaks
      socketListeners.forEach((listener, eventType) => {
        socket.removeEventListener(eventType, listener);
      });
      socketListeners.clear();

      // Close the socket
      socket.close();

      // Clear state
      set({ socket: null, socketListeners: new Map(), websocketStatus: 'idle', websocketAttempts: 0 });
    }
  },

  addScreenshot: (screenshot: Screenshot) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            screenshots: [...state.currentExecution.screenshots, screenshot],
          }
        : state.currentExecution,
    }));
  },

  addLog: (log: LogEntry) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            logs: [...state.currentExecution.logs, log],
          }
        : state.currentExecution,
    }));
  },

  updateExecutionStatus: (status: Execution['status'], error?: string) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            status,
            error,
            completedAt:
              status === 'completed' || status === 'failed' || status === 'cancelled'
                ? new Date()
                : state.currentExecution.completedAt,
          }
        : state.currentExecution,
    }));
  },

  updateProgress: (progress: number, currentStep?: string) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            progress,
            currentStep: currentStep ?? state.currentExecution.currentStep,
            lastHeartbeat: {
              step: currentStep ?? state.currentExecution.lastHeartbeat?.step,
              elapsedMs: 0,
              timestamp: new Date(),
            },
          }
        : state.currentExecution,
    }));
  },
  recordHeartbeat: (step?: string, elapsedMs?: number) => {
    set((state) => ({
      currentExecution: state.currentExecution
        ? {
            ...state.currentExecution,
            lastHeartbeat: {
              step,
              elapsedMs,
              timestamp: new Date(),
            },
          }
        : state.currentExecution,
    }));
  },

  clearCurrentExecution: () => {
    set({ currentExecution: null });
  },
}));
