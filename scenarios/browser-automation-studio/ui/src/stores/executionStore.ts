import { create } from 'zustand';
import axios from 'axios';
import { getConfig } from '../config';

type ExecutionEventType =
  | 'execution.started'
  | 'execution.progress'
  | 'execution.completed'
  | 'execution.failed'
  | 'execution.cancelled'
  | 'step.started'
  | 'step.completed'
  | 'step.failed'
  | 'step.screenshot'
  | 'step.log'
  | 'step.telemetry';

interface ExecutionEventMessage {
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

interface Screenshot {
  id: string;
  timestamp: Date;
  url: string;
  stepName: string;
}

interface LogEntry {
  id: string;
  timestamp: Date;
  level: 'info' | 'warning' | 'error' | 'success';
  message: string;
}

interface Execution {
  id: string;
  workflowId: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  startedAt: Date;
  completedAt?: Date;
  screenshots: Screenshot[];
  logs: LogEntry[];
  currentStep?: string;
  progress: number;
  error?: string;
}

interface ExecutionStore {
  executions: Execution[];
  currentExecution: Execution | null;
  socket: WebSocket | null;

  startExecution: (workflowId: string) => Promise<void>;
  stopExecution: (executionId: string) => Promise<void>;
  loadExecutions: (workflowId?: string) => Promise<void>;
  loadExecution: (executionId: string) => Promise<void>;
  connectWebSocket: (executionId: string) => Promise<void>;
  disconnectWebSocket: () => void;
  addScreenshot: (screenshot: Screenshot) => void;
  addLog: (log: LogEntry) => void;
  updateExecutionStatus: (status: Execution['status'], error?: string) => void;
  updateProgress: (progress: number, currentStep?: string) => void;
}

const createId = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return Math.random().toString(36).slice(2);
};

const parseTimestamp = (timestamp?: string) => (timestamp ? new Date(timestamp) : new Date());

const stepLabel = (event: ExecutionEventMessage) =>
  event.step_node_id || event.step_type || (typeof event.step_index === 'number' ? `Step ${event.step_index + 1}` : 'Step');

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
        logs: [],
        progress: 0,
      };

      set({ currentExecution: execution });
      await get().connectWebSocket(execution.id);
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
      set({ currentExecution: response.data });
    } catch (error) {
      console.error('Failed to load execution:', error);
    }
  },

  connectWebSocket: async (executionId: string) => {
    const config = await getConfig();
    const wsUrl = new URL('/ws', config.WS_URL);
    wsUrl.searchParams.set('execution_id', executionId);

    const socket = new WebSocket(wsUrl.toString());

    const handleEvent = (event: ExecutionEventMessage, fallbackTimestamp?: string, fallbackProgress?: number) => {
      const eventTimestamp = event.timestamp ?? fallbackTimestamp;
      if (typeof event.progress === 'number') {
        get().updateProgress(event.progress, event.step_node_id ?? event.step_type);
      } else if (typeof fallbackProgress === 'number') {
        get().updateProgress(fallbackProgress, event.step_node_id ?? event.step_type);
      }

      switch (event.type) {
        case 'execution.started':
          get().updateExecutionStatus('running');
          return;
        case 'execution.completed':
          get().updateExecutionStatus('completed');
          return;
        case 'execution.failed':
          get().updateExecutionStatus('failed', event.message);
          return;
        case 'execution.cancelled':
          get().updateExecutionStatus('failed', event.message ?? 'Execution cancelled');
          return;
        case 'execution.progress':
          if (typeof event.progress === 'number') {
            get().updateProgress(event.progress, event.step_node_id ?? event.step_type);
          }
          return;
        case 'step.started':
          if (typeof event.progress === 'number') {
            get().updateProgress(event.progress, event.step_node_id ?? event.step_type);
          }
          get().addLog({
            id: createId(),
            level: 'info',
            message: event.message ?? `${stepLabel(event)} started`,
            timestamp: parseTimestamp(eventTimestamp),
          });
          return;
        case 'step.completed':
        case 'step.failed': {
          const message = event.message ?? `${stepLabel(event)} ${event.type === 'step.completed' ? 'completed' : 'failed'}`;
          get().addLog({
            id: createId(),
            level: event.type === 'step.failed' ? 'error' : 'success',
            message,
            timestamp: parseTimestamp(eventTimestamp),
          });

          if (event.type === 'step.failed') {
            get().updateExecutionStatus('failed', message);
          }
          return;
        }
        case 'step.screenshot': {
          const payload = event.payload ?? {};
          const url = (payload.url as string | undefined) ??
            ((payload.base64 as string | undefined) ? `data:image/png;base64,${payload.base64 as string}` : undefined);
          if (!url) {
            return;
          }

          const screenshot: Screenshot = {
            id: (payload.screenshot_id as string | undefined) ?? createId(),
            url,
            stepName: stepLabel(event),
            timestamp: parseTimestamp((payload.timestamp as string | undefined) ?? eventTimestamp),
          };

          get().addScreenshot(screenshot);
          return;
        }
        case 'step.log': {
          const payload = event.payload ?? {};
          const message = (payload.message as string | undefined) ?? event.message ?? 'Step log';
          const level = (payload.level as LogEntry['level'] | undefined) ?? 'info';
          get().addLog({
            id: createId(),
            level,
            message,
            timestamp: parseTimestamp(eventTimestamp),
          });
          return;
        }
        default:
          return;
      }
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
}));
