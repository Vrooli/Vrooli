import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { act } from '@testing-library/react';
import { useExecutionStore } from '@stores/executionStore';
import type { Execution } from '@stores/executionStore';

// Mock fetch globally
global.fetch = vi.fn();

// Mock WebSocket
class MockWebSocket {
  url: string;
  onopen: ((event: Event) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  readyState = 0; // CONNECTING

  constructor(url: string) {
    this.url = url;
    // Simulate async connection
    setTimeout(() => {
      this.readyState = 1; // OPEN
      if (this.onopen) {
        this.onopen(new Event('open'));
      }
    }, 0);
  }

  send(_data: string) {
    // Mock send
  }

  close() {
    this.readyState = 3; // CLOSED
    if (this.onclose) {
      this.onclose(new CloseEvent('close'));
    }
  }

  addEventListener(event: string, handler: EventListener) {
    if (event === 'open') this.onopen = handler as (event: Event) => void;
    if (event === 'close') this.onclose = handler as (event: CloseEvent) => void;
    if (event === 'error') this.onerror = handler as (event: Event) => void;
    if (event === 'message') this.onmessage = handler as (event: MessageEvent) => void;
  }

  removeEventListener() {
    // Mock removeEventListener
  }
}

global.WebSocket = MockWebSocket as any;

// Mock getConfig
vi.mock('../config', () => ({
  getConfig: vi.fn().mockResolvedValue({
    API_URL: 'http://localhost:8080/api/v1',
    WS_URL: 'ws://localhost:8081',
  }),
}));

// Mock logger
vi.mock('../utils/logger', () => ({
  logger: {
    info: vi.fn(),
    error: vi.fn(),
    warn: vi.fn(),
    debug: vi.fn(),
  },
}));

// Mock execution event processor
vi.mock('./executionEventProcessor', () => ({
  processExecutionEvent: vi.fn(),
  createId: vi.fn(() => `id-${Math.random().toString(36).substr(2, 9)}`),
  parseTimestamp: vi.fn((ts) => (ts ? new Date(ts) : new Date())),
}));

function createFetchResponse<T>(data: T, ok = true, status = ok ? 200 : 400) {
  return Promise.resolve({
    ok,
    status,
    text: () => Promise.resolve(JSON.stringify(data)),
    json: () => Promise.resolve(data),
  } as Response);
}

describe('executionStore [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset store state
    useExecutionStore.setState({
      executions: [],
      currentExecution: null,
      viewerWorkflowId: null,
      socket: null,
      socketListeners: new Map(),
      websocketStatus: 'idle',
      websocketAttempts: 0,
    });
  });

  afterEach(() => {
    // Clean up any WebSocket connections
    useExecutionStore.getState().disconnectWebSocket();
  });

  describe('Execution Lifecycle', () => {
    it('starts execution successfully [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const mockExecution = {
        execution_id: 'exec-1',
        workflow_id: 'workflow-1',
        status: 'running',
        started_at: '2025-01-01T00:00:00Z',
      };

      (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
        createFetchResponse(mockExecution)
      );

      await act(async () => {
        await useExecutionStore.getState().startExecution('workflow-1');
      });

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/workflows/workflow-1/execute',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ wait_for_completion: false }),
        })
      );

      const state = useExecutionStore.getState();
      expect(state.currentExecution).toBeTruthy();
      expect(state.currentExecution?.id).toBe('exec-1');
      expect(state.currentExecution?.status).toBe('running');
      expect(state.viewerWorkflowId).toBe('workflow-1');
    });

    it('saves workflow before execution if save function provided [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const mockSaveFn = vi.fn().mockResolvedValue(undefined);
      const mockExecution = {
        execution_id: 'exec-2',
        workflow_id: 'workflow-2',
        status: 'running',
      };

      (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
        createFetchResponse(mockExecution)
      );

      await act(async () => {
        await useExecutionStore.getState().startExecution('workflow-2', mockSaveFn);
      });

      expect(mockSaveFn).toHaveBeenCalled();
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/workflows/workflow-2/execute'),
        expect.any(Object)
      );
    });

    it('stops execution successfully [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 50,
      };

      useExecutionStore.setState({ currentExecution: execution });

      (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
        createFetchResponse({ success: true })
      );

      await act(async () => {
        await useExecutionStore.getState().stopExecution('exec-1');
      });

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/executions/exec-1/stop',
        expect.objectContaining({
          method: 'POST',
        })
      );

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.status).toBe('cancelled');
      expect(state.currentExecution?.error).toBe('Execution cancelled by user');
      expect(state.currentExecution?.completedAt).toBeTruthy();
    });

    it('loads executions list [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const mockExecutions = [
        {
          id: 'exec-1',
          workflow_id: 'workflow-1',
          status: 'completed',
          started_at: '2025-01-01',
          completed_at: '2025-01-01',
        },
        {
          id: 'exec-2',
          workflow_id: 'workflow-1',
          status: 'failed',
          started_at: '2025-01-02',
          completed_at: '2025-01-02',
        },
      ];

      (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
        createFetchResponse({ executions: mockExecutions })
      );

      await act(async () => {
        await useExecutionStore.getState().loadExecutions('workflow-1');
      });

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/executions?workflow_id=workflow-1'
      );

      const state = useExecutionStore.getState();
      expect(state.executions).toHaveLength(2);
    });

    it('handles 404 when loading executions [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
        createFetchResponse({}, false, 404)
      );

      await act(async () => {
        await useExecutionStore.getState().loadExecutions('workflow-1');
      });

      const state = useExecutionStore.getState();
      expect(state.executions).toEqual([]);
    });

    it('loads single execution [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const mockExecution = {
        id: 'exec-1',
        workflow_id: 'workflow-1',
        status: 'completed',
        started_at: '2025-01-01',
        completed_at: '2025-01-01',
        progress: 100,
        screenshots: [],
        timeline: [],
        logs: [],
      };

      // Mock both execution and timeline fetch
      (global.fetch as ReturnType<typeof vi.fn>)
        .mockResolvedValueOnce(createFetchResponse(mockExecution))
        .mockResolvedValueOnce(createFetchResponse({ frames: [], logs: [] }));

      await act(async () => {
        await useExecutionStore.getState().loadExecution('exec-1');
      });

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/executions/exec-1'
      );

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.id).toBe('exec-1');
      expect(state.viewerWorkflowId).toBe('workflow-1');
    });
  });

  describe('Timeline and Telemetry [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
    it('refreshes timeline with frames and logs [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 0,
      };

      useExecutionStore.setState({ currentExecution: execution });

      const mockTimeline = {
        frames: [
          {
            step_index: 0,
            node_id: 'node-1',
            step_type: 'navigate',
            status: 'completed',
            success: true,
            duration_ms: 1000,
          },
          {
            step_index: 1,
            node_id: 'node-2',
            step_type: 'click',
            status: 'completed',
            success: true,
            duration_ms: 500,
          },
        ],
        logs: [
          {
            id: 'log-1',
            level: 'info',
            message: 'Navigation started',
            step_name: 'Navigate',
            timestamp: '2025-01-01T00:00:00Z',
          },
        ],
        status: 'running',
        progress: 50,
      };

      (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
        createFetchResponse(mockTimeline)
      );

      await act(async () => {
        await useExecutionStore.getState().refreshTimeline('exec-1');
      });

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:8080/api/v1/executions/exec-1/timeline'
      );

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.timeline).toHaveLength(2);
      expect(state.currentExecution?.logs).toHaveLength(1);
      expect(state.currentExecution?.progress).toBe(50);
      expect(state.currentExecution?.currentStep).toBe('node-2');
    });

    it('merges new logs with existing logs [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [
          {
            id: 'log-1',
            level: 'info',
            message: 'Existing log',
            timestamp: new Date('2025-01-01T00:00:00Z'),
          },
        ],
        progress: 0,
      };

      useExecutionStore.setState({ currentExecution: execution });

      const mockTimeline = {
        frames: [],
        logs: [
          {
            id: 'log-2',
            level: 'warning',
            message: 'New log',
            timestamp: '2025-01-01T00:00:01Z',
          },
        ],
      };

      (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
        createFetchResponse(mockTimeline)
      );

      await act(async () => {
        await useExecutionStore.getState().refreshTimeline('exec-1');
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.logs).toHaveLength(2);
      expect(state.currentExecution?.logs[0].message).toBe('Existing log');
      expect(state.currentExecution?.logs[1].message).toBe('New log');
    });

    it('updates execution status from timeline data [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 50,
      };

      useExecutionStore.setState({ currentExecution: execution });

      const mockTimeline = {
        frames: [],
        logs: [],
        status: 'completed',
        progress: 100,
        completed_at: '2025-01-01T00:05:00Z',
      };

      (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
        createFetchResponse(mockTimeline)
      );

      await act(async () => {
        await useExecutionStore.getState().refreshTimeline('exec-1');
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.status).toBe('completed');
      expect(state.currentExecution?.progress).toBe(100);
      expect(state.currentExecution?.completedAt).toBeTruthy();
    });

    it('handles error status in timeline [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 75,
      };

      useExecutionStore.setState({ currentExecution: execution });

      const mockTimeline = {
        frames: [],
        logs: [],
        status: 'failed',
        error: 'Selector not found: .submit-button',
        completed_at: '2025-01-01T00:03:00Z',
      };

      (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
        createFetchResponse(mockTimeline)
      );

      await act(async () => {
        await useExecutionStore.getState().refreshTimeline('exec-1');
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.status).toBe('failed');
      expect(state.currentExecution?.error).toBe('Selector not found: .submit-button');
    });
  });

  describe('State Management', () => {
    it('updates execution status [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 50,
      };

      useExecutionStore.setState({ currentExecution: execution });

      act(() => {
        useExecutionStore.getState().updateExecutionStatus('completed');
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.status).toBe('completed');
      expect(state.currentExecution?.completedAt).toBeTruthy();
    });

    it('updates execution status with error [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 30,
      };

      useExecutionStore.setState({ currentExecution: execution });

      act(() => {
        useExecutionStore.getState().updateExecutionStatus('failed', 'Network timeout');
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.status).toBe('failed');
      expect(state.currentExecution?.error).toBe('Network timeout');
      expect(state.currentExecution?.completedAt).toBeTruthy();
    });

    it('updates progress and current step [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 0,
      };

      useExecutionStore.setState({ currentExecution: execution });

      act(() => {
        useExecutionStore.getState().updateProgress(75, 'Clicking submit button');
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.progress).toBe(75);
      expect(state.currentExecution?.currentStep).toBe('Clicking submit button');
    });

    it('records heartbeat [REQ:BAS-EXEC-HEARTBEAT-DETECTION]', () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 50,
      };

      useExecutionStore.setState({ currentExecution: execution });

      act(() => {
        useExecutionStore.getState().recordHeartbeat('Waiting for element', 1500);
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.lastHeartbeat).toBeTruthy();
      expect(state.currentExecution?.lastHeartbeat?.step).toBe('Waiting for element');
      expect(state.currentExecution?.lastHeartbeat?.elapsedMs).toBe(1500);
      expect(state.currentExecution?.lastHeartbeat?.timestamp).toBeInstanceOf(Date);
    });

    it('adds screenshot to execution [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 25,
      };

      useExecutionStore.setState({ currentExecution: execution });

      const screenshot = {
        id: 'screenshot-1',
        url: 'https://storage/exec-1/screenshot-1.png',
        timestamp: new Date(),
        stepIndex: 2,
      };

      act(() => {
        useExecutionStore.getState().addScreenshot(screenshot);
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.screenshots).toHaveLength(1);
      expect(state.currentExecution?.screenshots[0].id).toBe('screenshot-1');
    });

    it('adds log to execution [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 10,
      };

      useExecutionStore.setState({ currentExecution: execution });

      const logEntry = {
        id: 'log-1',
        level: 'warning' as const,
        message: 'Retry attempt 2/3',
        timestamp: new Date(),
      };

      act(() => {
        useExecutionStore.getState().addLog(logEntry);
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution?.logs).toHaveLength(1);
      expect(state.currentExecution?.logs[0].level).toBe('warning');
      expect(state.currentExecution?.logs[0].message).toBe('Retry attempt 2/3');
    });

    it('clears current execution [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'completed',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 100,
      };

      useExecutionStore.setState({ currentExecution: execution });

      act(() => {
        useExecutionStore.getState().clearCurrentExecution();
      });

      const state = useExecutionStore.getState();
      expect(state.currentExecution).toBeNull();
    });
  });

  describe('Viewer Management', () => {
    it('opens viewer for workflow [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
      act(() => {
        useExecutionStore.getState().openViewer('workflow-1');
      });

      const state = useExecutionStore.getState();
      expect(state.viewerWorkflowId).toBe('workflow-1');
    });

    it('switches viewer to different workflow [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 50,
      };

      useExecutionStore.setState({
        currentExecution: execution,
        viewerWorkflowId: 'workflow-1',
      });

      act(() => {
        useExecutionStore.getState().openViewer('workflow-2');
      });

      const state = useExecutionStore.getState();
      expect(state.viewerWorkflowId).toBe('workflow-2');
      expect(state.currentExecution).toBeNull();
    });

    it('closes viewer and disconnects WebSocket [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 75,
      };

      useExecutionStore.setState({
        currentExecution: execution,
        viewerWorkflowId: 'workflow-1',
      });

      act(() => {
        useExecutionStore.getState().closeViewer();
      });

      const state = useExecutionStore.getState();
      expect(state.viewerWorkflowId).toBeNull();
      expect(state.currentExecution).toBeNull();
    });
  });

  describe('WebSocket Connection [REQ:BAS-EXEC-TELEMETRY-STREAM]', () => {
    it('connects to WebSocket for execution telemetry [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 0,
      };

      useExecutionStore.setState({ currentExecution: execution });

      await act(async () => {
        await useExecutionStore.getState().connectWebSocket('exec-1');
        await new Promise((resolve) => setTimeout(resolve, 0));
      });

      const state = useExecutionStore.getState();
      expect(state.socket).toBeTruthy();
      expect(state.websocketStatus).toBe('connected');
    });

    it('disconnects WebSocket [REQ:BAS-EXEC-TELEMETRY-STREAM]', async () => {
      const execution: Execution = {
        id: 'exec-1',
        workflowId: 'workflow-1',
        status: 'running',
        startedAt: new Date(),
        screenshots: [],
        timeline: [],
        logs: [],
        progress: 0,
      };

      useExecutionStore.setState({ currentExecution: execution });

      await act(async () => {
        await useExecutionStore.getState().connectWebSocket('exec-1');
      });

      act(() => {
        useExecutionStore.getState().disconnectWebSocket();
      });

      const state = useExecutionStore.getState();
      expect(state.socket).toBeNull();
      expect(state.websocketStatus).toBe('idle');
    });
  });
});
