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

interface TimelineBoundingBox {
  x?: number;
  y?: number;
  width?: number;
  height?: number;
}

interface TimelineRegion {
  selector?: string;
  bounding_box?: TimelineBoundingBox;
  padding?: number;
  color?: string;
  opacity?: number;
}

interface TimelineRetryHistoryEntry {
  attempt?: number;
  success?: boolean;
  duration_ms?: number;
  call_duration_ms?: number;
  error?: string;
}

interface TimelineScreenshot {
  artifact_id: string;
  url?: string;
  thumbnail_url?: string;
  width?: number;
  height?: number;
  content_type?: string;
  size_bytes?: number;
}

interface TimelineAssertion {
  mode?: string;
  selector?: string;
  expected?: unknown;
  actual?: unknown;
  success?: boolean;
  message?: string;
  negated?: boolean;
  caseSensitive?: boolean;
}

interface TimelineArtifact {
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

interface TimelineFrame {
  step_index: number;
  node_id?: string;
  step_type?: string;
  status?: string;
  success: boolean;
  duration_ms?: number;
  total_duration_ms?: number;
  progress?: number;
  started_at?: string;
  completed_at?: string;
  final_url?: string;
  error?: string;
  console_log_count?: number;
  network_event_count?: number;
  extracted_data_preview?: unknown;
  highlight_regions?: TimelineRegion[];
  mask_regions?: TimelineRegion[];
  focused_element?: { selector?: string; bounding_box?: TimelineBoundingBox };
  element_bounding_box?: TimelineBoundingBox;
  click_position?: { x?: number; y?: number };
  cursor_trail?: Array<{ x?: number; y?: number }>;
  zoom_factor?: number;
  screenshot?: TimelineScreenshot | null;
  artifacts?: TimelineArtifact[];
  assertion?: TimelineAssertion | null;
  retry_attempt?: number;
  retry_max_attempts?: number;
  retry_configured?: number;
  retry_delay_ms?: number;
  retry_backoff_factor?: number;
  retry_history?: TimelineRetryHistoryEntry[];
  dom_snapshot_preview?: string;
  dom_snapshot_artifact_id?: string;
  dom_snapshot?: TimelineArtifact | null;
}

interface Execution {
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
  socket: WebSocket | null;
  socketListeners: Map<string, EventListener>;

  startExecution: (workflowId: string, saveWorkflowFn?: () => Promise<void>) => Promise<void>;
  stopExecution: (executionId: string) => Promise<void>;
  loadExecutions: (workflowId?: string) => Promise<void>;
  loadExecution: (executionId: string) => Promise<void>;
  refreshTimeline: (executionId: string) => Promise<void>;
  connectWebSocket: (executionId: string) => Promise<void>;
  disconnectWebSocket: () => void;
  addScreenshot: (screenshot: Screenshot) => void;
  addLog: (log: LogEntry) => void;
  updateExecutionStatus: (status: Execution['status'], error?: string) => void;
  updateProgress: (progress: number, currentStep?: string) => void;
  recordHeartbeat: (step?: string, elapsedMs?: number) => void;
  clearCurrentExecution: () => void;
}

const normalizeExecution = (raw: any): Execution => {
  const startedAt = raw.started_at ?? raw.startedAt;
  const completedAt = raw.completed_at ?? raw.completedAt;
  return {
    id: raw.id ?? raw.execution_id ?? '',
    workflowId: raw.workflow_id ?? raw.workflowId ?? '',
    status: raw.status ?? 'pending',
    startedAt: startedAt ? new Date(startedAt) : new Date(),
    completedAt: completedAt ? new Date(completedAt) : undefined,
    screenshots: Array.isArray(raw.screenshots) ? raw.screenshots : [],
    timeline: Array.isArray(raw.timeline) ? (raw.timeline as TimelineFrame[]) : [],
    logs: Array.isArray(raw.logs) ? raw.logs : [],
    currentStep: raw.current_step ?? raw.currentStep,
    progress: typeof raw.progress === 'number' ? raw.progress : 0,
    error: raw.error ?? undefined,
    lastHeartbeat: undefined,
  };
};

export const useExecutionStore = create<ExecutionStore>((set, get) => ({
  executions: [],
  currentExecution: null,
  socket: null,
  socketListeners: new Map(),

  startExecution: async (workflowId: string, saveWorkflowFn?: () => Promise<void>) => {
    try {
      // Save workflow first if save function provided (to ensure latest changes are used)
      if (saveWorkflowFn) {
        await saveWorkflowFn();
      }

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
            ? { ...state.currentExecution, status: 'failed' as const }
            : state.currentExecution,
      }));
    } catch (error) {
      logger.error('Failed to stop execution', { component: 'ExecutionStore', action: 'stopExecution', executionId }, error);
    }
  },

  loadExecutions: async (workflowId?: string) => {
    try {
      const config = await getConfig();
      const url = workflowId
        ? `${config.API_URL}/workflows/${workflowId}/executions`
        : `${config.API_URL}/executions`;
      const response = await fetch(url);

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
      set({ currentExecution: execution });
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

      set((state) => ({
        currentExecution:
          state.currentExecution && state.currentExecution.id === executionId
            ? { ...state.currentExecution, timeline: frames }
            : state.currentExecution,
      }));
    } catch (error) {
      logger.error('Failed to load execution timeline', { component: 'ExecutionStore', action: 'refreshTimeline', executionId }, error);
    }
  },

  connectWebSocket: async (executionId: string) => {
    // Cleanup any existing connection first
    get().disconnectWebSocket();

    const config = await getConfig();
    if (!config.WS_URL) {
      logger.error('WebSocket base URL not configured', { component: 'ExecutionStore', action: 'connectWebSocket', executionId });
      return;
    }

    const wsBase = config.WS_URL.endsWith('/') ? config.WS_URL : `${config.WS_URL}/`;
    const wsUrl = new URL('ws', wsBase);
    wsUrl.searchParams.set('execution_id', executionId);

    const socket = new WebSocket(wsUrl.toString());
    const listeners = new Map<string, EventListener>();

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

    const errorListener: EventListener = (event) => {
      logger.error('WebSocket error', { component: 'ExecutionStore', action: 'handleWebSocketError', executionId }, event);
      get().updateExecutionStatus('failed', 'WebSocket error');
    };

    const closeListener: EventListener = () => {
      // Clean up listeners before clearing state
      listeners.forEach((listener, eventType) => {
        socket.removeEventListener(eventType, listener);
      });
      listeners.clear();
      set({ socket: null, socketListeners: new Map() });
    };

    // Track listeners for cleanup
    listeners.set('message', messageListener);
    listeners.set('error', errorListener);
    listeners.set('close', closeListener);

    // Add event listeners
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
      set({ socket: null, socketListeners: new Map() });
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
    get().disconnectWebSocket();
    set({ currentExecution: null });
  },
}));
