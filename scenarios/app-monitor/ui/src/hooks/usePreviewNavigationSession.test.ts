import { act, renderHook, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { usePreviewNavigationSession } from './usePreviewNavigationSession';

const {
  useIframeBridgeMock,
  usePreviewNavigationMock,
} = vi.hoisted(() => ({
  useIframeBridgeMock: vi.fn(),
  usePreviewNavigationMock: vi.fn(),
}));

vi.mock('@/hooks/useIframeBridge', () => ({
  useIframeBridge: useIframeBridgeMock,
}));

vi.mock('@/components/views/usePreviewNavigation', () => ({
  default: usePreviewNavigationMock,
}));

describe('usePreviewNavigationSession', () => {
  beforeEach(() => {
    useIframeBridgeMock.mockReset();
    usePreviewNavigationMock.mockReset();

    useIframeBridgeMock.mockReturnValue({
      state: {
        isSupported: true,
        href: null,
        canGoBack: false,
        canGoForward: false,
      },
      childOrigin: null,
      sendNav: vi.fn(),
      resetState: vi.fn(),
      runComplianceCheck: vi.fn(),
      requestScreenshot: vi.fn(),
      logState: null,
      configureLogs: vi.fn(),
      getRecentLogs: vi.fn(() => []),
      requestLogBatch: vi.fn(),
      networkState: null,
      configureNetwork: vi.fn(),
      getRecentNetworkEvents: vi.fn(() => []),
      requestNetworkBatch: vi.fn(),
      inspectState: {
        supported: false,
        active: false,
      },
      startInspect: vi.fn(),
      stopInspect: vi.fn(),
      setInspectTargetIndex: vi.fn(),
      shiftInspectTarget: vi.fn(),
    });

    usePreviewNavigationMock.mockImplementation((options: { previewUrl: string | null }) => ({
      canGoBack: false,
      canGoForward: false,
      handleUrlInputChange: vi.fn(),
      handleUrlInputKeyDown: vi.fn(),
      handleUrlInputBlur: vi.fn(),
      handleGoBack: vi.fn(),
      handleGoForward: vi.fn(),
      applyDefaultPreviewUrl: vi.fn(),
      applyPreviewUrlValue: vi.fn(),
      resetPreviewState: vi.fn(),
      syncFromBridge: vi.fn(() => options.previewUrl),
    }));
  });

  it('normalizes initial state and emits state snapshots', async () => {
    const onStateChange = vi.fn();
    const { result } = renderHook(() => usePreviewNavigationSession({
      iframeRef: { current: null },
      setStatusMessage: vi.fn(),
      initialState: {
        previewUrl: ' http://localhost:4310 ',
        previewUrlInput: 'http://localhost:4310',
        hasCustomPreviewUrl: true,
        history: ['http://localhost:4310', '   ', 'http://localhost:5000'],
        historyIndex: 99,
        initialPreviewUrl: 'http://localhost:4310',
      },
      onStateChange,
    }));

    expect(result.current.previewUrl).toBe('http://localhost:4310');
    expect(result.current.history).toEqual(['http://localhost:4310', 'http://localhost:5000']);
    expect(result.current.historyIndex).toBe(1);

    await waitFor(() => {
      expect(onStateChange).toHaveBeenCalledWith(expect.objectContaining({
        previewUrl: 'http://localhost:4310',
        hasCustomPreviewUrl: true,
        history: ['http://localhost:4310', 'http://localhost:5000'],
        historyIndex: 1,
      }));
    });
  });

  it('clears session state when clearNavigationSession is called', () => {
    const { result } = renderHook(() => usePreviewNavigationSession({
      iframeRef: { current: null },
      setStatusMessage: vi.fn(),
      initialState: {
        previewUrl: 'http://localhost:4310',
        previewUrlInput: 'http://localhost:4310',
        hasCustomPreviewUrl: true,
        history: ['http://localhost:4310'],
        historyIndex: 0,
        initialPreviewUrl: 'http://localhost:4310',
      },
    }));

    act(() => {
      result.current.clearNavigationSession();
    });

    expect(result.current.previewUrl).toBeNull();
    expect(result.current.previewUrlInput).toBe('');
    expect(result.current.history).toEqual([]);
    expect(result.current.historyIndex).toBe(-1);
    expect(result.current.hasCustomPreviewUrl).toBe(false);
    expect(result.current.initialPreviewUrlRef.current).toBeNull();
  });
});
