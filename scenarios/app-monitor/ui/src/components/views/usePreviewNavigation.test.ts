import { act, renderHook } from '@testing-library/react';
import { useRef, useState } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { usePreviewNavigation } from './usePreviewNavigation';

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

  it('updates URL input from bridge location without replacing iframe source URL', () => {
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
    expect(result.current.history).toEqual([
      'http://localhost:3000/apps/git-control-tower/proxy/',
      'http://localhost:3000/apps/git-control-tower/proxy/?path=src%2FApp.tsx',
    ]);
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
