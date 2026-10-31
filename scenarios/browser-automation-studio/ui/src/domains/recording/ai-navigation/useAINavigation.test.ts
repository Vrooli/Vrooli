import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useWebSocketMessage } from '@/contexts/WebSocketContext';
import { recordingApi } from '../api';
import { useAINavigation } from './useAINavigation';

vi.mock('@/contexts/WebSocketContext', () => ({ useWebSocketMessage: vi.fn() }));
vi.mock('@/utils/apiHeaders', () => ({ getAIRequestHeadersSync: () => ({}) }));
vi.mock('../api', () => ({
  recordingApi: {
    startAINavigation: vi.fn(),
    getAINavigationStatus: vi.fn(),
    abortAINavigation: vi.fn(),
    resumeAINavigation: vi.fn(),
  },
}));

const terminalStatus = (navigationId: string) => ({
  navigationId,
  status: 'completed',
  terminal: true,
  steps: [{ index: 1, actionType: 'navigate', url: 'https://example.com', description: 'done', success: true, at: { seconds: 1700000000n, nanos: 0 } }],
  totalTokens: 1,
  totalDurationMs: 10n,
  finalUrl: 'https://example.com',
  verifiedSuccess: true,
  extractedData: {},
});

function mockSuccessfulStart(navigationId: string): void {
  vi.mocked(recordingApi.startAINavigation).mockResolvedValueOnce({
    success: true,
    data: { navigationId },
  } as never);
}

function keepStatusObservationPending(): void {
  vi.mocked(recordingApi.getAINavigationStatus).mockImplementationOnce(() => new Promise(() => {}));
}

function emitHumanHandoff(navigationId: string, reason: string): (message: unknown) => void {
  const callback = vi.mocked(useWebSocketMessage).mock.calls[0][0] as (message: unknown) => void;
  act(() => callback({
    type: 'ai_navigation_awaiting_human',
    navigationId,
    sessionId: 'session-1',
    reason,
    timestamp: '2026-09-26T22:40:00.000Z',
  }));
  return callback;
}

describe('useAINavigation recovery [REQ:BAS-RH-J24]', () => {
  beforeEach(() => vi.clearAllMocks());

  it('publishes the accepted navigation identity before status observation', async () => {
    const order: string[] = [];
    mockSuccessfulStart('nav-ordered');
    vi.mocked(recordingApi.getAINavigationStatus).mockImplementationOnce(() => {
      order.push('observe');
      return Promise.resolve({ success: true, data: terminalStatus('nav-ordered') } as never);
    });
    const onStarted = vi.fn(() => order.push('started'));
    const onComplete = vi.fn();
    const { result } = renderHook(() => useAINavigation({
      sessionId: 'session-1',
      onStarted,
      onComplete,
    }));

    await act(async () => { await result.current.startNavigation('ordered start', 'local_first'); });
    await waitFor(() => expect(onComplete).toHaveBeenCalled());

    expect(onStarted).toHaveBeenCalledWith('nav-ordered', 2);
    expect(order).toEqual(['started', 'observe']);
  });

  it('rejects a second start while the first request is in flight', async () => {
    let resolveStart: ((value: unknown) => void) | undefined;
    keepStatusObservationPending();
    vi.mocked(recordingApi.startAINavigation).mockImplementationOnce(() => new Promise((resolve) => {
      resolveStart = resolve;
    }) as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    let firstStart: Promise<string | null> | undefined;
    act(() => {
      firstStart = result.current.startNavigation('first', 'local_first');
    });
    let secondStart: string | null | undefined;
    await act(async () => {
      secondStart = await result.current.startNavigation('second', 'local_first');
    });

    expect(secondStart).toBeNull();
    expect(recordingApi.startAINavigation).toHaveBeenCalledTimes(1);
    resolveStart?.({ success: true, data: { navigationId: 'nav-first' } });
    await act(async () => { await firstStart; });
  });

  it('does not let an aborted start overwrite reset state', async () => {
    let rejectStart: ((reason?: unknown) => void) | undefined;
    vi.mocked(recordingApi.startAINavigation).mockImplementationOnce(() => new Promise((_resolve, reject) => {
      rejectStart = reject;
    }) as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    let firstStart: Promise<string | null> | undefined;
    act(() => {
      firstStart = result.current.startNavigation('first', 'local_first');
    });
    act(() => result.current.reset());
    rejectStart?.(new DOMException('The operation was aborted', 'AbortError'));
    await act(async () => {
      await firstStart?.catch(() => undefined);
    });

    expect(result.current.state.status).toBe('idle');
    expect(result.current.state.error).toBeNull();
    expect(result.current.state.isNavigating).toBe(false);
  });

  it('keeps navigation recoverable when status observation fails', async () => {
    mockSuccessfulStart('nav-status-failure');
    vi.mocked(recordingApi.getAINavigationStatus).mockResolvedValueOnce({
      success: false,
      error: 'status unavailable',
    } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('observe', 'local_first'); });
    await waitFor(() => expect(result.current.state.status).toBe('observation_unavailable'));

    expect(result.current.state.isNavigating).toBe(true);
    expect(result.current.state.navigationId).toBe('nav-status-failure');
    expect(result.current.state.error).toBe('status unavailable');
  });

  it('keeps navigation recoverable when status observation throws', async () => {
    mockSuccessfulStart('nav-status-throw');
    vi.mocked(recordingApi.getAINavigationStatus).mockRejectedValueOnce(new Error('socket closed'));
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('observe thrown error', 'local_first'); });
    await waitFor(() => expect(result.current.state.status).toBe('observation_unavailable'));

    expect(result.current.state.isNavigating).toBe(true);
    expect(result.current.state.navigationId).toBe('nav-status-throw');
    expect(result.current.state.error).toBe('socket closed');
  });

  it('restores active status when a live step proves observation recovered', async () => {
    mockSuccessfulStart('nav-live-recovered');
    vi.mocked(recordingApi.getAINavigationStatus).mockResolvedValueOnce({
      success: false,
      error: 'status unavailable',
    } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('live recovery', 'local_first'); });
    await waitFor(() => expect(result.current.state.status).toBe('observation_unavailable'));

    const websocketCallback = vi.mocked(useWebSocketMessage).mock.calls[0][0] as (message: unknown) => void;
    act(() => websocketCallback({
      type: 'ai_navigation_step',
      navigationId: 'nav-live-recovered',
      sessionId: 'session-1',
      stepNumber: 1,
      action: { type: 'navigate', url: 'https://example.com' },
      reasoning: 'live event restored the stream',
      currentUrl: 'https://example.com',
      goalAchieved: false,
      tokensUsed: { promptTokens: 1, completionTokens: 1, totalTokens: 2 },
      durationMs: 5,
      timestamp: '2026-09-29T00:00:00.000Z',
    }));

    expect(result.current.state.status).toBe('navigating');
    expect(result.current.state.isNavigating).toBe(true);
    expect(result.current.state.error).toBeNull();
    expect(result.current.state.steps).toHaveLength(1);
  });

  it('allows an observed navigation to be stopped after the status connection is lost', async () => {
    mockSuccessfulStart('nav-observation-abort');
    vi.mocked(recordingApi.getAINavigationStatus)
      .mockResolvedValueOnce({ success: false, error: 'status unavailable' } as never)
      .mockResolvedValueOnce({
        success: true,
        data: { ...terminalStatus('nav-observation-abort'), status: 'aborted', terminal: true },
      } as never);
    vi.mocked(recordingApi.abortAINavigation).mockResolvedValueOnce({ success: true } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('abort after observation loss', 'local_first'); });
    await waitFor(() => expect(result.current.state.status).toBe('observation_unavailable'));

    await act(async () => { await result.current.abortNavigation(); });
    await waitFor(() => expect(result.current.state.status).toBe('aborted'));
    expect(result.current.state.navigationId).toBeNull();
    expect(recordingApi.abortAINavigation).toHaveBeenCalledWith('nav-observation-abort', expect.anything());
  });

  it('admits a successor after observation-loss recovery without stale completion', async () => {
    vi.mocked(recordingApi.startAINavigation)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-recovered' } } as never)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-successor' } } as never);
    vi.mocked(recordingApi.getAINavigationStatus)
      .mockResolvedValueOnce({ success: false, error: 'status unavailable' } as never)
      .mockResolvedValueOnce({
        success: true,
        data: { ...terminalStatus('nav-recovered'), status: 'aborted', terminal: true },
      } as never)
      .mockImplementationOnce(() => new Promise(() => {}));
    vi.mocked(recordingApi.abortAINavigation).mockResolvedValueOnce({ success: true } as never);
    const onComplete = vi.fn();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1', onComplete }));

    await act(async () => { await result.current.startNavigation('recover then continue', 'local_first'); });
    await waitFor(() => expect(result.current.state.status).toBe('observation_unavailable'));

    await act(async () => { await result.current.abortNavigation(); });
    await waitFor(() => expect(result.current.state.status).toBe('aborted'));
    expect(onComplete).toHaveBeenCalledTimes(1);
    expect(onComplete).toHaveBeenCalledWith('aborted', undefined, 'nav-recovered');

    await act(async () => {
      await expect(result.current.startNavigation('successor', 'local_first')).resolves.toBe('nav-successor');
    });
    expect(result.current.state.navigationId).toBe('nav-successor');
    expect(result.current.state.isNavigating).toBe(true);
    expect(onComplete).toHaveBeenCalledTimes(1);
  });

  it('fails closed when status recovery returns an unknown status', async () => {
    mockSuccessfulStart('nav-invalid-status');
    vi.mocked(recordingApi.getAINavigationStatus).mockResolvedValueOnce({
      success: true,
      data: { ...terminalStatus('nav-invalid-status'), status: 'future_status' },
    } as never);
    const onComplete = vi.fn();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1', onComplete }));

    await act(async () => { await result.current.startNavigation('invalid status', 'local_first'); });
    await waitFor(() => expect(result.current.state.status).toBe('observation_unavailable'));

    expect(result.current.state.isNavigating).toBe(true);
    expect(result.current.state.navigationId).toBe('nav-invalid-status');
    expect(result.current.state.error).toBe('Received invalid navigation status');
    expect(onComplete).not.toHaveBeenCalled();
  });

  it('ignores completion events with unknown statuses', async () => {
    mockSuccessfulStart('nav-invalid-complete');
    keepStatusObservationPending();
    const onComplete = vi.fn();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1', onComplete }));

    await act(async () => { await result.current.startNavigation('invalid completion', 'local_first'); });
    const websocketCallback = vi.mocked(useWebSocketMessage).mock.calls[0][0] as (message: unknown) => void;
    act(() => websocketCallback({
      type: 'ai_navigation_complete',
      navigationId: 'nav-invalid-complete',
      sessionId: 'session-1',
      status: 'future_status',
      totalSteps: 1,
      totalTokens: 1,
      totalDurationMs: 10,
      finalUrl: 'https://example.com',
    }));

    expect(result.current.state.isNavigating).toBe(true);
    expect(result.current.state.status).toBe('navigating');
    expect(onComplete).not.toHaveBeenCalled();
  });

  it('ignores same-navigation events from a different session', async () => {
    mockSuccessfulStart('nav-session-fence');
    keepStatusObservationPending();
    const onComplete = vi.fn();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1', onComplete }));

    await act(async () => { await result.current.startNavigation('session fence', 'local_first'); });
    const websocketCallback = vi.mocked(useWebSocketMessage).mock.calls[0][0] as (message: unknown) => void;
    act(() => websocketCallback({
      type: 'ai_navigation_complete',
      navigationId: 'nav-session-fence',
      sessionId: 'session-2',
      status: 'completed',
      totalSteps: 1,
      totalTokens: 1,
      totalDurationMs: 10,
      finalUrl: 'https://other-session.example.com',
    }));

    expect(result.current.state.status).toBe('navigating');
    expect(result.current.state.isNavigating).toBe(true);
    expect(onComplete).not.toHaveBeenCalled();

    act(() => websocketCallback({
      type: 'ai_navigation_complete',
      navigationId: 'nav-session-fence',
      sessionId: 'session-1',
      status: 'completed',
      totalSteps: 1,
      totalTokens: 1,
      totalDurationMs: 10,
      finalUrl: 'https://current-session.example.com',
    }));
    expect(result.current.state.status).toBe('completed');
    expect(onComplete).toHaveBeenCalledTimes(1);
  });

  it('admits a successor from a stale start callback after terminal retirement', async () => {
    vi.mocked(recordingApi.startAINavigation)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-terminal' } } as never)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-successor' } } as never);
    vi.mocked(recordingApi.getAINavigationStatus).mockImplementation(() => new Promise(() => {}));
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => {
      await result.current.startNavigation('terminal navigation', 'local_first');
    });
    const staleStart = result.current.startNavigation;
    const websocketCallback = vi.mocked(useWebSocketMessage).mock.calls[0][0] as (message: unknown) => void;

    let successorId: string | null | undefined;
    await act(async () => {
      websocketCallback({
        type: 'ai_navigation_complete',
        navigationId: 'nav-terminal',
        sessionId: 'session-1',
        status: 'completed',
        totalSteps: 0,
        totalTokens: 0,
        totalDurationMs: 1,
      });
      successorId = await staleStart('successor navigation', 'local_first');
    });

    expect(successorId).toBe('nav-successor');
    expect(recordingApi.startAINavigation).toHaveBeenCalledTimes(2);
  });

  it('pauses live navigation while awaiting human intervention', async () => {
    mockSuccessfulStart('nav-awaiting-human');
    keepStatusObservationPending();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('human handoff', 'local_first'); });
    const websocketCallback = vi.mocked(useWebSocketMessage).mock.calls[0][0] as (message: unknown) => void;
    act(() => websocketCallback({
      type: 'ai_navigation_awaiting_human',
      navigationId: 'nav-awaiting-human',
      sessionId: 'session-1',
      stepNumber: 1,
      reason: 'Captcha requires a human',
      instructions: 'Complete the captcha',
      interventionType: 'captcha',
      trigger: 'ai_requested',
      timestamp: '2026-09-26T22:40:00.000Z',
    }));

    expect(result.current.state.status).toBe('awaiting_human');
    expect(result.current.state.isNavigating).toBe(false);
    expect(result.current.state.humanIntervention).toMatchObject({
      reason: 'Captcha requires a human',
      instructions: 'Complete the captcha',
      interventionType: 'captcha',
      trigger: 'ai_requested',
    });
  });

  it('restores active state when human intervention resumes navigation', async () => {
    mockSuccessfulStart('nav-resume-state');
    vi.mocked(recordingApi.getAINavigationStatus)
      .mockImplementationOnce(() => new Promise(() => {}))
      .mockResolvedValueOnce({
        success: true,
        data: {
          ...terminalStatus('nav-resume-state'),
          status: 'navigating',
          terminal: false,
        },
      } as never);
    vi.mocked(recordingApi.resumeAINavigation).mockResolvedValueOnce({ success: true } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('resume state', 'local_first'); });
    const websocketCallback = emitHumanHandoff('nav-resume-state', 'Verification required');
    expect(result.current.state.isNavigating).toBe(false);

    await act(async () => { await result.current.resumeNavigation(); });
    await waitFor(() => expect(result.current.state.isNavigating).toBe(true));
    expect(result.current.state.status).toBe('navigating');
    act(() => websocketCallback({
      type: 'ai_navigation_resumed',
      navigationId: 'nav-resume-state',
      sessionId: 'session-1',
      timestamp: '2026-09-26T22:41:00.000Z',
    }));

    expect(recordingApi.resumeAINavigation).toHaveBeenCalledWith('nav-resume-state', expect.anything());
    expect(result.current.state.status).toBe('navigating');
    expect(result.current.state.isNavigating).toBe(true);
    expect(result.current.state.humanIntervention).toBeNull();
  });

  it('admits resume from a stale callback after human handoff arrives', async () => {
    mockSuccessfulStart('nav-resume-stale');
    keepStatusObservationPending();
    vi.mocked(recordingApi.resumeAINavigation).mockResolvedValueOnce({ success: true } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('resume stale', 'local_first'); });
    const staleResume = result.current.resumeNavigation;
    const websocketCallback = emitHumanHandoff('nav-resume-stale', 'Verification required');

    await act(async () => {
      expect(await staleResume()).toBe(true);
    });
    expect(recordingApi.resumeAINavigation).toHaveBeenCalledWith('nav-resume-stale', expect.anything());
  });

  it('keeps abort cleanup active and clears terminal navigation identity', async () => {
    mockSuccessfulStart('nav-abort-human');
    keepStatusObservationPending();
    vi.mocked(recordingApi.abortAINavigation).mockResolvedValueOnce({ success: true } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('abort handoff', 'local_first'); });
    const websocketCallback = emitHumanHandoff('nav-abort-human', 'Login required');

    await act(async () => { await result.current.abortNavigation(); });
    expect(recordingApi.abortAINavigation).toHaveBeenCalledWith('nav-abort-human', expect.anything());
    expect(result.current.state.status).toBe('aborting');
    expect(result.current.state.isNavigating).toBe(true);

    act(() => websocketCallback({
      type: 'ai_navigation_complete',
      navigationId: 'nav-abort-human',
      sessionId: 'session-1',
      status: 'aborted',
      totalSteps: 0,
      totalTokens: 0,
      totalDurationMs: 5,
      finalUrl: 'https://example.com',
    }));

    expect(result.current.state.status).toBe('aborted');
    expect(result.current.state.isNavigating).toBe(false);
    expect(result.current.state.navigationId).toBeNull();
    expect(vi.mocked(recordingApi.getAINavigationStatus).mock.calls[0][2]?.signal?.aborted).toBe(true);
  });

  it.each([
    { command: 'abort', navigationId: 'nav-abort-retry', reason: 'Login required', error: 'abort unavailable' },
    { command: 'resume', navigationId: 'nav-resume-retry', reason: 'Verification required', error: 'resume unavailable' },
  ] as const)('keeps a failed $command retryable during human handoff', async ({ command, navigationId, reason, error }) => {
    vi.mocked(recordingApi.startAINavigation).mockResolvedValueOnce({
      success: true,
      data: { navigationId },
    } as never);
    keepStatusObservationPending();
    const api = command === 'abort' ? recordingApi.abortAINavigation : recordingApi.resumeAINavigation;
    vi.mocked(api)
      .mockResolvedValueOnce({ success: false, error } as never)
      .mockResolvedValueOnce({ success: true } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation(`${command} retry`, 'local_first'); });
    const websocketCallback = emitHumanHandoff(navigationId, reason);

    const sendCommand = () => command === 'abort'
      ? result.current.abortNavigation()
      : result.current.resumeNavigation();
    await act(async () => { await sendCommand(); });
    expect(result.current.state.status).toBe('awaiting_human');
    expect(result.current.state.isNavigating).toBe(false);
    expect(result.current.state.error).toBe(error);

    await act(async () => { await sendCommand(); });
    expect(api).toHaveBeenCalledTimes(2);
    expect(result.current.state.error).toBeNull();
    if (command === 'abort') {
      expect(result.current.state.status).toBe('aborting');
      expect(result.current.state.isNavigating).toBe(true);
    }
  });

  it.each([
    { command: 'abort', navigationId: 'nav-abort-race', reason: 'Login required', failure: 'late abort failure' },
    { command: 'resume', navigationId: 'nav-resume-race', reason: 'Verification required', failure: 'late resume failure' },
  ] as const)('ignores a late $command failure after navigation resumes', async ({ command, navigationId, reason, failure }) => {
    let resolveCommand: ((value: unknown) => void) | undefined;
    vi.mocked(recordingApi.startAINavigation).mockResolvedValueOnce({
      success: true,
      data: { navigationId },
    } as never);
    keepStatusObservationPending();
    const pendingCommand = new Promise((resolve) => { resolveCommand = resolve; });
    if (command === 'abort') {
      vi.mocked(recordingApi.abortAINavigation).mockImplementationOnce(() => pendingCommand as never);
    } else {
      vi.mocked(recordingApi.resumeAINavigation).mockImplementationOnce(() => pendingCommand as never);
    }
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation(`${command} race`, 'local_first'); });
    const websocketCallback = emitHumanHandoff(navigationId, reason);

    let commandPromise: Promise<void> | undefined;
    act(() => {
      commandPromise = command === 'abort'
        ? result.current.abortNavigation()
        : result.current.resumeNavigation();
    });
    act(() => websocketCallback({
      type: 'ai_navigation_resumed',
      navigationId,
      sessionId: 'session-1',
      timestamp: '2026-09-26T22:41:00.000Z',
    }));
    resolveCommand?.({ success: false, error: failure });
    await act(async () => { await commandPromise; });

    expect(result.current.state.status).toBe('navigating');
    expect(result.current.state.isNavigating).toBe(true);
    expect(result.current.state.error).toBeNull();
  });

  it.each([
    { command: 'resume', navigationId: 'nav-resume-duplicate', reason: 'Verification required' },
    { command: 'abort', navigationId: 'nav-abort-duplicate', reason: 'Login required' },
  ] as const)('admits only one concurrent $command command for a handoff', async ({ command, navigationId, reason }) => {
    vi.mocked(recordingApi.startAINavigation).mockResolvedValueOnce({
      success: true,
      data: { navigationId },
    } as never);
    keepStatusObservationPending();
    const api = command === 'abort' ? recordingApi.abortAINavigation : recordingApi.resumeAINavigation;
    vi.mocked(api).mockResolvedValueOnce({ success: true } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation(`${command} duplicate`, 'local_first'); });
    const websocketCallback = emitHumanHandoff(navigationId, reason);

    let firstCommand: Promise<void> | undefined;
    let secondCommand: Promise<void> | undefined;
    act(() => {
      firstCommand = command === 'abort' ? result.current.abortNavigation() : result.current.resumeNavigation();
      secondCommand = command === 'abort' ? result.current.abortNavigation() : result.current.resumeNavigation();
    });
    await act(async () => { await Promise.all([firstCommand, secondCommand]); });

    expect(api).toHaveBeenCalledTimes(1);
  });

  it.each([
    { command: 'resume', navigationId: 'nav-resume-ack', reason: 'Verification required' },
    { command: 'abort', navigationId: 'nav-abort-ack' },
  ] as const)('does not resend $command after API acknowledgement', async ({ command, navigationId, reason }) => {
    mockSuccessfulStart(navigationId);
    keepStatusObservationPending();
    const api = command === 'abort' ? recordingApi.abortAINavigation : recordingApi.resumeAINavigation;
    vi.mocked(api).mockResolvedValueOnce({ success: true } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation(`${command} acknowledgement`, 'local_first'); });
    if (reason) emitHumanHandoff(navigationId, reason);
    const sendCommand = () => command === 'abort'
      ? result.current.abortNavigation()
      : result.current.resumeNavigation();
    await act(async () => { await sendCommand(); });
    await act(async () => { await sendCommand(); });

    expect(api).toHaveBeenCalledTimes(1);
  });

  it('recovers an acknowledged abort when the completion event is missed', async () => {
    mockSuccessfulStart('nav-abort-recovery');
    vi.mocked(recordingApi.getAINavigationStatus)
      .mockImplementationOnce(() => new Promise(() => {}))
      .mockResolvedValueOnce({
        success: true,
        data: {
          ...terminalStatus('nav-abort-recovery'),
          status: 'aborted',
          terminal: true,
        },
      } as never);
    vi.mocked(recordingApi.abortAINavigation).mockResolvedValueOnce({ success: true } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('abort recovery', 'local_first'); });
    await waitFor(() => expect(recordingApi.getAINavigationStatus).toHaveBeenCalledTimes(1));
    const initialObservationSignal = vi.mocked(recordingApi.getAINavigationStatus).mock.calls[0][2]?.signal;
    await act(async () => { await result.current.abortNavigation(); });

    await waitFor(() => expect(recordingApi.getAINavigationStatus).toHaveBeenCalledTimes(2));
    expect(initialObservationSignal?.aborted).toBe(true);
    expect(vi.mocked(recordingApi.getAINavigationStatus).mock.calls[1][2]?.signal?.aborted).toBe(false);
    expect(result.current.state.status).toBe('aborted');
    expect(result.current.state.navigationId).toBeNull();
    expect(result.current.state.isNavigating).toBe(false);
  });

  it('does not let an old command release a new navigation handoff guard', async () => {
    const resumeResolvers: Array<(value: unknown) => void> = [];
    vi.mocked(recordingApi.startAINavigation)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-old' } } as never)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-new' } } as never);
    vi.mocked(recordingApi.getAINavigationStatus).mockImplementation(() => new Promise(() => {}));
    vi.mocked(recordingApi.resumeAINavigation).mockImplementation(() => new Promise((resolve) => {
      resumeResolvers.push(resolve);
    }) as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('old handoff', 'local_first'); });
    const websocketCallback = emitHumanHandoff('nav-old', 'Old verification');

    let oldResume: Promise<boolean> | undefined;
    act(() => { oldResume = result.current.resumeNavigation(); });
    act(() => result.current.reset());

    await act(async () => { await result.current.startNavigation('new handoff', 'local_first'); });
    act(() => websocketCallback({
      type: 'ai_navigation_awaiting_human',
      navigationId: 'nav-new',
      sessionId: 'session-1',
      reason: 'New verification',
      timestamp: '2026-09-26T22:41:00.000Z',
    }));
    let newResume: Promise<void> | undefined;
    act(() => { newResume = result.current.resumeNavigation(); });

    resumeResolvers[0]?.({ success: true });
    let oldResumeResult: boolean | undefined;
    await act(async () => { oldResumeResult = await oldResume; });
    expect(oldResumeResult).toBe(false);
    let duplicateNewResume: Promise<void> | undefined;
    act(() => { duplicateNewResume = result.current.resumeNavigation(); });

    expect(recordingApi.resumeAINavigation).toHaveBeenCalledTimes(2);
    resumeResolvers[1]?.({ success: true });
    await act(async () => { await Promise.all([newResume, duplicateNewResume]); });
  });

  it('ignores resumed events after reset fences the handoff', async () => {
    mockSuccessfulStart('nav-reset-handoff');
    keepStatusObservationPending();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('reset handoff', 'local_first'); });
    const websocketCallback = emitHumanHandoff('nav-reset-handoff', 'Verification required');
    act(() => result.current.reset());

    act(() => websocketCallback({
      type: 'ai_navigation_resumed',
      navigationId: 'nav-reset-handoff',
      sessionId: 'session-1',
      timestamp: '2026-09-26T22:41:00.000Z',
    }));

    expect(result.current.state.status).toBe('idle');
    expect(result.current.state.isNavigating).toBe(false);
    expect(result.current.state.navigationId).toBeNull();
  });

  it('ignores live steps without a valid one-based step number', async () => {
    mockSuccessfulStart('nav-invalid-step');
    keepStatusObservationPending();
    const onStep = vi.fn();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1', onStep }));

    await act(async () => { await result.current.startNavigation('invalid step', 'local_first'); });
    const websocketCallback = vi.mocked(useWebSocketMessage).mock.calls[0][0] as (message: unknown) => void;
    act(() => websocketCallback({ type: '__proto__' }));
    act(() => websocketCallback({
      type: 'ai_navigation_step',
      navigationId: 'nav-invalid-step',
      sessionId: 'session-1',
      reasoning: 'malformed event',
    }));
    act(() => websocketCallback({
      type: 'ai_navigation_step',
      navigationId: 'nav-invalid-step',
      sessionId: 'session-1',
      stepNumber: 1,
      action: { type: 'click' },
      reasoning: 'valid event',
      currentUrl: 'https://example.com',
      goalAchieved: false,
      tokensUsed: { promptTokens: 1, completionTokens: 1, totalTokens: 2 },
      durationMs: 5,
      timestamp: '2026-09-26T22:40:00.000Z',
    }));

    expect(onStep).toHaveBeenCalledTimes(1);
    expect(onStep.mock.calls[0][0]).toMatchObject({ stepNumber: 1, reasoning: 'valid event' });
    expect(result.current.state.steps).toHaveLength(1);
  });

  it('preserves richer live steps when recovery returns an equal-length snapshot', async () => {
    let resolveStatus: ((value: unknown) => void) | undefined;
    mockSuccessfulStart('nav-merge-details');
    vi.mocked(recordingApi.getAINavigationStatus).mockImplementationOnce(() => new Promise((resolve) => {
      resolveStatus = resolve;
    }) as never);
    const onComplete = vi.fn();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1', onComplete }));

    await act(async () => { await result.current.startNavigation('merge details', 'local_first'); });
    const websocketCallback = vi.mocked(useWebSocketMessage).mock.calls[0][0] as (message: unknown) => void;
    act(() => websocketCallback({
      type: 'ai_navigation_step',
      navigationId: 'nav-merge-details',
      sessionId: 'session-1',
      stepNumber: 1,
      action: { type: 'click', selector: '#live', value: 'live-value' },
      reasoning: 'live reasoning',
      currentUrl: 'https://live.example.com',
      goalAchieved: false,
      tokensUsed: { promptTokens: 3, completionTokens: 4, totalTokens: 7 },
      durationMs: 42,
      timestamp: '2026-09-26T22:30:00.000Z',
    }));

    resolveStatus?.({
      success: true,
      data: {
        ...terminalStatus('nav-merge-details'),
        steps: [{
          index: 1,
          actionType: 'click',
          selector: '#recovered',
          value: '',
          url: 'https://recovered.example.com',
          description: 'recovered reasoning',
          success: true,
        }],
      },
    });
    await waitFor(() => expect(onComplete).toHaveBeenCalledWith('completed', undefined, 'nav-merge-details'));

    expect(result.current.state.steps).toHaveLength(1);
    expect(result.current.state.steps[0]).toMatchObject({
      reasoning: 'live reasoning',
      currentUrl: 'https://live.example.com',
      durationMs: 42,
      tokensUsed: { promptTokens: 3, completionTokens: 4, totalTokens: 7 },
      action: { type: 'click', selector: '#live', value: 'live-value' },
    });
    expect(result.current.state.navigationId).toBeNull();
  });

  it('allows the same step numbers in a replacement navigation', async () => {
    vi.mocked(recordingApi.startAINavigation)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-1' } } as never)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-2' } } as never);
    vi.mocked(recordingApi.getAINavigationStatus)
      .mockResolvedValueOnce({ success: true, data: terminalStatus('nav-1') } as never)
      .mockResolvedValueOnce({ success: true, data: terminalStatus('nav-2') } as never);
    const onStep = vi.fn();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1', onStep }));

    await act(async () => { await result.current.startNavigation('first', 'local_first'); });
    await waitFor(() => expect(onStep).toHaveBeenCalledTimes(1));
    act(() => result.current.reset());
    await act(async () => { await result.current.startNavigation('second', 'local_first'); });
    await waitFor(() => expect(onStep).toHaveBeenCalledTimes(2));

    expect(onStep.mock.calls[1][0].stepNumber).toBe(1);
    expect(onStep.mock.calls[1][0].timestamp.toISOString()).toBe('2023-11-14T22:13:20.000Z');
  });

  it('aborts a pending status wait and renews its signal on reset', async () => {
    vi.mocked(recordingApi.startAINavigation)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-1' } } as never)
      .mockResolvedValueOnce({ success: true, data: { navigationId: 'nav-2' } } as never);
    vi.mocked(recordingApi.getAINavigationStatus)
      .mockImplementationOnce(() => new Promise(() => {}))
      .mockResolvedValueOnce({ success: true, data: terminalStatus('nav-2') } as never);
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('first', 'local_first'); });
    await waitFor(() => expect(recordingApi.getAINavigationStatus).toHaveBeenCalledTimes(1));
    const firstSignal = vi.mocked(recordingApi.getAINavigationStatus).mock.calls[0][2]?.signal;
    expect(firstSignal?.aborted).toBe(false);

    act(() => result.current.reset());
    expect(firstSignal?.aborted).toBe(true);
    await act(async () => { await result.current.startNavigation('second', 'local_first'); });
    await waitFor(() => expect(recordingApi.getAINavigationStatus).toHaveBeenCalledTimes(2));
    const secondSignal = vi.mocked(recordingApi.getAINavigationStatus).mock.calls[1][2]?.signal;
    expect(secondSignal?.aborted).toBe(false);
  });

  it('aborts the owned status wait when the hook unmounts', async () => {
    mockSuccessfulStart('nav-unmount');
    keepStatusObservationPending();
    const { result, unmount } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    await act(async () => { await result.current.startNavigation('unmount', 'local_first'); });
    await waitFor(() => expect(recordingApi.getAINavigationStatus).toHaveBeenCalledTimes(1));
    const statusSignal = vi.mocked(recordingApi.getAINavigationStatus).mock.calls[0][2]?.signal;

    unmount();

    expect(statusSignal?.aborted).toBe(true);
  });

  it('does not create a status wait when start resolves after unmount', async () => {
    let resolveStart: ((value: unknown) => void) | undefined;
    vi.mocked(recordingApi.startAINavigation).mockImplementationOnce(() => new Promise((resolve) => {
      resolveStart = resolve;
    }) as never);
    const { result, unmount } = renderHook(() => useAINavigation({ sessionId: 'session-1' }));

    let startPromise: Promise<string | null> | undefined;
    act(() => { startPromise = result.current.startNavigation('late start', 'local_first'); });
    unmount();
    resolveStart?.({ success: true, data: { navigationId: 'nav-late-start' } });

    await expect(startPromise).resolves.toBeNull();
    expect(recordingApi.getAINavigationStatus).not.toHaveBeenCalled();
  });

  it('preserves recovered selector and value details for actionable steps', async () => {
    mockSuccessfulStart('nav-details');
    vi.mocked(recordingApi.getAINavigationStatus).mockResolvedValueOnce({
      success: true,
      data: {
        ...terminalStatus('nav-details'),
        steps: [
          {
            index: 1,
            actionType: 'click',
            selector: '#submit',
            value: '',
            url: 'https://example.com/form',
            description: 'click submit',
            success: true,
          },
          {
            index: 2,
            actionType: 'type',
            selector: '#email',
            value: 'user@example.com',
            url: 'https://example.com/form',
            description: 'fill email',
            success: true,
          },
          {
            index: 3,
            actionType: 'read',
            selector: '#confirmation',
            value: '',
            url: 'https://example.com/form',
            description: 'read confirmation',
            success: true,
          },
          {
            index: 4,
            actionType: 'type',
            selector: '#clearable',
            value: '',
            url: 'https://example.com/form',
            description: 'clear field',
            success: true,
          },
        ],
      },
    } as never);
    const onStep = vi.fn();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1', onStep }));

    await act(async () => { await result.current.startNavigation('recover details', 'local_first'); });
    await waitFor(() => expect(onStep).toHaveBeenCalledTimes(4));

    expect(onStep.mock.calls[0][0].action).toMatchObject({ type: 'click', selector: '#submit' });
    expect(onStep.mock.calls[1][0].action).toMatchObject({
      type: 'type',
      selector: '#email',
      value: 'user@example.com',
      text: 'user@example.com',
      success: true,
    });
    expect(onStep.mock.calls[2][0].action).toMatchObject({ type: 'read', selector: '#confirmation', success: true });
    expect(onStep.mock.calls[3][0].action).toMatchObject({
      type: 'type',
      selector: '#clearable',
      value: '',
      text: '',
    });
  });

  it('preserves selector and value details from live WebSocket actions', async () => {
    mockSuccessfulStart('nav-live-details');
    keepStatusObservationPending();
    const onStep = vi.fn();
    const { result } = renderHook(() => useAINavigation({ sessionId: 'session-1', onStep }));
    await act(async () => { await result.current.startNavigation('live details', 'local_first'); });

    const websocketCallback = vi.mocked(useWebSocketMessage).mock.calls[0][0] as (message: unknown) => void;
    act(() => websocketCallback({
      type: 'ai_navigation_step',
      navigationId: 'nav-live-details',
      sessionId: 'session-1',
      stepNumber: 1,
      action: { type: 'evaluate', selector: '#email', value: 'live@example.com' },
      reasoning: 'fill email',
      currentUrl: 'https://example.com/form',
      goalAchieved: false,
      tokensUsed: { promptTokens: 1, completionTokens: 1, totalTokens: 2 },
      durationMs: 5,
      timestamp: '2026-09-26T21:00:00.000Z',
    }));
    act(() => websocketCallback({
      type: 'ai_navigation_step',
      navigationId: 'nav-live-details',
      sessionId: 'session-1',
      stepNumber: 2,
      action: {
        type: 'type',
        success: false,
        input: { selector: '#email', value: 'typed@example.com' },
      },
      reasoning: 'type email',
      currentUrl: 'https://example.com/form',
      goalAchieved: false,
      tokensUsed: { promptTokens: 1, completionTokens: 1, totalTokens: 2 },
      durationMs: 5,
      timestamp: '2026-09-26T21:00:01.000Z',
    }));
    act(() => websocketCallback({
      type: 'ai_navigation_step',
      navigationId: 'nav-live-details',
      sessionId: 'session-1',
      stepNumber: 3,
      action: { type: 'click', value: 'button-value' },
      reasoning: 'click button',
      currentUrl: 'https://example.com/form',
      goalAchieved: false,
      tokensUsed: { promptTokens: 1, completionTokens: 1, totalTokens: 2 },
      durationMs: 5,
      timestamp: '2026-09-26T21:00:02.000Z',
    }));
    act(() => websocketCallback({
      type: 'ai_navigation_step',
      navigationId: 'nav-live-details',
      sessionId: 'session-1',
      stepNumber: 3,
      action: { type: 'click', value: 'button-value' },
      reasoning: 'replayed click button',
      currentUrl: 'https://example.com/form',
      goalAchieved: false,
      tokensUsed: { promptTokens: 99, completionTokens: 99, totalTokens: 198 },
      durationMs: 99,
      timestamp: '2026-09-26T21:00:03.000Z',
    }));

    expect(onStep.mock.calls[0][0]).toMatchObject({
      action: expect.objectContaining({
        type: 'evaluate',
        selector: '#email',
        value: 'live@example.com',
      }),
    });
    expect(onStep.mock.calls[1][0]).toMatchObject({
      action: expect.objectContaining({
        type: 'type',
        selector: '#email',
        value: 'typed@example.com',
        text: 'typed@example.com',
        success: false,
      }),
    });
    expect(onStep.mock.calls[2][0].action).toMatchObject({ type: 'click', value: 'button-value' });
    expect(onStep).toHaveBeenCalledTimes(3);
    expect(result.current.state.steps).toHaveLength(3);
  });
});
