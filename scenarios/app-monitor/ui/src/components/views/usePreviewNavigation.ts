import { useCallback, useEffect, useMemo, useRef } from 'react';
import type { ChangeEvent, KeyboardEvent, MutableRefObject } from 'react';
import { logger } from '@/services/logger';
import {
  isAppMonitorProxyPreviewTarget,
  isBlockedHostEmbedPreviewTarget,
  resolvePreviewUrlCandidate,
} from '@/utils/previewUrl';
import {
  PREVIEW_NAV_BLOCKED_HOST_MESSAGE,
  createPreviewNavigationPlan,
  isSameNormalizedUrl,
} from './previewNavigationPlanner';
import {
  previewNavigationActions,
  reducePreviewNavigationState,
  type PreviewNavigationState,
} from './previewNavigationStateMachine';

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

const BRIDGE_NAVIGATION_FALLBACK_MS = 1200;

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
  const latestStateRef = useRef<PreviewNavigationState>({
    previewUrl,
    previewUrlInput,
    hasCustomPreviewUrl,
    history,
    historyIndex,
    initialPreviewUrl: initialPreviewUrlRef.current,
  });
  const blockedHostEmbedAttemptsRef = useRef(0);
  const pendingBridgeNavigationRef = useRef<{
    target: string;
    timeoutId: number;
  } | null>(null);
  latestPreviewUrlInputRef.current = previewUrlInput;
  latestStateRef.current = {
    previewUrl,
    previewUrlInput,
    hasCustomPreviewUrl,
    history,
    historyIndex,
    initialPreviewUrl: initialPreviewUrlRef.current,
  };

  const applyNavigationState = useCallback((nextState: PreviewNavigationState) => {
    latestStateRef.current = nextState;
    initialPreviewUrlRef.current = nextState.initialPreviewUrl;
    setPreviewUrl(nextState.previewUrl);
    setPreviewUrlInput(nextState.previewUrlInput);
    setHasCustomPreviewUrl(nextState.hasCustomPreviewUrl);
    setHistory(nextState.history);
    setHistoryIndex(nextState.historyIndex);
  }, [
    initialPreviewUrlRef,
    setHasCustomPreviewUrl,
    setHistory,
    setHistoryIndex,
    setPreviewUrl,
    setPreviewUrlInput,
  ]);

  const transitionNavigationState = useCallback(
    (action: Parameters<typeof reducePreviewNavigationState>[1]): PreviewNavigationState => {
      const current = latestStateRef.current;
      const next = reducePreviewNavigationState(current, action);
      applyNavigationState(next);
      return next;
    },
    [applyNavigationState],
  );

  const clearPendingBridgeNavigationFallback = useCallback(() => {
    const pending = pendingBridgeNavigationRef.current;
    if (!pending) {
      return;
    }
    window.clearTimeout(pending.timeoutId);
    pendingBridgeNavigationRef.current = null;
  }, []);

  const canGoBack = useMemo(() => (
    bridgeState.isSupported ? bridgeState.canGoBack : historyIndex > 0
  ), [bridgeState.canGoBack, bridgeState.isSupported, historyIndex]);

  const canGoForward = useMemo(() => (
    bridgeState.isSupported ? bridgeState.canGoForward : (historyIndex >= 0 && historyIndex < history.length - 1)
  ), [bridgeState.canGoForward, bridgeState.isSupported, history.length, historyIndex]);

  const resetPreviewState = useCallback((options?: { force?: boolean }) => {
    transitionNavigationState(previewNavigationActions.reset(options?.force));
  }, [transitionNavigationState]);

  const applyDefaultPreviewUrl = useCallback((url: string) => {
    const reference = latestStateRef.current.previewUrl;
    const normalizedUrl = resolvePreviewUrlCandidate(url, reference) ?? url;
    transitionNavigationState(previewNavigationActions.applyDefaultUrl(normalizedUrl));
  }, [transitionNavigationState]);

  const clearStatusMessage = useCallback(() => {
    setStatusMessage(null);
  }, [setStatusMessage]);

  const commitLocalNavigation = useCallback((resolvedTarget: string) => {
    transitionNavigationState(previewNavigationActions.applyLocalNavigation(resolvedTarget));
    resetBridgeState();
    clearStatusMessage();
  }, [clearStatusMessage, resetBridgeState, transitionNavigationState]);

  const syncUrlInput = useCallback((value: string) => {
    if (value !== latestStateRef.current.previewUrlInput) {
      transitionNavigationState(previewNavigationActions.setInput(value));
    }
    latestPreviewUrlInputRef.current = value;
  }, [transitionNavigationState]);

  const applyPreviewUrlValue = useCallback((value: string) => {
    clearPendingBridgeNavigationFallback();
    const currentState = latestStateRef.current;
    const navigationReference = bridgeState.href || currentState.previewUrl || currentState.initialPreviewUrl;
    const hostOrigin = typeof window !== 'undefined' ? window.location.origin : null;
    const plan = createPreviewNavigationPlan({
      rawValue: value,
      navigationReference,
      hostOrigin,
      bridgeSupported: bridgeState.isSupported,
      childOrigin,
    });

    syncUrlInput(plan.nextInput);

    if (plan.kind === 'empty') {
      transitionNavigationState(previewNavigationActions.markDefaultCleared());
      return;
    }

    if (plan.kind === 'invalid') {
      setStatusMessage(plan.message);
      return;
    }

    if (plan.kind === 'blocked-host') {
      blockedHostEmbedAttemptsRef.current += 1;
      logger.warn('Blocked preview navigation to app-monitor shell target', {
        target: plan.resolvedTarget,
        blockedHostEmbedAttempts: blockedHostEmbedAttemptsRef.current,
      });
      setStatusMessage(plan.message);
      return;
    }

    if (plan.kind === 'bridge-go') {
      const sent = sendBridgeNav('GO', plan.resolvedTarget);
      if (sent) {
        const timeoutId = window.setTimeout(() => {
          if (latestPreviewUrlInputRef.current !== plan.nextInput) {
            return;
          }
          commitLocalNavigation(plan.resolvedTarget);
          pendingBridgeNavigationRef.current = null;
        }, BRIDGE_NAVIGATION_FALLBACK_MS);
        pendingBridgeNavigationRef.current = { target: plan.resolvedTarget, timeoutId };
        transitionNavigationState(previewNavigationActions.applyLocalNavigation(plan.resolvedTarget));
        // Bridge GO should not immediately replace iframe src; retain current src
        // while still reflecting address bar/history intent.
        applyNavigationState({
          ...latestStateRef.current,
          previewUrl: currentState.previewUrl,
        });
        clearStatusMessage();
        return;
      }
    }

    commitLocalNavigation(plan.resolvedTarget);
  }, [
    bridgeState.href,
    bridgeState.isSupported,
    childOrigin,
    applyNavigationState,
    commitLocalNavigation,
    clearPendingBridgeNavigationFallback,
    clearStatusMessage,
    sendBridgeNav,
    setStatusMessage,
    syncUrlInput,
    transitionNavigationState,
  ]);

  const applyPreviewUrlInput = useCallback(() => {
    applyPreviewUrlValue(latestPreviewUrlInputRef.current);
  }, [applyPreviewUrlValue]);

  const handleUrlInputChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    syncUrlInput(event.target.value);
  }, [syncUrlInput]);

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
    clearPendingBridgeNavigationFallback();
    onBeforeLocalNavigation();
    if (bridgeState.isSupported) {
      sendBridgeNav('BACK');
      return;
    }

    if (historyIndex <= 0) {
      return;
    }

    transitionNavigationState(previewNavigationActions.travelHistory('back'));
    clearStatusMessage();
  }, [bridgeState.isSupported, clearPendingBridgeNavigationFallback, clearStatusMessage, historyIndex, onBeforeLocalNavigation, sendBridgeNav, transitionNavigationState]);

  const handleGoForward = useCallback(() => {
    clearPendingBridgeNavigationFallback();
    if (bridgeState.isSupported) {
      sendBridgeNav('FWD');
      return;
    }

    if (historyIndex === -1 || historyIndex >= history.length - 1) {
      return;
    }

    transitionNavigationState(previewNavigationActions.travelHistory('forward'));
    clearStatusMessage();
  }, [bridgeState.isSupported, clearPendingBridgeNavigationFallback, clearStatusMessage, history, historyIndex, sendBridgeNav, transitionNavigationState]);

  const syncFromBridge = useCallback((href: string | null) => {
    if (!href) {
      return;
    }
    const currentState = latestStateRef.current;
    const normalizedHref = resolvePreviewUrlCandidate(
      href,
      bridgeState.href || currentState.previewUrl || currentState.initialPreviewUrl,
    ) ?? href;
    const hostOrigin = typeof window !== 'undefined' ? window.location.origin : null;
    if (
      isBlockedHostEmbedPreviewTarget(normalizedHref, hostOrigin)
      || isAppMonitorProxyPreviewTarget(normalizedHref)
    ) {
      blockedHostEmbedAttemptsRef.current += 1;
      logger.warn('Blocked preview bridge sync to app-monitor shell target', {
        target: normalizedHref,
        blockedHostEmbedAttempts: blockedHostEmbedAttemptsRef.current,
      });
      setStatusMessage(PREVIEW_NAV_BLOCKED_HOST_MESSAGE);
      return;
    }
    const pending = pendingBridgeNavigationRef.current;
    if (pending && isSameNormalizedUrl(pending.target, normalizedHref)) {
      clearPendingBridgeNavigationFallback();
    }
    transitionNavigationState(previewNavigationActions.syncFromBridge(normalizedHref));
  }, [
    bridgeState.href,
    clearPendingBridgeNavigationFallback,
    setStatusMessage,
    transitionNavigationState,
  ]);
  useEffect(() => {
    return () => {
      clearPendingBridgeNavigationFallback();
    };
  }, [clearPendingBridgeNavigationFallback]);

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
