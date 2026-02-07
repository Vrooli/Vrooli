import { useCallback, useEffect, useRef, useState } from 'react';
import type { RefObject } from 'react';
import type { BridgeShortcutIntent } from '@vrooli/iframe-bridge';
import { useIframeBridge } from '@/hooks/useIframeBridge';
import usePreviewNavigation from '@/components/views/usePreviewNavigation';

type ShortcutMessage = {
  intent: BridgeShortcutIntent;
};

interface UsePreviewNavigationSessionOptions {
  iframeRef: RefObject<HTMLIFrameElement>;
  setStatusMessage: (message: string | null) => void;
  onBeforeLocalNavigation?: () => void;
  onShortcut?: (message: ShortcutMessage) => void;
}

export function usePreviewNavigationSession({
  iframeRef,
  setStatusMessage,
  onBeforeLocalNavigation,
  onShortcut,
}: UsePreviewNavigationSessionOptions) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewUrlInput, setPreviewUrlInput] = useState('');
  const [hasCustomPreviewUrl, setHasCustomPreviewUrl] = useState(false);
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const initialPreviewUrlRef = useRef<string | null>(null);
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
