import { create } from 'zustand';
import { getConfig } from '../config';
import { logger } from '../utils/logger';
import {
  processExecutionEvent,
  createId,
  parseTimestamp,
  type ExecutionEventMessage,
  type ExecutionEventHandlers,
  type Screenshot,
  type LogEntry,
} from './executionEventProcessor';

const WS_RETRY_LIMIT = 5;
const WS_RETRY_BASE_DELAY_MS = 1500;

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
  status: 'pending' | 'running' | 'completed' | 'failed';
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

const normalizeExecution = (raw: unknown): Execution => {
  const rawData = raw as Record<string, unknown>;
  const startedAt = rawData.started_at ?? rawData.startedAt;
  const completedAt = rawData.completed_at ?? rawData.completedAt;
  const statusValue = rawData.status;
  const currentStepValue = rawData.current_step ?? rawData.currentStep;
  const errorValue = rawData.error;

  return {
    id: String(rawData.id ?? rawData.execution_id ?? ''),
    workflowId: String(rawData.workflow_id ?? rawData.workflowId ?? ''),
    status: (typeof statusValue === 'string' && ['pending', 'running', 'completed', 'failed'].includes(statusValue))
      ? statusValue as 'pending' | 'running' | 'completed' | 'failed'
      : 'pending',
    startedAt: (typeof startedAt === 'string' || typeof startedAt === 'number' || startedAt instanceof Date) ? new Date(startedAt) : new Date(),
    completedAt: (typeof completedAt === 'string' || typeof completedAt === 'number' || completedAt instanceof Date) ? new Date(completedAt) : undefined,
    screenshots: Array.isArray(rawData.screenshots) ? rawData.screenshots : [],
    timeline: Array.isArray(rawData.timeline) ? (rawData.timeline as TimelineFrame[]) : [],
    logs: Array.isArray(rawData.logs) ? rawData.logs : [],
    currentStep: typeof currentStepValue === 'string' ? currentStepValue : undefined,
    progress: typeof rawData.progress === 'number' ? rawData.progress : 0,
    error: typeof errorValue === 'string' ? errorValue : undefined,
    lastHeartbeat: undefined,
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
                status: 'failed' as const,
                error: 'Execution stopped by user',
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
      set({ executions: data.executions });
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
      const execution = normalizeExecution(data);
      set({ currentExecution: execution, viewerWorkflowId: execution.workflowId });
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
      const frames = Array.isArray(data?.frames)
        ? (data.frames as TimelineFrame[])
        : [];
      const rawLogs = Array.isArray(data?.logs)
        ? (data.logs as TimelineLog[])
        : [];

      const normalizeLogLevel = (value?: string): LogEntry['level'] => {
        switch ((value ?? '').toLowerCase()) {
          case 'error':
          case 'err':
            return 'error';
          case 'warning':
          case 'warn':
            return 'warning';
          case 'success':
          case 'ok':
          case 'passed':
            return 'success';
          default:
            return 'info';
        }
      };

      const normalizedLogs: LogEntry[] = rawLogs
        .map((log) => {
          const baseMessage = typeof log?.message === 'string' ? log.message.trim() : '';
          const stepName = typeof log?.step_name === 'string' ? log.step_name : typeof log?.stepName === 'string' ? log.stepName : '';
          const composedMessage = stepName && baseMessage
            ? `${stepName}: ${baseMessage}`
            : baseMessage || stepName;
          if (!composedMessage) {
            return null;
          }
          const rawTimestamp = typeof log?.timestamp === 'string' || typeof log?.timestamp === 'number'
            ? String(log.timestamp)
            : '';
          const timestamp = parseTimestamp(log?.timestamp);
          const fallbackId = `${stepName}|${composedMessage}|${rawTimestamp}`;
          const id = typeof log?.id === 'string' && log.id.trim().length > 0
            ? log.id.trim()
            : fallbackId || createId();
          return {
            id,
            level: normalizeLogLevel(log?.level),
            message: composedMessage,
            timestamp,
          } satisfies LogEntry;
        })
        .filter((entry): entry is LogEntry => Boolean(entry));

      const normalizeStatus = (value?: string): Execution['status'] | undefined => {
        switch ((value ?? '').toLowerCase()) {
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
            return undefined;
        }
      };

      const mappedStatus = normalizeStatus(data?.status);
      const progressValue = typeof data?.progress === 'number' ? data.progress : undefined;
      const completedAtRaw = data?.completed_at ?? data?.completedAt;
      const errorMessage = typeof data?.error === 'string' ? data.error : undefined;

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

        if (errorMessage && errorMessage.trim().length > 0) {
          updated.error = errorMessage.trim();
        }

        if (completedAtRaw && (updated.status === 'completed' || updated.status === 'failed')) {
          const completedDate = new Date(completedAtRaw);
          if (!Number.isNaN(completedDate.valueOf())) {
            updated.completedAt = completedDate;
          }
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
    let wsUrl: URL;
    try {
      wsUrl = new URL('ws', wsBase);
    } catch (err) {
      logger.error('Failed to construct WebSocket URL', { component: 'ExecutionStore', action: 'connectWebSocket', executionId, wsBase }, err);
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

    const handleUpdate = (raw: ExecutionUpdateMessage) => {
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
          get().updateExecutionStatus('failed', raw.message ?? 'Execution cancelled');
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
        const data = JSON.parse((event as MessageEvent).data) as ExecutionUpdateMessage;
        handleUpdate(data);
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
            completedAt: status === 'completed' || status === 'failed' ? new Date() : state.currentExecution.completedAt,
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
