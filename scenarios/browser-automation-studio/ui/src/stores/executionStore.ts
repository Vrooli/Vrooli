import { create } from 'zustand';
import axios from 'axios';
import { io, Socket } from 'socket.io-client';
import { getConfig } from '../config';

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
  socket: Socket | null;
  
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

export const useExecutionStore = create<ExecutionStore>((set, get) => ({
  executions: [],
  currentExecution: null,
  socket: null,
  
  startExecution: async (workflowId: string) => {
    try {
      const config = await getConfig();
      const response = await axios.post(`${config.API_URL}/workflows/${workflowId}/execute`, {
        wait_for_completion: false
      });
      
      const execution: Execution = {
        id: response.data.execution_id,
        workflowId,
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        logs: [],
        progress: 0
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
        currentExecution: state.currentExecution?.id === executionId 
          ? { ...state.currentExecution, status: 'failed' as const }
          : state.currentExecution
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
    const socket = io(config.WS_URL, {
      query: { executionId }
    });
    
    socket.on('screenshot', (screenshot: Screenshot) => {
      get().addScreenshot(screenshot);
    });
    
    socket.on('log', (log: LogEntry) => {
      get().addLog(log);
    });
    
    socket.on('progress', ({ progress, currentStep }) => {
      get().updateProgress(progress, currentStep);
    });
    
    socket.on('status', ({ status, error }) => {
      get().updateExecutionStatus(status, error);
    });
    
    socket.on('complete', (_result) => {
      get().updateExecutionStatus('completed');
      get().disconnectWebSocket();
    });
    
    socket.on('error', (error) => {
      get().updateExecutionStatus('failed', error.message);
      get().disconnectWebSocket();
    });
    
    set({ socket });
  },
  
  disconnectWebSocket: () => {
    const { socket } = get();
    if (socket) {
      socket.disconnect();
      set({ socket: null });
    }
  },
  
  addScreenshot: (screenshot: Screenshot) => {
    set((state) => ({
      currentExecution: state.currentExecution 
        ? {
            ...state.currentExecution,
            screenshots: [...state.currentExecution.screenshots, screenshot]
          }
        : state.currentExecution
    }));
  },
  
  addLog: (log: LogEntry) => {
    set((state) => ({
      currentExecution: state.currentExecution 
        ? {
            ...state.currentExecution,
            logs: [...state.currentExecution.logs, log]
          }
        : state.currentExecution
    }));
  },
  
  updateExecutionStatus: (status: Execution['status'], error?: string) => {
    set((state) => ({
      currentExecution: state.currentExecution 
        ? {
            ...state.currentExecution,
            status,
            error,
            completedAt: status === 'completed' || status === 'failed' ? new Date() : undefined
          }
        : state.currentExecution
    }));
  },
  
  updateProgress: (progress: number, currentStep?: string) => {
    set((state) => ({
      currentExecution: state.currentExecution 
        ? {
            ...state.currentExecution,
            progress,
            currentStep
          }
        : state.currentExecution
    }));
  }
}));