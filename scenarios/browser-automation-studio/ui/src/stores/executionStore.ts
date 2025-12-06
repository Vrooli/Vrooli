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
  step_type?: string;
  stepType?: string;
  status?: string;
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

const mapExecutionStatus = (status?: string): Execution['status'] => {
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
    case 'cancelled':
      return 'failed';
    default:
      return 'pending';
  }
};

const parseExecutionProto = (raw: unknown): Execution => {
  const proto = fromJson(ExecutionSchema, raw as any, { ignoreUnknownFields: true });
  const startedAt = timestampToDate(proto.startedAt) ?? coerceDate((raw as Record<string, unknown>)?.started_at) ?? new Date();
  const completedAt = timestampToDate(proto.completedAt) ?? coerceDate((raw as Record<string, unknown>)?.completed_at);
  const lastHeartbeat = proto.lastHeartbeat ? timestampToDate(proto.lastHeartbeat) : coerceDate((raw as Record<string, unknown>)?.last_heartbeat);

  return {
    id: proto.id || String((raw as Record<string, unknown>)?.execution_id ?? ''),
    workflowId: proto.workflowId || String((raw as Record<string, unknown>)?.workflow_id ?? ''),
    status: mapExecutionStatus(proto.status),
    startedAt,
    completedAt: completedAt || undefined,
    screenshots: [],
    timeline: [],
    logs: [],
    currentStep: proto.currentStep || undefined,
    progress: typeof proto.progress === 'number' ? proto.progress : 0,
    error: proto.error ?? undefined,
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

const mapTimelineArtifactFromProto = (artifact?: ProtoTimelineFrame['artifacts'][number]) => {
  if (!artifact) return undefined;
  const stepIndex = artifact.stepIndex ?? undefined;
  return {
    id: artifact.id,
    type: artifact.type,
    label: artifact.label || undefined,
    storage_url: artifact.storageUrl || undefined,
    thumbnail_url: artifact.thumbnailUrl || undefined,
    content_type: artifact.contentType || undefined,
    size_bytes: toNumber(artifact.sizeBytes),
    step_index: stepIndex,
    payload: artifact.payload ?? undefined,
  } satisfies TimelineArtifact;
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
    step_type: frame.stepType,
    stepType: frame.stepType,
    status: frame.status,
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
    level: (log.level ?? 'info') as LogEntry['level'],
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
      const ts = coerceDate(shot.timestamp) ?? parseTimestamp(shot.timestamp);
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

const extractProgressFromPayload = (payload: Record<string, unknown> | null | undefined): number | undefined => {
  if (!payload || typeof payload !== 'object') return undefined;
  const candidates = [
    payload.progress,
    payload.progress_percent,
    payload.progressPercent,
  ];
  for (const candidate of candidates) {
    if (typeof candidate === 'number' && Number.isFinite(candidate)) {
      return candidate;
    }
  }
  return undefined;
};

const envelopeToExecutionEvent = (envelope: ExecutionEventEnvelope | null | undefined): ExecutionEventMessage | null => {
  if (!envelope || typeof envelope !== 'object' || typeof envelope.kind !== 'string') {
    return null;
  }
  const payload = (typeof envelope.payload === 'object' && envelope.payload !== null)
    ? (envelope.payload as Record<string, unknown>)
    : undefined;
  const stepIndex = typeof envelope.stepIndex === 'number'
    ? envelope.stepIndex
    : undefined;

  const progress = extractProgressFromPayload(payload);

  const toStringOrEmpty = (value: unknown) => (typeof value === 'string' && value.trim().length > 0 ? value : '');
  const executionId = toStringOrEmpty(envelope.executionId);
  const workflowId = toStringOrEmpty(envelope.workflowId);

  const stepNodeId = (() => {
    if (typeof payload?.node_id === 'string') return payload.node_id;
    if (typeof (payload as Record<string, unknown> | undefined)?.nodeId === 'string') return (payload as Record<string, unknown>).nodeId as string;
    if (typeof payload?.step_node_id === 'string') return payload.step_node_id;
    return undefined;
  })();

  const stepType = (() => {
    if (typeof payload?.step_type === 'string') return payload.step_type;
    if (typeof (payload as Record<string, unknown> | undefined)?.stepType === 'string') return (payload as Record<string, unknown>).stepType as string;
    return undefined;
  })();

  const message = (() => {
    if (typeof payload?.message === 'string') return payload.message;
    return undefined;
  })();

  return {
    type: envelope.kind as ExecutionEventType,
    execution_id: executionId,
    workflow_id: workflowId,
    step_index: stepIndex,
    step_node_id: stepNodeId,
    step_type: stepType,
    progress,
    message,
    payload,
    timestamp: typeof envelope.timestamp === 'string' ? envelope.timestamp : undefined,
  };
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
      let protoTimeline: ProtoExecutionTimeline;
      try {
        protoTimeline = fromJson(ExecutionTimelineSchema, data, { ignoreUnknownFields: true });
      } catch (schemaError) {
        logger.error(
          'Timeline response failed proto validation',
          { component: 'ExecutionStore', action: 'refreshTimeline', executionId },
          schemaError,
        );
        throw schemaError;
      }

      const frames: TimelineFrame[] = (protoTimeline.frames ?? []).map((frame) => mapTimelineFrameFromProto(frame));
      const normalizedLogs: LogEntry[] = (protoTimeline.logs ?? [])
        .map((log) => mapTimelineLogFromProto(log))
        .filter((entry): entry is LogEntry => Boolean(entry));

      const mappedStatus = mapExecutionStatus(protoTimeline.status);
      const progressValue = typeof protoTimeline.progress === 'number' ? protoTimeline.progress : undefined;
      const completedAt = timestampToDate(protoTimeline.completedAt);
      const errorMessage = typeof protoTimeline.status === 'string' && protoTimeline.status.toLowerCase() === 'failed'
        ? protoTimeline.status
        : undefined;

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

        if (errorMessage && errorMessage.length > 0) {
          updated.error = errorMessage;
        }

        if (completedAt && (updated.status === 'completed' || updated.status === 'failed')) {
          updated.completedAt = completedAt;
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
      const progress = extractProgressFromPayload(
        (typeof envelope.payload === 'object' ? envelope.payload : undefined) as Record<string, unknown> | undefined,
      );
      handleEvent(event, envelope.timestamp, progress ?? event.progress);
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
