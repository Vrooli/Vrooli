import { useCallback, useEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';
import type { BridgeShortcutIntent } from '@vrooli/iframe-bridge';
import { useIframeBridge } from '@/hooks/useIframeBridge';
import usePreviewNavigation from '@/components/views/usePreviewNavigation';

type ShortcutMessage = {
  intent: BridgeShortcutIntent;
};

export interface PreviewNavigationSessionSnapshot {
  previewUrl: string | null;
  previewUrlInput: string;
  hasCustomPreviewUrl: boolean;
  history: string[];
  historyIndex: number;
  initialPreviewUrl: string | null;
}

interface UsePreviewNavigationSessionOptions {
  iframeRef: RefObject<HTMLIFrameElement>;
  setStatusMessage: (message: string | null) => void;
  onBeforeLocalNavigation?: () => void;
  onShortcut?: (message: ShortcutMessage) => void;
  initialState?: Partial<PreviewNavigationSessionSnapshot>;
  onStateChange?: (state: PreviewNavigationSessionSnapshot) => void;
}

export function usePreviewNavigationSession({
  iframeRef,
  setStatusMessage,
  onBeforeLocalNavigation,
  onShortcut,
  initialState,
  onStateChange,
}: UsePreviewNavigationSessionOptions) {
  const normalizedInitialHistory = Array.isArray(initialState?.history)
    ? initialState.history.filter((entry): entry is string => typeof entry === 'string' && entry.trim().length > 0)
    : [];
  const normalizedInitialHistoryIndex = typeof initialState?.historyIndex === 'number' && Number.isFinite(initialState.historyIndex)
    ? Math.max(-1, Math.min(normalizedInitialHistory.length - 1, Math.floor(initialState.historyIndex)))
    : normalizedInitialHistory.length - 1;
  const [previewUrl, setPreviewUrl] = useState<string | null>(
    () => (typeof initialState?.previewUrl === 'string' && initialState.previewUrl.trim().length > 0 ? initialState.previewUrl.trim() : null),
  );
  const [previewUrlInput, setPreviewUrlInput] = useState<string>(
    () => (typeof initialState?.previewUrlInput === 'string' ? initialState.previewUrlInput : ''),
  );
  const [hasCustomPreviewUrl, setHasCustomPreviewUrl] = useState<boolean>(() => Boolean(initialState?.hasCustomPreviewUrl));
  const [history, setHistory] = useState<string[]>(() => normalizedInitialHistory);
  const [historyIndex, setHistoryIndex] = useState<number>(() => normalizedInitialHistoryIndex);
  const initialPreviewUrlRef = useRef<string | null>(
    typeof initialState?.initialPreviewUrl === 'string' && initialState.initialPreviewUrl.trim().length > 0
      ? initialState.initialPreviewUrl.trim()
      : null,
  );
  const syncFromBridgeRef = useRef<(href: string | null) => void>(() => {});

  const handleBridgeLocation = useCallback((message: { href: string; title?: string | null }) => {
    syncFromBridgeRef.current(message.href ?? null);
    setStatusMessage(null);
  }, [setStatusMessage]);

  const bridge = useIframeBridge({
    iframeRef,
    previewUrl,
    onLocation: handleBridgeLocation,
    onShortcut,
  });

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
    bridgeState: {
      isSupported: bridge.state.isSupported,
      href: bridge.state.href,
      canGoBack: bridge.state.canGoBack,
      canGoForward: bridge.state.canGoForward,
    },
    childOrigin: bridge.childOrigin,
    sendBridgeNav: bridge.sendNav,
    resetBridgeState: bridge.resetState,
    setStatusMessage,
    onBeforeLocalNavigation: onBeforeLocalNavigation ?? (() => {}),
  });

  useEffect(() => {
    syncFromBridgeRef.current = navigation.syncFromBridge;
  }, [navigation.syncFromBridge]);

  useEffect(() => {
    if (!onStateChange) {
      return;
    }
    onStateChange({
      previewUrl,
      previewUrlInput,
      hasCustomPreviewUrl,
      history,
      historyIndex,
      initialPreviewUrl: initialPreviewUrlRef.current,
    });
  }, [hasCustomPreviewUrl, history, historyIndex, onStateChange, previewUrl, previewUrlInput]);

  const clearNavigationSession = useCallback(() => {
    setHasCustomPreviewUrl(false);
    setHistory([]);
    setHistoryIndex(-1);
    setPreviewUrl(null);
    setPreviewUrlInput('');
    initialPreviewUrlRef.current = null;
  }, []);

  return {
    bridge,
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
    clearNavigationSession,
    ...navigation,
  };
}

export default usePreviewNavigationSession;
