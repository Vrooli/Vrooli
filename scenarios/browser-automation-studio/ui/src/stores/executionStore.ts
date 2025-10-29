import { create } from 'zustand';
import axios from 'axios';
import { getConfig } from '../config';
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

  startExecution: (workflowId: string) => Promise<void>;
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

  startExecution: async (workflowId: string) => {
    try {
      const config = await getConfig();
      const response = await axios.post(`${config.API_URL}/workflows/${workflowId}/execute`, {
        wait_for_completion: false,
      });

      const execution: Execution = {
        id: response.data.execution_id,
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
      console.error('Failed to start execution:', error);
      throw error;
    }
  },

  stopExecution: async (executionId: string) => {
    try {
      const config = await getConfig();
      await axios.post(`${config.API_URL}/executions/${executionId}/stop`);
      get().disconnectWebSocket();
      set((state) => ({
        currentExecution:
          state.currentExecution?.id === executionId
            ? { ...state.currentExecution, status: 'failed' as const }
            : state.currentExecution,
      }));
    } catch (error) {
      console.error('Failed to stop execution:', error);
    }
  },

  loadExecutions: async (workflowId?: string) => {
    try {
      const config = await getConfig();
      const url = workflowId
        ? `${config.API_URL}/workflows/${workflowId}/executions`
        : `${config.API_URL}/executions`;
      const response = await axios.get(url);
      set({ executions: response.data.executions });
    } catch (error) {
      console.error('Failed to load executions:', error);
    }
  },

  loadExecution: async (executionId: string) => {
    try {
      const config = await getConfig();
      const response = await axios.get(`${config.API_URL}/executions/${executionId}`);
      const execution = normalizeExecution(response.data);
      set({ currentExecution: execution });
      void get().refreshTimeline(executionId);
    } catch (error) {
      console.error('Failed to load execution:', error);
    }
  },

  refreshTimeline: async (executionId: string) => {
    try {
      const config = await getConfig();
      const response = await axios.get(`${config.API_URL}/executions/${executionId}/timeline`);
      const frames = Array.isArray(response.data?.frames)
        ? (response.data.frames as TimelineFrame[])
        : [];

      set((state) => ({
        currentExecution:
          state.currentExecution && state.currentExecution.id === executionId
            ? { ...state.currentExecution, timeline: frames }
            : state.currentExecution,
      }));
    } catch (error) {
      console.error('Failed to load execution timeline:', error);
    }
  },

  connectWebSocket: async (executionId: string) => {
    const config = await getConfig();
    const wsUrl = new URL('/ws', config.WS_URL);
    wsUrl.searchParams.set('execution_id', executionId);

    const socket = new WebSocket(wsUrl.toString());

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

    socket.addEventListener('message', (event) => {
      try {
        const data = JSON.parse(event.data) as ExecutionUpdateMessage;
        handleUpdate(data);
      } catch (err) {
        console.error('Failed to parse execution update', err);
      }
    });

    socket.addEventListener('error', (event) => {
      console.error('WebSocket error', event);
      get().updateExecutionStatus('failed', 'WebSocket error');
    });

    socket.addEventListener('close', () => {
      set({ socket: null });
    });

    set({ socket });
  },

  disconnectWebSocket: () => {
    const { socket } = get();
    if (socket) {
      socket.close();
      set({ socket: null });
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
}));
