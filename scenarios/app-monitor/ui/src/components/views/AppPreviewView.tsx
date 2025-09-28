import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ChangeEvent, FormEvent, KeyboardEvent, MouseEvent, PointerEvent } from 'react';
import type { BridgeLogEvent, BridgeNetworkEvent } from '@vrooli/iframe-bridge';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import clsx from 'clsx';
import { ArrowLeft, ArrowRight, Bug, ExternalLink, Info, Loader2, Power, RefreshCw, RotateCcw, ScrollText, X } from 'lucide-react';
import { appService } from '@/services/api';
import type { ReportIssueConsoleLogEntry, ReportIssueNetworkEntry, ReportIssuePayload } from '@/services/api';
import { useAppsStore } from '@/state/appsStore';

type Html2CanvasFn = (element: HTMLElement, options?: Record<string, unknown>) => Promise<HTMLCanvasElement>;

declare global {
  interface Window {
    html2canvas?: Html2CanvasFn;
  }
}

const loadHtml2Canvas = (() => {
  let loader: Promise<Html2CanvasFn> | null = null;
  return async (): Promise<Html2CanvasFn> => {
    if (typeof window !== 'undefined' && typeof window.html2canvas === 'function') {
      return window.html2canvas;
    }

    if (!loader) {
      loader = new Promise<Html2CanvasFn>((resolve, reject) => {
        const existingScript = document.querySelector<HTMLScriptElement>('script[data-html2canvas="true"]');
        if (existingScript) {
          existingScript.addEventListener('load', () => {
            if (window.html2canvas) {
              resolve(window.html2canvas);
            } else {
              reject(new Error('html2canvas failed to initialize.'));
            }
          });
          existingScript.addEventListener('error', () => reject(new Error('Failed to load html2canvas script.')));
          return;
        }

        const script = document.createElement('script');
        script.src = 'https://cdn.jsdelivr.net/npm/html2canvas@1.4.1/dist/html2canvas.min.js';
        script.async = true;
        script.crossOrigin = 'anonymous';
        script.dataset.html2canvas = 'true';
        script.onload = () => {
          if (window.html2canvas) {
            resolve(window.html2canvas);
          } else {
            reject(new Error('html2canvas failed to initialize.'));
          }
        };
        script.onerror = () => reject(new Error('Failed to load html2canvas script.'));
        document.head.appendChild(script);
      });
    }

    return loader;
  };
})();
import { logger } from '@/services/logger';
import type { App } from '@/types';
import AppModal from '../AppModal';
import { buildPreviewUrl, isRunningStatus, isStoppedStatus, locateAppByIdentifier } from '@/utils/appPreview';
import { useIframeBridge } from '@/hooks/useIframeBridge';
import type { BridgeComplianceResult } from '@/hooks/useIframeBridge';
import './AppPreviewView.css';

type ConsoleSeverity = 'error' | 'warn' | 'info' | 'log' | 'debug' | 'trace';

interface ReportConsoleEntry {
  display: string;
  severity: ConsoleSeverity;
  payload: ReportIssueConsoleLogEntry;
}

interface ReportNetworkEntry {
  display: string;
  payload: ReportIssueNetworkEntry;
}

interface ReportAppLogStream {
  key: string;
  label: string;
  type: 'lifecycle' | 'background';
  lines: string[];
  total: number;
  command?: string;
}

const normalizeConsoleLevel = (level: string | null | undefined): ConsoleSeverity => {
  const normalized = (level ?? '').toString().toLowerCase();
  if (['error', 'err', 'fatal', 'severe'].includes(normalized)) {
    return 'error';
  }
  if (['warn', 'warning'].includes(normalized)) {
    return 'warn';
  }
  if (['info', 'information', 'notice'].includes(normalized)) {
    return 'info';
  }
  if (['debug', 'verbose'].includes(normalized)) {
    return 'debug';
  }
  if (normalized === 'trace') {
    return 'trace';
  }
  if (['log', 'default'].includes(normalized)) {
    return 'log';
  }
  return 'log';
};

const AppPreviewView = () => {
  const apps = useAppsStore(state => state.apps);
  const setAppsState = useAppsStore(state => state.setAppsState);
  const loadApps = useAppsStore(state => state.loadApps);
  const loadingInitial = useAppsStore(state => state.loadingInitial);
  const hasInitialized = useAppsStore(state => state.hasInitialized);
  const navigate = useNavigate();
  const { appId } = useParams<{ appId: string }>();
  const location = useLocation();
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
  const [reportMessage, setReportMessage] = useState('');
  const [reportIncludeScreenshot, setReportIncludeScreenshot] = useState(false);
  const [reportSubmitting, setReportSubmitting] = useState(false);
  const [reportError, setReportError] = useState<string | null>(null);
  const [reportResult, setReportResult] = useState<{ issueId?: string; issueUrl?: string; message?: string } | null>(null);
  const [reportScreenshotData, setReportScreenshotData] = useState<string | null>(null);
  const [reportScreenshotOriginalData, setReportScreenshotOriginalData] = useState<string | null>(null);
  const [reportScreenshotLoading, setReportScreenshotLoading] = useState(false);
  const [reportScreenshotError, setReportScreenshotError] = useState<string | null>(null);
  const [reportScreenshotOriginalDimensions, setReportScreenshotOriginalDimensions] = useState<{ width: number; height: number } | null>(null);
  const [reportSelectionRect, setReportSelectionRect] = useState<{ x: number; y: number; width: number; height: number } | null>(null);
  const [reportScreenshotClip, setReportScreenshotClip] = useState<{ x: number; y: number; width: number; height: number } | null>(null);
  const [reportScreenshotInfo, setReportScreenshotInfo] = useState<string | null>(null);
  const [reportAppLogs, setReportAppLogs] = useState<string[]>([]);
  const [reportAppLogsTotal, setReportAppLogsTotal] = useState<number | null>(null);
  const [reportAppLogsLoading, setReportAppLogsLoading] = useState(false);
  const [reportAppLogsError, setReportAppLogsError] = useState<string | null>(null);
  const [reportAppLogsExpanded, setReportAppLogsExpanded] = useState(false);
  const [reportAppLogsFetchedAt, setReportAppLogsFetchedAt] = useState<number | null>(null);
  const [reportAppLogStreams, setReportAppLogStreams] = useState<ReportAppLogStream[]>([]);
  const [reportAppLogSelections, setReportAppLogSelections] = useState<Record<string, boolean>>({});
  const [reportIncludeAppLogs, setReportIncludeAppLogs] = useState(true);
  const [reportConsoleLogs, setReportConsoleLogs] = useState<ReportConsoleEntry[]>([]);
  const [reportConsoleLogsTotal, setReportConsoleLogsTotal] = useState<number | null>(null);
  const [reportConsoleLogsLoading, setReportConsoleLogsLoading] = useState(false);
  const [reportConsoleLogsError, setReportConsoleLogsError] = useState<string | null>(null);
  const [reportConsoleLogsExpanded, setReportConsoleLogsExpanded] = useState(false);
  const [reportConsoleLogsFetchedAt, setReportConsoleLogsFetchedAt] = useState<number | null>(null);
  const [reportIncludeConsoleLogs, setReportIncludeConsoleLogs] = useState(true);
  const [reportNetworkEvents, setReportNetworkEvents] = useState<ReportNetworkEntry[]>([]);
  const [reportNetworkTotal, setReportNetworkTotal] = useState<number | null>(null);
  const [reportNetworkLoading, setReportNetworkLoading] = useState(false);
  const [reportNetworkError, setReportNetworkError] = useState<string | null>(null);
  const [reportNetworkExpanded, setReportNetworkExpanded] = useState(false);
  const [reportNetworkFetchedAt, setReportNetworkFetchedAt] = useState<number | null>(null);
  const [reportIncludeNetworkRequests, setReportIncludeNetworkRequests] = useState(true);
  const [previewReloadToken, setPreviewReloadToken] = useState(0);
  const [previewOverlay, setPreviewOverlay] = useState<null | { type: 'restart' | 'waiting' | 'error'; message: string }>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const [bridgeCompliance, setBridgeCompliance] = useState<BridgeComplianceResult | null>(null);
  const complianceRunRef = useRef(false);
  const initialPreviewUrlRef = useRef<string | null>(null);
  const reportScreenshotContainerRef = useRef<HTMLDivElement | null>(null);
  const reportScreenshotContainerSizeRef = useRef<{ width: number; height: number } | null>(null);
  const reportDragStateRef = useRef<{ pointerId: number; startX: number; startY: number; containerWidth: number; containerHeight: number } | null>(null);
  const restartMonitorRef = useRef<{ cancel: () => void } | null>(null);
  const lastRefreshRequestRef = useRef(0);
  const lastRecordedViewRef = useRef<{ id: string | null; timestamp: number }>({ id: null, timestamp: 0 });
  const reportAppLogsFetchedForRef = useRef<string | null>(null);
  const reportAppLogsPanelId = 'app-report-dialog-logs';
  const REPORT_APP_LOGS_MAX_LINES = 200;
  const reportConsoleLogsFetchedForRef = useRef<string | null>(null);
  const reportNetworkFetchedForRef = useRef<string | null>(null);
  const reportConsoleLogsPanelId = 'app-report-dialog-console';
  const reportNetworkPanelId = 'app-report-dialog-network';
  const REPORT_CONSOLE_LOGS_MAX_LINES = 150;
  const REPORT_NETWORK_MAX_EVENTS = 150;
  const MAX_CONSOLE_TEXT_LENGTH = 2000;
  const MAX_NETWORK_URL_LENGTH = 2048;
  const MAX_NETWORK_ERROR_LENGTH = 1500;
  const MAX_NETWORK_REQUEST_ID_LENGTH = 128;

  useEffect(() => {
    if (!hasInitialized && !loadingInitial) {
      void loadApps();
    }
  }, [hasInitialized, loadApps, loadingInitial]);
  const logsTruncated = useMemo(() => (
    typeof reportAppLogsTotal === 'number' && reportAppLogsTotal > reportAppLogs.length
  ), [reportAppLogs.length, reportAppLogsTotal]);
  const formattedReportLogsTime = useMemo(() => {
    if (!reportAppLogsFetchedAt) {
      return null;
    }
    try {
      return new Date(reportAppLogsFetchedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [reportAppLogsFetchedAt]);

  const consoleLogsTruncated = useMemo(() => (
    typeof reportConsoleLogsTotal === 'number' && reportConsoleLogsTotal > reportConsoleLogs.length
  ), [reportConsoleLogs.length, reportConsoleLogsTotal]);

  const formattedConsoleLogsTime = useMemo(() => {
    if (!reportConsoleLogsFetchedAt) {
      return null;
    }
    try {
      return new Date(reportConsoleLogsFetchedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [reportConsoleLogsFetchedAt]);

  const networkEventsTruncated = useMemo(() => (
    typeof reportNetworkTotal === 'number' && reportNetworkTotal > reportNetworkEvents.length
  ), [reportNetworkEvents.length, reportNetworkTotal]);

  const selectedReportLogStreams = useMemo(() => (
    reportAppLogStreams.filter(stream => reportAppLogSelections[stream.key] !== false)
  ), [reportAppLogSelections, reportAppLogStreams]);

  const selectedReportLogCount = selectedReportLogStreams.length;
  const totalReportLogStreamCount = reportAppLogStreams.length;

  const formattedNetworkCapturedAt = useMemo(() => {
    if (!reportNetworkFetchedAt) {
      return null;
    }
    try {
      return new Date(reportNetworkFetchedAt).toLocaleTimeString();
    } catch {
      return null;
    }
  }, [reportNetworkFetchedAt]);

  const trimForPayload = (value: string, max: number): string => {
    if (typeof value !== 'string' || value.length === 0) {
      return value;
    }
    if (!Number.isFinite(max) || max <= 0 || value.length <= max) {
      return value;
    }
    return `${value.slice(0, max)}...`;
  };

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

  const resetPreviewState = useCallback((options?: { force?: boolean }) => {
    if (!options?.force && hasCustomPreviewUrl) {
      return;
    }

    setPreviewUrl(null);
    setPreviewUrlInput('');
    setHistory([]);
    setHistoryIndex(-1);
    initialPreviewUrlRef.current = null;
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

  const stopRestartMonitor = useCallback(() => {
    if (restartMonitorRef.current) {
      restartMonitorRef.current.cancel();
      restartMonitorRef.current = null;
    }
  }, []);

  const reloadPreview = useCallback(() => {
    resetState();
    setPreviewReloadToken(prev => prev + 1);
  }, [resetState]);

  const beginRestartMonitor = useCallback((appIdentifier: string) => {
    stopRestartMonitor();

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
                setStatusMessage('Application restarted. Refreshing preview...');
                setPreviewOverlay(null);
                setLoading(false);
                reloadPreview();
                stopRestartMonitor();
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
          logger.warn('Restart monitor poll failed', error);
        }

        const delay = attempt < 5 ? 1000 : 2000;
        await new Promise(resolve => window.setTimeout(resolve, delay));
      }

      if (!cancelled) {
        setLoading(false);
        setPreviewOverlay({ type: 'error', message: 'Application has not come back online yet. Try refreshing.' });
      }
    };

    void poll();
  }, [applyDefaultPreviewUrl, commitAppUpdate, hasCustomPreviewUrl, reloadPreview, setLoading, setPreviewOverlay, setStatusMessage, stopRestartMonitor]);

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

  const captureIframeScreenshot = useCallback(async () => {
    if (!reportDialogOpen || !reportIncludeScreenshot || !canCaptureScreenshot) {
      return;
    }

    setReportScreenshotLoading(true);
    setReportScreenshotError(null);
    setReportScreenshotInfo(null);
    setReportScreenshotClip(null);
    setReportSelectionRect(null);
    setReportScreenshotOriginalData(null);
    setReportScreenshotData(null);
    setReportScreenshotOriginalDimensions(null);
    reportScreenshotContainerSizeRef.current = null;

    const produceFromCanvas = (canvas: HTMLCanvasElement, infoMessage?: string) => {
      const dataUrl = canvas.toDataURL('image/png');
      const base64 = dataUrl.replace(/^data:image\/png;base64,/, '');
      setReportScreenshotOriginalData(base64);
      setReportScreenshotData(base64);
      setReportScreenshotOriginalDimensions({ width: canvas.width, height: canvas.height });
      if (infoMessage) {
        setReportScreenshotInfo(infoMessage);
      }
      return true;
    };

    const wrapText = (
      context: CanvasRenderingContext2D,
      text: string,
      x: number,
      y: number,
      maxWidth: number,
      lineHeight: number,
    ) => {
      const words = text.split(/\s+/);
      let line = '';
      let currentY = y;
      for (const word of words) {
        const testLine = line ? `${line} ${word}` : word;
        const metrics = context.measureText(testLine);
        if (metrics.width > maxWidth && line) {
          context.fillText(line, x, currentY);
          line = word;
          currentY += lineHeight;
        } else {
          line = testLine;
        }
      }
      if (line) {
        context.fillText(line, x, currentY);
        currentY += lineHeight;
      }
      return currentY;
    };

    const createPlaceholderCanvas = (
      width: number,
      height: number,
      reason: 'cross-origin' | 'fallback',
    ): HTMLCanvasElement | null => {
      const canvas = document.createElement('canvas');
      const paddedWidth = Math.max(320, Math.round(width) || 0) || 640;
      const paddedHeight = Math.max(220, Math.round(height) || 0) || 360;
      canvas.width = paddedWidth;
      canvas.height = paddedHeight;
      const ctx = canvas.getContext('2d');
      if (!ctx) {
        return null;
      }

      const gradient = ctx.createLinearGradient(0, 0, paddedWidth, paddedHeight);
      gradient.addColorStop(0, '#01161b');
      gradient.addColorStop(1, '#042a31');
      ctx.fillStyle = gradient;
      ctx.fillRect(0, 0, paddedWidth, paddedHeight);

      ctx.strokeStyle = 'rgba(0, 255, 255, 0.5)';
      ctx.lineWidth = 2;
      ctx.strokeRect(18, 18, paddedWidth - 36, paddedHeight - 36);

      ctx.fillStyle = 'rgba(224, 255, 249, 0.9)';
      ctx.font = '20px "Inter", "Segoe UI", sans-serif';
      ctx.textBaseline = 'top';
      ctx.fillText('Preview capture not available', 32, 34);

      ctx.font = '15px "Inter", "Segoe UI", sans-serif';
      ctx.fillStyle = 'rgba(224, 255, 249, 0.75)';
      const reasonText = reason === 'cross-origin'
        ? 'The application preview runs on a different origin, so browsers block direct screenshots.'
        : 'The preview could not be captured automatically. A fallback frame is attached instead.';
      const detailsBaseline = wrapText(ctx, reasonText, 32, 66, paddedWidth - 64, 22);

      if (activePreviewUrl) {
        ctx.fillStyle = 'rgba(0, 255, 255, 0.85)';
        ctx.font = '15px "Inter", "Segoe UI", sans-serif';
        wrapText(ctx, activePreviewUrl, 32, Math.min(detailsBaseline + 20, paddedHeight - 80), paddedWidth - 64, 20);
      }

      return canvas;
    };

    const captureContainerCanvas = async (
      reason: 'cross-origin' | 'missing-iframe' | 'fallback',
    ): Promise<boolean> => {
      const container = previewContainerRef.current;
      if (!container) {
        setReportScreenshotError('Preview container is not available for capture.');
        if (reason !== 'fallback') {
          setReportIncludeScreenshot(false);
        }
        return false;
      }

      try {
        const html2canvas = await loadHtml2Canvas();
        const containerBackground = window.getComputedStyle(container).backgroundColor || '#060a06';
        const canvas = await html2canvas(container, {
          backgroundColor: containerBackground,
          logging: false,
          useCORS: true,
          scale: window.devicePixelRatio || 1,
        });
        const infoMessage = reason === 'cross-origin'
          ? 'Captured the preview frame using a fallback because the app runs on a different origin. The embedded content may appear blank.'
          : 'Captured the preview frame using a fallback of the surrounding interface.';
        return produceFromCanvas(canvas, infoMessage);
      } catch (error) {
        console.error('Failed to capture preview frame container', error);
        const bounds = container.getBoundingClientRect();
        const placeholder = createPlaceholderCanvas(bounds.width, bounds.height, reason === 'cross-origin' ? 'cross-origin' : 'fallback');
        if (placeholder) {
          const infoMessage = reason === 'cross-origin'
            ? 'Attached a placeholder frame because the app runs on a different origin.'
            : 'Attached a placeholder frame because the live preview could not be captured automatically.';
          return produceFromCanvas(placeholder, infoMessage);
        }
        setReportScreenshotError('Unable to capture this preview automatically. Try opening the app in a new tab and attach a manual screenshot.');
        setReportIncludeScreenshot(false);
        return false;
      }
    };

    const captureIframeCanvas = async (): Promise<'captured' | 'cross-origin' | 'missing' | 'failed'> => {
      const iframe = iframeRef.current;
      if (!iframe) {
        return 'missing';
      }

      let iframeDocument: Document | null = null;

      try {
        iframeDocument = iframe.contentDocument;
      } catch (error) {
        console.warn('Restricted from accessing iframe document (likely cross-origin).', error);
        return 'cross-origin';
      }

      if (!iframeDocument) {
        try {
          iframeDocument = iframe.contentWindow ? iframe.contentWindow.document : null;
        } catch (error) {
          console.warn('Restricted from accessing iframe window document (cross-origin).', error);
          return 'cross-origin';
        }
      }

      if (!iframeDocument) {
        return 'missing';
      }

      try {
        const html2canvas = await loadHtml2Canvas();
        const target = iframeDocument.documentElement as HTMLElement;
        const computedBackground = iframeDocument.body ? window.getComputedStyle(iframeDocument.body).backgroundColor : undefined;
        const backgroundColor = computedBackground && computedBackground !== 'rgba(0, 0, 0, 0)'
          ? computedBackground
          : iframeDocument.body?.style?.backgroundColor || undefined;
        const canvas = await html2canvas(target, {
          backgroundColor,
          logging: false,
          useCORS: true,
          scale: window.devicePixelRatio || 1,
        });
        produceFromCanvas(canvas);
        return 'captured';
      } catch (error) {
        console.error('Failed to capture iframe screenshot', error);
        return 'failed';
      }
    };

    let handled = false;

    if (bridgeSupportsScreenshot) {
      try {
        const result = await requestScreenshot({ scale: window.devicePixelRatio || 1 });
        setReportScreenshotOriginalData(result.data);
        setReportScreenshotData(result.data);
        setReportScreenshotOriginalDimensions({ width: result.width, height: result.height });
        setReportScreenshotInfo(result.note ?? null);
        handled = true;
      } catch (error) {
        logger.warn('Bridge screenshot capture failed, falling back to client capture', error);
      }
    }

    if (handled) {
      setReportScreenshotLoading(false);
      return;
    }

    try {
      if (!isPreviewSameOrigin) {
        await captureContainerCanvas('cross-origin');
      } else {
        const result = await captureIframeCanvas();
        if (result === 'captured') {
          handled = true;
          return;
        }
        if (result === 'cross-origin') {
          await captureContainerCanvas('cross-origin');
        } else if (result === 'missing') {
          await captureContainerCanvas('missing-iframe');
        } else {
          await captureContainerCanvas('fallback');
        }
      }
    } finally {
      setReportScreenshotLoading(false);
    }
  }, [
    activePreviewUrl,
    bridgeSupportsScreenshot,
    canCaptureScreenshot,
    isPreviewSameOrigin,
    reportDialogOpen,
    reportIncludeScreenshot,
    requestScreenshot,
  ]);

  useEffect(() => {
    if (!appId) {
      navigate({
        pathname: '/apps',
        search: location.search || undefined,
      }, { replace: true });
    }
  }, [appId, location.search, navigate]);

  useEffect(() => {
    setFetchAttempted(false);
    stopRestartMonitor();
    setPreviewOverlay(null);
  }, [appId, stopRestartMonitor]);

  useEffect(() => {
    return () => {
      stopRestartMonitor();
    };
  }, [stopRestartMonitor]);

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
    await executeAppAction(appToControl, action);
  }, [executeAppAction]);

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

    const targetId = currentApp.id;
    setPreviewOverlay({ type: 'restart', message: 'Restarting application...' });
    setLoading(true);
    reloadPreview();

    void executeAppAction(targetId, 'restart').then(success => {
      if (!success) {
        setLoading(false);
        setPreviewOverlay({ type: 'error', message: 'Unable to restart the application. Check logs for details.' });
        return;
      }

      setPreviewOverlay({ type: 'waiting', message: 'Waiting for application to restart...' });
      beginRestartMonitor(targetId);
    });
  }, [beginRestartMonitor, currentApp, executeAppAction, pendingAction, reloadPreview]);

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
  }, [bridgeState.isSupported, history, historyIndex, sendBridgeNav]);

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
      navigate(`/logs/${currentApp.id}`);
    }
  }, [currentApp, navigate]);

  const resolveReportLogIdentifier = useCallback(() => {
    const candidates = [currentApp?.scenario_name, currentApp?.id, appId]
      .map(value => (typeof value === 'string' ? value.trim() : ''))
      .filter(value => value.length > 0);

    if (candidates.length === 0) {
      return null;
    }

    return candidates[0];
  }, [appId, currentApp]);

  const describeLogValue = useCallback((value: unknown): string => {
    if (value === null) {
      return 'null';
    }
    if (value === undefined) {
      return 'undefined';
    }
    if (typeof value === 'string') {
      return value;
    }
    if (typeof value === 'number' || typeof value === 'boolean') {
      return String(value);
    }
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }, []);

  const buildConsoleEventBody = useCallback((event: BridgeLogEvent): string => {
    const segments: string[] = [];
    if (event.message) {
      segments.push(event.message);
    }
    if (Array.isArray(event.args) && event.args.length > 0) {
      segments.push(event.args.map(describeLogValue).join(' '));
    }
    if (event.context && Object.keys(event.context).length > 0) {
      segments.push(describeLogValue(event.context));
    }
    const body = segments.filter(Boolean).join(' ');
    return body || '(no console output)';
  }, [describeLogValue]);

  const formatBridgeLogEvent = useCallback((event: BridgeLogEvent): string => {
    const timestamp = (() => {
      try {
        return new Date(event.ts).toLocaleTimeString();
      } catch {
        return String(event.ts);
      }
    })();
    const level = event.level.toUpperCase();
    const source = event.source;
    const body = buildConsoleEventBody(event);
    return `${timestamp} [${source}/${level}] ${body}`.trim();
  }, [buildConsoleEventBody]);

  const toConsoleEntry = (event: BridgeLogEvent): ReportConsoleEntry => ({
    display: formatBridgeLogEvent(event),
    severity: normalizeConsoleLevel(event.level),
    payload: {
      ts: event.ts,
      level: event.level,
      source: event.source,
      text: trimForPayload(buildConsoleEventBody(event), MAX_CONSOLE_TEXT_LENGTH),
    },
  });

  const formatBridgeNetworkEvent = (event: BridgeNetworkEvent): string => {
    const timestamp = (() => {
      try {
        return new Date(event.ts).toLocaleTimeString();
      } catch {
        return String(event.ts);
      }
    })();
    const method = event.method ? event.method.toUpperCase() : 'GET';
    const url = (event.url && event.url.trim()) || '(unknown URL)';
    let status = 'pending';
    if (typeof event.status === 'number') {
      status = String(event.status);
    } else if (typeof event.ok === 'boolean') {
      status = event.ok ? 'ok' : 'error';
    }
    const extras: string[] = [];
    if (typeof event.durationMs === 'number' && Number.isFinite(event.durationMs)) {
      extras.push(`${Math.max(0, Math.round(event.durationMs))}ms`);
    }
    if (event.error) {
      extras.push(`error: ${event.error}`);
    }
    if (event.requestId) {
      extras.push(`id=${event.requestId}`);
    }
    const tail = extras.length > 0 ? ` (${extras.join(', ')})` : '';
    return `${timestamp} ${method} ${url} -> ${status}${tail}`;
  };

  const toNetworkEntry = (event: BridgeNetworkEvent): ReportNetworkEntry => {
    const sanitizedURL = trimForPayload((event.url || '').trim() || '(unknown URL)', MAX_NETWORK_URL_LENGTH);
    const sanitizedError = trimForPayload(event.error ?? '', MAX_NETWORK_ERROR_LENGTH).trim();
    const sanitizedRequestId = trimForPayload(event.requestId ?? '', MAX_NETWORK_REQUEST_ID_LENGTH).trim();
    const duration = typeof event.durationMs === 'number' && Number.isFinite(event.durationMs)
      ? Math.max(0, Math.round(event.durationMs))
      : undefined;

    return {
      display: formatBridgeNetworkEvent({ ...event, url: sanitizedURL, error: sanitizedError, requestId: sanitizedRequestId }),
      payload: {
        ts: event.ts,
        kind: event.kind,
        method: event.method ? event.method.toUpperCase() : 'GET',
        url: sanitizedURL,
        status: typeof event.status === 'number' ? event.status : undefined,
        ok: typeof event.ok === 'boolean' ? event.ok : undefined,
        durationMs: duration,
        error: sanitizedError || undefined,
        requestId: sanitizedRequestId || undefined,
      },
    };
  };

  const fetchReportAppLogs = useCallback(async (options?: { force?: boolean }) => {
    const identifier = resolveReportLogIdentifier();
    if (!identifier) {
      setReportAppLogs([]);
      setReportAppLogsTotal(null);
      setReportAppLogsError('Unable to determine which logs to include.');
      setReportAppLogsFetchedAt(null);
      reportAppLogsFetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && reportAppLogsFetchedForRef.current === normalizedIdentifier) {
      return;
    }

    setReportAppLogsLoading(true);
    setReportAppLogsError(null);

    try {
      const result = await appService.getAppLogs(identifier, 'both');
      const rawLogs = Array.isArray(result.logs) ? result.logs : [];
      const trimmedLogs = rawLogs.slice(-REPORT_APP_LOGS_MAX_LINES);

      setReportAppLogs(trimmedLogs);
      setReportAppLogsTotal(rawLogs.length);
      const streams = Array.isArray(result.streams) ? result.streams : [];
      const normalizedStreams: ReportAppLogStream[] = streams.map((stream) => {
        const safeLines = Array.isArray(stream.lines) ? stream.lines : [];
        const trimmedLines = safeLines.slice(-REPORT_APP_LOGS_MAX_LINES);
        let label = stream.label || stream.key;
        if (!label) {
          label = stream.type === 'lifecycle' ? 'Lifecycle' : stream.step || 'Background';
        }
        if (stream.type === 'lifecycle') {
          label = 'Lifecycle';
        }
        return {
          key: stream.key,
          label,
          type: stream.type,
          lines: trimmedLines,
          total: safeLines.length,
          command: stream.command,
        };
      });

      setReportAppLogStreams(normalizedStreams);
      setReportAppLogSelections(previous => normalizedStreams.reduce<Record<string, boolean>>((acc, stream) => {
        const prior = previous?.[stream.key];
        acc[stream.key] = prior !== undefined ? prior : true;
        return acc;
      }, {}));
      setReportAppLogsFetchedAt(Date.now());
      reportAppLogsFetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.error('Failed to load logs for issue report', error);
      setReportAppLogs([]);
      setReportAppLogsTotal(null);
      setReportAppLogStreams([]);
      setReportAppLogSelections({});
      setReportAppLogsError('Unable to load logs. Try again.');
      setReportAppLogsFetchedAt(null);
      reportAppLogsFetchedForRef.current = null;
    } finally {
      setReportAppLogsLoading(false);
    }
  }, [REPORT_APP_LOGS_MAX_LINES, logger, resolveReportLogIdentifier]);

  const fetchReportConsoleLogs = useCallback(async (options?: { force?: boolean }) => {
    const identifier = resolveReportLogIdentifier();
    if (!identifier) {
      setReportConsoleLogs([]);
      setReportConsoleLogsTotal(null);
      setReportConsoleLogsError('Unable to determine which console logs to include.');
      setReportConsoleLogsFetchedAt(null);
      reportConsoleLogsFetchedForRef.current = null;
      return;
    }

    if (!bridgeState.isSupported || !bridgeState.caps.includes('logs')) {
      setReportConsoleLogs([]);
      setReportConsoleLogsTotal(null);
      setReportConsoleLogsError('Console log capture is not available for this preview.');
      setReportConsoleLogsFetchedAt(null);
      reportConsoleLogsFetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && reportConsoleLogsFetchedForRef.current === normalizedIdentifier) {
      return;
    }

    setReportConsoleLogsLoading(true);
    setReportConsoleLogsError(null);

    try {
      if (logState && configureLogs && logState.enabled === false) {
        configureLogs({ enable: true });
      }

      let events: BridgeLogEvent[] = [];
      try {
        events = await requestLogBatch({ limit: REPORT_CONSOLE_LOGS_MAX_LINES });
      } catch (error) {
        logger.warn('Console log snapshot failed; using buffered events', error);
        events = getRecentLogs();
      }

      if (!Array.isArray(events)) {
        events = [];
      }

      const limited = events.slice(-REPORT_CONSOLE_LOGS_MAX_LINES);
      const entries = limited.map(toConsoleEntry);

      setReportConsoleLogs(entries);
      setReportConsoleLogsTotal(events.length);
      setReportConsoleLogsFetchedAt(Date.now());
      reportConsoleLogsFetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.warn('Failed to load console logs for issue report', error);
      setReportConsoleLogs([]);
      setReportConsoleLogsTotal(null);
      setReportConsoleLogsError('Unable to load console logs from the preview iframe.');
      setReportConsoleLogsFetchedAt(null);
      reportConsoleLogsFetchedForRef.current = null;
    } finally {
      setReportConsoleLogsLoading(false);
    }
  }, [
    REPORT_CONSOLE_LOGS_MAX_LINES,
    bridgeState.caps,
    bridgeState.isSupported,
    configureLogs,
    getRecentLogs,
    logger,
    logState,
    requestLogBatch,
    resolveReportLogIdentifier,
    toConsoleEntry,
  ]);

  const fetchReportNetworkEvents = useCallback(async (options?: { force?: boolean }) => {
    const identifier = resolveReportLogIdentifier();
    if (!identifier) {
      setReportNetworkEvents([]);
      setReportNetworkTotal(null);
      setReportNetworkError('Unable to determine which network events to include.');
      setReportNetworkFetchedAt(null);
      reportNetworkFetchedForRef.current = null;
      return;
    }

    if (!bridgeState.isSupported || !bridgeState.caps.includes('network')) {
      setReportNetworkEvents([]);
      setReportNetworkTotal(null);
      setReportNetworkError('Network capture is not available for this preview.');
      setReportNetworkFetchedAt(null);
      reportNetworkFetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && reportNetworkFetchedForRef.current === normalizedIdentifier) {
      return;
    }

    setReportNetworkLoading(true);
    setReportNetworkError(null);

    try {
      if (networkState && configureNetwork && networkState.enabled === false) {
        configureNetwork({ enable: true });
      }

      let events: BridgeNetworkEvent[] = [];
      try {
        events = await requestNetworkBatch({ limit: REPORT_NETWORK_MAX_EVENTS });
      } catch (error) {
        logger.warn('Network snapshot failed; using buffered events', error);
        events = getRecentNetworkEvents();
      }

      if (!Array.isArray(events)) {
        events = [];
      }

      const limited = events.slice(-REPORT_NETWORK_MAX_EVENTS);
      const entries = limited.map(toNetworkEntry);

      setReportNetworkEvents(entries);
      setReportNetworkTotal(events.length);
      setReportNetworkFetchedAt(Date.now());
      reportNetworkFetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.warn('Failed to load network events for issue report', error);
      setReportNetworkEvents([]);
      setReportNetworkTotal(null);
      setReportNetworkError('Unable to load network requests from the preview iframe.');
      setReportNetworkFetchedAt(null);
      reportNetworkFetchedForRef.current = null;
    } finally {
      setReportNetworkLoading(false);
    }
  }, [
    REPORT_NETWORK_MAX_EVENTS,
    bridgeState.caps,
    bridgeState.isSupported,
    configureNetwork,
    getRecentNetworkEvents,
    logger,
    networkState,
    requestNetworkBatch,
    resolveReportLogIdentifier,
    toNetworkEntry,
  ]);

  const handleToggleReportAppLogs = useCallback(() => {
    setReportAppLogsExpanded(prev => {
      const next = !prev;
      if (next && !reportAppLogsLoading && reportAppLogs.length === 0 && !reportAppLogsError) {
        void fetchReportAppLogs({ force: true });
      }
      return next;
    });
  }, [fetchReportAppLogs, reportAppLogs.length, reportAppLogsError, reportAppLogsLoading]);

  const handleReportLogStreamToggle = useCallback((key: string, checked: boolean) => {
    setReportAppLogSelections(prev => ({
      ...prev,
      [key]: checked,
    }));
  }, []);

  const handleRefreshReportAppLogs = useCallback(() => {
    void fetchReportAppLogs({ force: true });
  }, [fetchReportAppLogs]);

  const handleToggleReportConsoleLogs = useCallback(() => {
    setReportConsoleLogsExpanded(prev => {
      const next = !prev;
      if (next && !reportConsoleLogsLoading && reportConsoleLogs.length === 0 && !reportConsoleLogsError) {
        void fetchReportConsoleLogs({ force: true });
      }
      return next;
    });
  }, [fetchReportConsoleLogs, reportConsoleLogs.length, reportConsoleLogsError, reportConsoleLogsLoading]);

  const handleRefreshReportConsoleLogs = useCallback(() => {
    void fetchReportConsoleLogs({ force: true });
  }, [fetchReportConsoleLogs]);

  const handleToggleReportNetworkEvents = useCallback(() => {
    setReportNetworkExpanded(prev => {
      const next = !prev;
      if (next && !reportNetworkLoading && reportNetworkEvents.length === 0 && !reportNetworkError) {
        void fetchReportNetworkEvents({ force: true });
      }
      return next;
    });
  }, [fetchReportNetworkEvents, reportNetworkError, reportNetworkEvents.length, reportNetworkLoading]);

  const handleRefreshReportNetworkEvents = useCallback(() => {
    void fetchReportNetworkEvents({ force: true });
  }, [fetchReportNetworkEvents]);

  const handleOpenReportDialog = useCallback(() => {
    setReportDialogOpen(true);
    setReportMessage('');
    setReportError(null);
    setReportResult(null);
    setReportScreenshotOriginalData(null);
    setReportScreenshotData(null);
    setReportScreenshotError(null);
    setReportScreenshotInfo(null);
    setReportScreenshotLoading(false);
    setReportScreenshotOriginalDimensions(null);
    setReportSelectionRect(null);
    setReportScreenshotClip(null);
    reportScreenshotContainerSizeRef.current = null;
    setReportAppLogs([]);
    setReportAppLogsTotal(null);
    setReportAppLogsError(null);
    setReportAppLogsExpanded(false);
    setReportAppLogsFetchedAt(null);
    reportAppLogsFetchedForRef.current = null;
    setReportConsoleLogs([]);
    setReportConsoleLogsTotal(null);
    setReportConsoleLogsError(null);
    setReportConsoleLogsExpanded(false);
    setReportConsoleLogsFetchedAt(null);
    reportConsoleLogsFetchedForRef.current = null;
    setReportNetworkEvents([]);
    setReportNetworkTotal(null);
    setReportNetworkError(null);
    setReportNetworkExpanded(false);
    setReportNetworkFetchedAt(null);
    reportNetworkFetchedForRef.current = null;
    setReportIncludeAppLogs(true);
    setReportIncludeConsoleLogs(true);
    setReportIncludeNetworkRequests(true);
    setReportIncludeScreenshot(canCaptureScreenshot);
    if (canCaptureScreenshot) {
      void captureIframeScreenshot();
    }
    void fetchReportAppLogs({ force: true });
    void fetchReportConsoleLogs({ force: true });
    void fetchReportNetworkEvents({ force: true });
  }, [
    canCaptureScreenshot,
    captureIframeScreenshot,
    fetchReportAppLogs,
    fetchReportConsoleLogs,
    fetchReportNetworkEvents,
  ]);

  const handleCloseReportDialog = useCallback(() => {
    setReportDialogOpen(false);
    setReportSubmitting(false);
    setReportError(null);
    setReportResult(null);
    setReportMessage('');
    setReportIncludeScreenshot(canCaptureScreenshot);
    setReportScreenshotOriginalData(null);
    setReportScreenshotData(null);
    setReportScreenshotError(null);
    setReportScreenshotInfo(null);
    setReportScreenshotLoading(false);
    setReportScreenshotOriginalDimensions(null);
    setReportSelectionRect(null);
    setReportScreenshotClip(null);
    reportScreenshotContainerSizeRef.current = null;
    setReportAppLogs([]);
    setReportAppLogsTotal(null);
    setReportAppLogsError(null);
    setReportAppLogsExpanded(false);
    setReportAppLogsFetchedAt(null);
    setReportAppLogsLoading(false);
    setReportAppLogStreams([]);
    setReportAppLogSelections({});
    reportAppLogsFetchedForRef.current = null;
    setReportConsoleLogs([]);
    setReportConsoleLogsTotal(null);
    setReportConsoleLogsError(null);
    setReportConsoleLogsExpanded(false);
    setReportConsoleLogsFetchedAt(null);
    setReportConsoleLogsLoading(false);
    reportConsoleLogsFetchedForRef.current = null;
    setReportNetworkEvents([]);
    setReportNetworkTotal(null);
    setReportNetworkError(null);
    setReportNetworkExpanded(false);
    setReportNetworkFetchedAt(null);
    setReportNetworkLoading(false);
    reportNetworkFetchedForRef.current = null;
  }, [canCaptureScreenshot]);

  const handleReportMessageChange = useCallback((event: ChangeEvent<HTMLTextAreaElement>) => {
    setReportMessage(event.target.value);
    if (reportError) {
      setReportError(null);
    }
  }, [reportError]);

  const handleReportIncludeScreenshotChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const nextValue = event.target.checked;
    setReportIncludeScreenshot(nextValue);
    if (nextValue) {
      setReportSelectionRect(null);
      setReportScreenshotClip(null);
      reportScreenshotContainerSizeRef.current = null;
      setReportScreenshotError(null);
      setReportScreenshotInfo(null);
      setReportScreenshotOriginalData(null);
      setReportScreenshotData(null);
      setReportScreenshotOriginalDimensions(null);
      if (canCaptureScreenshot) {
        void captureIframeScreenshot();
      }
    } else {
      setReportScreenshotOriginalData(null);
      setReportScreenshotData(null);
      setReportScreenshotError(null);
      setReportScreenshotInfo(null);
      setReportScreenshotLoading(false);
      setReportSelectionRect(null);
      setReportScreenshotOriginalDimensions(null);
      setReportScreenshotClip(null);
      reportScreenshotContainerSizeRef.current = null;
    }
  }, [canCaptureScreenshot, captureIframeScreenshot]);

  const handleRetryScreenshotCapture = useCallback(() => {
    if (!reportScreenshotLoading) {
      setReportScreenshotClip(null);
      setReportSelectionRect(null);
      reportScreenshotContainerSizeRef.current = null;
      setReportScreenshotOriginalData(null);
      setReportScreenshotData(null);
      setReportScreenshotOriginalDimensions(null);
      setReportScreenshotInfo(null);
      void captureIframeScreenshot();
    }
  }, [captureIframeScreenshot, reportScreenshotLoading]);

  const handleResetScreenshotSelection = useCallback(() => {
    if (reportScreenshotLoading || !reportScreenshotOriginalData) {
      return;
    }
    setReportScreenshotData(reportScreenshotOriginalData);
    setReportSelectionRect(null);
    setReportScreenshotClip(null);
    setReportScreenshotError(null);
    reportScreenshotContainerSizeRef.current = null;
  }, [reportScreenshotLoading, reportScreenshotOriginalData]);

  const clampValue = useCallback((value: number, min: number, max: number) => {
    if (Number.isNaN(value)) {
      return min;
    }
    return Math.max(min, Math.min(max, value));
  }, []);

  const cropScreenshot = useCallback(async (clip: { x: number; y: number; width: number; height: number }) => {
    if (!reportScreenshotOriginalData) {
      return null;
    }

    const image = new Image();
    image.src = `data:image/png;base64,${reportScreenshotOriginalData}`;

    await new Promise<void>((resolve, reject) => {
      image.onload = () => resolve();
      image.onerror = () => reject(new Error('Failed to load base screenshot.'));
    });

    const canvas = document.createElement('canvas');
    canvas.width = clip.width;
    canvas.height = clip.height;

    const context = canvas.getContext('2d');
    if (!context) {
      return null;
    }

    context.drawImage(
      image,
      clip.x,
      clip.y,
      clip.width,
      clip.height,
      0,
      0,
      clip.width,
      clip.height,
    );

    const croppedDataUrl = canvas.toDataURL('image/png');
    return croppedDataUrl.replace(/^data:image\/png;base64,/, '');
  }, [reportScreenshotOriginalData]);

  const handleScreenshotImageLoad = useCallback(() => {
    const container = reportScreenshotContainerRef.current;
    if (container) {
      const rect = container.getBoundingClientRect();
      if (rect.width > 0 && rect.height > 0) {
        reportScreenshotContainerSizeRef.current = { width: rect.width, height: rect.height };
      }
    }
  }, []);

  const handleScreenshotPointerDown = useCallback((event: PointerEvent<HTMLDivElement>) => {
    if (!reportScreenshotData || reportScreenshotLoading || !reportScreenshotOriginalDimensions) {
      return;
    }

    const container = reportScreenshotContainerRef.current;
    if (!container) {
      return;
    }

    const rect = container.getBoundingClientRect();
    if (rect.width <= 0 || rect.height <= 0) {
      return;
    }

    event.preventDefault();

    const x = clampValue(event.clientX - rect.left, 0, rect.width);
    const y = clampValue(event.clientY - rect.top, 0, rect.height);

    reportDragStateRef.current = {
      pointerId: event.pointerId,
      startX: x,
      startY: y,
      containerWidth: rect.width,
      containerHeight: rect.height,
    };
    reportScreenshotContainerSizeRef.current = { width: rect.width, height: rect.height };

    setReportSelectionRect({ x, y, width: 0, height: 0 });

    try {
      container.setPointerCapture(event.pointerId);
    } catch {
      // ignore pointer capture errors
    }
  }, [clampValue, reportScreenshotData, reportScreenshotOriginalDimensions, reportScreenshotLoading]);

  const handleScreenshotPointerMove = useCallback((event: PointerEvent<HTMLDivElement>) => {
    const dragState = reportDragStateRef.current;
    if (!dragState) {
      return;
    }

    const container = reportScreenshotContainerRef.current;
    if (!container) {
      return;
    }

    const rect = container.getBoundingClientRect();
    const currentX = clampValue(event.clientX - rect.left, 0, rect.width);
    const currentY = clampValue(event.clientY - rect.top, 0, rect.height);

    const x = Math.min(dragState.startX, currentX);
    const y = Math.min(dragState.startY, currentY);
    const width = Math.abs(currentX - dragState.startX);
    const height = Math.abs(currentY - dragState.startY);

    setReportSelectionRect({ x, y, width, height });

    reportDragStateRef.current = {
      ...dragState,
      containerWidth: rect.width,
      containerHeight: rect.height,
    };
    reportScreenshotContainerSizeRef.current = { width: rect.width, height: rect.height };
  }, [clampValue]);

  const finalizeScreenshotSelection = useCallback((event: PointerEvent<HTMLDivElement>, cancelledDrag = false) => {
    const dragState = reportDragStateRef.current;
    const selection = reportSelectionRect;
    reportDragStateRef.current = null;

    const container = reportScreenshotContainerRef.current;
    if (container) {
      try {
        container.releasePointerCapture(event.pointerId);
      } catch {
        // ignore
      }
    }

    if (cancelledDrag || !selection || !dragState || !reportScreenshotOriginalDimensions) {
      setReportSelectionRect(null);
      return;
    }

    const minDisplaySize = 12;
    if (selection.width < minDisplaySize || selection.height < minDisplaySize) {
      setReportSelectionRect(null);
      return;
    }

    const activeClip = reportScreenshotClip ?? {
      x: 0,
      y: 0,
      width: reportScreenshotOriginalDimensions.width,
      height: reportScreenshotOriginalDimensions.height,
    };

    const scaleX = activeClip.width / dragState.containerWidth;
    const scaleY = activeClip.height / dragState.containerHeight;

    const rawClip = {
      x: Math.max(0, Math.round(activeClip.x + (selection.x * scaleX))),
      y: Math.max(0, Math.round(activeClip.y + (selection.y * scaleY))),
      width: Math.max(1, Math.round(selection.width * scaleX)),
      height: Math.max(1, Math.round(selection.height * scaleY)),
    };

    rawClip.width = Math.min(rawClip.width, Math.max(1, reportScreenshotOriginalDimensions.width-rawClip.x));
    rawClip.height = Math.min(rawClip.height, Math.max(1, reportScreenshotOriginalDimensions.height-rawClip.y));

    const minClipSize = 24;
    if (rawClip.width < minClipSize || rawClip.height < minClipSize) {
      setReportSelectionRect(null);
      return;
    }

    setReportSelectionRect(null);
    setReportScreenshotLoading(true);
    setReportScreenshotError(null);

    void (async () => {
      try {
        const cropped = await cropScreenshot(rawClip);
        if (!cropped) {
          setReportScreenshotError('Failed to crop screenshot.');
          setReportScreenshotLoading(false);
          return;
        }

        setReportScreenshotData(cropped);
        setReportScreenshotClip(rawClip);
      } catch (error) {
        console.error('Failed to crop screenshot', error);
        setReportScreenshotError('Unable to crop the screenshot. Try again.');
      } finally {
        setReportScreenshotLoading(false);
      }
    })();
  }, [cropScreenshot, reportScreenshotClip, reportScreenshotOriginalDimensions, reportSelectionRect]);

  const handleScreenshotPointerUp = useCallback((event: PointerEvent<HTMLDivElement>) => {
    finalizeScreenshotSelection(event, false);
  }, [finalizeScreenshotSelection]);

  const handleScreenshotPointerLeave = useCallback((event: PointerEvent<HTMLDivElement>) => {
    if (!reportDragStateRef.current) {
      return;
    }
    finalizeScreenshotSelection(event, true);
  }, [finalizeScreenshotSelection]);

  const handleSubmitReport = useCallback(async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const trimmed = reportMessage.trim();
    if (!trimmed) {
      setReportError('Please add a short description of the issue.');
      return;
    }

    const targetAppId = currentApp?.id ?? appId ?? '';
    if (!targetAppId) {
      setReportError('Unable to determine which application to report.');
      return;
    }

    const includeScreenshot = reportIncludeScreenshot && canCaptureScreenshot;
    if (includeScreenshot && !reportScreenshotData) {
      setReportError('Capture a screenshot before sending the report.');
      return;
    }

    setReportSubmitting(true);
    setReportError(null);

    try {
      const previewContextUrl = activePreviewUrl || null;
      const payload: ReportIssuePayload = {
        message: trimmed,
        includeScreenshot,
        previewUrl: previewContextUrl,
        appName: currentApp?.name ?? null,
        scenarioName: currentApp?.scenario_name ?? null,
        source: 'app-monitor',
        screenshotData: includeScreenshot ? reportScreenshotData ?? null : null,
      };

      if (reportIncludeAppLogs) {
        if (selectedReportLogStreams.length > 0) {
          const combinedLogs: string[] = [];

          selectedReportLogStreams.forEach((stream) => {
            if (combinedLogs.length >= REPORT_APP_LOGS_MAX_LINES) {
              return;
            }

            const contextLabel = stream.type === 'background'
              ? `Background: ${stream.label}`
              : 'Lifecycle';
            combinedLogs.push(`--- ${contextLabel} ---`);

            if (stream.type === 'background' && stream.command && combinedLogs.length < REPORT_APP_LOGS_MAX_LINES) {
              combinedLogs.push(`# ${stream.command}`);
            }

            const remaining = REPORT_APP_LOGS_MAX_LINES - combinedLogs.length;
            if (remaining > 0) {
              const tail = stream.lines.slice(-remaining);
              combinedLogs.push(...tail);
            }
          });

          payload.logs = combinedLogs;
          payload.logsTotal = selectedReportLogStreams.reduce((total, stream) => total + stream.total, 0);
          if (reportAppLogsFetchedAt) {
            payload.logsCapturedAt = new Date(reportAppLogsFetchedAt).toISOString();
          }
        } else if (reportAppLogs.length > 0) {
          payload.logs = reportAppLogs;
          payload.logsTotal = typeof reportAppLogsTotal === 'number' ? reportAppLogsTotal : reportAppLogs.length;
          if (reportAppLogsFetchedAt) {
            payload.logsCapturedAt = new Date(reportAppLogsFetchedAt).toISOString();
          }
        }
      }

      if (reportIncludeConsoleLogs && reportConsoleLogs.length > 0) {
        payload.consoleLogs = reportConsoleLogs.map(entry => entry.payload);
        payload.consoleLogsTotal = typeof reportConsoleLogsTotal === 'number'
          ? reportConsoleLogsTotal
          : reportConsoleLogs.length;
        if (reportConsoleLogsFetchedAt) {
          payload.consoleLogsCapturedAt = new Date(reportConsoleLogsFetchedAt).toISOString();
        }
      }

      if (reportIncludeNetworkRequests && reportNetworkEvents.length > 0) {
        payload.networkRequests = reportNetworkEvents.map(entry => entry.payload);
        payload.networkRequestsTotal = typeof reportNetworkTotal === 'number'
          ? reportNetworkTotal
          : reportNetworkEvents.length;
        if (reportNetworkFetchedAt) {
          payload.networkCapturedAt = new Date(reportNetworkFetchedAt).toISOString();
        }
      }

      const response = await appService.reportAppIssue(targetAppId, payload);

      const issueId = response.data?.issue_id;
      const issueUrl = response.data?.issue_url;
      setReportResult({
        issueId,
        issueUrl,
        message: response.message ?? 'Issue report sent successfully.',
      });
      setReportMessage('');
    } catch (error: unknown) {
      const fallbackMessage = (error as { message?: string })?.message ?? 'Failed to send issue report.';
      setReportError(fallbackMessage);
    } finally {
      setReportSubmitting(false);
    }
  }, [
    activePreviewUrl,
    appId,
    canCaptureScreenshot,
    currentApp,
    reportIncludeScreenshot,
    reportAppLogs,
    reportAppLogsFetchedAt,
    reportAppLogsTotal,
    selectedReportLogStreams,
    reportConsoleLogs,
    reportConsoleLogsFetchedAt,
    reportConsoleLogsTotal,
    reportNetworkEvents,
    reportNetworkFetchedAt,
    reportNetworkTotal,
    reportIncludeAppLogs,
    reportIncludeConsoleLogs,
    reportIncludeNetworkRequests,
    reportMessage,
    reportScreenshotData,
  ]);

  useEffect(() => {
    if (!reportDialogOpen) {
      return;
    }

    const keyListener = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'Escape' && !reportSubmitting) {
        event.preventDefault();
        handleCloseReportDialog();
      }
    };

    window.addEventListener('keydown', keyListener);
    return () => {
      window.removeEventListener('keydown', keyListener);
    };
  }, [handleCloseReportDialog, reportDialogOpen, reportSubmitting]);

  useEffect(() => {
    if (!reportDialogOpen || !reportIncludeScreenshot) {
      return;
    }

    if (!canCaptureScreenshot) {
      setReportScreenshotError('Load the preview to capture a screenshot.');
      return;
    }

    if (reportScreenshotLoading || reportScreenshotOriginalData) {
      return;
    }

    void captureIframeScreenshot();
  }, [
    captureIframeScreenshot,
    canCaptureScreenshot,
    reportDialogOpen,
    reportIncludeScreenshot,
    reportScreenshotLoading,
    reportScreenshotOriginalData,
    activePreviewUrl,
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
  const scenarioDisplayName = useMemo(() => {
    if (!currentApp) {
      return 'application';
    }
    return currentApp.name || currentApp.scenario_name || currentApp.id || 'application';
  }, [currentApp]);

  const toggleActionLabel = isAppRunning ? 'Stop scenario' : 'Start scenario';
  const restartActionLabel = `Restart ${scenarioDisplayName}`;
  const toggleTooltip = `${toggleActionLabel}${currentApp ? ` (${currentApp.id})` : ''}`;
  const actionInProgress = pendingAction !== null;

  const bridgeIssueMessage = useMemo(() => {
    if (!bridgeState.isSupported || !bridgeCompliance || bridgeCompliance.ok) {
      return null;
    }
    const detail = bridgeCompliance.failures.join(', ');
    return `Preview bridge diagnostics failed (${detail}). History syncing may be unreliable.`;
  }, [bridgeCompliance, bridgeState.isSupported]);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }

    let animationFrame: number | null = null;

    const updatePreviewHeight = () => {
      const container = previewContainerRef.current;
      if (!container) {
        return;
      }

      const isCompactLayout = window.innerWidth <= 900;
      if (!isCompactLayout) {
        container.style.removeProperty('min-height');
        return;
      }

      const viewportHeight = window.visualViewport?.height ?? window.innerHeight;
      const rect = container.getBoundingClientRect();
      const clearance = 16; // breathing room below the iframe
      const available = Math.max(viewportHeight - rect.top - clearance, 320);
      container.style.minHeight = `${available}px`;
    };

    const scheduleUpdate = () => {
      if (animationFrame !== null) {
        cancelAnimationFrame(animationFrame);
      }
      animationFrame = window.requestAnimationFrame(updatePreviewHeight);
    };

    scheduleUpdate();

    const handleResize = () => {
      scheduleUpdate();
    };

    window.addEventListener('resize', handleResize);
    window.addEventListener('orientationchange', handleResize);
    window.visualViewport?.addEventListener('resize', handleResize);

    return () => {
      if (animationFrame !== null) {
        cancelAnimationFrame(animationFrame);
      }
      window.removeEventListener('resize', handleResize);
      window.removeEventListener('orientationchange', handleResize);
      window.visualViewport?.removeEventListener('resize', handleResize);
    };
  }, [previewUrl, loading, previewOverlay, reportDialogOpen, bridgeIssueMessage, statusMessage, currentApp]);

  const canGoBack = bridgeState.isSupported ? bridgeState.canGoBack : historyIndex > 0;
  const canGoForward = bridgeState.isSupported ? bridgeState.canGoForward : (historyIndex >= 0 && historyIndex < history.length - 1);
  const openPreviewTarget = bridgeState.isSupported && bridgeState.href ? bridgeState.href : previewUrl;

  const selectionDimensionLabel = useMemo(() => {
    if (!reportSelectionRect || !reportScreenshotOriginalDimensions) {
      return null;
    }

    const containerSize = reportScreenshotContainerSizeRef.current;
    if (!containerSize || containerSize.width === 0 || containerSize.height === 0) {
      return null;
    }

    const width = Math.round((reportScreenshotOriginalDimensions.width / containerSize.width) * reportSelectionRect.width);
    const height = Math.round((reportScreenshotOriginalDimensions.height / containerSize.height) * reportSelectionRect.height);

    if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
      return null;
    }

    return `${width} × ${height}px`;
  }, [reportSelectionRect, reportScreenshotOriginalDimensions]);

  return (
    <div className="app-preview-view">
      <div className="preview-toolbar">
        <div className="preview-toolbar__group preview-toolbar__group--left">
          <button
            type="button"
            className="preview-toolbar__icon-btn"
            onClick={handleGoBack}
            disabled={!canGoBack}
            aria-label={canGoBack ? 'Go back' : 'No previous page'}
            title={canGoBack ? 'Go back' : 'No previous page'}
          >
            <ArrowLeft aria-hidden size={18} />
          </button>
          <button
            type="button"
            className="preview-toolbar__icon-btn"
            onClick={handleGoForward}
            disabled={!canGoForward}
            aria-label={canGoForward ? 'Go forward' : 'No forward page'}
            title={canGoForward ? 'Go forward' : 'No forward page'}
          >
            <ArrowRight aria-hidden size={18} />
          </button>
          <button
            type="button"
            className="preview-toolbar__icon-btn"
            onClick={handleRefresh}
            disabled={loading}
            aria-label={loading ? 'Refreshing application status' : 'Refresh application'}
            title={loading ? 'Refreshing…' : 'Refresh'}
          >
            <RefreshCw aria-hidden size={18} className={clsx({ spinning: loading })} />
          </button>
          <div className="preview-toolbar__title">
            <div
              className={clsx('preview-toolbar__url-wrapper', urlStatusClass)}
              title={urlStatusTitle}
            >
              <button
                type="button"
                className="preview-toolbar__url-action-btn"
                onClick={() => setModalOpen(true)}
                disabled={!currentApp}
                aria-label="Application details"
                title="Application details"
              >
                <Info aria-hidden size={16} />
              </button>
              <input
                type="text"
                className="preview-toolbar__url-input"
                value={previewUrlInput}
                onChange={handleUrlInputChange}
                onBlur={handleUrlInputBlur}
                onKeyDown={handleUrlInputKeyDown}
                placeholder="Enter preview URL"
                aria-label="Preview URL"
                autoComplete="off"
                spellCheck={false}
                inputMode="url"
              />
              <button
                type="button"
                className="preview-toolbar__url-action-btn"
                onClick={handleOpenPreviewInNewTab}
                disabled={!openPreviewTarget}
                aria-label={openPreviewTarget ? 'Open preview in new tab' : 'Preview unavailable'}
                title={openPreviewTarget ? 'Open in new tab' : 'Preview unavailable'}
              >
                <ExternalLink aria-hidden size={16} />
              </button>
            </div>
          </div>
        </div>
        <div className="preview-toolbar__group preview-toolbar__group--right">
          <button
            type="button"
            className={clsx(
              'preview-toolbar__icon-btn',
              isAppRunning && 'preview-toolbar__icon-btn--danger',
              (pendingAction === 'start' || pendingAction === 'stop') && 'preview-toolbar__icon-btn--waiting',
            )}
            onClick={handleToggleApp}
            disabled={!currentApp || actionInProgress}
            aria-label={toggleTooltip}
            title={toggleTooltip}
          >
            {(pendingAction === 'start' || pendingAction === 'stop') ? (
              <Loader2 aria-hidden size={18} className="spinning" />
            ) : (
              <Power aria-hidden size={18} />
            )}
          </button>
          <button
            type="button"
            className={clsx(
              'preview-toolbar__icon-btn',
              'preview-toolbar__icon-btn--secondary',
              pendingAction === 'restart' && 'preview-toolbar__icon-btn--waiting',
            )}
            onClick={handleRestartApp}
            disabled={!currentApp || !isAppRunning || actionInProgress}
            aria-label={restartActionLabel}
            title={restartActionLabel}
          >
            {pendingAction === 'restart' ? (
              <Loader2 aria-hidden size={18} className="spinning" />
            ) : (
              <RotateCcw aria-hidden size={18} />
            )}
          </button>
          <button
            type="button"
            className="preview-toolbar__icon-btn"
            onClick={handleViewLogs}
            disabled={!currentApp}
            aria-label="View logs"
            title="View logs"
          >
            <ScrollText aria-hidden size={18} />
          </button>
          <button
            type="button"
            className={clsx('preview-toolbar__icon-btn', 'preview-toolbar__icon-btn--report')}
            onClick={handleOpenReportDialog}
            disabled={!currentApp}
            aria-label="Report an issue"
            title="Report an issue"
          >
            <Bug aria-hidden size={18} />
          </button>
        </div>
      </div>

      {bridgeIssueMessage && (
        <div className="preview-status">
          {bridgeIssueMessage}
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
            navigate(`/logs/${appIdentifier}`);
          }}
        />
      )}

      {reportDialogOpen && (
        <div
          className="report-dialog__overlay"
          role="presentation"
          onClick={() => {
            if (!reportSubmitting) {
              handleCloseReportDialog();
            }
          }}
        >
          <div
            className="report-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="app-report-dialog-title"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="report-dialog__header">
              <h2 id="app-report-dialog-title" className="report-dialog__title">
                <Bug aria-hidden size={20} />
                <span>Report an Issue</span>
              </h2>
              <button
                type="button"
                className="report-dialog__close"
                onClick={handleCloseReportDialog}
                disabled={reportSubmitting}
                aria-label="Close report dialog"
              >
                <X aria-hidden size={16} />
              </button>
            </div>

            {reportResult ? (
              <div className="report-dialog__state">
                <p className="report-dialog__success">
                  {reportResult.message ?? 'Issue report sent successfully.'}
                </p>
                {reportResult.issueId && (
                  <p className="report-dialog__success-id">
                    Tracking ID: <span>{reportResult.issueId}</span>
                  </p>
                )}
                {reportResult.issueUrl && (
                  <div className="report-dialog__success-link">
                    <button
                      type="button"
                      className="report-dialog__button"
                      onClick={() => window.open(reportResult.issueUrl, '_blank', 'noopener,noreferrer')}
                    >
                      Open in Issue Tracker
                    </button>
                  </div>
                )}
                <div className="report-dialog__actions">
                  <button
                    type="button"
                    className="report-dialog__button report-dialog__button--primary"
                    onClick={handleCloseReportDialog}
                  >
                    Close
                  </button>
                </div>
              </div>
            ) : (
              <form className="report-dialog__form" onSubmit={handleSubmitReport}>
                <div className="report-dialog__layout">
                  <div className="report-dialog__lane report-dialog__lane--primary">
                    <label htmlFor="app-report-message" className="report-dialog__label">
                      Describe the issue
                    </label>
                    <textarea
                      id="app-report-message"
                      className="report-dialog__textarea"
                      value={reportMessage}
                      onChange={handleReportMessageChange}
                      rows={6}
                      placeholder="What are you seeing? Include steps to reproduce if possible."
                      disabled={reportSubmitting}
                      required
                    />
                    <div className="report-dialog__logs">
                      <section className="report-dialog__logs-section">
                        <div className="report-dialog__logs-header">
                          <label
                            className={clsx(
                              'report-dialog__logs-include',
                              !reportIncludeAppLogs && 'report-dialog__logs-include--off',
                            )}
                          >
                            <input
                              type="checkbox"
                              checked={reportIncludeAppLogs}
                              onChange={(event) => setReportIncludeAppLogs(event.target.checked)}
                              aria-label="Include app logs in report"
                            />
                            <span className="report-dialog__logs-title">App logs</span>
                          </label>
                          <button
                            type="button"
                            className="report-dialog__logs-toggle"
                            onClick={handleToggleReportAppLogs}
                            aria-expanded={reportAppLogsExpanded}
                            aria-controls={reportAppLogsPanelId}
                          >
                            {reportAppLogsExpanded ? 'Hide' : 'Show'}
                          </button>
                        </div>
                        <p className="report-dialog__logs-note">
                          {reportAppLogsError
                            ? 'Logs failed to load. Expand to retry.'
                            : reportAppLogsLoading
                              ? 'Fetching recent logs…'
                              : 'Lifecycle and background logs from the running scenario.'}
                          {!reportIncludeAppLogs && (
                            <span className="report-dialog__logs-note-status">Excluded from report</span>
                          )}
                        </p>
                        {reportAppLogStreams.length > 0 && (
                          <div className="report-dialog__logs-streams">
                            <div className="report-dialog__logs-streams-list">
                              {reportAppLogStreams.map(stream => {
                                const checked = reportAppLogSelections[stream.key] !== false;
                                const labelText = stream.type === 'background'
                                  ? `Background · ${stream.label}`
                                  : 'Lifecycle';
                                const linesLabel = stream.total === 1 ? '1 line' : `${stream.total} lines`;
                                return (
                                  <label
                                    key={stream.key}
                                    className={clsx(
                                      'report-dialog__logs-stream',
                                      !reportIncludeAppLogs && 'report-dialog__logs-stream--disabled',
                                    )}
                                  >
                                    <input
                                      type="checkbox"
                                      checked={checked}
                                      onChange={(event) => handleReportLogStreamToggle(stream.key, event.target.checked)}
                                      disabled={!reportIncludeAppLogs}
                                      aria-label={`Include ${labelText.toLowerCase()} log stream`}
                                    />
                                    <span className="report-dialog__logs-stream-details">
                                      <span className="report-dialog__logs-stream-label">{labelText}</span>
                                      <span className="report-dialog__logs-stream-meta">{linesLabel}</span>
                                    </span>
                                  </label>
                                );
                              })}
                            </div>
                            <div className="report-dialog__logs-streams-meta">
                              {reportIncludeAppLogs ? (
                                <span>{`${selectedReportLogCount} of ${totalReportLogStreamCount} log streams selected`}</span>
                              ) : (
                                <span>Enable app logs to include these streams.</span>
                              )}
                              {reportIncludeAppLogs && selectedReportLogCount === 0 && (
                                <span className="report-dialog__logs-streams-warning">No log streams selected</span>
                              )}
                            </div>
                          </div>
                        )}
                        <div
                          id={reportAppLogsPanelId}
                          className="report-dialog__logs-panel"
                          style={reportAppLogsExpanded ? undefined : { display: 'none' }}
                          aria-hidden={!reportAppLogsExpanded}
                        >
                          <div className="report-dialog__logs-meta">
                            <span>
                              {reportAppLogsLoading
                                ? 'Loading logs…'
                                : reportAppLogs.length > 0
                                  ? `Showing last ${reportAppLogs.length}${logsTruncated ? ` of ${reportAppLogsTotal}` : ''} lines${formattedReportLogsTime ? ` (captured ${formattedReportLogsTime})` : ''}.`
                                  : reportAppLogsError
                                    ? 'Logs unavailable.'
                                    : 'No logs captured yet.'}
                            </span>
                            <button
                              type="button"
                              className="report-dialog__logs-refresh"
                              onClick={handleRefreshReportAppLogs}
                              disabled={reportAppLogsLoading}
                            >
                              {reportAppLogsLoading ? (
                                <Loader2 aria-hidden size={14} className="spinning" />
                              ) : (
                                <RefreshCw aria-hidden size={14} />
                              )}
                              <span>{reportAppLogsLoading ? 'Loading' : 'Refresh'}</span>
                            </button>
                          </div>
                          {reportAppLogsLoading ? (
                            <div className="report-dialog__logs-loading">
                              <Loader2 aria-hidden size={18} className="spinning" />
                              <span>Fetching logs…</span>
                            </div>
                          ) : reportAppLogsError ? (
                            <div className="report-dialog__logs-message">
                              <p>{reportAppLogsError}</p>
                              <button
                                type="button"
                                className="report-dialog__button report-dialog__button--ghost"
                                onClick={handleRefreshReportAppLogs}
                                disabled={reportAppLogsLoading}
                              >
                                Retry
                              </button>
                            </div>
                          ) : reportAppLogs.length === 0 ? (
                            <p className="report-dialog__logs-empty">No logs available for this scenario.</p>
                          ) : (
                            <pre className="report-dialog__logs-content">{reportAppLogs.join('\n')}</pre>
                          )}
                        </div>
                      </section>

                      <section className="report-dialog__logs-section">
                        <div className="report-dialog__logs-header">
                          <label
                            className={clsx(
                              'report-dialog__logs-include',
                              !reportIncludeConsoleLogs && 'report-dialog__logs-include--off',
                              !bridgeState.caps.includes('logs') && 'report-dialog__logs-include--disabled',
                            )}
                          >
                            <input
                              type="checkbox"
                              checked={reportIncludeConsoleLogs}
                              onChange={(event) => setReportIncludeConsoleLogs(event.target.checked)}
                              aria-label="Include console logs in report"
                              disabled={!bridgeState.caps.includes('logs')}
                            />
                            <span className="report-dialog__logs-title">Console logs</span>
                          </label>
                          <button
                            type="button"
                            className="report-dialog__logs-toggle"
                            onClick={handleToggleReportConsoleLogs}
                            aria-expanded={reportConsoleLogsExpanded}
                            aria-controls={reportConsoleLogsPanelId}
                          >
                            {reportConsoleLogsExpanded ? 'Hide' : 'Show'}
                          </button>
                        </div>
                        <p className="report-dialog__logs-note">
                          {bridgeState.caps.includes('logs')
                            ? reportConsoleLogsError
                              ? 'Console events failed to load. Expand to retry.'
                              : reportConsoleLogsLoading
                                ? 'Fetching console logs…'
                                : 'Captured directly from the iframe console (log, warn, error).'
                            : 'Console capture is unavailable for this preview.'}
                          {(!reportIncludeConsoleLogs || !bridgeState.caps.includes('logs')) && (
                            <span className="report-dialog__logs-note-status">
                              {bridgeState.caps.includes('logs') ? 'Excluded from report' : 'Unavailable'}
                            </span>
                          )}
                        </p>
                        <div
                          id={reportConsoleLogsPanelId}
                          className="report-dialog__logs-panel"
                          style={reportConsoleLogsExpanded ? undefined : { display: 'none' }}
                          aria-hidden={!reportConsoleLogsExpanded}
                        >
                          <div className="report-dialog__logs-meta">
                            <span>
                              {reportConsoleLogsLoading
                                ? 'Loading console logs…'
                                : reportConsoleLogs.length > 0
                                  ? `Showing last ${reportConsoleLogs.length}${consoleLogsTruncated ? ` of ${reportConsoleLogsTotal}` : ''} events${formattedConsoleLogsTime ? ` (captured ${formattedConsoleLogsTime})` : ''}.`
                                  : reportConsoleLogsError
                                    ? 'Console logs unavailable.'
                                    : 'No console events captured yet.'}
                            </span>
                            <button
                              type="button"
                              className="report-dialog__logs-refresh"
                              onClick={handleRefreshReportConsoleLogs}
                              disabled={reportConsoleLogsLoading}
                            >
                              {reportConsoleLogsLoading ? (
                                <Loader2 aria-hidden size={14} className="spinning" />
                              ) : (
                                <RefreshCw aria-hidden size={14} />
                              )}
                              <span>{reportConsoleLogsLoading ? 'Loading' : 'Refresh'}</span>
                            </button>
                          </div>
                          {reportConsoleLogsLoading ? (
                            <div className="report-dialog__logs-loading">
                              <Loader2 aria-hidden size={18} className="spinning" />
                              <span>Fetching console logs…</span>
                            </div>
                          ) : reportConsoleLogsError ? (
                            <div className="report-dialog__logs-message">
                              <p>{reportConsoleLogsError}</p>
                              <button
                                type="button"
                                className="report-dialog__button report-dialog__button--ghost"
                                onClick={handleRefreshReportConsoleLogs}
                                disabled={reportConsoleLogsLoading}
                              >
                                Retry
                              </button>
                            </div>
                          ) : reportConsoleLogs.length === 0 ? (
                            <p className="report-dialog__logs-empty">No console events captured.</p>
                          ) : (
                            <pre className={clsx('report-dialog__logs-content', 'report-dialog__logs-content--console')}>
                              {reportConsoleLogs.map((entry, index) => (
                                <span
                                  key={`${entry.payload.ts}-${index}`}
                                  className={clsx(
                                    'report-dialog__console-line',
                                    `report-dialog__console-line--${entry.severity}`,
                                  )}
                                >
                                  {entry.display}
                                </span>
                              ))}
                            </pre>
                          )}
                        </div>
                      </section>

                      <section className="report-dialog__logs-section">
                        <div className="report-dialog__logs-header">
                          <label
                            className={clsx(
                              'report-dialog__logs-include',
                              !reportIncludeNetworkRequests && 'report-dialog__logs-include--off',
                              !bridgeState.caps.includes('network') && 'report-dialog__logs-include--disabled',
                            )}
                          >
                            <input
                              type="checkbox"
                              checked={reportIncludeNetworkRequests}
                              onChange={(event) => setReportIncludeNetworkRequests(event.target.checked)}
                              aria-label="Include network requests in report"
                              disabled={!bridgeState.caps.includes('network')}
                            />
                            <span className="report-dialog__logs-title">Network requests</span>
                          </label>
                          <button
                            type="button"
                            className="report-dialog__logs-toggle"
                            onClick={handleToggleReportNetworkEvents}
                            aria-expanded={reportNetworkExpanded}
                            aria-controls={reportNetworkPanelId}
                          >
                            {reportNetworkExpanded ? 'Hide' : 'Show'}
                          </button>
                        </div>
                        <p className="report-dialog__logs-note">
                          {bridgeState.caps.includes('network')
                            ? reportNetworkError
                              ? 'Network capture failed. Expand to retry.'
                              : reportNetworkLoading
                                ? 'Fetching recent requests…'
                                : 'Recent fetch/xhr activity observed inside the iframe.'
                            : 'Network capture is unavailable for this preview.'}
                          {(!reportIncludeNetworkRequests || !bridgeState.caps.includes('network')) && (
                            <span className="report-dialog__logs-note-status">
                              {bridgeState.caps.includes('network') ? 'Excluded from report' : 'Unavailable'}
                            </span>
                          )}
                        </p>
                        <div
                          id={reportNetworkPanelId}
                          className="report-dialog__logs-panel"
                          style={reportNetworkExpanded ? undefined : { display: 'none' }}
                          aria-hidden={!reportNetworkExpanded}
                        >
                          <div className="report-dialog__logs-meta">
                            <span>
                              {reportNetworkLoading
                                ? 'Loading network requests…'
                                : reportNetworkEvents.length > 0
                                  ? `Showing last ${reportNetworkEvents.length}${networkEventsTruncated ? ` of ${reportNetworkTotal}` : ''} requests${formattedNetworkCapturedAt ? ` (captured ${formattedNetworkCapturedAt})` : ''}.`
                                  : reportNetworkError
                                    ? 'Network requests unavailable.'
                                    : 'No network activity captured.'}
                            </span>
                            <button
                              type="button"
                              className="report-dialog__logs-refresh"
                              onClick={handleRefreshReportNetworkEvents}
                              disabled={reportNetworkLoading}
                            >
                              {reportNetworkLoading ? (
                                <Loader2 aria-hidden size={14} className="spinning" />
                              ) : (
                                <RefreshCw aria-hidden size={14} />
                              )}
                              <span>{reportNetworkLoading ? 'Loading' : 'Refresh'}</span>
                            </button>
                          </div>
                          {reportNetworkLoading ? (
                            <div className="report-dialog__logs-loading">
                              <Loader2 aria-hidden size={18} className="spinning" />
                              <span>Fetching requests…</span>
                            </div>
                          ) : reportNetworkError ? (
                            <div className="report-dialog__logs-message">
                              <p>{reportNetworkError}</p>
                              <button
                                type="button"
                                className="report-dialog__button report-dialog__button--ghost"
                                onClick={handleRefreshReportNetworkEvents}
                                disabled={reportNetworkLoading}
                              >
                                Retry
                              </button>
                            </div>
                          ) : reportNetworkEvents.length === 0 ? (
                            <p className="report-dialog__logs-empty">No network requests captured.</p>
                          ) : (
                            <pre className="report-dialog__logs-content">
                              {reportNetworkEvents.map(entry => entry.display).join('\n')}
                            </pre>
                          )}
                        </div>
                      </section>
                    </div>
                  </div>
                  <div className="report-dialog__lane report-dialog__lane--secondary">
                    <label className="report-dialog__checkbox">
                      <input
                        type="checkbox"
                        checked={reportIncludeScreenshot && canCaptureScreenshot}
                        onChange={handleReportIncludeScreenshotChange}
                        disabled={!canCaptureScreenshot || reportSubmitting}
                      />
                      <span>Include screenshot of the current preview</span>
                    </label>
                    {!canCaptureScreenshot && (
                      <p className="report-dialog__hint">Load the preview to capture a screenshot.</p>
                    )}
                    {reportIncludeScreenshot && canCaptureScreenshot && (
                      <div className="report-dialog__preview" aria-live="polite">
                        {reportScreenshotLoading && (
                          <div className="report-dialog__preview-loading">
                            <Loader2 aria-hidden size={18} className="spinning" />
                            <span>Capturing screenshot…</span>
                          </div>
                        )}

                        {!reportScreenshotLoading && reportScreenshotError && (
                          <div className="report-dialog__preview-error">
                            <p>{reportScreenshotError}</p>
                            <button
                              type="button"
                              className="report-dialog__button report-dialog__button--ghost"
                              onClick={handleRetryScreenshotCapture}
                              disabled={reportScreenshotLoading}
                            >
                              Retry capture
                            </button>
                          </div>
                        )}

                        {!reportScreenshotLoading && !reportScreenshotError && reportScreenshotInfo && (
                          <p className="report-dialog__preview-info">{reportScreenshotInfo}</p>
                        )}

                        {!reportScreenshotLoading && !reportScreenshotError && reportScreenshotData && (
                          <>
                            <div
                              className={clsx(
                                'report-dialog__preview-image',
                                'report-dialog__preview-image--selectable',
                              )}
                              ref={reportScreenshotContainerRef}
                              onPointerDown={handleScreenshotPointerDown}
                              onPointerMove={handleScreenshotPointerMove}
                              onPointerUp={handleScreenshotPointerUp}
                              onPointerLeave={handleScreenshotPointerLeave}
                              onPointerCancel={handleScreenshotPointerLeave}
                            >
                              <img
                                src={`data:image/png;base64,${reportScreenshotData}`}
                                alt="Preview screenshot"
                                loading="lazy"
                                draggable={false}
                                onLoad={handleScreenshotImageLoad}
                              />
                              {reportSelectionRect && (
                                <div
                                  className="report-dialog__selection"
                                  style={{
                                    left: `${reportSelectionRect.x}px`,
                                    top: `${reportSelectionRect.y}px`,
                                    width: `${reportSelectionRect.width}px`,
                                    height: `${reportSelectionRect.height}px`,
                                  }}
                                >
                                  {selectionDimensionLabel && (
                                    <span className="report-dialog__selection-label">{selectionDimensionLabel}</span>
                                  )}
                                </div>
                              )}
                              {reportScreenshotClip && (
                                <div className="report-dialog__selection-indicator">
                                  Selection saved. Drag again to refine or reset for a clean slate.
                                </div>
                              )}
                            </div>

                            <div className="report-dialog__preview-actions">
                              <div className="report-dialog__preview-meta">
                                {reportScreenshotOriginalDimensions && (
                                  <span className="report-dialog__preview-meta-item">
                                    Base: {reportScreenshotOriginalDimensions.width} × {reportScreenshotOriginalDimensions.height}px
                                  </span>
                                )}
                                {reportScreenshotClip && (
                                  <span className="report-dialog__preview-meta-item">
                                    Crop: {reportScreenshotClip.width} × {reportScreenshotClip.height}px
                                  </span>
                                )}
                              </div>
                              <div className="report-dialog__preview-buttons">
                                <button
                                  type="button"
                                  className="report-dialog__button report-dialog__button--ghost"
                                  onClick={handleRetryScreenshotCapture}
                                  disabled={reportScreenshotLoading}
                                >
                                  Re-capture
                                </button>
                                <button
                                  type="button"
                                  className="report-dialog__button report-dialog__button--ghost"
                                  onClick={handleResetScreenshotSelection}
                                  disabled={reportScreenshotLoading || !reportScreenshotClip}
                                >
                                  Reset area
                                </button>
                              </div>
                              <p className="report-dialog__preview-hint">
                                {reportScreenshotClip === null
                                  ? 'Drag on the screenshot to focus on the area that needs attention.'
                                  : 'Crop saved. Drag again to fine-tune or reset to capture a different view.'}
                              </p>
                            </div>
                          </>
                        )}

                        {!reportScreenshotLoading && !reportScreenshotError && !reportScreenshotData && (
                          <p className="report-dialog__preview-hint">Ready to capture the current preview.</p>
                        )}
                      </div>
                    )}
                  </div>
                </div>

                {reportError && (
                  <p className="report-dialog__error" role="alert">
                    {reportError}
                  </p>
                )}

                <div className="report-dialog__actions">
                  <button
                    type="button"
                    className="report-dialog__button"
                    onClick={handleCloseReportDialog}
                    disabled={reportSubmitting}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="report-dialog__button report-dialog__button--primary"
                    disabled={
                      reportSubmitting || (
                        reportIncludeScreenshot && canCaptureScreenshot && (reportScreenshotLoading || !reportScreenshotData)
                      )
                    }
                  >
                    {reportSubmitting ? (
                      <>
                        <Loader2 aria-hidden size={16} className="spinning" />
                        Sending…
                      </>
                    ) : (
                      'Send Report'
                    )}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

export default AppPreviewView;
