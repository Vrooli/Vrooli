import { useCallback, useEffect, useMemo, useRef, useState, useId } from 'react';
import type { ChangeEvent, KeyboardEvent, MouseEvent, PointerEvent as ReactPointerEvent } from 'react';
import { useLocation, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import clsx from 'clsx';
import { Info, Loader2, X, ChevronDown, Image, BoxSelect } from 'lucide-react';
import { appService } from '@/services/api';
import { useAppsStore } from '@/state/appsStore';
import { useScenarioEngagementStore } from '@/state/scenarioEngagementStore';
import { logger } from '@/services/logger';
import type { App, AppProxyMetadata, LocalhostUsageReport } from '@/types';
import { useShellOverlayStore } from '@/state/shellOverlayStore';
import { useSurfaceMediaStore } from '@/state/surfaceMediaStore';
import AppModal from '../AppModal';
import AppPreviewToolbar from '../AppPreviewToolbar';
import ReportIssueDialog from '../report/ReportIssueDialog';
import {
  buildPreviewUrl,
  isRunningStatus,
  isStoppedStatus,
  locateAppByIdentifier,
  resolveAppIdentifier,
} from '@/utils/appPreview';
import { ensureDataUrl } from '@/utils/dataUrl';
import { useIframeBridge } from '@/hooks/useIframeBridge';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import { useDeviceEmulation } from '@/hooks/useDeviceEmulation';
import DeviceEmulationToolbar from '../device-emulation/DeviceEmulationToolbar';
import DeviceEmulationViewport from '../device-emulation/DeviceEmulationViewport';
import DeviceVisionFilterDefs from '../device-emulation/DeviceVisionFilterDefs';
import { useAppLogs } from '@/hooks/useAppLogs';
import AppLogsPanel from '../logs/AppLogsPanel';
import './AppPreviewView.css';

const PREVIEW_LOAD_TIMEOUT_MS = 6000;
const PREVIEW_CONNECTING_LABEL = 'Connecting to preview...';
const PREVIEW_TIMEOUT_MESSAGE = 'Preview did not respond. Ensure the application UI is running and reachable from App Monitor.';
const PREVIEW_MIXED_CONTENT_MESSAGE = 'Preview blocked: browser refused to load HTTP content inside an HTTPS dashboard. Expose the UI through the tunnel hostname or enable HTTPS for the scenario.';
const IOS_AUTOBACK_GUARD_MS = 15000;
const PREVIEW_SCREENSHOT_STABILITY_MS = 1000;
const INSPECTOR_FLOATING_MARGIN = 12;
type InspectorPosition = { x: number; y: number };
const DEFAULT_INSPECTOR_POSITION: InspectorPosition = { x: 24, y: 104 };
const DRAG_THRESHOLD_PX = 3;
type InspectorScreenshot = {
  dataUrl: string;
  width: number;
  height: number;
  filename: string;
  capturedAt: number;
  note?: string | null;
  mode?: string | null;
  clip?: { x: number; y: number; width: number; height: number } | null;
};
const AppPreviewView = () => {
  const apps = useAppsStore(state => state.apps);
  const setAppsState = useAppsStore(state => state.setAppsState);
  const loadApps = useAppsStore(state => state.loadApps);
  const loadingInitial = useAppsStore(state => state.loadingInitial);
  const hasInitialized = useAppsStore(state => state.hasInitialized);
  const canOpenTabsOverlay = apps.length > 0;
  const navigate = useNavigate();
  const { appId } = useParams<{ appId: string }>();
  type PreviewLocationState = {
    fromAppsList?: boolean;
    originAppId?: string;
    navTimestamp?: number;
    suppressedAutoBack?: boolean;
    autoSelected?: boolean;
    autoSelectedAt?: number;
  };
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
  const overlayStateRef = useRef<typeof activeOverlay>(null);
  const captureInFlightRef = useRef(false);
  const lastScreenshotRef = useRef<{ surfaceId: string | null; capturedAt: number }>({ surfaceId: null, capturedAt: 0 });
  useEffect(() => {
    setIsLogsPanelOpen(overlayQuery === 'logs');
  }, [overlayQuery]);
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
  const [previewInteractionSignal, setPreviewInteractionSignal] = useState(0);
  const [previewOverlay, setPreviewOverlay] = useState<null | { type: 'restart' | 'waiting' | 'error'; message: string }>(null);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isLayoutFullscreen, setIsLayoutFullscreen] = useState(false);
  const [iframeLoadedAt, setIframeLoadedAt] = useState<number | null>(null);
  const [iframeLoadError, setIframeLoadError] = useState<string | null>(null);
  const [inspectStatusMessage, setInspectStatusMessage] = useState<string | null>(null);
  const [inspectCopyFeedback, setInspectCopyFeedback] = useState<'selector' | 'text' | null>(null);
  const [isInspectorDialogOpen, setIsInspectorDialogOpen] = useState(false);
  const [inspectorDetailsExpanded, setInspectorDetailsExpanded] = useState(false);
  const [inspectorScreenshot, setInspectorScreenshot] = useState<InspectorScreenshot | null>(null);
  const [inspectorScreenshotExpanded, setInspectorScreenshotExpanded] = useState(false);
  const [inspectorDialogPosition, setInspectorDialogPosition] = useState<InspectorPosition>(() => ({ ...DEFAULT_INSPECTOR_POSITION }));
  const [isInspectorDragging, setIsInspectorDragging] = useState(false);
  useEffect(() => {
    if (!isInspectorDialogOpen) {
      return;
    }
    setInspectorDetailsExpanded(false);
    setInspectorScreenshotExpanded(false);
  }, [isInspectorDialogOpen]);
  const inspectorDialogRef = useRef<HTMLDivElement | null>(null);
  const inspectorDragStateRef = useRef<{
    pointerId: number;
    offsetX: number;
    offsetY: number;
    startClientX: number;
    startClientY: number;
    pointerCaptured: boolean;
    dragging: boolean;
    width: number;
    height: number;
    containerRect: DOMRect;
  } | null>(null);
  const inspectorSuppressClickRef = useRef(false);
  const inspectorDialogTitleId = useId();
  const inspectorDetailsSectionId = useId();
  const inspectorScreenshotSectionId = useId();
  const inspectorDetailsContentId = `${inspectorDetailsSectionId}-content`;
  const inspectorScreenshotContentId = `${inspectorScreenshotSectionId}-content`;
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
  const inspectMessageLockRef = useRef(false);
  const inspectMessageUnlockTimerRef = useRef<number | null>(null);

  const scenarioDisplayName = useMemo(() => {
    const fallback = appId?.trim() || 'Scenario';
    if (!currentApp) {
      return fallback;
    }
    const preferred = (currentApp.scenario_name ?? currentApp.name ?? '').trim();
    return preferred.length > 0 ? preferred : fallback;
  }, [appId, currentApp]);

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
    if (!inspectCopyFeedback) {
      return;
    }
    if (typeof window === 'undefined') {
      return;
    }
    const timeoutId = window.setTimeout(() => setInspectCopyFeedback(null), 1800);
    return () => window.clearTimeout(timeoutId);
  }, [inspectCopyFeedback]);

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
    inspectState,
    startInspect,
    stopInspect,
  } = useIframeBridge({
    iframeRef,
    previewUrl,
    onLocation: handleBridgeLocation,
  });

  const inspectTarget = useMemo(() => inspectState.hover ?? inspectState.result, [inspectState.hover, inspectState.result]);
  const inspectMeta = inspectTarget?.meta ?? null;
  const inspectRect = inspectTarget?.documentRect ?? null;
  const inspectSizeLabel = inspectRect
    ? `${Math.round(inspectRect.width)} × ${Math.round(inspectRect.height)} px`
    : null;
  const inspectPositionLabel = inspectRect
    ? `x ${Math.round(inspectRect.x)}, y ${Math.round(inspectRect.y)}`
    : null;
  const inspectClassTokens = useMemo<string[]>(
    () => (inspectMeta?.classes ? inspectMeta.classes.slice(0, 3) : []),
    [inspectMeta?.classes],
  );
  const inspectSelectorValue = inspectMeta?.selector ?? null;
  const inspectLabelValue = inspectMeta?.ariaLabel ?? inspectMeta?.label ?? null;
  const inspectTextValue = inspectMeta?.text ?? null;
  const inspectAriaDescription = inspectMeta?.ariaDescription ?? null;
  const inspectTitleValue = inspectMeta?.title ?? null;
  const inspectTextPreview = inspectTextValue && inspectTextValue.length > 220
    ? `${inspectTextValue.slice(0, 220)}…`
    : inspectTextValue;
  const inspectResultMethod = inspectState.result?.method ?? null;
  let inspectMethodLabel: string | null = null;
  if (inspectResultMethod === 'keyboard') {
    inspectMethodLabel = 'Keyboard selection';
  } else if (inspectResultMethod === 'pointer') {
    inspectMethodLabel = 'Pointer selection';
  } else if (inspectState.active && inspectTarget?.pointerType) {
    inspectMethodLabel = `${inspectTarget.pointerType} pointer`;
  }
  const shouldRenderInspectorDialog = isInspectorDialogOpen;

  const clampInspectorPosition = useCallback((
    x: number,
    y: number,
    dimensions?: { width?: number; height?: number; containerRect?: DOMRect },
  ): InspectorPosition => {
    const container = previewViewRef.current;
    const containerRect = dimensions?.containerRect ?? container?.getBoundingClientRect();
    const width = dimensions?.width ?? inspectorDialogRef.current?.offsetWidth ?? 360;
    const height = dimensions?.height ?? inspectorDialogRef.current?.offsetHeight ?? 360;
    if (!containerRect) {
      return { x, y };
    }
    const maxX = Math.max(INSPECTOR_FLOATING_MARGIN, containerRect.width - width - INSPECTOR_FLOATING_MARGIN);
    const maxY = Math.max(INSPECTOR_FLOATING_MARGIN, containerRect.height - height - INSPECTOR_FLOATING_MARGIN);
    const clampedX = Math.min(Math.max(x, INSPECTOR_FLOATING_MARGIN), maxX);
    const clampedY = Math.min(Math.max(y, INSPECTOR_FLOATING_MARGIN), maxY);
    return { x: clampedX, y: clampedY };
  }, []);

  const clearInspectMessageLock = useCallback(() => {
    if (inspectMessageUnlockTimerRef.current !== null && typeof window !== 'undefined') {
      window.clearTimeout(inspectMessageUnlockTimerRef.current);
    }
    inspectMessageUnlockTimerRef.current = null;
    inspectMessageLockRef.current = false;
  }, []);

  const setLockedInspectMessage = useCallback((message: string, durationMs = 3200) => {
    clearInspectMessageLock();
    setInspectStatusMessage(message);
    if (typeof window === 'undefined') {
      return;
    }
    inspectMessageLockRef.current = true;
    inspectMessageUnlockTimerRef.current = window.setTimeout(() => {
      inspectMessageLockRef.current = false;
      inspectMessageUnlockTimerRef.current = null;
      setInspectStatusMessage(null);
    }, durationMs);
  }, [clearInspectMessageLock]);

  const handleInspectorPointerDown = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    if (!isInspectorDialogOpen) {
      return;
    }
    if (event.button !== 0) {
      return;
    }
    const target = event.target as HTMLElement | null;
    if (!target) {
      return;
    }
    const handleNode = target.closest('.preview-inspector__handle');
    if (!handleNode) {
      return;
    }
    if (target.closest('button, a, input, textarea, select')) {
      return;
    }
    const dialog = inspectorDialogRef.current;
    const container = previewViewRef.current;
    if (!dialog || !container) {
      return;
    }
    const containerRect = container.getBoundingClientRect();
    const dialogRect = dialog.getBoundingClientRect();
    inspectorDragStateRef.current = {
      pointerId: event.pointerId,
      offsetX: event.clientX - dialogRect.left,
      offsetY: event.clientY - dialogRect.top,
      startClientX: event.clientX,
      startClientY: event.clientY,
      pointerCaptured: false,
      dragging: false,
      width: dialogRect.width,
      height: dialogRect.height,
      containerRect,
    };
    if (typeof dialog.setPointerCapture === 'function') {
      try {
        dialog.setPointerCapture(event.pointerId);
        inspectorDragStateRef.current.pointerCaptured = true;
      } catch {
        inspectorDragStateRef.current.pointerCaptured = false;
      }
    }
    event.preventDefault();
  }, [isInspectorDialogOpen]);

  const handleInspectorPointerMove = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    const state = inspectorDragStateRef.current;
    if (!state || state.pointerId !== event.pointerId) {
      return;
    }
    const container = previewViewRef.current;
    const containerRect = container?.getBoundingClientRect() ?? state.containerRect;
    const dialogRect = inspectorDialogRef.current?.getBoundingClientRect();
    const width = dialogRect?.width ?? state.width;
    const height = dialogRect?.height ?? state.height;

    if (!state.dragging) {
      const diffX = Math.abs(event.clientX - state.startClientX);
      const diffY = Math.abs(event.clientY - state.startClientY);
      if (diffX < DRAG_THRESHOLD_PX && diffY < DRAG_THRESHOLD_PX) {
        return;
      }
      state.dragging = true;
      setIsInspectorDragging(true);
    }

    if (!state.pointerCaptured) {
      const dialog = inspectorDialogRef.current;
      if (dialog && typeof dialog.setPointerCapture === 'function') {
        try {
          dialog.setPointerCapture(event.pointerId);
          state.pointerCaptured = true;
        } catch {
          state.pointerCaptured = false;
        }
      }
    }

    event.preventDefault();
    const rawX = event.clientX - containerRect.left - state.offsetX;
    const rawY = event.clientY - containerRect.top - state.offsetY;
    const next = clampInspectorPosition(rawX, rawY, { width, height, containerRect });
    setInspectorDialogPosition(prev => (prev.x === next.x && prev.y === next.y ? prev : next));
  }, [clampInspectorPosition]);

  const handleInspectorPointerUp = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    const state = inspectorDragStateRef.current;
    if (!state || state.pointerId !== event.pointerId) {
      return;
    }
    const dialog = inspectorDialogRef.current;
    if (state.pointerCaptured && dialog && typeof dialog.releasePointerCapture === 'function') {
      try {
        dialog.releasePointerCapture(event.pointerId);
      } catch {
        // ignore release failures
      }
    }
    if (state.dragging) {
      event.preventDefault();
      inspectorSuppressClickRef.current = true;
      if (typeof window !== 'undefined') {
        window.setTimeout(() => {
          inspectorSuppressClickRef.current = false;
        }, 0);
      } else {
        inspectorSuppressClickRef.current = false;
      }
    }
    inspectorDragStateRef.current = null;
    setIsInspectorDragging(false);
  }, []);

  const handleInspectorClickCapture = useCallback((event: MouseEvent<HTMLDivElement>) => {
    if (inspectorSuppressClickRef.current) {
      event.preventDefault();
      event.stopPropagation();
      inspectorSuppressClickRef.current = false;
    }
  }, []);

  const handleInspectorDialogClose = useCallback(() => {
    if (inspectState.active) {
      stopInspect();
    }
    inspectorDragStateRef.current = null;
    inspectorSuppressClickRef.current = false;
    setIsInspectorDragging(false);
    setInspectCopyFeedback(null);
    clearInspectMessageLock();
    setInspectStatusMessage(null);
    setIsInspectorDialogOpen(false);
  }, [clearInspectMessageLock, inspectState.active, setInspectStatusMessage, stopInspect]);

  const handleCopyValue = useCallback(async (value: string, kind: 'selector' | 'text') => {
    try {
      if (typeof navigator !== 'undefined' && navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
        await navigator.clipboard.writeText(value);
      } else if (typeof document !== 'undefined') {
        const textarea = document.createElement('textarea');
        textarea.value = value;
        textarea.setAttribute('readonly', 'true');
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.focus();
        textarea.select();
        document.execCommand('copy');
        document.body.removeChild(textarea);
      } else {
        throw new Error('clipboard-unavailable');
      }
      setInspectCopyFeedback(kind);
      setLockedInspectMessage(kind === 'selector' ? 'Selector copied to clipboard.' : 'Text snippet copied to clipboard.', 2200);
    } catch (error) {
      logger.warn('Failed to copy inspector value', error);
      setLockedInspectMessage('Unable to copy to clipboard.', 3200);
    }
  }, [setLockedInspectMessage]);

  const handleCopySelector = useCallback(() => {
    if (!inspectSelectorValue) {
      setLockedInspectMessage('Selector unavailable for this element.', 2600);
      return;
    }
    void handleCopyValue(inspectSelectorValue, 'selector');
  }, [handleCopyValue, inspectSelectorValue, setLockedInspectMessage]);

  const handleCopyText = useCallback(() => {
    if (!inspectTextValue) {
      setLockedInspectMessage('Text snippet unavailable for this element.', 2600);
      return;
    }
    void handleCopyValue(inspectTextValue, 'text');
  }, [handleCopyValue, inspectTextValue, setLockedInspectMessage]);

  const handleCaptureElementScreenshot = useCallback(async () => {
    if (!inspectRect) {
      setLockedInspectMessage('Element bounds unavailable for screenshots.', 3000);
      return;
    }
    try {
      const scale = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1;
      const result = await requestScreenshot({
        mode: 'clip',
        clip: inspectRect,
        selector: inspectSelectorValue ?? undefined,
        scale,
      });
      const sanitizedSelector = inspectSelectorValue
        ? inspectSelectorValue.replace(/[^a-z0-9-_]+/gi, '-').slice(0, 40)
        : 'element';
      const filename = `${(currentAppIdentifier ?? 'preview').replace(/[^a-z0-9-_]+/gi, '-').slice(0, 24)}-${sanitizedSelector || 'element'}-${Date.now()}.png`;
      const dataUrl = ensureDataUrl(result.data);
      if (!dataUrl) {
        setLockedInspectMessage('Failed to capture element screenshot.', 3400);
        return;
      }
      const capturedAt = Date.now();
      if (typeof document !== 'undefined') {
        const anchor = document.createElement('a');
        anchor.href = dataUrl;
        anchor.download = filename;
        document.body.appendChild(anchor);
        anchor.click();
        document.body.removeChild(anchor);
      }
      setInspectorScreenshot({
        dataUrl,
        width: result.width,
        height: result.height,
        filename,
        capturedAt,
        note: result.note ?? null,
        mode: result.mode ?? null,
        clip: result.clip ?? null,
      });
      setInspectorScreenshotExpanded(true);
      setLockedInspectMessage('Element screenshot downloaded.', 3200);
    } catch (error) {
      logger.error('Failed to capture element screenshot', error);
      setLockedInspectMessage('Failed to capture element screenshot.', 3400);
    }
  }, [
    currentAppIdentifier,
    inspectRect,
    inspectSelectorValue,
    requestScreenshot,
    setInspectorScreenshot,
    setInspectorScreenshotExpanded,
    setLockedInspectMessage,
  ]);

  const handleToggleInspectMode = useCallback(() => {
    if (inspectState.active) {
      const stopped = stopInspect();
      if (!stopped) {
        setLockedInspectMessage('Unable to exit inspect mode.', 3200);
      }
      return;
    }
    if (!inspectState.supported) {
      setLockedInspectMessage('Element inspector requires the latest bridge in the previewed app.', 3600);
      return;
    }
    if (!previewUrl) {
      setLockedInspectMessage('Load the preview before inspecting elements.', 3200);
      return;
    }
    const started = startInspect();
    if (!started) {
      setLockedInspectMessage('Element inspector is unavailable for this preview.', 3600);
      return;
    }
    clearInspectMessageLock();
    setInspectStatusMessage('Select an element in the preview. Press Esc to cancel.');
  }, [clearInspectMessageLock, inspectState.active, inspectState.supported, previewUrl, setLockedInspectMessage, startInspect, stopInspect]);

  useEffect(() => {
    if (inspectState.active && !isInspectorDialogOpen) {
      setIsInspectorDialogOpen(true);
    }
  }, [inspectState.active, isInspectorDialogOpen]);

  useEffect(() => {
    if (inspectStatusMessage && !isInspectorDialogOpen) {
      setIsInspectorDialogOpen(true);
    }
  }, [inspectStatusMessage, isInspectorDialogOpen]);

  useEffect(() => {
    if (!isInspectorDialogOpen) {
      inspectorDragStateRef.current = null;
      setIsInspectorDragging(false);
      return;
    }

    setInspectorDialogPosition(prev => clampInspectorPosition(prev.x, prev.y));

    if (typeof ResizeObserver === 'undefined') {
      return;
    }

    const container = previewViewRef.current;
    if (!container) {
      return;
    }

    const observer = new ResizeObserver(() => {
      setInspectorDialogPosition(prev => clampInspectorPosition(prev.x, prev.y));
    });
    observer.observe(container);
    return () => {
      observer.disconnect();
    };
  }, [clampInspectorPosition, isInspectorDialogOpen]);

  useEffect(() => {
    if (!previewViewNode) {
      return;
    }
    setInspectorDialogPosition(prev => clampInspectorPosition(prev.x, prev.y));
  }, [clampInspectorPosition, previewViewNode]);

  useEffect(() => {
    if (!inspectState.supported) {
      return;
    }
    if (inspectMessageLockRef.current) {
      return;
    }
    if (inspectState.error) {
      setLockedInspectMessage(`Element inspector error: ${inspectState.error}`, 3600);
      return;
    }
    if (inspectState.active) {
      clearInspectMessageLock();
      setInspectStatusMessage('Select an element in the preview. Press Esc to cancel.');
      return;
    }
    if (inspectState.lastReason === 'complete') {
      setLockedInspectMessage('Element captured. Use “Inspect element” to capture another.', 3400);
      return;
    }
    if (inspectState.lastReason === 'cancel') {
      setLockedInspectMessage('Selection cancelled.', 2400);
      return;
    }
    if (!inspectTarget) {
      clearInspectMessageLock();
      setInspectStatusMessage(null);
    }
  }, [clearInspectMessageLock, inspectState.active, inspectState.error, inspectState.lastReason, inspectState.supported, inspectTarget, setLockedInspectMessage]);

  useEffect(() => {
    if (!previewUrl && inspectState.active) {
      stopInspect();
    }
  }, [inspectState.active, previewUrl, stopInspect]);

  useEffect(() => {
    if (!inspectState.active || typeof window === 'undefined') {
      return () => {};
    }

    const handleKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        stopInspect();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [inspectState.active, stopInspect]);

  useEffect(() => {
    return () => {
      stopInspect();
      clearInspectMessageLock();
    };
  }, [clearInspectMessageLock, stopInspect]);

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

  const scheduleScreenshotCapture = useCallback((delayMs = 0, options?: { force?: boolean }) => {
    if (!currentAppIdentifier || !bridgeSupportsScreenshot || !canCaptureScreenshot) {
      return undefined;
    }

    if (!bridgeState.isReady || !iframeLoadedAt) {
      return undefined;
    }

    const now = Date.now();
    const captureFreshness = now - lastScreenshotRef.current.capturedAt;
    const isStaleCapture = lastScreenshotRef.current.surfaceId !== currentAppIdentifier
      || captureFreshness >= 3000;

    if (!options?.force) {
      if (captureInFlightRef.current) {
        return undefined;
      }
      if (!isStaleCapture) {
        return undefined;
      }
    }

    const stabilityAnchor = bridgeState.lastReadyAt ?? iframeLoadedAt;
    const enforcedDelay = !options?.force && stabilityAnchor
      ? Math.max(0, (stabilityAnchor + PREVIEW_SCREENSHOT_STABILITY_MS) - now)
      : 0;
    const totalDelay = options?.force ? delayMs : Math.max(delayMs, enforcedDelay);

    let cancelled = false;
    let timeoutId: number | null = null;

    const runCapture = () => {
      if (cancelled) {
        return;
      }

      if (captureInFlightRef.current && !options?.force) {
        return;
      }

      captureInFlightRef.current = true;
      const scale = typeof window !== 'undefined' ? window.devicePixelRatio || 1 : 1;

      void (async () => {
        try {
          const result = await requestScreenshot({ mode: 'viewport', scale });
          if (cancelled) {
            return;
          }
          const dataUrl = ensureDataUrl(result.data);
          if (dataUrl) {
            setSurfaceScreenshot('app', currentAppIdentifier, {
              dataUrl,
              width: result.width,
              height: result.height,
              capturedAt: Date.now(),
              note: result.note ?? null,
              source: 'bridge',
            });
            lastScreenshotRef.current = { surfaceId: currentAppIdentifier, capturedAt: Date.now() };
          }
        } catch (error) {
          logger.debug('Preview screenshot capture failed', error);
        } finally {
          captureInFlightRef.current = false;
        }
      })();
    };

    if (totalDelay > 0 && typeof window !== 'undefined') {
      timeoutId = window.setTimeout(runCapture, totalDelay);
    } else {
      runCapture();
    }

    return () => {
      cancelled = true;
      if (timeoutId !== null && typeof window !== 'undefined') {
        window.clearTimeout(timeoutId);
      }
    };
  }, [
    bridgeState.isReady,
    bridgeState.lastReadyAt,
    bridgeSupportsScreenshot,
    canCaptureScreenshot,
    currentAppIdentifier,
    iframeLoadedAt,
    requestScreenshot,
    setSurfaceScreenshot,
  ]);

  useEffect(() => {
    setIframeLoadedAt(null);
    setIframeLoadError(null);
  }, [previewUrl, previewReloadToken]);

  useEffect(() => {
    if (lastScreenshotRef.current.surfaceId !== currentAppIdentifier) {
      lastScreenshotRef.current = { surfaceId: currentAppIdentifier, capturedAt: 0 };
    }
  }, [currentAppIdentifier]);

  useEffect(() => {
    const previous = overlayStateRef.current;
    overlayStateRef.current = activeOverlay;

    if (activeOverlay !== 'tabs' || previous === 'tabs') {
      return;
    }

    const now = Date.now();
    const delay = iframeLoadedAt ? Math.max(0, 250 - (now - iframeLoadedAt)) : 0;
    return scheduleScreenshotCapture(delay);
  }, [activeOverlay, iframeLoadedAt, scheduleScreenshotCapture]);

  useEffect(() => {
    if (!iframeLoadedAt) {
      return;
    }

    return scheduleScreenshotCapture(200);
  }, [iframeLoadedAt, scheduleScreenshotCapture]);

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
      setStatusMessage(`${scenarioDisplayName} is not running`);
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
    scenarioDisplayName,
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
        onToggleInspect={handleToggleInspectMode}
        menuPortalContainer={previewViewNode}
        canOpenTabsOverlay={canOpenTabsOverlay}
        previewInteractionSignal={previewInteractionSignal}
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

      {shouldRenderInspectorDialog && (
        <div
          ref={inspectorDialogRef}
          className={clsx(
            'preview-inspector-dialog',
            isInspectorDragging && 'preview-inspector-dialog--dragging',
          )}
          style={{ transform: `translate3d(${Math.round(inspectorDialogPosition.x)}px, ${Math.round(inspectorDialogPosition.y)}px, 0)` }}
          role="dialog"
          aria-modal="false"
          aria-labelledby={inspectorDialogTitleId}
          onPointerDown={handleInspectorPointerDown}
          onPointerMove={handleInspectorPointerMove}
          onPointerUp={handleInspectorPointerUp}
          onPointerCancel={handleInspectorPointerUp}
          onClickCapture={handleInspectorClickCapture}
        >
          <section
            className={clsx(
              'preview-inspector',
              inspectState.active && 'preview-inspector--active',
              !inspectState.supported && 'preview-inspector--unsupported',
            )}
            aria-live="polite"
          >
            <header className="preview-inspector__header">
              <div className="preview-inspector__handle">
                <span className="preview-inspector__title" id={inspectorDialogTitleId}>
                  {inspectState.active
                    ? 'Inspecting element'
                    : inspectState.supported
                      ? 'Element inspector'
                      : 'Element inspector unavailable'}
                </span>
                <button
                  type="button"
                  className="preview-inspector__close"
                  onClick={handleInspectorDialogClose}
                  aria-label="Close element inspector"
                >
                  <X aria-hidden size={16} />
                </button>
              </div>
              <div className="preview-inspector__actions">
                {inspectTarget && (
                  <button
                    type="button"
                    className="preview-inspector__button"
                    onClick={handleCaptureElementScreenshot}
                    disabled={!inspectTarget}
                  >
                    Capture screenshot
                  </button>
                )}
                <button
                  type="button"
                  className={clsx('preview-inspector__button', inspectState.active && 'preview-inspector__button--active')}
                  onClick={handleToggleInspectMode}
                  disabled={!inspectState.supported || (!inspectState.active && !previewUrl)}
                >
                  {inspectState.active ? 'Stop' : 'Inspect element'}
                </button>
              </div>
            </header>
            {inspectStatusMessage && (
              <p
                className={clsx(
                  'preview-inspector__status',
                  (inspectState.error || (inspectStatusMessage && inspectStatusMessage.toLowerCase().includes('failed')))
                    && 'preview-inspector__status--error',
                )}
              >
                {inspectStatusMessage}
              </p>
            )}
            {inspectTarget && (
              <div className="preview-inspector__section">
                <button
                  type="button"
                  className={clsx(
                    'preview-inspector__section-toggle',
                    inspectorDetailsExpanded && 'preview-inspector__section-toggle--expanded',
                  )}
                  onClick={() => setInspectorDetailsExpanded(prev => !prev)}
                  aria-expanded={inspectorDetailsExpanded}
                  aria-controls={inspectorDetailsContentId}
                  id={inspectorDetailsSectionId}
                >
                  <span className="preview-inspector__section-label">
                    <BoxSelect aria-hidden size={16} className="preview-inspector__section-symbol" />
                    Element details
                  </span>
                  <ChevronDown aria-hidden size={16} className="preview-inspector__section-icon" />
                </button>
                <div
                  id={inspectorDetailsContentId}
                  className="preview-inspector__section-content"
                  hidden={!inspectorDetailsExpanded}
                  role="region"
                  aria-labelledby={inspectorDetailsSectionId}
                >
                  <dl className="preview-inspector__details">
                    <div className="preview-inspector__detail">
                      <dt>Element</dt>
                      <dd>
                        <code>{inspectMeta?.tag ?? 'unknown'}</code>
                        {inspectMeta?.id && (
                          <span className="preview-inspector__element-token">#{inspectMeta.id}</span>
                        )}
                        {inspectClassTokens.map(token => (
                          <span key={token} className="preview-inspector__element-token">.{token}</span>
                        ))}
                      </dd>
                    </div>
                    <div className="preview-inspector__detail">
                      <dt>Selector</dt>
                      <dd>
                        {inspectSelectorValue ? (
                          <>
                            <code>{inspectSelectorValue}</code>
                            <button type="button" className="preview-inspector__copy" onClick={handleCopySelector}>
                              Copy
                            </button>
                            {inspectCopyFeedback === 'selector' && (
                              <span className="preview-inspector__copy-feedback">Copied</span>
                            )}
                          </>
                        ) : (
                          <span className="preview-inspector__muted">Unavailable</span>
                        )}
                      </dd>
                    </div>
                    {inspectLabelValue && (
                      <div className="preview-inspector__detail">
                        <dt>Label</dt>
                        <dd>{inspectLabelValue}</dd>
                      </div>
                    )}
                    {inspectAriaDescription && (
                      <div className="preview-inspector__detail">
                        <dt>ARIA description</dt>
                        <dd>{inspectAriaDescription}</dd>
                      </div>
                    )}
                    {inspectTitleValue && (
                      <div className="preview-inspector__detail">
                        <dt>Title attribute</dt>
                        <dd>{inspectTitleValue}</dd>
                      </div>
                    )}
                    {inspectMeta?.role && (
                      <div className="preview-inspector__detail">
                        <dt>Role</dt>
                        <dd>{inspectMeta.role}</dd>
                      </div>
                    )}
                    {inspectTextPreview && (
                      <div className="preview-inspector__detail">
                        <dt>Text</dt>
                        <dd>
                          {inspectTextPreview}
                          <button type="button" className="preview-inspector__copy" onClick={handleCopyText}>
                            Copy
                          </button>
                          {inspectCopyFeedback === 'text' && (
                            <span className="preview-inspector__copy-feedback">Copied</span>
                          )}
                        </dd>
                      </div>
                    )}
                    {inspectSizeLabel && (
                      <div className="preview-inspector__detail">
                        <dt>Size</dt>
                        <dd>{inspectSizeLabel}</dd>
                      </div>
                    )}
                    {inspectPositionLabel && (
                      <div className="preview-inspector__detail">
                        <dt>Position</dt>
                        <dd>{inspectPositionLabel}</dd>
                      </div>
                    )}
                    {inspectMethodLabel && (
                      <div className="preview-inspector__detail">
                        <dt>Selection</dt>
                        <dd>{inspectMethodLabel}</dd>
                      </div>
                    )}
                  </dl>
                </div>
              </div>
            )}
            {inspectorScreenshot && (
              <div className="preview-inspector__section">
                <button
                  type="button"
                  className={clsx(
                    'preview-inspector__section-toggle',
                    'preview-inspector__section-toggle--screenshot',
                    inspectorScreenshotExpanded && 'preview-inspector__section-toggle--expanded',
                  )}
                  onClick={() => setInspectorScreenshotExpanded(prev => !prev)}
                  aria-expanded={inspectorScreenshotExpanded}
                  aria-controls={inspectorScreenshotContentId}
                  id={inspectorScreenshotSectionId}
                >
                  <span className="preview-inspector__section-label">
                    <Image aria-hidden size={16} className="preview-inspector__section-symbol" />
                    Captured screenshot
                  </span>
                  <span className="preview-inspector__section-hint">
                    {inspectorScreenshot.width} × {inspectorScreenshot.height} px
                  </span>
                  <ChevronDown aria-hidden size={16} className="preview-inspector__section-icon" />
                </button>
                <div
                  id={inspectorScreenshotContentId}
                  className="preview-inspector__section-content"
                  hidden={!inspectorScreenshotExpanded}
                  role="region"
                  aria-labelledby={inspectorScreenshotSectionId}
                >
                  <figure className="preview-inspector__screenshot">
                    <div className="preview-inspector__screenshot-frame">
                      <img
                        src={inspectorScreenshot.dataUrl}
                        alt={`Captured element screenshot (${inspectorScreenshot.width} × ${inspectorScreenshot.height} pixels)`}
                      />
                    </div>
                    <figcaption className="preview-inspector__screenshot-meta">
                      <span className="preview-inspector__screenshot-dimensions">
                        {inspectorScreenshot.width} × {inspectorScreenshot.height} px
                      </span>
                      {inspectorScreenshot.note && (
                        <span className="preview-inspector__screenshot-note">{inspectorScreenshot.note}</span>
                      )}
                      <span
                        className="preview-inspector__screenshot-filename"
                        title={inspectorScreenshot.filename}
                      >
                        {inspectorScreenshot.filename}
                      </span>
                    </figcaption>
                  </figure>
                </div>
              </div>
            )}
          </section>
        </div>
      )}

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
        />
      )}
    </div>
  );
};


export default AppPreviewView;
