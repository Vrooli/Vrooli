import { fromJson } from '@bufbuild/protobuf';
import {
  ExecutionSchema,
  GetScreenshotsResponseSchema,
  type GetScreenshotsResponse as ProtoGetScreenshotsResponse,
} from '@vrooli/proto-types/browser-automation-studio/v1/execution_pb';
import {
  ExecutionTimelineSchema,
  type ExecutionTimeline as ProtoExecutionTimeline,
  type TimelineFrame as ProtoTimelineFrame,
  type TimelineLog as ProtoTimelineLog,
} from '@vrooli/proto-types/browser-automation-studio/v1/timeline_pb';
import { ExecuteWorkflowResponseSchema } from '@vrooli/proto-types/browser-automation-studio/v1/workflow_service_pb';
import { create } from 'zustand';
import { getConfig } from '../config';
import { logger } from '../utils/logger';
import { parseProtoStrict } from '../utils/proto';
import type { ReplayFrame, ReplayPoint, ReplayRegion, ReplayScreenshot } from '../features/execution/ReplayPlayer';
import {
  ExecutionEventsClient,
  type ExecutionUpdateMessage as WsExecutionUpdateMessage,
} from '../features/execution/ws/executionEvents';
import {
  mapArtifactType,
  mapExecutionStatus,
  mapProtoLogLevel,
  mapStepStatus,
  mapStepType,
  timestampToDate,
} from '../features/execution/utils/mappers';
import {
  processExecutionEvent,
  createId,
  parseTimestamp,
  type ExecutionEventHandlers,
  type Screenshot,
  type LogEntry,
} from './executionEventProcessor';
import type { ExecutionEventMessage } from '../features/execution/ws/executionEvents';

const WS_RETRY_LIMIT = 5;
const WS_RETRY_BASE_DELAY_MS = 1500;
let legacyWsWarningSent = false;

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

export interface TimelineBoundingBox {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
}

export interface TimelineRegion {
  selector?: string;
  boundingBox?: TimelineBoundingBox;
  padding?: number;
  color?: string;
  opacity?: number;
}

export interface TimelineRetryHistoryEntry {
  attempt?: number;
  success?: boolean;
  durationMs?: number;
  callDurationMs?: number;
  error?: string;
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
  timestamp?: string;
}

export type TimelineFrame = ReplayFrame & {
  artifacts?: TimelineArtifact[];
  domSnapshotArtifactId?: string;
};

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

const parseExecuteWorkflowResponse = (raw: unknown) =>
  parseProtoStrict(ExecuteWorkflowResponseSchema, raw) as ReturnType<typeof fromJson<typeof ExecuteWorkflowResponseSchema>>;

const parseExecutionProto = (raw: unknown): Execution => {
  const proto = parseProtoStrict(ExecutionSchema, raw) as ReturnType<typeof fromJson<typeof ExecutionSchema>>;
  const startedAt = timestampToDate(proto.startedAt) ?? new Date();
  const completedAt = timestampToDate(proto.completedAt);
  const lastHeartbeat = proto.lastHeartbeat ? timestampToDate(proto.lastHeartbeat) : undefined;

  return {
    id: proto.executionId || '',
    workflowId: proto.workflowId || '',
    status: mapExecutionStatus(proto.status),
    startedAt,
    completedAt: completedAt || undefined,
    screenshots: [],
    timeline: [],
    logs: [],
    currentStep: proto.currentStep || undefined,
    progress: typeof proto.progress === 'number' && proto.progress > 0 ? proto.progress : 0,
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

const mapProtoScreenshot = (shot?: ProtoTimelineFrame['screenshot'] | null): ReplayScreenshot | undefined => {
  if (!shot) return undefined;
  return {
    artifactId: shot.artifactId || createId(),
    url: shot.url,
    thumbnailUrl: shot.thumbnailUrl,
    width: toNumber(shot.width),
    height: toNumber(shot.height),
    contentType: shot.contentType || undefined,
    sizeBytes: toNumber(shot.sizeBytes),
  };
};

const mapProtoRegion = (region?: ProtoTimelineFrame['highlightRegions'][number]): ReplayRegion | undefined => {
  if (!region) return undefined;
  const boundingBox = mapBoundingBoxFromProto(region.boundingBox);
  if (!boundingBox && !region.selector) {
    return undefined;
  }
  return {
    selector: region.selector || undefined,
    boundingBox,
    padding: region.padding ?? undefined,
    color: region.color || undefined,
    opacity: undefined,
  };
};

const mapProtoMaskRegion = (region?: ProtoTimelineFrame['maskRegions'][number]): ReplayRegion | undefined => {
  if (!region) return undefined;
  const boundingBox = mapBoundingBoxFromProto(region.boundingBox);
  if (!boundingBox && !region.selector) {
    return undefined;
  }
  return {
    selector: region.selector || undefined,
    boundingBox,
    opacity: region.opacity ?? undefined,
  };
};

const mapProtoPoint = (point?: ProtoTimelineFrame['cursorTrail'][number]): ReplayPoint | undefined => {
  if (!point) return undefined;
  if (point.x == null || point.y == null) return undefined;
  return { x: point.x, y: point.y };
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

// Exported for targeted unit testing to ensure API↔UI contract stays aligned.
export const mapTimelineFrameFromProto = (frame: ProtoTimelineFrame): TimelineFrame => {
  const screenshot = mapProtoScreenshot(frame.screenshot);
  const focusBox = mapBoundingBoxFromProto(frame.focusedElement?.boundingBox);
  const elementBox = mapBoundingBoxFromProto(frame.elementBoundingBox);
  const artifacts = frame.artifacts?.map((artifact) => mapTimelineArtifactFromProto(artifact)!).filter(Boolean) ?? [];
  const domSnapshotArtifact = artifacts.find((artifact) => artifact?.type === 'dom_snapshot');

  return {
    id: screenshot?.artifactId || `frame-${frame.stepIndex}`,
    stepIndex: frame.stepIndex,
    nodeId: frame.nodeId || undefined,
    stepType: mapStepType(frame.stepType),
    status: mapStepStatus(frame.status),
    success: frame.success,
    durationMs: frame.durationMs ?? undefined,
    totalDurationMs: frame.totalDurationMs ?? undefined,
    progress: frame.progress ?? undefined,
    finalUrl: frame.finalUrl || undefined,
    error: frame.error || undefined,
    extractedDataPreview: frame.extractedDataPreview,
    consoleLogCount: frame.consoleLogCount ?? undefined,
    networkEventCount: frame.networkEventCount ?? undefined,
    screenshot: screenshot ?? undefined,
    highlightRegions: frame.highlightRegions?.map(mapProtoRegion).filter(Boolean) as ReplayRegion[] ?? [],
    maskRegions: frame.maskRegions?.map(mapProtoMaskRegion).filter(Boolean) as ReplayRegion[] ?? [],
    focusedElement: frame.focusedElement
      ? { selector: frame.focusedElement.selector || undefined, boundingBox: focusBox ?? undefined }
      : null,
    elementBoundingBox: elementBox ?? null,
    clickPosition: frame.clickPosition ? { x: frame.clickPosition.x, y: frame.clickPosition.y } : null,
    cursorTrail: frame.cursorTrail?.map(mapProtoPoint).filter(Boolean) as ReplayPoint[] ?? [],
    zoomFactor: frame.zoomFactor ?? undefined,
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
      : undefined,
    retryAttempt: frame.retryAttempt ?? undefined,
    retryMaxAttempts: frame.retryMaxAttempts ?? undefined,
    retryConfigured: frame.retryConfigured ?? undefined,
    retryDelayMs: frame.retryDelayMs ?? undefined,
    retryBackoffFactor: frame.retryBackoffFactor ?? undefined,
    retryHistory: frame.retryHistory?.map((entry) => ({
      attempt: entry.attempt ?? undefined,
      success: entry.success ?? undefined,
      durationMs: entry.durationMs ?? undefined,
      callDurationMs: entry.callDurationMs ?? undefined,
      error: entry.error || undefined,
    })) ?? [],
    domSnapshotPreview: frame.domSnapshotPreview || undefined,
    domSnapshotArtifactId: domSnapshotArtifact?.id || undefined,
    domSnapshotHtml: domSnapshotArtifact?.payload?.html ? String(domSnapshotArtifact.payload.html) : undefined,
    artifacts,
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

const mapScreenshotsFromProto = (raw: unknown): Screenshot[] => {
  const proto = parseProtoStrict(GetScreenshotsResponseSchema, raw) as ProtoGetScreenshotsResponse;
  if (!proto.screenshots || proto.screenshots.length === 0) {
    return [];
  }
  return proto.screenshots
    .map((shot) => {
      const ts = timestampToDate(shot.timestamp) ?? coerceDate(shot.timestamp) ?? parseTimestamp(shot.timestamp as any);
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
      const protoPayload = parseExecuteWorkflowResponse(data);

      const execution: Execution = {
        id: protoPayload.executionId || '',
        workflowId,
        status: mapExecutionStatus(protoPayload.status),
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

      // Strict proto parsing; bubble errors so contract drift is visible.
      // Preserve raw error for UI since the proto contract omits it.
      const { error: rawError, ...timelinePayload } = (data && typeof data === 'object'
        ? (data as Record<string, unknown>)
        : {}) as Record<string, unknown>;

      const protoTimeline = parseProtoStrict(
        ExecutionTimelineSchema,
        timelinePayload
      ) as ProtoExecutionTimeline;

      const frames = (protoTimeline.frames ?? []).map((frame) => mapTimelineFrameFromProto(frame));

      const normalizedLogs = (protoTimeline.logs ?? [])
        .map((log) => mapTimelineLogFromProto(log))
        .filter((entry): entry is LogEntry => Boolean(entry));

      // Extract status with fallback to raw data
      const mappedStatus = mapExecutionStatus(protoTimeline.status);

      // Extract progress with fallback to raw data
      const progressValue = typeof protoTimeline.progress === 'number'
        ? protoTimeline.progress
        : undefined;

      // Extract completedAt with fallback to raw data
      const completedAt = timestampToDate(protoTimeline.completedAt);

      // Extract error from raw data (ExecutionTimeline proto doesn't have error field)
      const errorValue = rawError ? String(rawError) : undefined;

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
          const label = lastFrame?.nodeId ?? lastFrame?.stepType;
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

    const handleLegacyUpdate = (raw: WsExecutionUpdateMessage) => {
      if (!legacyWsWarningSent) {
        logger.warn('Received legacy execution stream payload; please update server to proto envelopes', {
          component: 'ExecutionStore',
          action: 'handleWebSocketMessage',
          executionId,
        });
        legacyWsWarningSent = true;
      }
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

    const eventsClient = new ExecutionEventsClient({
      onEvent: (event, ctx) => handleEvent(event, ctx.fallbackTimestamp, ctx.fallbackProgress),
      onLegacy: handleLegacyUpdate,
      logger,
    });

    const messageListener = eventsClient.createMessageListener();

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
