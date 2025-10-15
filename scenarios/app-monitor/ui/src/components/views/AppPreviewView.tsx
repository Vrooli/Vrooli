import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ChangeEvent, KeyboardEvent, MouseEvent } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import clsx from 'clsx';
import { Info, Loader2, X } from 'lucide-react';
import { appService } from '@/services/api';
import { useAppsStore } from '@/state/appsStore';
import { logger } from '@/services/logger';
import type { App, AppProxyMetadata, LocalhostUsageReport } from '@/types';
import AppModal from '../AppModal';
import AppPreviewToolbar from '../AppPreviewToolbar';
import ReportIssueDialog from '../report/ReportIssueDialog';
import { buildPreviewUrl, isRunningStatus, isStoppedStatus, locateAppByIdentifier } from '@/utils/appPreview';
import { useIframeBridge } from '@/hooks/useIframeBridge';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import './AppPreviewView.css';

const PREVIEW_LOAD_TIMEOUT_MS = 6000;
const PREVIEW_CONNECTING_LABEL = 'Connecting to preview...';
const PREVIEW_TIMEOUT_MESSAGE = 'Preview did not respond. Ensure the application UI is running and reachable from App Monitor.';
const PREVIEW_MIXED_CONTENT_MESSAGE = 'Preview blocked: browser refused to load HTTP content inside an HTTPS dashboard. Expose the UI through the tunnel hostname or enable HTTPS for the scenario.';
const IOS_AUTOBACK_GUARD_MS = 15000;

const AppPreviewView = () => {
  const apps = useAppsStore(state => state.apps);
  const setAppsState = useAppsStore(state => state.setAppsState);
  const loadApps = useAppsStore(state => state.loadApps);
  const loadingInitial = useAppsStore(state => state.loadingInitial);
  const hasInitialized = useAppsStore(state => state.hasInitialized);
  const navigate = useNavigate();
  const { appId } = useParams<{ appId: string }>();
  type PreviewLocationState = {
    fromAppsList?: boolean;
    originAppId?: string;
    navTimestamp?: number;
    suppressedAutoBack?: boolean;
  };
  const location = useLocation();
  const locationState: PreviewLocationState | null = location.state && typeof location.state === 'object'
    ? location.state as PreviewLocationState
    : null;
  const isIosSafari = useMemo(() => {
    if (typeof navigator === 'undefined') {
      return false;
    }
    const ua = navigator.userAgent || '';
    const isIOS = /iP(ad|hone|od)/i.test(ua);
    const isWebKit = /WebKit/i.test(ua);
    const isExcluded = /CriOS|FxiOS|OPiOS|EdgiOS/i.test(ua);
    return isIOS && isWebKit && !isExcluded;
  }, []);
  const previewLocationRef = useRef<{ pathname: string; search: string }>({ pathname: location.pathname, search: location.search || '' });
  const lastAppIdRef = useRef<string | null>(appId ?? null);
  const [currentApp, setCurrentApp] = useState<App | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewUrlInput, setPreviewUrlInput] = useState('');
  const [hasCustomPreviewUrl, setHasCustomPreviewUrl] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>('Loading application preview...');
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [fetchAttempted, setFetchAttempted] = useState(false);
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [pendingAction, setPendingAction] = useState<null | 'start' | 'stop' | 'restart'>(null);
  const [reportDialogOpen, setReportDialogOpen] = useState(false);
  const [previewReloadToken, setPreviewReloadToken] = useState(0);
  const [previewOverlay, setPreviewOverlay] = useState<null | { type: 'restart' | 'waiting' | 'error'; message: string }>(null);
  const [iframeLoadedAt, setIframeLoadedAt] = useState<number | null>(null);
  const [iframeLoadError, setIframeLoadError] = useState<string | null>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const [bridgeCompliance, setBridgeCompliance] = useState<BridgeComplianceResult | null>(null);
  const [proxyMetadata, setProxyMetadata] = useState<AppProxyMetadata | null>(null);
  const [localhostReport, setLocalhostReport] = useState<LocalhostUsageReport | null>(null);
  const [bridgeMessageDismissed, setBridgeMessageDismissed] = useState(false);
  const [localhostMessageDismissed, setLocalhostMessageDismissed] = useState(false);
  const complianceRunRef = useRef(false);
  const initialPreviewUrlRef = useRef<string | null>(null);
  const restartMonitorRef = useRef<{ cancel: () => void } | null>(null);
  const lastRefreshRequestRef = useRef(0);
  const lastRecordedViewRef = useRef<{ id: string | null; timestamp: number }>({ id: null, timestamp: 0 });
  const iosPopGuardRef = useRef<{ active: boolean; timeoutId: number | null; activatedAt: number; handledCount: number }>({
    active: false,
    timeoutId: null,
    activatedAt: 0,
    handledCount: 0,
  });
  const iosGuardedLocationKeyRef = useRef<string | null>(null);
  const lastStateSnapshotRef = useRef<string>('');

  const recordDebugEvent = useCallback((event: string, detail?: Record<string, unknown>) => {
    try {
      if (typeof window === 'undefined') {
        return;
      }
      const payload = {
        event,
        timestamp: Date.now(),
        detail: detail ?? null,
        userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : undefined,
      };
      const body = JSON.stringify(payload);
      if (navigator && typeof navigator.sendBeacon === 'function') {
        const blob = new Blob([body], { type: 'application/json' });
        navigator.sendBeacon('/__debug/client-event', blob);
      } else {
        void fetch('/__debug/client-event', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body,
          keepalive: true,
        });
      }
    } catch (error) {
      // Best-effort debug logging; ignore failures
    }
  }, []);

  const recordNavigateEvent = useCallback((info: Record<string, unknown>) => {
    recordDebugEvent('navigate-event', {
      appId,
      ...info,
    });
  }, [appId, recordDebugEvent]);

  const updatePreviewGuard = useCallback((patch: Record<string, unknown>) => {
    if (typeof window === 'undefined') {
      return;
    }
    const globalWindow = window as typeof window & {
      __appMonitorPreviewGuard?: {
        active: boolean;
        armedAt: number;
        ttl: number;
        key: string | null;
        appId: string | null;
        recoverPath: string | null;
      ignoreNextPopstate?: boolean;
      lastSuppressedAt?: number;
      recoverState?: unknown;
      };
    };
    const guard = globalWindow.__appMonitorPreviewGuard ?? {
      active: false,
      armedAt: 0,
      ttl: IOS_AUTOBACK_GUARD_MS,
      key: null,
      appId: null,
      recoverPath: null,
      ignoreNextPopstate: false,
      lastSuppressedAt: 0,
    };
    Object.assign(guard, patch);
    guard.ttl = typeof guard.ttl === 'number' ? guard.ttl : IOS_AUTOBACK_GUARD_MS;
    globalWindow.__appMonitorPreviewGuard = guard;
    recordDebugEvent('preview-guard-update', guard);
  }, [recordDebugEvent]);

  useEffect(() => {
    if (!hasInitialized && !loadingInitial) {
      void loadApps();
    }
  }, [hasInitialized, loadApps, loadingInitial]);

  useEffect(() => {
    recordDebugEvent('preview-mount', {
      appId,
      locationState,
      isIosSafari,
      pathname: location.pathname,
    });
    return () => {
      recordDebugEvent('preview-unmount', {
        appId: lastAppIdRef.current,
      });
      updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
    };
  }, [appId, isIosSafari, location.pathname, locationState, recordDebugEvent, updatePreviewGuard]);

  useEffect(() => {
    if (appId) {
      lastAppIdRef.current = appId;
      previewLocationRef.current = {
        pathname: `/apps/${encodeURIComponent(appId)}/preview`,
        search: location.search || '',
      };
    }
  }, [appId, location.search]);

  useEffect(() => {
    if (!isIosSafari) {
      iosGuardedLocationKeyRef.current = null;
      if (iosPopGuardRef.current.timeoutId) {
        window.clearTimeout(iosPopGuardRef.current.timeoutId);
        iosPopGuardRef.current.timeoutId = null;
      }
      iosPopGuardRef.current.active = false;
      return;
    }

    if (!locationState?.fromAppsList || locationState?.suppressedAutoBack) {
      iosGuardedLocationKeyRef.current = null;
      if (iosPopGuardRef.current.timeoutId) {
        window.clearTimeout(iosPopGuardRef.current.timeoutId);
        iosPopGuardRef.current.timeoutId = null;
      }
      iosPopGuardRef.current.active = false;
      return;
    }

    if (iosGuardedLocationKeyRef.current === location.key) {
      return;
    }

    iosGuardedLocationKeyRef.current = location.key;
    const guard = iosPopGuardRef.current;
    guard.active = true;
    guard.activatedAt = Date.now();
    guard.handledCount = 0;
    if (guard.timeoutId) {
      window.clearTimeout(guard.timeoutId);
    }
    guard.timeoutId = window.setTimeout(() => {
      guard.active = false;
      guard.timeoutId = null;
    }, IOS_AUTOBACK_GUARD_MS);
    recordDebugEvent('ios-guard-armed', {
      key: location.key,
      appId,
      navTimestamp: locationState?.navTimestamp,
    });
    updatePreviewGuard({
      active: true,
      armedAt: Date.now(),
      ttl: IOS_AUTOBACK_GUARD_MS,
      key: location.key,
      appId: appId ?? null,
      recoverPath: `${location.pathname}${location.search ?? ''}`,
      ignoreNextPopstate: false,
      recoverState: typeof window !== 'undefined' ? window.history.state : undefined,
    });
  }, [
    appId,
    isIosSafari,
    location.key,
    location.pathname,
    location.search,
    locationState,
    recordDebugEvent,
    updatePreviewGuard,
  ]);

  useEffect(() => {
    if (!isIosSafari) {
      return;
    }

    const guard = iosPopGuardRef.current;
    const handlePopState = () => {
      if (!guard.active) {
        return;
      }

      const elapsed = Date.now() - guard.activatedAt;
      const shouldSuppress = guard.handledCount === 0 && elapsed >= 0 && elapsed <= IOS_AUTOBACK_GUARD_MS;

      if (shouldSuppress) {
        guard.handledCount = 1;
        guard.active = false;
        if (guard.timeoutId) {
          window.clearTimeout(guard.timeoutId);
          guard.timeoutId = null;
        }
        const targetLocation = previewLocationRef.current;
        const hasTargetPath = Boolean(targetLocation.pathname);
        recordDebugEvent('ios-popstate-suppressed', {
          elapsed,
          appId,
          targetPath: targetLocation.pathname,
        });
        if (hasTargetPath) {
          const nextState: PreviewLocationState = {
            ...(locationState ?? {}),
            fromAppsList: true,
            originAppId: locationState?.originAppId ?? lastAppIdRef.current ?? undefined,
            navTimestamp: locationState?.navTimestamp ?? Date.now(),
            suppressedAutoBack: true,
          };
          window.requestAnimationFrame(() => {
            recordNavigateEvent({
              reason: 'ios-guard-restore',
              targetPath: targetLocation.pathname,
              targetSearch: targetLocation.search,
              replace: true,
            });
            navigate({
              pathname: targetLocation.pathname,
              search: targetLocation.search || undefined,
            }, {
              replace: true,
              state: nextState,
            });
            recordDebugEvent('ios-guard-restore', {
              pathname: targetLocation.pathname,
              search: targetLocation.search,
              appId,
            });
          });
        }
        return;
      }

      guard.active = false;
      if (guard.timeoutId) {
        window.clearTimeout(guard.timeoutId);
        guard.timeoutId = null;
      }
      recordDebugEvent('ios-popstate-allowed', {
        elapsed,
        appId,
      });
    };

    window.addEventListener('popstate', handlePopState);
    return () => {
      window.removeEventListener('popstate', handlePopState);
      if (guard.timeoutId) {
        window.clearTimeout(guard.timeoutId);
        guard.timeoutId = null;
      }
      guard.active = false;
      guard.handledCount = 0;
      recordDebugEvent('ios-guard-disposed', {
        appId,
      });
      updatePreviewGuard({ active: false, key: null });
    };
  }, [appId, isIosSafari, locationState, navigate, recordDebugEvent, recordNavigateEvent, updatePreviewGuard]);

  const matchesAppIdentifier = useCallback((app: App, identifier?: string | null) => {
    if (!identifier) {
      return false;
    }

    const normalized = identifier.trim().toLowerCase();
    if (!normalized) {
      return false;
    }

    const candidates = [app.id, app.scenario_name]
      .map(value => (typeof value === 'string' ? value.trim().toLowerCase() : ''))
      .filter(value => value.length > 0);

    return candidates.includes(normalized);
  }, []);

  const handleBridgeLocation = useCallback((message: { href: string; title?: string | null }) => {
    if (message.href) {
      setPreviewUrlInput(message.href);
      if (!initialPreviewUrlRef.current) {
        initialPreviewUrlRef.current = message.href;
      }
      setHasCustomPreviewUrl(prev => {
        if (prev) {
          return prev;
        }
        const base = initialPreviewUrlRef.current;
        if (!base) {
          return prev;
        }
        const normalize = (value: string) => value.replace(/\/$/, '');
        return normalize(message.href) !== normalize(base);
      });
    }
    setStatusMessage(null);
  }, []);

  const {
    state: bridgeState,
    childOrigin,
    sendNav: sendBridgeNav,
    runComplianceCheck,
    resetState,
    requestScreenshot,
    logState,
    requestLogBatch,
    getRecentLogs,
    configureLogs,
    networkState,
    requestNetworkBatch,
    getRecentNetworkEvents,
    configureNetwork,
  } = useIframeBridge({
    iframeRef,
    previewUrl,
    onLocation: handleBridgeLocation,
  });

  useEffect(() => {
    if (!isIosSafari) {
      return;
    }

    if (!locationState?.fromAppsList || !locationState?.suppressedAutoBack) {
      return;
    }

    const nextState: PreviewLocationState = {
      ...(locationState ?? {}),
      fromAppsList: false,
      suppressedAutoBack: false,
    };

    recordNavigateEvent({
      reason: 'ios-guard-reset-state',
      targetPath: location.pathname,
      targetSearch: location.search,
      replace: true,
    });
    navigate({
      pathname: location.pathname,
      search: location.search || undefined,
    }, {
      replace: true,
      state: nextState,
    });
    recordDebugEvent('ios-guard-reset-state', {
      pathname: location.pathname,
      search: location.search,
      appId,
    });
    updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
  }, [appId, isIosSafari, location.pathname, location.search, locationState, navigate, recordDebugEvent, recordNavigateEvent, updatePreviewGuard]);

  const resetPreviewState = useCallback((options?: { force?: boolean }) => {
    if (!options?.force && hasCustomPreviewUrl) {
      return;
    }

    setPreviewUrl(null);
    setPreviewUrlInput('');
    setHistory([]);
    setHistoryIndex(-1);
    initialPreviewUrlRef.current = null;
    setIframeLoadedAt(null);
    setIframeLoadError(null);
  }, [hasCustomPreviewUrl]);

  const applyDefaultPreviewUrl = useCallback((url: string) => {
    initialPreviewUrlRef.current = url;
    setPreviewUrl(url);
    setPreviewUrlInput(url);
    setHistory(prevHistory => {
      if (prevHistory.length === 0) {
        setHistoryIndex(0);
        return [url];
      }

      if (prevHistory[prevHistory.length - 1] === url) {
        setHistoryIndex(prevHistory.length - 1);
        return prevHistory;
      }

      const nextHistory = [...prevHistory, url];
      setHistoryIndex(nextHistory.length - 1);
      return nextHistory;
    });
  }, [setHistoryIndex]);

  const commitAppUpdate = useCallback((nextApp: App) => {
    setAppsState(prev => {
      const index = prev.findIndex(app => app.id === nextApp.id);
      if (index === -1) {
        return [...prev, nextApp];
      }

      const updated = [...prev];
      updated[index] = nextApp;
      return updated;
    });

    setCurrentApp(prev => {
      if (!prev) {
        return !appId || appId === nextApp.id ? nextApp : prev;
      }

      return prev.id === nextApp.id ? nextApp : prev;
    });
  }, [appId, setAppsState]);

  const stopLifecycleMonitor = useCallback(() => {
    if (restartMonitorRef.current) {
      restartMonitorRef.current.cancel();
      restartMonitorRef.current = null;
    }
  }, []);

  const reloadPreview = useCallback(() => {
    resetState();
    setIframeLoadedAt(null);
    setIframeLoadError(null);
    setPreviewReloadToken(prev => prev + 1);
  }, [resetState]);

  const beginLifecycleMonitor = useCallback((appIdentifier: string, lifecycle: 'start' | 'restart') => {
    stopLifecycleMonitor();

    let cancelled = false;
    restartMonitorRef.current = {
      cancel: () => {
        cancelled = true;
      },
    };

    const poll = async () => {
      const maxAttempts = 30;
      for (let attempt = 0; attempt < maxAttempts; attempt += 1) {
        if (cancelled) {
          return;
        }

        try {
          const fetched = await appService.getApp(appIdentifier);
          if (cancelled) {
            return;
          }

          if (fetched) {
            commitAppUpdate(fetched);

            if (isRunningStatus(fetched.status) && !isStoppedStatus(fetched.status)) {
              const candidateUrl = buildPreviewUrl(fetched);
              if (candidateUrl) {
                if (!hasCustomPreviewUrl) {
                  applyDefaultPreviewUrl(candidateUrl);
                }
                setStatusMessage(lifecycle === 'restart'
                  ? 'Application restarted. Refreshing preview...'
                  : 'Application started. Refreshing preview...');
                setPreviewOverlay(null);
                setLoading(false);
                reloadPreview();
                stopLifecycleMonitor();
                window.setTimeout(() => {
                  if (!cancelled) {
                    setStatusMessage(null);
                  }
                }, 1500);
                return;
              }
            }
          }
        } catch (error) {
          logger.warn('Lifecycle monitor poll failed', error);
        }

        const delay = attempt < 5 ? 1000 : 2000;
        await new Promise(resolve => window.setTimeout(resolve, delay));
      }

      if (!cancelled) {
        setLoading(false);
        setPreviewOverlay({ type: 'error', message: lifecycle === 'restart'
          ? 'Application has not come back online yet. Try refreshing.'
          : 'Application is still starting. Try refreshing in a moment.' });
      }
    };

    void poll();
  }, [applyDefaultPreviewUrl, commitAppUpdate, hasCustomPreviewUrl, reloadPreview, setLoading, setPreviewOverlay, setStatusMessage, stopLifecycleMonitor]);

  const activePreviewUrl = useMemo(() => bridgeState.href || previewUrl || '', [bridgeState.href, previewUrl]);
  const canCaptureScreenshot = useMemo(() => Boolean(activePreviewUrl), [activePreviewUrl]);
  const isPreviewSameOrigin = useMemo(() => {
    if (typeof window === 'undefined' || !activePreviewUrl) {
      return false;
    }

    try {
      const targetOrigin = new URL(activePreviewUrl, window.location.href).origin;
      return targetOrigin === window.location.origin;
    } catch (error) {
      logger.warn('Failed to evaluate preview origin', { activePreviewUrl, error });
      return false;
    }
  }, [activePreviewUrl]);
  const bridgeSupportsScreenshot = useMemo(
    () => bridgeState.isSupported && bridgeState.caps.includes('screenshot'),
    [bridgeState.caps, bridgeState.isSupported],
  );
  const localhostIssueMessage = useMemo(() => {
    if (!localhostReport) {
      return null;
    }
    const count = localhostReport.findings?.length ?? 0;
    if (count > 0) {
      return `${count} hard-coded localhost reference${count === 1 ? '' : 's'} detected. Update requests to use the proxy base.`;
    }
    return null;
  }, [localhostReport]);

  useEffect(() => {
    if (localhostIssueMessage) {
      setLocalhostMessageDismissed(false);
    }
  }, [localhostIssueMessage]);

  useEffect(() => {
    setIframeLoadedAt(null);
    setIframeLoadError(null);
  }, [previewUrl, previewReloadToken]);

  useEffect(() => {
    if (!previewUrl) {
      setPreviewOverlay(prev => {
        if (!prev) {
          return prev;
        }
        if (
          prev.type === 'waiting' && prev.message === PREVIEW_CONNECTING_LABEL
        ) {
          return null;
        }
        if (
          prev.type === 'error' &&
          (prev.message === PREVIEW_TIMEOUT_MESSAGE || prev.message === PREVIEW_MIXED_CONTENT_MESSAGE)
        ) {
          return null;
        }
        return prev;
      });
      return;
    }

    if (bridgeState.isReady || iframeLoadedAt) {
      setPreviewOverlay(prev => {
        if (prev && prev.type === 'waiting' && prev.message === PREVIEW_CONNECTING_LABEL) {
          return null;
        }
        return prev;
      });
      return;
    }

    let cancelled = false;
    let waitingApplied = false;

    setPreviewOverlay(prev => {
      if (prev && prev.type === 'restart') {
        return prev;
      }
      waitingApplied = true;
      return { type: 'waiting', message: PREVIEW_CONNECTING_LABEL };
    });

    const timeoutId = window.setTimeout(() => {
      if (cancelled || bridgeState.isReady || iframeLoadedAt) {
        return;
      }

      const isMixedContent =
        typeof window !== 'undefined' &&
        window.location.protocol === 'https:' &&
        previewUrl.startsWith('http://');

      const message = iframeLoadError
        ? iframeLoadError
        : isMixedContent
          ? PREVIEW_MIXED_CONTENT_MESSAGE
          : PREVIEW_TIMEOUT_MESSAGE;

      setPreviewOverlay(current => {
        if (current && current.type === 'restart') {
          return current;
        }
        return { type: 'error', message };
      });
    }, PREVIEW_LOAD_TIMEOUT_MS);

    return () => {
      cancelled = true;
      window.clearTimeout(timeoutId);
      if (waitingApplied) {
        setPreviewOverlay(prev => {
          if (prev && prev.type === 'waiting' && prev.message === PREVIEW_CONNECTING_LABEL) {
            return null;
          }
          return prev;
        });
      }
    };
  }, [previewUrl, previewReloadToken, bridgeState.isReady, iframeLoadedAt, iframeLoadError]);

  const urlStatusClass = useMemo(() => {
    if (!currentApp) {
      return 'unknown';
    }
    return currentApp.status?.toLowerCase() || 'unknown';
  }, [currentApp]);

  const urlStatusTitle = useMemo(() => {
    if (!currentApp) {
      return 'Status: Unknown';
    }
    const status = currentApp.status ?? 'Unknown';
    return `Status: ${status}`;
  }, [currentApp]);

  const isAppRunning = useMemo(() => (currentApp ? isRunningStatus(currentApp.status) : false), [currentApp]);
  const scenarioDisplayName = useMemo(() => {
    if (!currentApp) {
      return 'application';
    }
    return currentApp.name || currentApp.scenario_name || currentApp.id || 'application';
  }, [currentApp]);

  const toggleActionLabel = isAppRunning ? 'Stop scenario' : 'Start scenario';
  const restartActionLabel = `Restart ${scenarioDisplayName}`;
  const appStatusLabel = currentApp?.status ?? 'Unknown';
  const actionInProgress = pendingAction !== null;

  const bridgeIssueMessage = useMemo(() => {
    if (!bridgeState.isSupported || !bridgeCompliance || bridgeCompliance.ok) {
      return null;
    }
    const detail = bridgeCompliance.failures.join(', ');
    return `Preview bridge diagnostics failed (${detail}). History syncing may be unreliable.`;
  }, [bridgeCompliance, bridgeState.isSupported]);

  useEffect(() => {
    if (bridgeIssueMessage) {
      setBridgeMessageDismissed(false);
    }
  }, [bridgeIssueMessage]);

  const canGoBack = bridgeState.isSupported ? bridgeState.canGoBack : historyIndex > 0;
  const canGoForward = bridgeState.isSupported ? bridgeState.canGoForward : (historyIndex >= 0 && historyIndex < history.length - 1);
  const openPreviewTarget = bridgeState.isSupported && bridgeState.href ? bridgeState.href : previewUrl;

  useEffect(() => {
    if (!appId) {
      recordNavigateEvent({
        reason: 'missing-app-id',
        targetPath: '/apps',
        targetSearch: location.search || undefined,
        replace: true,
        locationState,
      });
      updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
      navigate({
        pathname: '/apps',
        search: location.search || undefined,
      }, { replace: true });
    }
  }, [appId, location.search, locationState, navigate, recordNavigateEvent, updatePreviewGuard]);

  useEffect(() => {
    setFetchAttempted(false);
    stopLifecycleMonitor();
    setPreviewOverlay(null);
  }, [appId, stopLifecycleMonitor]);

  useEffect(() => {
    return () => {
      stopLifecycleMonitor();
    };
  }, [stopLifecycleMonitor]);

  useEffect(() => {
    setHasCustomPreviewUrl(false);
    setHistory([]);
    setHistoryIndex(-1);
    complianceRunRef.current = false;
    setBridgeCompliance(null);
    resetState();
  }, [appId, resetState]);

  useEffect(() => {
    complianceRunRef.current = false;
    setBridgeCompliance(null);
  }, [previewUrl]);

  useEffect(() => {
    if (!appId) {
      return;
    }

    const located = locateAppByIdentifier(apps, appId);
    if (located) {
      setCurrentApp(located);
      if (!located.is_partial) {
        setLoading(false);
        return;
      }

      setStatusMessage('Loading application details...');
      setLoading(true);
    }

    if (fetchAttempted) {
      return;
    }

    setFetchAttempted(true);
    const fetchApp = async () => {
      try {
        const fetched = await appService.getApp(appId);
        if (fetched) {
          commitAppUpdate(fetched);
          setStatusMessage(null);
        } else {
          setStatusMessage('Application not found.');
        }
      } catch (error) {
        logger.error('Failed to load application for preview', error);
        setStatusMessage('Failed to load application details.');
      } finally {
        setLoading(false);
      }
    };

    fetchApp().catch((error) => {
      logger.error('Preview fetch failed', error);
    });
  }, [appId, apps, commitAppUpdate, fetchAttempted]);

  useEffect(() => {
    if (!currentApp) {
      resetPreviewState({ force: true });
      setStatusMessage('Loading application preview...');
      setLoading(true);
      return;
    }

    const nextUrl = buildPreviewUrl(currentApp);
    const hasPreviewCandidate = Boolean(nextUrl);
    const isExplicitlyStopped = isStoppedStatus(currentApp.status);

    if (hasPreviewCandidate) {
      const resolvedUrl = nextUrl as string;
      if (!hasCustomPreviewUrl) {
        applyDefaultPreviewUrl(resolvedUrl);
      } else if (previewUrl === null) {
        initialPreviewUrlRef.current = resolvedUrl;
        setPreviewUrl(resolvedUrl);
      }
    } else if (!hasCustomPreviewUrl) {
      resetPreviewState();
    }

    if (isExplicitlyStopped) {
      if (!hasCustomPreviewUrl) {
        resetPreviewState();
      }
      setLoading(false);
      setStatusMessage('Application is not running. Start it from the Applications view to access the UI preview.');
      return;
    }

    if (!hasPreviewCandidate) {
      if (currentApp.is_partial) {
        setStatusMessage('Loading application details...');
        setLoading(true);
      } else {
        setStatusMessage('This application does not expose a UI endpoint to preview.');
        setLoading(false);
      }
      return;
    }

    setLoading(false);

    if (currentApp.is_partial && !currentApp.status) {
      setStatusMessage('Loading application details...');
    } else {
      setStatusMessage(null);
    }
  }, [
    applyDefaultPreviewUrl,
    currentApp,
    hasCustomPreviewUrl,
    previewUrl,
    resetPreviewState,
  ]);

  useEffect(() => {
    if (!appId) {
      return;
    }

    const match = locateAppByIdentifier(apps, appId);
    if (match) {
      setCurrentApp(match);
    }
  }, [apps, appId]);

  useEffect(() => {
    const storeMatch = appId ? locateAppByIdentifier(apps, appId) : null;
    const snapshot = {
      appId,
      hasCurrentApp: Boolean(currentApp),
      currentStatus: currentApp?.status ?? null,
      currentIsPartial: currentApp?.is_partial ?? null,
      appsCount: apps.length,
      hasStoreMatch: Boolean(storeMatch),
      matchIsPartial: storeMatch?.is_partial ?? null,
      locationKey: location.key,
      locationState,
    };
    const snapshotKey = JSON.stringify(snapshot);
    if (snapshotKey !== lastStateSnapshotRef.current) {
      lastStateSnapshotRef.current = snapshotKey;
      recordDebugEvent('preview-state', snapshot);
    }
  }, [appId, apps, currentApp, location.key, locationState, recordDebugEvent]);

  useEffect(() => {
    const appIdentifier = currentApp?.id;
    if (!appIdentifier) {
      setProxyMetadata(null);
      setLocalhostReport(null);
      return;
    }

    let cancelled = false;
    setProxyMetadata(prev => (prev && prev.appId === appIdentifier ? prev : null));
    setLocalhostReport(prev => {
      if (!prev) {
        return prev;
      }
      return prev.scenario === appIdentifier ? prev : null;
    });

    const loadDiagnostics = async () => {
      const [metadata, localhostDiagnostics] = await Promise.all([
        appService.getAppProxyMetadata(appIdentifier),
        appService.getAppLocalhostReport(appIdentifier),
      ]);

      if (!cancelled) {
        setProxyMetadata(metadata);
        setLocalhostReport(localhostDiagnostics);
      }
    };

    loadDiagnostics().catch((error) => {
      logger.warn('Failed to load proxy diagnostics', error);
    });

    return () => {
      cancelled = true;
    };
  }, [currentApp?.id]);

  useEffect(() => {
    if (!appId) {
      lastRecordedViewRef.current = { id: null, timestamp: 0 };
      return;
    }

    const now = Date.now();
    const { id: lastId, timestamp } = lastRecordedViewRef.current;
    if (lastId === appId && now - timestamp < 1000) {
      return;
    }

    lastRecordedViewRef.current = { id: appId, timestamp: now };

    void (async () => {
      const stats = await appService.recordAppView(appId);
      if (!stats) {
        return;
      }

      const targets = [appId, stats.scenario_name];

      setAppsState(prev => prev.map(app => {
        if (!targets.some(target => matchesAppIdentifier(app, target))) {
          return app;
        }

        return {
          ...app,
          view_count: stats.view_count,
          last_viewed_at: stats.last_viewed_at ?? app.last_viewed_at ?? null,
          first_viewed_at: stats.first_viewed_at ?? app.first_viewed_at ?? null,
        };
      }));

      setCurrentApp(prev => {
        if (!prev) {
          return prev;
        }

        if (!targets.some(target => matchesAppIdentifier(prev, target))) {
          return prev;
        }

        return {
          ...prev,
          view_count: stats.view_count,
          last_viewed_at: stats.last_viewed_at ?? prev.last_viewed_at ?? null,
          first_viewed_at: stats.first_viewed_at ?? prev.first_viewed_at ?? null,
        };
      });
    })();
  }, [appId, matchesAppIdentifier, setAppsState, setCurrentApp]);

  useEffect(() => {
    if (!bridgeState.isSupported || !bridgeState.isReady || !bridgeState.href) {
      return;
    }
    if (complianceRunRef.current) {
      return;
    }

    let cancelled = false;
    complianceRunRef.current = true;
    runComplianceCheck()
      .then(result => {
        if (!cancelled) {
          setBridgeCompliance(result);
        }
      })
      .catch(error => {
        logger.warn('Bridge compliance check failed', error);
        if (!cancelled) {
          setBridgeCompliance({ ok: false, failures: ['CHECK_FAILED'], checkedAt: Date.now() });
        }
      });

    return () => {
      cancelled = true;
    };
  }, [bridgeState.href, bridgeState.isReady, bridgeState.isSupported, runComplianceCheck]);

  const executeAppAction = useCallback(async (appToControl: string, action: 'start' | 'stop' | 'restart') => {
    setPendingAction(action);
    const actionInProgressMessage = action === 'stop'
      ? 'Stopping application...'
      : action === 'start'
        ? 'Starting application...'
        : 'Restarting application...';
    setStatusMessage(actionInProgressMessage);

    try {
      const success = await appService.controlApp(appToControl, action);
      if (!success) {
        setStatusMessage(`Unable to ${action} the application. Check logs for details.`);
        return false;
      }

      const timestamp = new Date().toISOString();
      if (action === 'start' || action === 'stop') {
        const nextStatus: App['status'] = action === 'stop' ? 'stopped' : 'running';
        setAppsState(prev => prev.map(app => (app.id === appToControl ? { ...app, status: nextStatus, updated_at: timestamp } : app)));
        setCurrentApp(prev => (prev && prev.id === appToControl ? { ...prev, status: nextStatus, updated_at: timestamp } : prev));
        setStatusMessage(action === 'stop'
          ? 'Application stopped. Start it again to relaunch the UI preview.'
          : 'Application started. Preview will refresh automatically.');
      } else {
        setStatusMessage('Restart command sent. Waiting for application to return...');
      }

      return true;
    } catch (error) {
      logger.error(`Failed to ${action} app ${appToControl}`, error);
      setStatusMessage(`Unable to ${action} the application. Check logs for details.`);
      return false;
    } finally {
      setPendingAction(null);
    }
  }, [setAppsState]);

  const handleAppAction = useCallback(async (appToControl: string, action: 'start' | 'stop' | 'restart') => {
    if (action === 'restart') {
      setPreviewOverlay({ type: 'restart', message: 'Restarting application...' });
      setLoading(true);
      reloadPreview();
    } else if (action === 'start') {
      setPreviewOverlay({ type: 'waiting', message: 'Waiting for application to start...' });
      setLoading(true);
    }

    const success = await executeAppAction(appToControl, action);
    if (!success) {
      if (action === 'start') {
        setPreviewOverlay({ type: 'error', message: 'Unable to start the application. Check logs for details.' });
        setLoading(false);
      } else if (action === 'restart') {
        setPreviewOverlay({ type: 'error', message: 'Unable to restart the application. Check logs for details.' });
        setLoading(false);
      }
      return;
    }

    if (action === 'start') {
      beginLifecycleMonitor(appToControl, 'start');
    } else if (action === 'restart') {
      setPreviewOverlay({ type: 'waiting', message: 'Waiting for application to restart...' });
      beginLifecycleMonitor(appToControl, 'restart');
    } else if (action === 'stop') {
      setPreviewOverlay(prev => (prev && prev.type === 'waiting' ? null : prev));
      setLoading(false);
    }
  }, [beginLifecycleMonitor, executeAppAction, reloadPreview, setLoading]);

  const handleToggleApp = useCallback(() => {
    if (!currentApp || pendingAction) {
      return;
    }

    const action: 'start' | 'stop' = isRunningStatus(currentApp.status) ? 'stop' : 'start';
    void handleAppAction(currentApp.id, action);
  }, [currentApp, handleAppAction, pendingAction]);

  const handleRestartApp = useCallback(() => {
    if (!currentApp || pendingAction || !isRunningStatus(currentApp.status)) {
      return;
    }
    void handleAppAction(currentApp.id, 'restart');
  }, [currentApp, handleAppAction, pendingAction]);

  const applyPreviewUrlInput = useCallback(() => {
    const trimmed = previewUrlInput.trim();

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

    if (bridgeState.isSupported) {
      try {
        const reference = bridgeState.href || previewUrl || window.location.href;
        const resolved = new URL(trimmed, reference);
        if (!childOrigin || resolved.origin === childOrigin) {
          const sent = sendBridgeNav('GO', resolved.href);
          if (sent) {
            setStatusMessage(null);
            return;
          }
        }
      } catch (error) {
        logger.warn('Bridge navigation failed to parse URL', error);
      }
    }

    setHasCustomPreviewUrl(true);
    setPreviewUrl(trimmed);
    initialPreviewUrlRef.current = trimmed;
    resetState();
    setStatusMessage(null);
    const baseHistory = historyIndex >= 0 ? history.slice(0, historyIndex + 1) : [];
    if (baseHistory[baseHistory.length - 1] === trimmed) {
      setHistory(baseHistory);
      setHistoryIndex(baseHistory.length - 1);
    } else {
      const updatedHistory = [...baseHistory, trimmed];
      setHistory(updatedHistory);
      setHistoryIndex(updatedHistory.length - 1);
    }
  }, [bridgeState.href, bridgeState.isSupported, childOrigin, history, historyIndex, previewUrl, previewUrlInput, resetState, sendBridgeNav]);

  const handleUrlInputChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    setPreviewUrlInput(event.target.value);
  }, []);

  const handleUrlInputKeyDown = useCallback((event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      event.preventDefault();
      applyPreviewUrlInput();
    }
  }, [applyPreviewUrlInput]);

  const handleUrlInputBlur = useCallback(() => {
    applyPreviewUrlInput();
  }, [applyPreviewUrlInput]);

  const handleRefresh = useCallback(() => {
    if (!appId) {
      return;
    }

    const requestId = Date.now();
    lastRefreshRequestRef.current = requestId;

    setPreviewOverlay(null);
    setLoading(true);
    setStatusMessage('Refreshing application status...');

    if (previewUrl || bridgeState.href || hasCustomPreviewUrl) {
      reloadPreview();
    }

    appService.getApp(appId)
      .then(fetched => {
        if (lastRefreshRequestRef.current !== requestId) {
          return;
        }

        if (fetched) {
          commitAppUpdate(fetched);
          setStatusMessage(null);
        } else {
          setStatusMessage('Application not found.');
        }
      })
      .catch(error => {
        if (lastRefreshRequestRef.current !== requestId) {
          return;
        }
        logger.error('Failed to refresh application preview', error);
        setStatusMessage('Failed to refresh application preview.');
      })
      .finally(() => {
        if (lastRefreshRequestRef.current === requestId) {
          setLoading(false);
        }
      });
  }, [appId, bridgeState.href, commitAppUpdate, hasCustomPreviewUrl, previewUrl, reloadPreview]);

  const handleGoBack = useCallback(() => {
    updatePreviewGuard({ active: false, key: null });
    if (bridgeState.isSupported) {
      sendBridgeNav('BACK');
      return;
    }

    if (historyIndex <= 0) {
      return;
    }

    const targetIndex = historyIndex - 1;
    const targetUrl = history[targetIndex];
    setHistoryIndex(targetIndex);
    setPreviewUrl(targetUrl);
    setPreviewUrlInput(targetUrl);
    setHasCustomPreviewUrl(true);
    setStatusMessage(null);
  }, [bridgeState.isSupported, history, historyIndex, sendBridgeNav, updatePreviewGuard]);

  const handleGoForward = useCallback(() => {
    if (bridgeState.isSupported) {
      sendBridgeNav('FWD');
      return;
    }

    if (historyIndex === -1 || historyIndex >= history.length - 1) {
      return;
    }

    const targetIndex = historyIndex + 1;
    const targetUrl = history[targetIndex];
    setHistoryIndex(targetIndex);
    setPreviewUrl(targetUrl);
    setPreviewUrlInput(targetUrl);
    setHasCustomPreviewUrl(true);
    setStatusMessage(null);
  }, [bridgeState.isSupported, history, historyIndex, sendBridgeNav]);

  const handleOpenPreviewInNewTab = useCallback((event: MouseEvent<HTMLButtonElement>) => {
    const target = bridgeState.isSupported && bridgeState.href ? bridgeState.href : previewUrl;
    if (!target) {
      return;
    }

    event.preventDefault();
    window.open(target, '_blank', 'noopener,noreferrer');
  }, [bridgeState.href, bridgeState.isSupported, previewUrl]);

  const handleViewLogs = useCallback(() => {
    if (currentApp) {
      recordNavigateEvent({
        reason: 'toolbar-view-logs',
        targetPath: `/logs/${currentApp.id}`,
      });
      updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
      navigate(`/logs/${currentApp.id}`);
    }
  }, [currentApp, navigate, recordNavigateEvent, updatePreviewGuard]);

  const handleIframeLoad = useCallback(() => {
    setIframeLoadError(null);
    setIframeLoadedAt(Date.now());
    setPreviewOverlay(prev => {
      if (!prev) {
        return prev;
      }

      if (prev.type === 'waiting' && prev.message === PREVIEW_CONNECTING_LABEL) {
        return null;
      }

      if (
        prev.type === 'error' &&
        (prev.message === PREVIEW_TIMEOUT_MESSAGE || prev.message === PREVIEW_MIXED_CONTENT_MESSAGE)
      ) {
        return null;
      }

      return prev;
    });
  }, []);

  const handleIframeError = useCallback(() => {
    const isMixedContent =
      typeof window !== 'undefined' &&
      window.location.protocol === 'https:' &&
      previewUrl?.startsWith('http://');

    const message = isMixedContent
      ? PREVIEW_MIXED_CONTENT_MESSAGE
      : 'Preview failed to load. Verify the application UI is reachable from the App Monitor host.';

    setIframeLoadedAt(null);
    setIframeLoadError(message);
    setPreviewOverlay(current => {
      if (current && current.type === 'restart') {
        return current;
      }
      return { type: 'error', message };
    });
  }, [previewUrl]);

  const handleOpenReportDialog = useCallback(() => {
    setReportDialogOpen(true);
  }, []);

  const handleCloseReportDialog = useCallback(() => {
    setReportDialogOpen(false);
  }, []);



  return (
    <div className="app-preview-view">
      <AppPreviewToolbar
        canGoBack={canGoBack}
        canGoForward={canGoForward}
        onGoBack={handleGoBack}
        onGoForward={handleGoForward}
        onRefresh={handleRefresh}
        isRefreshing={loading}
        onOpenDetails={() => setModalOpen(true)}
        previewUrlInput={previewUrlInput}
        onPreviewUrlInputChange={handleUrlInputChange}
        onPreviewUrlInputBlur={handleUrlInputBlur}
        onPreviewUrlInputKeyDown={handleUrlInputKeyDown}
        onOpenInNewTab={handleOpenPreviewInNewTab}
        openPreviewTarget={openPreviewTarget}
        urlStatusClass={urlStatusClass}
        urlStatusTitle={urlStatusTitle}
        hasCurrentApp={Boolean(currentApp)}
        isAppRunning={isAppRunning}
        pendingAction={pendingAction}
        actionInProgress={actionInProgress}
        toggleActionLabel={toggleActionLabel}
        onToggleApp={handleToggleApp}
        restartActionLabel={restartActionLabel}
        onRestartApp={handleRestartApp}
        onViewLogs={handleViewLogs}
        onReportIssue={handleOpenReportDialog}
        appStatusLabel={appStatusLabel}
      />

      {bridgeIssueMessage && !bridgeMessageDismissed && (
        <div className="preview-status" role="status">
          <span className="preview-status__message">{bridgeIssueMessage}</span>
          <button
            type="button"
            className="preview-status__dismiss"
            onClick={() => setBridgeMessageDismissed(true)}
            aria-label="Dismiss bridge diagnostics message"
          >
            <X aria-hidden size={16} />
          </button>
        </div>
      )}

      {localhostIssueMessage && !localhostMessageDismissed && (
        <div className="preview-status" role="status">
          <span className="preview-status__message">{localhostIssueMessage}</span>
          <button
            type="button"
            className="preview-status__dismiss"
            onClick={() => setLocalhostMessageDismissed(true)}
            aria-label="Dismiss localhost reference warning"
          >
            <X aria-hidden size={16} />
          </button>
        </div>
      )}

      {previewUrl ? (
        <div className="preview-iframe-container" ref={previewContainerRef}>
          <iframe
            key={previewReloadToken}
            src={previewUrl}
            title={`${currentApp?.name ?? 'Application'} preview`}
            className="preview-iframe"
            loading="lazy"
            ref={iframeRef}
            onLoad={handleIframeLoad}
            onError={handleIframeError}
          />
          {previewOverlay && (
            <div
              className={clsx('preview-iframe-overlay', `preview-iframe-overlay--${previewOverlay.type}`)}
              aria-live="polite"
            >
              {(previewOverlay.type === 'restart' || previewOverlay.type === 'waiting') ? (
                <Loader2 aria-hidden size={26} className="spinning" />
              ) : (
                <Info aria-hidden size={26} />
              )}
              <span>{previewOverlay.message}</span>
            </div>
          )}
        </div>
      ) : (
        <div className="preview-placeholder">
          {loading ? 'Fetching application details…' : statusMessage ?? 'Preview unavailable.'}
        </div>
      )}

      {modalOpen && currentApp && (
        <AppModal
          app={currentApp}
          isOpen={modalOpen}
          onClose={() => setModalOpen(false)}
          onAction={handleAppAction}
          onViewLogs={(appIdentifier) => {
            setModalOpen(false);
            recordNavigateEvent({
              reason: 'modal-view-logs',
              targetPath: `/logs/${appIdentifier}`,
            });
            updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
            navigate(`/logs/${appIdentifier}`);
          }}
          proxyMetadata={proxyMetadata}
          localhostReport={localhostReport}
          previewUrl={activePreviewUrl || null}
        />
      )}

      {reportDialogOpen && (
        <ReportIssueDialog
          isOpen={reportDialogOpen}
          onClose={handleCloseReportDialog}
          appId={currentApp?.id ?? appId ?? undefined}
          app={currentApp}
          activePreviewUrl={activePreviewUrl || null}
          canCaptureScreenshot={canCaptureScreenshot}
          previewContainerRef={previewContainerRef}
          iframeRef={iframeRef}
          isPreviewSameOrigin={isPreviewSameOrigin}
          bridgeSupportsScreenshot={bridgeSupportsScreenshot}
          requestScreenshot={requestScreenshot}
          bridgeState={bridgeState}
          logState={logState}
          configureLogs={configureLogs}
          getRecentLogs={getRecentLogs}
          requestLogBatch={requestLogBatch}
          networkState={networkState}
          configureNetwork={configureNetwork}
          getRecentNetworkEvents={getRecentNetworkEvents}
          requestNetworkBatch={requestNetworkBatch}
          bridgeCompliance={bridgeCompliance}
        />
      )}
    </div>
  );
};

export default AppPreviewView;
