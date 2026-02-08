import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { MouseEvent } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import clsx from 'clsx';
import { X } from 'lucide-react';
import ErrorBoundary, { SectionErrorFallback } from '@/components/ErrorBoundary';
import { appService } from '@/services/api';
import { useAutoNextScenario } from '@/hooks/useAutoNextScenario';
import { useAppsStore } from '@/state/appsStore';
import { useScenarioEngagementStore } from '@/state/scenarioEngagementStore';
import { logger } from '@/services/logger';
import type { App } from '@/types';
import { useShellOverlayStore } from '@/state/shellOverlayStore';
import { useSurfaceMediaStore } from '@/state/surfaceMediaStore';
import { usePrevious } from '@/hooks/usePrevious';
import AppModal from '../AppModal';
import AppPreviewToolbar from '../AppPreviewToolbar';
import ReportIssueDialog from '../report/ReportIssueDialog';
import PreviewInspectorPanel from './PreviewInspectorPanel';
import usePreviewInspector from './usePreviewInspector';
import {
  buildPreviewUrl,
  buildProxyPreviewUrl,
  isRunningStatus,
  isScenarioExplicitlyStopped,
  locateAppByIdentifier,
  normalizeIdentifier,
  resolveAppIdentifier,
} from '@/utils/appPreview';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import { useDeviceEmulation } from '@/hooks/useDeviceEmulation';
import DeviceEmulationToolbar from '../device-emulation/DeviceEmulationToolbar';
import DeviceEmulationViewport from '../device-emulation/DeviceEmulationViewport';
import DeviceVisionFilterDefs from '../device-emulation/DeviceVisionFilterDefs';
import { useAppLogs } from '@/hooks/useAppLogs';
import AppLogsPanel from '../logs/AppLogsPanel';
import { usePreviewCapture } from './usePreviewCapture';
import { useScheduledTimeout } from '@/hooks/useTimeout';
import { usePreviewOverlay } from '@/hooks/usePreviewOverlay';
import { usePreviewBackgroundColor } from '@/hooks/usePreviewBackgroundColor';
import { useAppLifecycleMonitor } from '@/hooks/useAppLifecycleMonitor';
import { usePreviewInteractionTracking } from '@/hooks/usePreviewInteractionTracking';
import { useProxyMetadataSynchronization } from '@/hooks/useProxyMetadataSynchronization';
import { useAppViewRecording } from '@/hooks/useAppViewRecording';
import { useAppDiagnostics } from '@/hooks/useAppDiagnostics';
import { useLighthouseHistory } from '@/hooks/useLighthouseHistory';
import { useAppCompleteness } from '@/hooks/useAppCompleteness';
import { useOverlayRouter } from '@/hooks/useOverlayRouter';
import { useKeyboardScope } from '@/hooks/useKeyboardScopes';
import { usePreviewAppLifecycle } from '@/hooks/usePreviewAppLifecycle';
import { usePreviewBridgeComplianceCheck } from '@/hooks/usePreviewBridgeComplianceCheck';
import { usePreviewIframeReadinessFallback } from '@/hooks/usePreviewIframeReadinessFallback';
import { usePreviewNavigationSession } from '@/hooks/usePreviewNavigationSession';
import { usePreviewReportSession } from '@/hooks/usePreviewReportSession';
import { usePreviewToolbarSession } from '@/hooks/usePreviewToolbarSession';
import { usePreviewUrlOrchestration } from '@/hooks/usePreviewUrlOrchestration';
import { PREVIEW_TIMEOUTS, PREVIEW_MESSAGES } from './previewConstants';
import type { PreviewLocationState } from '@/types/preview';
import { isPreviewLocationState } from '@/types/preview';
import { HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER, type BridgeShortcutIntent } from '@vrooli/iframe-bridge';
import { isPreviewDebugEventsEnabled } from '@/utils/previewDebug';
import PreviewFallbackState from '@/components/preview/PreviewFallbackState';
import './AppPreviewView.css';

const AppPreviewView = () => {
  const apps = useAppsStore(state => state.apps);
  const setAppsState = useAppsStore(state => state.setAppsState);
  const loadApps = useAppsStore(state => state.loadApps);
  const loadingInitial = useAppsStore(state => state.loadingInitial);
  const hasInitialized = useAppsStore(state => state.hasInitialized);
  const canOpenTabsOverlay = apps.length > 0;
  const { prepareAutoNext } = useAutoNextScenario();
  const navigate = useNavigate();
  const { openOverlay, closeOverlay } = useOverlayRouter();
  const { appId } = useParams<{ appId: string }>();
  const location = useLocation();
  const activeOverlay = useShellOverlayStore(state => state.activeView);
  const registerOverlayHost = useShellOverlayStore(state => state.registerHost);
  const setSurfaceScreenshot = useSurfaceMediaStore(state => state.setScreenshot);
  const beginScenarioSession = useScenarioEngagementStore(state => state.beginSession);
  const endScenarioSession = useScenarioEngagementStore(state => state.endSession);
  const locationState: PreviewLocationState | null = isPreviewLocationState(location.state)
    ? location.state
    : null;
  const autoSelectedFromTabs = Boolean(locationState?.autoSelected);
  const shouldOpenLogsFromQuery = useMemo(() => {
    const params = new URLSearchParams(location.search);
    return params.get('paneLogs') === '1';
  }, [location.search]);
  const [isLogsPanelOpen, setIsLogsPanelOpen] = useState(shouldOpenLogsFromQuery);
  const { schedule: scheduleAutoNextPrepare, clear: clearAutoNextPrepare } = useScheduledTimeout();
  useEffect(() => {
    if (shouldOpenLogsFromQuery) {
      setIsLogsPanelOpen(true);
    }
  }, [shouldOpenLogsFromQuery]);
  const [currentApp, setCurrentApp] = useState<App | null>(null);

  // Preview navigation state
  const [statusMessage, setStatusMessage] = useState<string | null>('Loading application preview...');
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [fetchAttempted, setFetchAttempted] = useState(false);
  const [reportDialogOpen, setReportDialogOpen] = useState(false);
  const [previewReloadToken, setPreviewReloadToken] = useState(0);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isLayoutFullscreen, setIsLayoutFullscreen] = useState(false);
  const [iframeLoadedAt, setIframeLoadedAt] = useState<number | null>(null);
  const [iframeLoadError, setIframeLoadError] = useState<string | null>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const shouldFocusPreviewOnLoadRef = useRef(Boolean(locationState?.fromAppsList));
  const previewViewRef = useRef<HTMLDivElement | null>(null);
  const [previewViewNode, setPreviewViewNode] = useState<HTMLDivElement | null>(null);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const [previewContainerNode, setPreviewContainerNode] = useState<HTMLDivElement | null>(null);
  const previousIsFullscreenRef = useRef(false);
  const deviceEmulation = useDeviceEmulation({
    container: previewContainerNode,
    storageNamespace: 'app-preview-view',
  });
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
  const [bridgeMessageDismissed, setBridgeMessageDismissed] = useState(false);
  const handleBridgeShortcut = useCallback((message: { intent: BridgeShortcutIntent }) => {
    const intent = message.intent;
    if (!intent || intent.action !== HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER) {
      return;
    }

    if (intent.outcome !== 'noop' && intent.outcome !== 'unhandled') {
      return;
    }

    if (activeOverlay === 'tabs') {
      return;
    }

    openOverlay('tabs');
  }, [activeOverlay, openOverlay]);
  const previewSession = usePreviewNavigationSession({
    iframeRef,
    setStatusMessage,
    onShortcut: handleBridgeShortcut,
  });
  const {
    bridge,
    previewUrl,
    setPreviewUrl,
    previewUrlInput,
    hasCustomPreviewUrl,
    initialPreviewUrlRef,
    clearNavigationSession,
    canGoBack,
    canGoForward,
    history,
    handleUrlInputChange,
    handleUrlInputKeyDown,
    handleUrlInputBlur,
    handleGoBack,
    handleGoForward,
    resetPreviewState: resetNavigationState,
    applyDefaultPreviewUrl: applyNavigationDefaultPreviewUrl,
    applyPreviewUrlValue,
  } = previewSession;
  const {
    state: bridgeState,
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
    setInspectTargetIndex,
    shiftInspectTarget,
  } = bridge;

  const focusPreviewIframe = useCallback((): boolean => {
    const iframe = iframeRef.current;
    if (!iframe) {
      return false;
    }

    try {
      iframe.focus({ preventScroll: true });
      iframe.contentWindow?.focus();
      return true;
    } catch (error) {
      logger.debug('Unable to focus preview iframe', error);
      return false;
    }
  }, []);
  const lastRefreshRequestRef = useRef(0);
  const lastStateSnapshotRef = useRef<string>('');

  // Use extracted hooks for cleaner code organization
  const { previewInteractionSignal } = usePreviewInteractionTracking({
    iframeRef,
    previewUrl,
    previewReloadToken,
  });

  const { proxyMetadata, localhostReport } = useProxyMetadataSynchronization({
    currentAppId: currentApp?.id ?? null,
  });

  useAppViewRecording({
    appId: appId ?? null,
    appSnapshot: currentApp,
    setAppsState,
    setCurrentApp,
  });

  // Preload diagnostics and Lighthouse history for the current app
  // This ensures data is ready when the user opens the app details modal
  const { diagnostics: preloadedDiagnostics, loading: diagnosticsLoading, error: diagnosticsError, refetch: refetchDiagnostics } = useAppDiagnostics(
    currentApp?.id ?? null,
    { enabled: true, refetchOnOpen: false }
  );

  const { history: preloadedLighthouseHistory, loading: lighthouseLoading, error: lighthouseError, refetch: refetchLighthouse } = useLighthouseHistory(
    currentApp?.id ?? null,
    { enabled: true }
  );

  const { completeness: preloadedCompleteness, loading: completenessLoading, error: completenessError, refetch: refetchCompleteness } = useAppCompleteness(
    currentApp?.id ?? null,
    { enabled: true, refetchOnOpen: false }
  );

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

  const isExplicitlyStopped = useMemo(
    () => isScenarioExplicitlyStopped(currentApp),
    [currentApp],
  );

  // Memoize currentApp properties to avoid unnecessary effect re-runs
  const currentAppForPreview = useMemo(() => {
    if (!currentApp) {
      return null;
    }
    // Only extract properties that affect preview URL generation
    return {
      id: currentApp.id,
      is_partial: currentApp.is_partial,
      status: currentApp.status,
      port: currentApp.port,
      port_mappings: currentApp.port_mappings,
      config: currentApp.config,
      environment: currentApp.environment,
    };
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

  // Only create a deterministic proxy URL if the app actually has a UI port
  // Otherwise we'd create a URL pointing to a non-existent UI
  const deterministicProxyUrl = useMemo(() => {
    if (!previewIdentifier) return null;
    if (!currentApp) return null;

    // Check if this app actually has a UI port
    const previewUrl = buildPreviewUrl(currentApp);
    if (!previewUrl) return null;  // No UI port, don't create proxy URL

    return buildProxyPreviewUrl(previewIdentifier);
  }, [previewIdentifier, currentApp]);

  // Track previous values for dialog cleanup logic
  const previousAppId = usePrevious(appId);
  const previousPreviewIdentifier = usePrevious(previewIdentifier);
  const previousReportScenario = usePrevious(currentAppIdentifier);

  useEffect(() => {
    clearAutoNextPrepare();

    if (apps.length === 0 || !currentAppIdentifier) {
      return;
    }

    const normalizedKey = normalizeIdentifier(currentAppIdentifier);
    if (!normalizedKey) {
      return;
    }

    scheduleAutoNextPrepare(() => {
      prepareAutoNext({ apps, currentAppId: normalizedKey }).catch((error) => {
        logger.warn('[appPreview] Failed to precompute auto-next scenario', error);
      });
    }, PREVIEW_TIMEOUTS.AUTO_NEXT_PREPARE);
  }, [apps, currentAppIdentifier, prepareAutoNext, scheduleAutoNextPrepare, clearAutoNextPrepare]);

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

      // Check if current app still matches the appId
      const stillMatches = locateAppByIdentifier([prev], trimmed) !== null;
      return stillMatches ? prev : null;
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
      if (!isPreviewDebugEventsEnabled()) {
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
    } catch {
      // Best-effort debug logging; ignore failures
    }
  }, []);

  const recordNavigateEvent = useCallback((info: Record<string, unknown>) => {
    recordDebugEvent('navigate-event', {
      appId,
      ...info,
    });
  }, [appId, recordDebugEvent]);

  const toggleLogsPanel = useCallback((nextState?: boolean) => {
    const shouldOpen = typeof nextState === 'boolean' ? nextState : !isLogsPanelOpen;
    setIsLogsPanelOpen(shouldOpen);
    recordNavigateEvent({
      reason: shouldOpen ? 'logs-overlay-open' : 'logs-overlay-close',
      currentAppId: currentApp?.id ?? null,
    });
  }, [currentApp?.id, isLogsPanelOpen, recordNavigateEvent]);

  useEffect(() => {
    if (!hasInitialized && !loadingInitial) {
      void loadApps();
    }
  }, [hasInitialized, loadApps, loadingInitial]);

  // Consolidated fullscreen management: API listener, layout class, and keyboard handler
  useEffect(() => {
    if (typeof document === 'undefined' || typeof window === 'undefined') {
      return () => {};
    }

    // Track native fullscreen API state
    const handleFullscreenChange = () => {
      setIsFullscreen(document.fullscreenElement === previewViewRef.current);
    };
    document.addEventListener('fullscreenchange', handleFullscreenChange);
    handleFullscreenChange();

    // Manage layout fullscreen body class
    const className = 'app-preview-immersive';
    const { body } = document;
    if (isLayoutFullscreen) {
      body.classList.add(className);
    } else {
      body.classList.remove(className);
    }

    return () => {
      document.removeEventListener('fullscreenchange', handleFullscreenChange);
      body.classList.remove(className);
    };
  }, [isLayoutFullscreen]);

  useKeyboardScope({
    id: 'app-preview-layout-fullscreen-escape',
    priority: 200,
    enabled: isLayoutFullscreen,
    onKeyDown: (event) => {
      if (event.defaultPrevented || event.key !== 'Escape') {
        return false;
      }
      setIsLayoutFullscreen(false);
      return true;
    },
  });

  useEffect(() => {
    const wasFullscreen = previousIsFullscreenRef.current;
    previousIsFullscreenRef.current = isFullscreen;

    if (!wasFullscreen || isFullscreen) {
      return;
    }

    if (activeOverlay === 'tabs') {
      closeOverlay({ preserve: ['segment'] });
    }
  }, [activeOverlay, closeOverlay, isFullscreen]);

  useEffect(() => {
    recordDebugEvent('preview-mount', {
      appId,
      locationState,
      pathname: location.pathname,
    });
    return () => {
      recordDebugEvent('preview-unmount', {
        appId: previousAppId,
      });
    };
  }, [appId, previousAppId, location.pathname, locationState, recordDebugEvent]);

  // matchesAppIdentifier is now imported from utils/appPreview

  const preserveErrorMessages = useMemo(
    () => [PREVIEW_MESSAGES.NO_UI, scenarioStoppedMessage],
    [scenarioStoppedMessage],
  );

  // Use shared preview overlay hook
  const { setPreviewOverlay, fallbackState } = usePreviewOverlay({
    previewUrl,
    previewReloadToken,
    loading,
    statusMessage,
    defaultEmptyMessage: 'Preview unavailable.',
    bridgeIsReady: bridgeState.isReady,
    iframeLoadedAt,
    iframeLoadError,
    preserveErrorMessages,
  });

  // Use shared preview background color hook
  const getPreviewBackgroundColor = usePreviewBackgroundColor(iframeRef, previewViewRef);

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
  const syncPreviewUrl = usePreviewUrlOrchestration({
    hasCustomPreviewUrl,
    previewUrl,
    applyDefaultPreviewUrl,
    resetPreviewState,
    setPreviewUrl,
    initialPreviewUrlRef,
  });

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

  const reloadPreview = useCallback(() => {
    resetState();
    setIframeLoadedAt(null);
    setIframeLoadError(null);
    setPreviewReloadToken(prev => prev + 1);
  }, [resetState]);

  const { beginLifecycleMonitor, stopLifecycleMonitor } = useAppLifecycleMonitor({
    currentAppIdentifier,
    hasCustomPreviewUrl,
    applyDefaultPreviewUrl,
    commitAppUpdate,
    reloadPreview,
    setLoading,
    setPreviewOverlay,
    setStatusMessage,
  });

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
  }, [appId, bridgeState.href, commitAppUpdate, hasCustomPreviewUrl, previewUrl, reloadPreview, setPreviewOverlay]);

  const handleOpenReportDialog = useCallback(() => {
    setReportDialogOpen(true);
  }, []);

  const handleCloseReportDialog = useCallback(() => {
    setReportDialogOpen(false);
  }, []);

  const activePreviewUrl = useMemo(() => bridgeState.href || previewUrl || '', [bridgeState.href, previewUrl]);
  const {
    reportElementCaptures,
    setHasPrimaryCaptureDraft,
    stagedCaptureCount,
    canCaptureScreenshot,
    bridgeSupportsScreenshot,
    isPreviewSameOrigin,
    handleInspectorCaptureAdded,
    handleElementCaptureNoteChange,
    handleRemoveElementCapture,
    resetElementCaptures: handleResetElementCaptures,
    resetReportDraftState,
  } = usePreviewReportSession({
    activePreviewUrl,
    bridgeState: {
      isSupported: bridgeState.isSupported,
      caps: bridgeState.caps,
    },
    logPrefix: '[app-preview]',
  });

  const inspector = usePreviewInspector({
    inspectState,
    startInspect,
    stopInspect,
    setInspectTargetIndex,
    shiftInspectTarget,
    requestScreenshot,
    previewUrl,
    currentAppIdentifier,
    iframeRef,
    previewViewRef,
    previewViewNode,
    onCaptureAdd: handleInspectorCaptureAdded,
    onViewReportRequest: handleOpenReportDialog,
  });
  const inspectorHandleDialogClose = inspector.handleInspectorDialogClose;

  // Consolidated dialog cleanup when app changes
  useEffect(() => {
    // Close inspector dialog when preview identifier changes
    const identifier = previewIdentifier ?? null;
    if (previousPreviewIdentifier && previousPreviewIdentifier !== identifier) {
      inspectorHandleDialogClose();
    }

    // Reset report dialog when scenario changes
    const scenarioKey = currentAppIdentifier ?? null;
    if (previousReportScenario !== null && previousReportScenario !== scenarioKey) {
      setReportDialogOpen(false);
      resetReportDraftState();
    }
  }, [
    previewIdentifier,
    previousPreviewIdentifier,
    currentAppIdentifier,
    previousReportScenario,
    inspectorHandleDialogClose,
    resetReportDraftState,
  ]);

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
    resolvePreviewBackgroundColor: getPreviewBackgroundColor,
    setSurfaceScreenshot,
    surfaceType: 'app',
  });

  useEffect(() => {
    setIframeLoadedAt(null);
    setIframeLoadError(null);
  }, [previewUrl, previewReloadToken]);

  const urlStatusTitle = useMemo(() => {
    if (!currentApp) {
      return 'Status: Unknown';
    }
    const status = currentApp.status ?? 'Unknown';
    return `Status: ${status}`;
  }, [currentApp]);

  const lifecycle = usePreviewAppLifecycle({
    currentApp,
    setStatusMessage,
    toggleLabels: {
      start: 'Start scenario',
      stop: 'Stop scenario',
    },
    restartLabel: 'Restart scenario',
    failureMessageForAction: (action) => (
      action === 'start'
        ? `Unable to start ${scenarioDisplayName}. Check logs for details.`
        : `Unable to ${action} the application. Check logs for details.`
    ),
    onSuccess: ({ appId: controlledAppId, action }) => {
      const timestamp = new Date().toISOString();
      if (action === 'start' || action === 'stop') {
        const nextStatus: App['status'] = action === 'stop' ? 'stopped' : 'running';
        setAppsState(prev => prev.map(app => (
          app.id === controlledAppId
            ? { ...app, status: nextStatus, updated_at: timestamp }
            : app
        )));
        setCurrentApp(prev => (
          prev && prev.id === controlledAppId
            ? { ...prev, status: nextStatus, updated_at: timestamp }
            : prev
        ));
        setStatusMessage(
          action === 'stop'
            ? 'Application stopped. Start it again to relaunch the UI preview.'
            : 'Application started. Preview will refresh automatically.',
        );
        return;
      }
      setStatusMessage('Restart command sent. Waiting for application to return...');
    },
  });

  const bridgeIssueMessage = useMemo(() => {
    if (!bridgeState.isSupported || !bridgeCompliance || bridgeCompliance.ok) {
      return null;
    }
    const detail = Array.isArray(bridgeCompliance.failures) ? bridgeCompliance.failures.join(', ') : 'unknown';
    return `Preview bridge diagnostics failed (${detail}). History syncing may be unreliable.`;
  }, [bridgeCompliance, bridgeState.isSupported]);

  useEffect(() => {
    if (bridgeIssueMessage) {
      setBridgeMessageDismissed(false);
    }
  }, [bridgeIssueMessage]);

  const {
    openPreviewTarget,
    urlSuggestions: toolbarUrlSuggestions,
    handleOpenScenarioSelector,
    handleOpenPreviewInNewTab: handleOpenPreviewInNewTabRaw,
  } = usePreviewToolbarSession({
    bridgeHref: bridgeState.isSupported ? bridgeState.href : null,
    previewUrl,
    history,
    apps,
    openOverlay,
    appOpenMode: 'single-preview',
  });

  useEffect(() => {
    if (!appId) {
      recordNavigateEvent({
        reason: 'missing-app-id',
        targetPath: '/apps',
        targetSearch: location.search || undefined,
        replace: true,
        locationState,
      });
      navigate({
        pathname: '/apps',
        search: location.search || undefined,
      }, { replace: true });
    }
  }, [appId, location.search, locationState, navigate, recordNavigateEvent]);

  // Consolidated appId-related state reset and cleanup
  useEffect(() => {
    // Reset state when appId changes
    setFetchAttempted(false);
    stopLifecycleMonitor();
    setPreviewOverlay(null);
    clearNavigationSession();
    setBridgeCompliance(null);
    resetState();
    setIframeLoadedAt(null);
    setIframeLoadError(null);

    // Cleanup lifecycle monitor on unmount
    return () => {
      stopLifecycleMonitor();
    };
  }, [appId, clearNavigationSession, resetState, stopLifecycleMonitor, setPreviewOverlay]);

  // Reset bridge compliance when preview URL changes
  useEffect(() => {
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
    if (locationState?.fromAppsList) {
      shouldFocusPreviewOnLoadRef.current = true;
    }
  }, [location.key, locationState?.fromAppsList]);

  useEffect(() => {
    if (!shouldFocusPreviewOnLoadRef.current) {
      return;
    }

    const rafId = window.requestAnimationFrame(() => {
      if (focusPreviewIframe()) {
        shouldFocusPreviewOnLoadRef.current = false;
      }
    });

    return () => {
      window.cancelAnimationFrame(rafId);
    };
  }, [currentAppIdentifier, focusPreviewIframe]);

  useEffect(() => {
    if (!currentAppForPreview) {
      syncPreviewUrl({
        appForPreview: null,
        fallbackPreviewUrl: deterministicProxyUrl,
        forceResetWhenMissingApp: true,
      });
      setStatusMessage('Loading application preview...');
      setLoading(true);
      return;
    }

    const { hasPreviewCandidate } = syncPreviewUrl({
      appForPreview: currentAppForPreview as App,
      fallbackPreviewUrl: deterministicProxyUrl,
    });

    if (isExplicitlyStopped) {
      setLoading(false);
      setStatusMessage(scenarioStoppedMessage);
      setPreviewOverlay(prev => {
        if (prev && (prev.type === 'restart' || (prev.type === 'error' && prev.message === scenarioStoppedMessage))) {
          return prev;
        }
        return { type: 'error', message: scenarioStoppedMessage };
      });
      return;
    }

    if (!hasPreviewCandidate) {
      if (currentAppForPreview.is_partial) {
        setStatusMessage('Loading application details...');
        setLoading(true);
        return;
      }

      setStatusMessage(PREVIEW_MESSAGES.NO_UI);
      setLoading(false);
      setPreviewOverlay(prev => {
        if (prev && (prev.type === 'restart' || (prev.type === 'error' && prev.message === PREVIEW_MESSAGES.NO_UI))) {
          return prev;
        }
        return { type: 'error', message: PREVIEW_MESSAGES.NO_UI };
      });
      return;
    }

    setLoading(false);

    if (currentAppForPreview.is_partial && !currentAppForPreview.status) {
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
        (prev.message === scenarioStoppedMessage || prev.message === PREVIEW_MESSAGES.NO_UI)
      ) {
        return null;
      }
      return prev;
    });
  }, [
    currentAppForPreview,
    deterministicProxyUrl,
    isExplicitlyStopped,
    syncPreviewUrl,
    scenarioStoppedMessage,
    setPreviewOverlay,
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

  usePreviewBridgeComplianceCheck({
    enabled: bridgeState.isSupported && bridgeState.isReady && Boolean(bridgeState.href),
    runComplianceCheck,
    onSuccess: setBridgeCompliance,
    onError: (error) => {
      logger.warn('Bridge compliance check failed', error);
      setBridgeCompliance({ ok: false, failures: ['CHECK_FAILED'], checkedAt: Date.now() });
    },
    runOnceWhileEnabled: true,
    resetKey: previewUrl,
  });

  const handleAppAction = useCallback(async (appToControl: string, action: 'start' | 'stop' | 'restart') => {
    if (action === 'restart') {
      setPreviewOverlay({ type: 'restart', message: 'Restarting application...' });
      setLoading(true);
      reloadPreview();
    } else if (action === 'start') {
      setPreviewOverlay({ type: 'waiting', message: 'Waiting for application to start...' });
      setLoading(true);
    }

    const success = await lifecycle.runAction(appToControl, action);
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
  }, [beginLifecycleMonitor, lifecycle, reloadPreview, scenarioDisplayName, setLoading, setPreviewOverlay]);

  const handleToggleApp = useCallback(() => {
    if (!currentApp || lifecycle.actionInProgress) {
      return;
    }

    const action: 'start' | 'stop' = isRunningStatus(currentApp.status) ? 'stop' : 'start';
    void handleAppAction(currentApp.id, action);
  }, [currentApp, handleAppAction, lifecycle.actionInProgress]);

  const handleRestartApp = useCallback(() => {
    if (!currentApp || lifecycle.actionInProgress || !isRunningStatus(currentApp.status)) {
      return;
    }
    void handleAppAction(currentApp.id, 'restart');
  }, [currentApp, handleAppAction, lifecycle.actionInProgress]);

  const handleOpenPreviewInNewTab = useCallback((event: MouseEvent<HTMLButtonElement>) => {
    // Keep the Referer header so the proxy can map shared asset URLs to the active scenario.
    handleOpenPreviewInNewTabRaw(event);
  }, [handleOpenPreviewInNewTabRaw]);

  const handleToggleLogsFromToolbar = useCallback(() => {
    toggleLogsPanel();
  }, [toggleLogsPanel]);

  const handleIframeLoad = useCallback(() => {
    setIframeLoadError(null);
    setIframeLoadedAt(Date.now());
    setPreviewOverlay(prev => {
      if (!prev) return prev;
      if (prev.type === 'waiting' && prev.message === PREVIEW_MESSAGES.CONNECTING) return null;
      if (prev.type === 'error' && (prev.message === PREVIEW_MESSAGES.TIMEOUT || prev.message === PREVIEW_MESSAGES.MIXED_CONTENT)) {
        return null;
      }
      return prev;
    });
    if (shouldFocusPreviewOnLoadRef.current) {
      const rafId = window.requestAnimationFrame(() => {
        if (focusPreviewIframe()) {
          shouldFocusPreviewOnLoadRef.current = false;
        }
      });
      window.setTimeout(() => window.cancelAnimationFrame(rafId), 300);
    }
  }, [focusPreviewIframe, setPreviewOverlay]);

  const handleIframeError = useCallback(() => {
    const isMixedContent =
      typeof window !== 'undefined' &&
      window.location.protocol === 'https:' &&
      previewUrl?.startsWith('http://');

    const message = isMixedContent
      ? PREVIEW_MESSAGES.MIXED_CONTENT
      : 'Preview failed to load. Verify the application UI is reachable from the App Monitor host.';

    setIframeLoadedAt(null);
    setIframeLoadError(message);
    setPreviewOverlay(current => {
      if (current && current.type === 'restart') {
        return current;
      }
      return { type: 'error', message };
    });
  }, [previewUrl, setPreviewOverlay]);

  usePreviewIframeReadinessFallback({
    iframeRef,
    enabled: loading && Boolean(previewUrl) && !iframeLoadedAt && !iframeLoadError,
    onReady: () => {
      setLoading(false);
      setIframeLoadError(null);
      setIframeLoadedAt(Date.now());
      setStatusMessage(null);
    },
  });

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
        urlStatusClass={lifecycle.urlStatusClass}
        urlStatusTitle={urlStatusTitle}
        hasDetailsWarning={hasLocalhostWarning}
        hasCurrentApp={Boolean(currentApp)}
        isAppRunning={lifecycle.isAppRunning}
        pendingAction={lifecycle.pendingAction}
        actionInProgress={lifecycle.actionInProgress}
        toggleActionLabel={lifecycle.toggleActionLabel}
        onToggleApp={handleToggleApp}
        restartActionLabel={lifecycle.restartActionLabel}
        onRestartApp={handleRestartApp}
        onToggleLogs={handleToggleLogsFromToolbar}
        areLogsVisible={isLogsPanelOpen}
        onReportIssue={handleOpenReportDialog}
        appStatusLabel={lifecycle.appStatusLabel}
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
        urlSuggestions={toolbarUrlSuggestions}
        onSelectUrlSuggestion={applyPreviewUrlValue}
        onOpenScenarioSelector={handleOpenScenarioSelector}
        scenarioSelectorLabel="Open scenario selector"
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

      <ErrorBoundary fallback={SectionErrorFallback}>
        <PreviewInspectorPanel
          inspectState={inspectState}
          previewUrl={previewUrl}
          inspector={inspector}
        />
      </ErrorBoundary>


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
          {fallbackState && <PreviewFallbackState state={fallbackState} variant="overlay" />}
        </div>
      ) : (
        fallbackState ? (
          <PreviewFallbackState state={fallbackState} variant="panel" />
        ) : (
          <div className="preview-placeholder">
            {loading ? `Fetching ${scenarioDisplayName} details` : statusMessage ?? 'Preview unavailable.'}
          </div>
        )
      )}

      {modalOpen && currentApp && (
        <ErrorBoundary fallback={SectionErrorFallback}>
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
              navigate(`/logs/${appIdentifier}`);
            }}
            proxyMetadata={proxyMetadata}
            previewUrl={activePreviewUrl || null}
            preloadedDiagnostics={preloadedDiagnostics}
            diagnosticsLoading={diagnosticsLoading}
            preloadedLighthouseHistory={preloadedLighthouseHistory}
            lighthouseLoading={lighthouseLoading}
            lighthouseError={lighthouseError}
            onRefetchLighthouse={refetchLighthouse}
            preloadedCompleteness={preloadedCompleteness}
            completenessLoading={completenessLoading}
            diagnosticsError={diagnosticsError}
            onRefetchDiagnostics={refetchDiagnostics}
            completenessError={completenessError}
            onRefetchCompleteness={refetchCompleteness}
          />
        </ErrorBoundary>
      )}

      {reportDialogOpen && (
        <ErrorBoundary fallback={SectionErrorFallback}>
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
        </ErrorBoundary>
      )}
    </div>
  );
};


export default AppPreviewView;
