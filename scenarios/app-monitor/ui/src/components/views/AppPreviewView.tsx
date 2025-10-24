import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { MouseEvent } from 'react';
import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import clsx from 'clsx';
import { Info, Loader2, X } from 'lucide-react';
import { appService } from '@/services/api';
import { useAutoNextScenario } from '@/hooks/useAutoNextScenario';
import { useAppsStore } from '@/state/appsStore';
import { useScenarioEngagementStore } from '@/state/scenarioEngagementStore';
import { logger } from '@/services/logger';
import type { App, AppProxyMetadata, LocalhostUsageReport } from '@/types';
import { useShellOverlayStore } from '@/state/shellOverlayStore';
import { useSurfaceMediaStore } from '@/state/surfaceMediaStore';
import AppModal from '../AppModal';
import AppPreviewToolbar from '../AppPreviewToolbar';
import ReportIssueDialog from '../report/ReportIssueDialog';
import type { ReportElementCapture } from '../report/reportTypes';
import PreviewInspectorPanel from './PreviewInspectorPanel';
import usePreviewInspector from './usePreviewInspector';
import usePreviewNavigation from './usePreviewNavigation';
import {
  buildPreviewUrl,
  buildProxyPreviewUrl,
  isRunningStatus,
  isStoppedStatus,
  locateAppByIdentifier,
  normalizeIdentifier,
  resolveAppIdentifier,
} from '@/utils/appPreview';
import { useIframeBridge } from '@/hooks/useIframeBridge';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import { useDeviceEmulation } from '@/hooks/useDeviceEmulation';
import DeviceEmulationToolbar from '../device-emulation/DeviceEmulationToolbar';
import DeviceEmulationViewport from '../device-emulation/DeviceEmulationViewport';
import DeviceVisionFilterDefs from '../device-emulation/DeviceVisionFilterDefs';
import { useAppLogs } from '@/hooks/useAppLogs';
import AppLogsPanel from '../logs/AppLogsPanel';
import { useIosAutobackGuard, type PreviewLocationState } from './useIosAutobackGuard';
import { usePreviewCapture } from './usePreviewCapture';
import './AppPreviewView.css';

const PREVIEW_LOAD_TIMEOUT_MS = 6000;
const PREVIEW_WAITING_DELAY_MS = 400;
const PREVIEW_CONNECTING_LABEL = 'Connecting to preview...';
const PREVIEW_TIMEOUT_MESSAGE = 'Preview did not respond. Ensure the application UI is running and reachable from App Monitor.';
const PREVIEW_MIXED_CONTENT_MESSAGE = 'Preview blocked: browser refused to load HTTP content inside an HTTPS dashboard. Expose the UI through the tunnel hostname or enable HTTPS for the scenario.';
const PREVIEW_NO_UI_MESSAGE = 'This application does not expose a UI endpoint to preview.';
const IOS_AUTOBACK_GUARD_MS = 15000;
const AUTO_NEXT_PREPARE_DELAY_MS = 4500;
const AppPreviewView = () => {
  const apps = useAppsStore(state => state.apps);
  const setAppsState = useAppsStore(state => state.setAppsState);
  const loadApps = useAppsStore(state => state.loadApps);
  const loadingInitial = useAppsStore(state => state.loadingInitial);
  const hasInitialized = useAppsStore(state => state.hasInitialized);
  const canOpenTabsOverlay = apps.length > 0;
  const { prepareAutoNext } = useAutoNextScenario();
  const navigate = useNavigate();
  const { appId } = useParams<{ appId: string }>();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const activeOverlay = useShellOverlayStore(state => state.activeView);
  const registerOverlayHost = useShellOverlayStore(state => state.registerHost);
  const setSurfaceScreenshot = useSurfaceMediaStore(state => state.setScreenshot);
  const beginScenarioSession = useScenarioEngagementStore(state => state.beginSession);
  const endScenarioSession = useScenarioEngagementStore(state => state.endSession);
  const locationState: PreviewLocationState | null = location.state && typeof location.state === 'object'
    ? location.state as PreviewLocationState
    : null;
  const autoSelectedFromTabs = Boolean(locationState?.autoSelected);
  const overlayQuery = searchParams.get('overlay');
  const [isLogsPanelOpen, setIsLogsPanelOpen] = useState(() => overlayQuery === 'logs');
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
  const autoNextPrepareTimerRef = useRef<number | null>(null);
  useEffect(() => {
    setIsLogsPanelOpen(overlayQuery === 'logs');
  }, [overlayQuery]);

  const [currentApp, setCurrentApp] = useState<App | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [previewUrlInput, setPreviewUrlInput] = useState('');
  const [hasCustomPreviewUrl, setHasCustomPreviewUrl] = useState(false);
  const [history, setHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [statusMessage, setStatusMessage] = useState<string | null>('Loading application preview...');
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [fetchAttempted, setFetchAttempted] = useState(false);
  const [pendingAction, setPendingAction] = useState<null | 'start' | 'stop' | 'restart'>(null);
  const [reportDialogOpen, setReportDialogOpen] = useState(false);
  const [reportElementCaptures, setReportElementCaptures] = useState<ReportElementCapture[]>([]);
  const [hasPrimaryCaptureDraft, setHasPrimaryCaptureDraft] = useState(false);
  const [previewReloadToken, setPreviewReloadToken] = useState(0);
  const [previewInteractionSignal, setPreviewInteractionSignal] = useState(0);
  const [previewOverlay, setPreviewOverlay] = useState<null | { type: 'restart' | 'waiting' | 'error'; message: string }>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isLayoutFullscreen, setIsLayoutFullscreen] = useState(false);
  const [iframeLoadedAt, setIframeLoadedAt] = useState<number | null>(null);
  const [iframeLoadError, setIframeLoadError] = useState<string | null>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const previewViewRef = useRef<HTMLDivElement | null>(null);
  const [previewViewNode, setPreviewViewNode] = useState<HTMLDivElement | null>(null);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const [previewContainerNode, setPreviewContainerNode] = useState<HTMLDivElement | null>(null);
  const deviceEmulation = useDeviceEmulation({ container: previewContainerNode });
  const {
    isActive: isDeviceEmulationActive,
    toggleActive: toggleDeviceEmulation,
    toolbar: deviceToolbar,
    viewport: deviceViewport,
  } = deviceEmulation;
  const currentAppIdentifier = useMemo(() => {
    if (currentApp) {
      const resolved = resolveAppIdentifier(currentApp) ?? currentApp.id;
      if (resolved && resolved.trim().length > 0) {
        return resolved.trim();
      }
    }
    if (appId && appId.trim().length > 0) {
      return appId.trim();
    }
    return null;
  }, [appId, currentApp]);
  const [bridgeCompliance, setBridgeCompliance] = useState<BridgeComplianceResult | null>(null);
  const [proxyMetadata, setProxyMetadata] = useState<AppProxyMetadata | null>(null);
  const [localhostReport, setLocalhostReport] = useState<LocalhostUsageReport | null>(null);
  const [bridgeMessageDismissed, setBridgeMessageDismissed] = useState(false);
  const complianceRunRef = useRef(false);
  const initialPreviewUrlRef = useRef<string | null>(null);
  const syncFromBridgeRef = useRef<(href: string | null) => void>(() => {});
  const restartMonitorRef = useRef<{ cancel: () => void } | null>(null);
  const lastRefreshRequestRef = useRef(0);
  const lastRecordedViewRef = useRef<{ id: string | null; timestamp: number }>({ id: null, timestamp: 0 });
  const lastStateSnapshotRef = useRef<string>('');

  const scenarioDisplayName = useMemo(() => {
    const fallback = appId?.trim() || 'Scenario';
    if (!currentApp) {
      return fallback;
    }
    const preferred = (currentApp.scenario_name ?? currentApp.name ?? '').trim();
    return preferred.length > 0 ? preferred : fallback;
  }, [appId, currentApp]);

  const scenarioStoppedMessage = useMemo(
    () => `${scenarioDisplayName} is not running`,
    [scenarioDisplayName],
  );

  const isExplicitlyStopped = useMemo(() => {
    if (!currentApp) {
      return false;
    }

    if (currentApp.is_partial) {
      return false;
    }

    return isStoppedStatus(currentApp.status);
  }, [currentApp]);

  const previewIdentifier = useMemo(() => {
    if (currentApp) {
      const resolved = resolveAppIdentifier(currentApp);
      if (resolved && resolved.trim().length > 0) {
        return resolved.trim();
      }
      if (typeof currentApp.id === 'string' && currentApp.id.trim().length > 0) {
        return currentApp.id.trim();
      }
    }

    if (appId && appId.trim().length > 0) {
      return appId.trim();
    }

    return null;
  }, [appId, currentApp]);

  const deterministicProxyUrl = useMemo(() => (
    previewIdentifier ? buildProxyPreviewUrl(previewIdentifier) : null
  ), [previewIdentifier]);

  const previousPreviewIdentifierRef = useRef<string | null>(null);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return undefined;
    }

    if (autoNextPrepareTimerRef.current !== null) {
      window.clearTimeout(autoNextPrepareTimerRef.current);
      autoNextPrepareTimerRef.current = null;
    }

    if (apps.length === 0 || !currentAppIdentifier) {
      return undefined;
    }

    const normalizedKey = normalizeIdentifier(currentAppIdentifier);
    if (!normalizedKey) {
      return undefined;
    }

    autoNextPrepareTimerRef.current = window.setTimeout(() => {
      prepareAutoNext({ apps, currentAppId: normalizedKey }).catch((error) => {
        console.warn('[appPreview] Failed to precompute auto-next scenario', error);
      });
    }, AUTO_NEXT_PREPARE_DELAY_MS);

    return () => {
      if (autoNextPrepareTimerRef.current !== null) {
        window.clearTimeout(autoNextPrepareTimerRef.current);
        autoNextPrepareTimerRef.current = null;
      }
    };
  }, [apps, currentAppIdentifier, prepareAutoNext]);

  const handleInspectorCaptureAdded = useCallback((capture: ReportElementCapture) => {
    setReportElementCaptures(prev => [...prev, capture]);
  }, []);

  const handleElementCaptureNoteChange = useCallback((captureId: string, note: string) => {
    setReportElementCaptures(prev => prev.map(capture => (
      capture.id === captureId ? { ...capture, note } : capture
    )));
  }, []);

  const handleRemoveElementCapture = useCallback((captureId: string) => {
    setReportElementCaptures(prev => prev.filter(capture => capture.id !== captureId));
  }, []);

  const handleResetElementCaptures = useCallback(() => {
    setReportElementCaptures([]);
  }, []);

  useEffect(() => {
    if (!appId) {
      return;
    }
    beginScenarioSession(appId, { viaAutoNext: autoSelectedFromTabs });
    return () => {
      endScenarioSession(appId);
    };
  }, [appId, autoSelectedFromTabs, beginScenarioSession, endScenarioSession]);

  useEffect(() => {
    const trimmed = appId?.trim() ?? '';
    if (!trimmed) {
      setCurrentApp(null);
      return;
    }

    setCurrentApp(prev => {
      if (!prev) {
        return prev;
      }

      const identifiers: Array<string | undefined> = [prev.id, prev.scenario_name, prev.name];
      const matches = identifiers.some(candidate => typeof candidate === 'string' && candidate.trim() === trimmed);
      return matches ? prev : null;
    });
  }, [appId]);

  useEffect(() => {
    const host = (isFullscreen || isLayoutFullscreen) ? previewViewNode : null;
    registerOverlayHost(host);
    return () => {
      registerOverlayHost(null);
    };
  }, [isFullscreen, isLayoutFullscreen, previewViewNode, registerOverlayHost]);

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

  const setLogsOverlayParam = useCallback((open: boolean) => {
    const current = searchParams.get('overlay');
    const next = new URLSearchParams(searchParams);
    if (open) {
      if (current === 'logs') {
        return;
      }
      next.set('overlay', 'logs');
    } else {
      if (current !== 'logs') {
        return;
      }
      next.delete('overlay');
    }
    setSearchParams(next, { replace: true });
  }, [searchParams, setSearchParams]);

  const toggleLogsPanel = useCallback((nextState?: boolean) => {
    const shouldOpen = typeof nextState === 'boolean' ? nextState : !isLogsPanelOpen;
    setIsLogsPanelOpen(shouldOpen);
    setLogsOverlayParam(shouldOpen);
    recordNavigateEvent({
      reason: shouldOpen ? 'logs-overlay-open' : 'logs-overlay-close',
      overlay: 'logs',
      currentAppId: currentApp?.id ?? null,
    });
  }, [currentApp?.id, isLogsPanelOpen, recordNavigateEvent, setLogsOverlayParam]);

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

  const disablePreviewGuard = useCallback(() => {
    updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
  }, [updatePreviewGuard]);

  useEffect(() => {
    if (!hasInitialized && !loadingInitial) {
      void loadApps();
    }
  }, [hasInitialized, loadApps, loadingInitial]);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const handleFullscreenChange = () => {
      setIsFullscreen(document.fullscreenElement === previewViewRef.current);
    };

    document.addEventListener('fullscreenchange', handleFullscreenChange);
    handleFullscreenChange();

    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
    };
  }, []);

  useEffect(() => {
    if (typeof document === 'undefined') {
      return () => {};
    }

    const className = 'app-preview-immersive';
    const { body } = document;
    if (isLayoutFullscreen) {
      body.classList.add(className);
    } else {
      body.classList.remove(className);
    }

    return () => {
      body.classList.remove(className);
    };
  }, [isLayoutFullscreen]);

  useEffect(() => {
    if (!isLayoutFullscreen) {
      return () => {};
    }

    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        setIsLayoutFullscreen(false);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [isLayoutFullscreen]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const handleWindowBlur = () => {
      setPreviewInteractionSignal(value => value + 1);
    };

    window.addEventListener('blur', handleWindowBlur);

    return () => {
      window.removeEventListener('blur', handleWindowBlur);
    };
  }, []);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    const iframe = iframeRef.current;
    if (!iframe) {
      return;
    }

    const handlePointerDown = () => {
      setPreviewInteractionSignal(value => value + 1);
    };

    iframe.addEventListener('pointerdown', handlePointerDown);

    return () => {
      iframe.removeEventListener('pointerdown', handlePointerDown);
    };
  }, [previewReloadToken, previewUrl]);

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

  useIosAutobackGuard({
    appId: appId ?? null,
    guardTtlMs: IOS_AUTOBACK_GUARD_MS,
    isIosSafari,
    lastAppIdRef,
    location,
    locationState,
    navigate,
    previewLocationRef,
    recordDebugEvent,
    recordNavigateEvent,
    updatePreviewGuard,
  });

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
    syncFromBridgeRef.current(message.href ?? null);
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
    inspectState,
    startInspect,
    stopInspect,
  } = useIframeBridge({
    iframeRef,
    previewUrl,
    onLocation: handleBridgeLocation,
  });
  const {
    canGoBack,
    canGoForward,
    handleUrlInputChange,
    handleUrlInputKeyDown,
    handleUrlInputBlur,
    handleGoBack,
    handleGoForward,
    resetPreviewState: resetNavigationState,
    applyDefaultPreviewUrl: applyNavigationDefaultPreviewUrl,
    syncFromBridge,
  } = usePreviewNavigation({
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
      isSupported: bridgeState.isSupported,
      href: bridgeState.href,
      canGoBack: bridgeState.canGoBack,
      canGoForward: bridgeState.canGoForward,
    },
    childOrigin,
    sendBridgeNav,
    resetBridgeState: resetState,
    setStatusMessage,
    onBeforeLocalNavigation: disablePreviewGuard,
  });

  useEffect(() => {
    syncFromBridgeRef.current = syncFromBridge;
  }, [syncFromBridge]);

  const resetPreviewState = useCallback((options?: { force?: boolean }) => {
    if (!options?.force && hasCustomPreviewUrl) {
      return;
    }
    resetNavigationState(options);
    setIframeLoadedAt(null);
    setIframeLoadError(null);
  }, [hasCustomPreviewUrl, resetNavigationState, setIframeLoadedAt, setIframeLoadError]);

  const applyDefaultPreviewUrl = useCallback((url: string) => {
    applyNavigationDefaultPreviewUrl(url);
  }, [applyNavigationDefaultPreviewUrl]);

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
        setPreviewOverlay({
          type: 'error',
          message: lifecycle === 'restart'
            ? 'Application has not come back online yet. Try refreshing.'
            : 'Application is still starting. Try refreshing in a moment.',
        });
      }
    };

    void poll();
  }, [applyDefaultPreviewUrl, commitAppUpdate, hasCustomPreviewUrl, reloadPreview, setLoading, setPreviewOverlay, setStatusMessage, stopLifecycleMonitor]);

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

  const inspector = usePreviewInspector({
    inspectState,
    startInspect,
    stopInspect,
    requestScreenshot,
    previewUrl,
    currentAppIdentifier,
    iframeRef,
    previewViewRef,
    previewViewNode,
    onCaptureAdd: handleInspectorCaptureAdded,
  });

  useEffect(() => {
    const identifier = previewIdentifier ?? null;
    if (previousPreviewIdentifierRef.current && previousPreviewIdentifierRef.current !== identifier) {
      inspector.handleInspectorDialogClose();
    }
    previousPreviewIdentifierRef.current = identifier;
  }, [previewIdentifier, inspector.handleInspectorDialogClose]);

  const resolvePreviewBackgroundColor = useCallback(() => {
    if (typeof window === 'undefined') {
      return undefined;
    }

    const iframe = iframeRef.current;
    if (iframe) {
      try {
        const iframeWindow = iframe.contentWindow;
        const iframeDocument = iframe.contentDocument ?? iframeWindow?.document ?? null;
        if (iframeDocument && iframeWindow) {
          const candidates: Element[] = [];
          if (iframeDocument.body) {
            candidates.push(iframeDocument.body);
          }
          if (iframeDocument.documentElement && iframeDocument.documentElement !== iframeDocument.body) {
            candidates.push(iframeDocument.documentElement);
          }
          for (const element of candidates) {
            const style = iframeWindow.getComputedStyle(element);
            if (!style) {
              continue;
            }
            const color = style.backgroundColor;
            if (color && color !== 'rgba(0, 0, 0, 0)' && color.toLowerCase() !== 'transparent') {
              return color;
            }
          }
        }
      } catch (error) {
        logger.debug('Unable to inspect iframe background color', error);
      }
    }

    if (previewViewRef.current) {
      try {
        const style = window.getComputedStyle(previewViewRef.current);
        const color = style.backgroundColor;
        if (color && color !== 'rgba(0, 0, 0, 0)' && color.toLowerCase() !== 'transparent') {
          return color;
        }
      } catch (error) {
        logger.debug('Unable to inspect preview container background color', error);
      }
    }

    return undefined;
  }, []);

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
  const logsState = useAppLogs({ app: currentApp, appId: appId ?? null, active: isLogsPanelOpen });
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
  const hasLocalhostWarning = Boolean(localhostIssueMessage);

  usePreviewCapture({
    activeOverlay,
    bridgeIsReady: bridgeState.isReady,
    bridgeLastReadyAt: bridgeState.lastReadyAt,
    bridgeSupportsScreenshot,
    canCaptureScreenshot,
    currentAppIdentifier,
    iframeLoadedAt,
    requestScreenshot,
    resolvePreviewBackgroundColor,
    setSurfaceScreenshot,
    surfaceType: 'app',
  });

  useEffect(() => {
    setIframeLoadedAt(null);
    setIframeLoadError(null);
  }, [previewUrl, previewReloadToken]);

  useEffect(() => {
    const preserveErrorOverlay = previewOverlay?.type === 'error' && (
      previewOverlay.message === PREVIEW_NO_UI_MESSAGE ||
      previewOverlay.message === scenarioStoppedMessage
    );

    if (preserveErrorOverlay) {
      return;
    }

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
    const waitingTimeoutId = window.setTimeout(() => {
      setPreviewOverlay(prev => {
        if (cancelled || bridgeState.isReady || iframeLoadedAt) {
          return prev;
        }
        if (prev && prev.type === 'restart') {
          return prev;
        }
        waitingApplied = true;
        return { type: 'waiting', message: PREVIEW_CONNECTING_LABEL };
      });
    }, PREVIEW_WAITING_DELAY_MS);
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
      window.clearTimeout(waitingTimeoutId);
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
  }, [
    previewUrl,
    previewReloadToken,
    bridgeState.isReady,
    iframeLoadedAt,
    iframeLoadError,
    previewOverlay,
    scenarioStoppedMessage,
  ]);

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
  const toggleActionLabel = isAppRunning ? 'Stop scenario' : 'Start scenario';
  const restartActionLabel = 'Restart scenario';
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
    setPreviewUrl(null);
    setPreviewUrlInput('');
    initialPreviewUrlRef.current = null;
    setIframeLoadedAt(null);
    setIframeLoadError(null);
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
      if (!hasCustomPreviewUrl) {
        if (deterministicProxyUrl && previewUrl !== deterministicProxyUrl) {
          resetPreviewState({ force: true });
          applyDefaultPreviewUrl(deterministicProxyUrl);
        } else if (!deterministicProxyUrl) {
          resetPreviewState({ force: true });
        }
      }
      setStatusMessage('Loading application preview...');
      setLoading(true);
      return;
    }

    const nextUrl = buildPreviewUrl(currentApp);
    const hasPreviewCandidate = Boolean(nextUrl);
    const defaultPreviewUrl = hasPreviewCandidate
      ? (nextUrl as string)
      : deterministicProxyUrl;

    if (!hasCustomPreviewUrl) {
      if (defaultPreviewUrl && previewUrl !== defaultPreviewUrl) {
        applyDefaultPreviewUrl(defaultPreviewUrl);
      } else if (!defaultPreviewUrl) {
        resetPreviewState();
      }
    } else if (hasPreviewCandidate && previewUrl === null) {
      const resolvedUrl = nextUrl as string;
      initialPreviewUrlRef.current = resolvedUrl;
      setPreviewUrl(resolvedUrl);
    }

    if (isExplicitlyStopped) {
      setLoading(false);
      setStatusMessage(scenarioStoppedMessage);
      setPreviewOverlay(prev => {
        if (prev && prev.type === 'restart') {
          return prev;
        }
        if (prev && prev.type === 'error' && prev.message === scenarioStoppedMessage) {
          return prev;
        }
        return { type: 'error', message: scenarioStoppedMessage };
      });
      return;
    }

    if (!hasPreviewCandidate) {
      if (currentApp.is_partial) {
        setStatusMessage('Loading application details...');
        setLoading(true);
        return;
      }

      setStatusMessage(PREVIEW_NO_UI_MESSAGE);
      setLoading(false);
      setPreviewOverlay(prev => {
        if (prev && prev.type === 'restart') {
          return prev;
        }
        if (prev && prev.type === 'error' && prev.message === PREVIEW_NO_UI_MESSAGE) {
          return prev;
        }
        return { type: 'error', message: PREVIEW_NO_UI_MESSAGE };
      });
      return;
    }

    setLoading(false);

    if (currentApp.is_partial && !currentApp.status) {
      setStatusMessage('Loading application details...');
    } else {
      setStatusMessage(null);
    }

    setPreviewOverlay(prev => {
      if (!prev) {
        return prev;
      }
      if (
        prev.type === 'error' &&
        (prev.message === scenarioStoppedMessage || prev.message === PREVIEW_NO_UI_MESSAGE)
      ) {
        return null;
      }
      return prev;
    });
  }, [
    applyDefaultPreviewUrl,
    currentApp,
    deterministicProxyUrl,
    hasCustomPreviewUrl,
    isExplicitlyStopped,
    previewUrl,
    scenarioStoppedMessage,
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
        const failureMessage = action === 'start'
          ? `Unable to start ${scenarioDisplayName}. Check logs for details.`
          : `Unable to ${action} the application. Check logs for details.`;
        setStatusMessage(failureMessage);
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
      const failureMessage = action === 'start'
        ? `Unable to start ${scenarioDisplayName}. Check logs for details.`
        : `Unable to ${action} the application. Check logs for details.`;
      setStatusMessage(failureMessage);
      return false;
    } finally {
      setPendingAction(null);
    }
  }, [scenarioDisplayName, setAppsState]);

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
        setPreviewOverlay({ type: 'error', message: `Unable to start ${scenarioDisplayName}. Check logs for details.` });
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
  }, [beginLifecycleMonitor, executeAppAction, reloadPreview, scenarioDisplayName, setLoading]);

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

  const handleOpenPreviewInNewTab = useCallback((event: MouseEvent<HTMLButtonElement>) => {
    const target = bridgeState.isSupported && bridgeState.href ? bridgeState.href : previewUrl;
    if (!target) {
      return;
    }

    event.preventDefault();
    // Keep the Referer header so the proxy can map shared asset URLs to the active scenario.
    window.open(target, '_blank', 'noopener');
  }, [bridgeState.href, bridgeState.isSupported, previewUrl]);

  const handleToggleLogsFromToolbar = useCallback(() => {
    updatePreviewGuard({ active: false, key: null, recoverPath: null, recoverState: null });
    toggleLogsPanel();
  }, [toggleLogsPanel, updatePreviewGuard]);

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

  const handleToggleFullscreen = useCallback(() => {
    if (typeof document === 'undefined') {
      return;
    }

    const container = previewViewRef.current;
    if (!container) {
      return;
    }

    if (document.fullscreenElement === container) {
      const exitFullscreen = typeof document.exitFullscreen === 'function'
        ? document.exitFullscreen.bind(document)
        : null;
      if (exitFullscreen) {
        exitFullscreen().catch(error => {
          logger.error('Failed to exit fullscreen preview', error);
        });
      }
      return;
    }

    if (isLayoutFullscreen) {
      setIsLayoutFullscreen(false);
      return;
    }

    const enterNativeFullscreen = () => {
      if (typeof container.requestFullscreen === 'function') {
        return container.requestFullscreen();
      }
      return Promise.reject(new Error('Fullscreen API unavailable'));
    };

    if (document.fullscreenElement && document.fullscreenElement !== container) {
      const exitFullscreen = typeof document.exitFullscreen === 'function'
        ? document.exitFullscreen.bind(document)
        : null;
      if (exitFullscreen) {
        exitFullscreen()
          .then(() => enterNativeFullscreen().catch(error => {
            logger.error('Failed to enter fullscreen preview after releasing existing element', error);
            setIsLayoutFullscreen(true);
          }))
          .catch(error => {
            logger.error('Failed to switch fullscreen element', error);
            setIsLayoutFullscreen(true);
          });
        return;
      }
      setIsLayoutFullscreen(true);
      return;
    }

    enterNativeFullscreen()
      .catch(error => {
        logger.error('Fullscreen API unavailable or failed; falling back to immersive layout', error);
        setIsLayoutFullscreen(true);
      });
  }, [isLayoutFullscreen]);

  const isFullView = isFullscreen || isLayoutFullscreen;
  const stagedCaptureCount = reportElementCaptures.length + (hasPrimaryCaptureDraft ? 1 : 0);

  return (
    <div
      className={clsx('app-preview-view', isLayoutFullscreen && 'app-preview-view--immersive')}
      ref={node => {
        previewViewRef.current = node;
        setPreviewViewNode(node);
      }}
    >
      <DeviceVisionFilterDefs />
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
        hasDetailsWarning={hasLocalhostWarning}
        hasCurrentApp={Boolean(currentApp)}
        isAppRunning={isAppRunning}
        pendingAction={pendingAction}
        actionInProgress={actionInProgress}
        toggleActionLabel={toggleActionLabel}
        onToggleApp={handleToggleApp}
        restartActionLabel={restartActionLabel}
        onRestartApp={handleRestartApp}
        onToggleLogs={handleToggleLogsFromToolbar}
        areLogsVisible={isLogsPanelOpen}
        onReportIssue={handleOpenReportDialog}
        appStatusLabel={appStatusLabel}
        isFullView={isFullView}
        onToggleFullView={handleToggleFullscreen}
        isDeviceEmulationActive={isDeviceEmulationActive}
        onToggleDeviceEmulation={toggleDeviceEmulation}
        canInspect={inspectState.supported}
        isInspecting={inspectState.active}
        onToggleInspect={inspector.handleToggleInspectMode}
        menuPortalContainer={previewViewNode}
        canOpenTabsOverlay={canOpenTabsOverlay}
        previewInteractionSignal={previewInteractionSignal}
        issueCaptureCount={stagedCaptureCount}
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

      {isDeviceEmulationActive && !isLogsPanelOpen && <DeviceEmulationToolbar {...deviceToolbar} />}

      <PreviewInspectorPanel
        inspectState={inspectState}
        previewUrl={previewUrl}
        inspector={inspector}
      />


      {isLogsPanelOpen ? (
        <div
          className={clsx('preview-logs-container', (isFullscreen || isLayoutFullscreen) && 'preview-logs-container--immersive')}
        >
          <AppLogsPanel
            app={currentApp}
            onClose={() => toggleLogsPanel(false)}
            {...logsState}
          />
        </div>
      ) : previewUrl ? (
        <div
          className={clsx('preview-iframe-container', isDeviceEmulationActive && 'preview-iframe-container--emulated')}
          ref={node => {
            previewContainerRef.current = node;
            setPreviewContainerNode(node);
          }}
        >
          {isDeviceEmulationActive ? (
            <DeviceEmulationViewport {...deviceViewport}>
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
            </DeviceEmulationViewport>
          ) : (
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
          )}
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
          {loading ? `Fetching ${scenarioDisplayName} details` : statusMessage ?? 'Preview unavailable.'}
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
          elementCaptures={reportElementCaptures}
          onElementCaptureNoteChange={handleElementCaptureNoteChange}
          onElementCaptureRemove={handleRemoveElementCapture}
          onElementCapturesReset={handleResetElementCaptures}
          onPrimaryCaptureDraftChange={setHasPrimaryCaptureDraft}
        />
      )}
    </div>
  );
};


export default AppPreviewView;
