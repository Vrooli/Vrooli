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
    resetBridgeState: vi.fn(),
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
      previewUrl: 'http://localhost:4310',
      previewUrlInput: 'http://localhost:4310',
      hasCustomPreviewUrl: false,
      history: ['http://localhost:4310'],
      historyIndex: 0,
      bridgeState: {
        isSupported: true,
        href: null,
        canGoBack: false,
        canGoForward: false,
      },
      sendBridgeNav,
    }));

    act(() => {
      result.current.navigation.applyPreviewUrlValue('https://example.com/settings');
    });

    expect(sendBridgeNav).toHaveBeenCalledWith('GO', 'https://example.com/settings');
    expect(result.current.hasCustomPreviewUrl).toBe(true);
    expect(result.current.previewUrl).toBe('https://example.com/settings');
    expect(result.current.historyIndex).toBe(1);
    expect(result.current.history).toEqual(['http://localhost:4310', 'https://example.com/settings']);
    expect(result.current.statusMessage).toBeNull();
  });

  it('rejects relative URL input when no preview reference exists', () => {
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
    expect(result.current.previewUrl).toBeNull();
    expect(result.current.hasCustomPreviewUrl).toBe(false);
    expect(result.current.statusMessage).toContain('absolute URL');
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
});
