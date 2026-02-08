import { useCallback, useMemo, useRef } from 'react';
import type { ChangeEvent, KeyboardEvent, MutableRefObject } from 'react';
import { logger } from '@/services/logger';
import { resolvePreviewUrlCandidate } from '@/utils/previewUrl';

type BridgeSnapshot = {
  isSupported: boolean;
  href: string | null;
  canGoBack: boolean;
  canGoForward: boolean;
};

type StateRefs = {
  previewUrl: string | null;
  setPreviewUrl: React.Dispatch<React.SetStateAction<string | null>>;
  previewUrlInput: string;
  setPreviewUrlInput: React.Dispatch<React.SetStateAction<string>>;
  hasCustomPreviewUrl: boolean;
  setHasCustomPreviewUrl: React.Dispatch<React.SetStateAction<boolean>>;
  history: string[];
  setHistory: React.Dispatch<React.SetStateAction<string[]>>;
  historyIndex: number;
  setHistoryIndex: React.Dispatch<React.SetStateAction<number>>;
};

type UsePreviewNavigationOptions = StateRefs & {
  initialPreviewUrlRef: MutableRefObject<string | null>;
  bridgeState: BridgeSnapshot;
  childOrigin: string | null;
  sendBridgeNav: (cmd: 'GO' | 'BACK' | 'FWD', href?: string) => boolean;
  resetBridgeState: () => void;
  setStatusMessage: (message: string | null) => void;
  onBeforeLocalNavigation: () => void;
};

type UsePreviewNavigationResult = {
  canGoBack: boolean;
  canGoForward: boolean;
  applyPreviewUrlValue: (value: string) => void;
  applyPreviewUrlInput: () => void;
  handleUrlInputChange: (event: ChangeEvent<HTMLInputElement>) => void;
  handleUrlInputKeyDown: (event: KeyboardEvent<HTMLInputElement>) => void;
  handleUrlInputBlur: () => void;
  handleGoBack: () => void;
  handleGoForward: () => void;
  resetPreviewState: (options?: { force?: boolean }) => void;
  applyDefaultPreviewUrl: (url: string) => void;
  syncFromBridge: (href: string | null) => void;
};

const normalizeUrl = (value: string): string => value.replace(/\/$/, '');
const isSameUrl = (left: string, right: string): boolean => normalizeUrl(left) === normalizeUrl(right);

export const usePreviewNavigation = ({
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
  bridgeState,
  childOrigin,
  sendBridgeNav,
  resetBridgeState,
  setStatusMessage,
  onBeforeLocalNavigation,
}: UsePreviewNavigationOptions): UsePreviewNavigationResult => {
  const latestPreviewUrlInputRef = useRef(previewUrlInput);
  latestPreviewUrlInputRef.current = previewUrlInput;

  const canGoBack = useMemo(() => (
    bridgeState.isSupported ? bridgeState.canGoBack : historyIndex > 0
  ), [bridgeState.canGoBack, bridgeState.isSupported, historyIndex]);

  const canGoForward = useMemo(() => (
    bridgeState.isSupported ? bridgeState.canGoForward : (historyIndex >= 0 && historyIndex < history.length - 1)
  ), [bridgeState.canGoForward, bridgeState.isSupported, history.length, historyIndex]);

  const resetPreviewState = useCallback((options?: { force?: boolean }) => {
    if (!options?.force && hasCustomPreviewUrl) {
      return;
    }
    setPreviewUrl(null);
    setPreviewUrlInput('');
    setHistory([]);
    setHistoryIndex(-1);
    initialPreviewUrlRef.current = null;
  }, [hasCustomPreviewUrl, initialPreviewUrlRef, setHistory, setPreviewUrl, setPreviewUrlInput, setHistoryIndex]);

  const applyDefaultPreviewUrl = useCallback((url: string) => {
    const normalizedUrl = resolvePreviewUrlCandidate(url, previewUrl) ?? url;
    initialPreviewUrlRef.current = normalizedUrl;
    setPreviewUrl(normalizedUrl);
    setPreviewUrlInput(normalizedUrl);
    setHasCustomPreviewUrl(false);
    setHistory(prevHistory => {
      if (prevHistory.length === 0) {
        setHistoryIndex(0);
        return [normalizedUrl];
      }
      if (prevHistory[prevHistory.length - 1] === normalizedUrl) {
        setHistoryIndex(prevHistory.length - 1);
        return prevHistory;
      }
      const nextHistory = [...prevHistory, normalizedUrl];
      setHistoryIndex(nextHistory.length - 1);
      return nextHistory;
    });
  }, [
    initialPreviewUrlRef,
    previewUrl,
    setHasCustomPreviewUrl,
    setHistory,
    setHistoryIndex,
    setPreviewUrl,
    setPreviewUrlInput,
  ]);

  const applyPreviewUrlValue = useCallback((value: string) => {
    const trimmed = value.trim();

    if (!trimmed) {
      if (previewUrlInput !== '') {
        setPreviewUrlInput('');
      }
      setHasCustomPreviewUrl(false);
      return;
    }

    if (trimmed !== previewUrlInput) {
      setPreviewUrlInput(trimmed);
    }
    latestPreviewUrlInputRef.current = trimmed;

    const navigationReference = bridgeState.href || previewUrl || initialPreviewUrlRef.current;
    const resolvedTarget = resolvePreviewUrlCandidate(trimmed, navigationReference);

    if (!resolvedTarget) {
      setStatusMessage('Enter an absolute URL or wait for preview to load before using relative paths.');
      return;
    }

    if (bridgeState.isSupported) {
      try {
        if (!childOrigin || new URL(resolvedTarget).origin === childOrigin) {
          const sent = sendBridgeNav('GO', resolvedTarget);
          if (sent) {
            setHasCustomPreviewUrl(true);
            setPreviewUrl(resolvedTarget);
            setHistory(prevHistory => {
              const baseHistory = historyIndex >= 0 ? prevHistory.slice(0, historyIndex + 1) : [];
              const last = baseHistory[baseHistory.length - 1];
              const nextHistory = last === resolvedTarget ? baseHistory : [...baseHistory, resolvedTarget];
              setHistoryIndex(nextHistory.length - 1);
              return nextHistory;
            });
            setStatusMessage(null);
            return;
          }
        }
      } catch (error) {
        logger.warn('Bridge navigation failed to parse URL', error);
      }
    }

    setHasCustomPreviewUrl(true);
    setPreviewUrl(resolvedTarget);
    initialPreviewUrlRef.current = resolvedTarget;
    resetBridgeState();
    setStatusMessage(null);

    setHistory(prevHistory => {
      const baseHistory = historyIndex >= 0 ? prevHistory.slice(0, historyIndex + 1) : [];
      const last = baseHistory[baseHistory.length - 1];
      const nextHistory = last === resolvedTarget ? baseHistory : [...baseHistory, resolvedTarget];
      setHistoryIndex(nextHistory.length - 1);
      return nextHistory;
    });
  }, [
    bridgeState.href,
    bridgeState.isSupported,
    childOrigin,
    historyIndex,
    initialPreviewUrlRef,
    previewUrl,
    previewUrlInput,
    resetBridgeState,
    sendBridgeNav,
    setHasCustomPreviewUrl,
    setHistory,
    setHistoryIndex,
    setPreviewUrl,
    setPreviewUrlInput,
    setStatusMessage,
  ]);

  const applyPreviewUrlInput = useCallback(() => {
    applyPreviewUrlValue(latestPreviewUrlInputRef.current);
  }, [applyPreviewUrlValue]);

  const handleUrlInputChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    setPreviewUrlInput(event.target.value);
  }, [setPreviewUrlInput]);

  const handleUrlInputKeyDown = useCallback((event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      event.preventDefault();
      applyPreviewUrlInput();
    }
  }, [applyPreviewUrlInput]);

  const handleUrlInputBlur = useCallback(() => {
    applyPreviewUrlInput();
  }, [applyPreviewUrlInput]);

  const handleGoBack = useCallback(() => {
    onBeforeLocalNavigation();
    if (bridgeState.isSupported) {
      sendBridgeNav('BACK');
      return;
    }

    if (historyIndex <= 0) {
      return;
    }

    const targetIndex = historyIndex - 1;
    const targetUrl = history[targetIndex] ?? null;
    setHistoryIndex(targetIndex);
    setPreviewUrl(targetUrl);
    setPreviewUrlInput(targetUrl ?? '');
    setHasCustomPreviewUrl(true);
    setStatusMessage(null);
  }, [bridgeState.isSupported, history, historyIndex, onBeforeLocalNavigation, sendBridgeNav, setHasCustomPreviewUrl, setHistoryIndex, setPreviewUrl, setPreviewUrlInput, setStatusMessage]);

  const handleGoForward = useCallback(() => {
    if (bridgeState.isSupported) {
      sendBridgeNav('FWD');
      return;
    }

    if (historyIndex === -1 || historyIndex >= history.length - 1) {
      return;
    }

    const targetIndex = historyIndex + 1;
    const targetUrl = history[targetIndex] ?? null;
    setHistoryIndex(targetIndex);
    setPreviewUrl(targetUrl);
    setPreviewUrlInput(targetUrl ?? '');
    setHasCustomPreviewUrl(true);
    setStatusMessage(null);
  }, [bridgeState.isSupported, history, historyIndex, sendBridgeNav, setHasCustomPreviewUrl, setHistoryIndex, setPreviewUrl, setPreviewUrlInput, setStatusMessage]);

  const syncFromBridge = useCallback((href: string | null) => {
    if (!href) {
      return;
    }
    const normalizedHref = resolvePreviewUrlCandidate(
      href,
      bridgeState.href || previewUrl || initialPreviewUrlRef.current,
    ) ?? href;
    setPreviewUrl(normalizedHref);
    setPreviewUrlInput(normalizedHref);
    setHistory(prevHistory => {
      const baseHistory = historyIndex >= 0 ? prevHistory.slice(0, historyIndex + 1) : [];
      const last = baseHistory[baseHistory.length - 1];
      if (last && isSameUrl(last, normalizedHref)) {
        setHistoryIndex(baseHistory.length - 1);
        return baseHistory;
      }

      const nextHistory = [...baseHistory, normalizedHref];
      setHistoryIndex(nextHistory.length - 1);
      return nextHistory;
    });
    if (!initialPreviewUrlRef.current) {
      initialPreviewUrlRef.current = normalizedHref;
    }
    setHasCustomPreviewUrl(prev => {
      if (prev) {
        return prev;
      }
      const base = initialPreviewUrlRef.current;
      if (!base) {
        return prev;
      }
      return !isSameUrl(normalizedHref, base);
    });
  }, [
    bridgeState.href,
    historyIndex,
    initialPreviewUrlRef,
    previewUrl,
    setHasCustomPreviewUrl,
    setHistory,
    setHistoryIndex,
    setPreviewUrl,
    setPreviewUrlInput,
  ]);

  return {
    canGoBack,
    canGoForward,
    applyPreviewUrlValue,
    applyPreviewUrlInput,
    handleUrlInputChange,
    handleUrlInputKeyDown,
    handleUrlInputBlur,
    handleGoBack,
    handleGoForward,
    resetPreviewState,
    applyDefaultPreviewUrl,
    syncFromBridge,
  };
};

export default usePreviewNavigation;
