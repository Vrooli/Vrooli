import { act, renderHook } from '@testing-library/react';
import { useRef, useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { usePreviewNavigation } from './usePreviewNavigation';

const { loggerWarnMock } = vi.hoisted(() => ({
  loggerWarnMock: vi.fn(),
}));

vi.mock('@/services/logger', () => ({
  logger: {
    debug: vi.fn(),
    info: vi.fn(),
    warn: loggerWarnMock,
    error: vi.fn(),
  },
}));

interface HarnessOptions {
  previewUrl: string | null;
  previewUrlInput: string;
  hasCustomPreviewUrl: boolean;
  history: string[];
  historyIndex: number;
  bridgeState?: {
    isSupported: boolean;
    href: string | null;
    canGoBack: boolean;
    canGoForward: boolean;
  };
  childOrigin?: string | null;
  sendBridgeNav?: (cmd: 'GO' | 'BACK' | 'FWD', href?: string) => boolean;
  resetBridgeState?: () => void;
}

function useNavigationHarness(options: HarnessOptions) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(options.previewUrl);
  const [previewUrlInput, setPreviewUrlInput] = useState<string>(options.previewUrlInput);
  const [hasCustomPreviewUrl, setHasCustomPreviewUrl] = useState<boolean>(options.hasCustomPreviewUrl);
  const [history, setHistory] = useState<string[]>(options.history);
  const [historyIndex, setHistoryIndex] = useState<number>(options.historyIndex);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const initialPreviewUrlRef = useRef<string | null>(options.previewUrl);

  const navigation = usePreviewNavigation({
    previewUrl,
    setPreviewUrl,
    previewUrlInput,
    setPreviewUrlInput,
    hasCustomPreviewUrl,
    setHasCustomPreviewUrl,
    history,
    setHistory,
    historyIndex,
    setHistoryIndex,
    initialPreviewUrlRef,
    bridgeState: options.bridgeState ?? {
      isSupported: false,
      href: null,
      canGoBack: false,
      canGoForward: false,
    },
    childOrigin: options.childOrigin ?? null,
    sendBridgeNav: options.sendBridgeNav ?? (() => false),
    resetBridgeState: options.resetBridgeState ?? vi.fn(),
    setStatusMessage,
    onBeforeLocalNavigation: vi.fn(),
  });

  return {
    navigation,
    previewUrl,
    previewUrlInput,
    hasCustomPreviewUrl,
    history,
    historyIndex,
    statusMessage,
  };
}

describe('usePreviewNavigation', () => {
  beforeEach(() => {
    loggerWarnMock.mockReset();
  });

  it('increments telemetry counter when blocking host shell embed targets', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('/apps/workspace');
      result.current.navigation.applyPreviewUrlValue('/apps/workspace');
    });

    expect(loggerWarnMock).toHaveBeenNthCalledWith(
      1,
      'Blocked preview navigation to app-monitor shell target',
      expect.objectContaining({
        blockedHostEmbedAttempts: 1,
      }),
    );
    expect(loggerWarnMock).toHaveBeenNthCalledWith(
      2,
      'Blocked preview navigation to app-monitor shell target',
      expect.objectContaining({
        blockedHostEmbedAttempts: 2,
      }),
    );
  });

  it('marks manual URL navigation as custom immediately when bridge navigation succeeds', () => {
    const sendBridgeNav = vi.fn().mockReturnValue(true);
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
      bridgeState: {
        isSupported: true,
        href: 'http://localhost:3000/apps/git-control-tower/proxy/',
        canGoBack: false,
        canGoForward: false,
      },
      childOrigin: 'http://localhost:3000',
      sendBridgeNav,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('http://localhost:3000/apps/git-control-tower/proxy/?path=README.md');
    });

    expect(sendBridgeNav).toHaveBeenCalledWith('GO', 'http://localhost:3000/apps/git-control-tower/proxy/?path=README.md');
    expect(result.current.hasCustomPreviewUrl).toBe(true);
    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(result.current.historyIndex).toBe(1);
    expect(result.current.history).toEqual([
      'http://localhost:3000/apps/git-control-tower/proxy/',
      'http://localhost:3000/apps/git-control-tower/proxy/?path=README.md',
    ]);
    expect(result.current.statusMessage).toBeNull();
  });

  it('falls back to iframe src navigation when bridge GO is acknowledged but no location sync arrives', () => {
    vi.useFakeTimers();
    const sendBridgeNav = vi.fn().mockReturnValue(true);
    const resetBridgeState = vi.fn();
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
      bridgeState: {
        isSupported: true,
        href: 'http://localhost:3000/apps/git-control-tower/proxy/',
        canGoBack: false,
        canGoForward: false,
      },
      childOrigin: 'http://localhost:3000',
      sendBridgeNav,
      resetBridgeState,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('http://localhost:3000/apps/git-control-tower/proxy/?path=README.md');
    });

    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(resetBridgeState).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(1200);
    });

    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/?path=README.md');
    expect(resetBridgeState).toHaveBeenCalledTimes(1);
    vi.useRealTimers();
  });

  it('cancels bridge fallback navigation when location sync arrives in time', () => {
    vi.useFakeTimers();
    const sendBridgeNav = vi.fn().mockReturnValue(true);
    const resetBridgeState = vi.fn();
    const target = 'http://localhost:3000/apps/git-control-tower/proxy/?path=README.md';
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
      bridgeState: {
        isSupported: true,
        href: 'http://localhost:3000/apps/git-control-tower/proxy/',
        canGoBack: false,
        canGoForward: false,
      },
      childOrigin: 'http://localhost:3000',
      sendBridgeNav,
      resetBridgeState,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue(target);
      result.current.navigation.syncFromBridge(target);
      vi.advanceTimersByTime(1200);
    });

    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(result.current.previewUrlInput).toBe(target);
    expect(result.current.hasCustomPreviewUrl).toBe(true);
    expect(resetBridgeState).not.toHaveBeenCalled();
    vi.useRealTimers();
  });

  it('does not send bridge GO when switching to a different scenario proxy URL', () => {
    const sendBridgeNav = vi.fn().mockReturnValue(true);
    const resetBridgeState = vi.fn();
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
      bridgeState: {
        isSupported: true,
        href: 'http://localhost:3000/apps/git-control-tower/proxy/',
        canGoBack: false,
        canGoForward: false,
      },
      childOrigin: 'http://localhost:3000',
      sendBridgeNav,
      resetBridgeState,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('http://localhost:3000/apps/agent-inbox/proxy/');
    });

    expect(sendBridgeNav).not.toHaveBeenCalled();
    expect(resetBridgeState).toHaveBeenCalledTimes(1);
    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/agent-inbox/proxy/');
  });

  it('uses iframe src navigation when changing path within same scenario proxy', () => {
    const sendBridgeNav = vi.fn().mockReturnValue(true);
    const resetBridgeState = vi.fn();
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
      bridgeState: {
        isSupported: true,
        href: 'http://localhost:3000/apps/git-control-tower/proxy/',
        canGoBack: false,
        canGoForward: false,
      },
      childOrigin: 'http://localhost:3000',
      sendBridgeNav,
      resetBridgeState,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('http://localhost:3000/apps/git-control-tower/proxy/settings');
    });

    expect(sendBridgeNav).not.toHaveBeenCalled();
    expect(resetBridgeState).toHaveBeenCalledTimes(1);
    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/settings');
  });

  it('falls back to iframe src navigation for root-relative input when no bridge scope exists', () => {
    const sendBridgeNav = vi.fn().mockReturnValue(true);
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: null,
      previewUrlInput: '',
      hasCustomPreviewUrl: false,
      history: [],
      historyIndex: -1,
      bridgeState: {
        isSupported: true,
        href: null,
        canGoBack: false,
        canGoForward: false,
      },
      sendBridgeNav,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('/settings');
    });

    expect(sendBridgeNav).not.toHaveBeenCalled();
    expect(result.current.previewUrl).toBe('http://localhost:3000/settings');
    expect(result.current.hasCustomPreviewUrl).toBe(true);
    expect(result.current.statusMessage).toBeNull();
  });

  it('resolves relative URLs against active preview URL instead of host page URL', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:4310/app/',
      previewUrlInput: 'http://localhost:4310/app/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:4310/app/'],
      historyIndex: 0,
      bridgeState: {
        isSupported: false,
        href: null,
        canGoBack: false,
        canGoForward: false,
      },
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('settings');
    });

    expect(result.current.previewUrl).toBe('http://localhost:4310/app/settings');
    expect(result.current.hasCustomPreviewUrl).toBe(true);
    expect(result.current.history).toEqual(['http://localhost:4310/app/', 'http://localhost:4310/app/settings']);
  });

  it('accepts localhost URLs without an explicit protocol', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: null,
      previewUrlInput: '',
      hasCustomPreviewUrl: false,
      history: [],
      historyIndex: -1,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('localhost:4310/admin');
    });

    expect(result.current.previewUrl).toBe('http://localhost:4310/admin');
    expect(result.current.hasCustomPreviewUrl).toBe(true);
    expect(result.current.history).toEqual(['http://localhost:4310/admin']);
  });

  it('normalizes bare scenario identifiers to proxy URLs', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: null,
      previewUrlInput: '',
      hasCustomPreviewUrl: false,
      history: [],
      historyIndex: -1,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('agent-inbox');
    });

    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/agent-inbox/proxy/');
    expect(result.current.previewUrlInput).toBe('/apps/agent-inbox/proxy/');
    expect(result.current.hasCustomPreviewUrl).toBe(true);
    expect(result.current.statusMessage).toBeNull();
  });

  it('rewrites /apps/<scenario> paths to proxy URLs', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: null,
      previewUrlInput: '',
      hasCustomPreviewUrl: false,
      history: [],
      historyIndex: -1,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('/apps/git-control-tower');
    });

    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(result.current.previewUrlInput).toBe('/apps/git-control-tower/proxy/');
    expect(result.current.hasCustomPreviewUrl).toBe(true);
    expect(result.current.statusMessage).toBeNull();
  });

  it('blocks app-monitor shell URLs to avoid recursive iframe embedding', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('/apps/workspace');
    });

    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(result.current.statusMessage).toBe(
      'Preview URL points to App Monitor shell. Use a scenario proxy URL like /apps/<scenario>/proxy/.',
    );
  });

  it('blocks app-monitor proxy URLs to avoid recursive iframe embedding', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('/apps/app-monitor/proxy/');
    });

    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(result.current.statusMessage).toBe(
      'Preview URL points to App Monitor shell. Use a scenario proxy URL like /apps/<scenario>/proxy/.',
    );
  });

  it('updates persisted URL input/history from bridge location without replacing iframe source URL', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
    }));

    act(() => {
      result.current.navigation.syncFromBridge('http://localhost:3000/apps/git-control-tower/proxy/?path=src%2FApp.tsx');
    });

    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(result.current.previewUrlInput).toBe('http://localhost:3000/apps/git-control-tower/proxy/?path=src%2FApp.tsx');
    expect(result.current.hasCustomPreviewUrl).toBe(true);
    expect(result.current.history).toEqual([
      'http://localhost:3000/apps/git-control-tower/proxy/',
      'http://localhost:3000/apps/git-control-tower/proxy/?path=src%2FApp.tsx',
    ]);
  });

  it('ignores bridge sync updates that point to app-monitor proxy URLs', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:3000/apps/git-control-tower/proxy/',
      previewUrlInput: 'http://localhost:3000/apps/git-control-tower/proxy/',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:3000/apps/git-control-tower/proxy/'],
      historyIndex: 0,
    }));

    act(() => {
      result.current.navigation.syncFromBridge('http://localhost:3000/apps/app-monitor/proxy/');
    });

    expect(result.current.previewUrl).toBe('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(result.current.previewUrlInput).toBe('http://localhost:3000/apps/git-control-tower/proxy/');
    expect(result.current.history).toEqual(['http://localhost:3000/apps/git-control-tower/proxy/']);
    expect(result.current.statusMessage).toBe(
      'Preview URL points to App Monitor shell. Use a scenario proxy URL like /apps/<scenario>/proxy/.',
    );
  });

  it('keeps selected URL when blur fires before state re-render', () => {
    const { result } = renderHook(() => useNavigationHarness({
      previewUrl: 'http://localhost:4310',
      previewUrlInput: '',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:4310'],
      historyIndex: 0,
      bridgeState: {
        isSupported: false,
        href: null,
        canGoBack: false,
        canGoForward: false,
      },
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('http://localhost:4310/settings');
      result.current.navigation.handleUrlInputBlur();
    });

    expect(result.current.previewUrl).toBe('http://localhost:4310/settings');
    expect(result.current.previewUrlInput).toBe('http://localhost:4310/settings');
    expect(result.current.history).toEqual([
      'http://localhost:4310',
      'http://localhost:4310/settings',
    ]);
    expect(result.current.historyIndex).toBe(1);
  });
});
