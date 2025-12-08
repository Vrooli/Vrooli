import { vi } from 'vitest';
import type { Execution } from '@stores/executionStore';

export type MockExecutionStoreState = {
  currentExecution: Execution | null;
  viewerWorkflowId: string | null;
  executions: Execution[];
  socket: WebSocket | null;
  websocketStatus: 'idle' | 'connecting' | 'connected' | 'error';
  connectWebSocket: ReturnType<typeof vi.fn>;
  closeViewer: ReturnType<typeof vi.fn>;
  openViewer: ReturnType<typeof vi.fn>;
  startExecution: ReturnType<typeof vi.fn>;
  stopExecution: ReturnType<typeof vi.fn>;
  refreshTimeline: ReturnType<typeof vi.fn>;
  loadExecution: ReturnType<typeof vi.fn>;
  loadExecutions: ReturnType<typeof vi.fn>;
  addScreenshot: ReturnType<typeof vi.fn>;
  addLog: ReturnType<typeof vi.fn>;
  updateExecutionStatus: ReturnType<typeof vi.fn>;
  updateProgress: ReturnType<typeof vi.fn>;
  recordHeartbeat: ReturnType<typeof vi.fn>;
  clearCurrentExecution: ReturnType<typeof vi.fn>;
};

export const createMockExecutionStoreState = (): MockExecutionStoreState => ({
  currentExecution: null,
  viewerWorkflowId: null,
  executions: [],
  socket: null,
  websocketStatus: 'idle',
  connectWebSocket: vi.fn(),
  closeViewer: vi.fn(),
  openViewer: vi.fn(),
  startExecution: vi.fn(),
  stopExecution: vi.fn(),
  refreshTimeline: vi.fn(),
  loadExecution: vi.fn(),
  loadExecutions: vi.fn(),
  addScreenshot: vi.fn(),
  addLog: vi.fn(),
  updateExecutionStatus: vi.fn(),
  updateProgress: vi.fn(),
  recordHeartbeat: vi.fn(),
  clearCurrentExecution: vi.fn(),
});

export const resetMockExecutionStoreState = (state: MockExecutionStoreState) => {
  state.currentExecution = null;
  state.viewerWorkflowId = null;
  state.executions = [];
  state.socket = null;
  state.websocketStatus = 'idle';
  state.connectWebSocket = vi.fn();
  state.closeViewer = vi.fn();
  state.openViewer = vi.fn();
  state.startExecution = vi.fn();
  state.stopExecution = vi.fn();
  state.refreshTimeline = vi.fn();
  state.loadExecution = vi.fn();
  state.loadExecutions = vi.fn();
  state.addScreenshot = vi.fn();
  state.addLog = vi.fn();
  state.updateExecutionStatus = vi.fn();
  state.updateProgress = vi.fn();
  state.recordHeartbeat = vi.fn();
  state.clearCurrentExecution = vi.fn();
};
