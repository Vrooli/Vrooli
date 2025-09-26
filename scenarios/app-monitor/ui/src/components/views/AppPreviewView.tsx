import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ChangeEvent, FormEvent, KeyboardEvent, MouseEvent, PointerEvent } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import clsx from 'clsx';
import { ArrowLeft, ArrowRight, Bug, ExternalLink, Info, Loader2, Power, RefreshCw, RotateCcw, ScrollText, X } from 'lucide-react';
import { appService } from '@/services/api';

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

interface AppPreviewViewProps {
  apps: App[];
  setApps: React.Dispatch<React.SetStateAction<App[]>>;
}

const AppPreviewView = ({ apps, setApps }: AppPreviewViewProps) => {
  const navigate = useNavigate();
  const { appId } = useParams<{ appId: string }>();
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
  const [reportResult, setReportResult] = useState<{ issueId?: string; message?: string } | null>(null);
  const [reportScreenshotData, setReportScreenshotData] = useState<string | null>(null);
  const [reportScreenshotOriginalData, setReportScreenshotOriginalData] = useState<string | null>(null);
  const [reportScreenshotLoading, setReportScreenshotLoading] = useState(false);
  const [reportScreenshotError, setReportScreenshotError] = useState<string | null>(null);
  const [reportScreenshotOriginalDimensions, setReportScreenshotOriginalDimensions] = useState<{ width: number; height: number } | null>(null);
  const [reportSelectionRect, setReportSelectionRect] = useState<{ x: number; y: number; width: number; height: number } | null>(null);
  const [reportScreenshotClip, setReportScreenshotClip] = useState<{ x: number; y: number; width: number; height: number } | null>(null);
  const [reportScreenshotInfo, setReportScreenshotInfo] = useState<string | null>(null);
  const iframeRef = useRef<HTMLIFrameElement | null>(null);
  const previewContainerRef = useRef<HTMLDivElement | null>(null);
  const [bridgeCompliance, setBridgeCompliance] = useState<BridgeComplianceResult | null>(null);
  const complianceRunRef = useRef(false);
  const initialPreviewUrlRef = useRef<string | null>(null);
  const reportScreenshotContainerRef = useRef<HTMLDivElement | null>(null);
  const reportScreenshotContainerSizeRef = useRef<{ width: number; height: number } | null>(null);
  const reportDragStateRef = useRef<{ pointerId: number; startX: number; startY: number; containerWidth: number; containerHeight: number } | null>(null);

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
  } = useIframeBridge({
    iframeRef,
    previewUrl,
    onLocation: handleBridgeLocation,
  });

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
      navigate('/apps', { replace: true });
    }
  }, [appId, navigate]);

  useEffect(() => {
    setFetchAttempted(false);
  }, [appId]);

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
          setCurrentApp(fetched);
          setApps(prev => {
            const index = prev.findIndex(app => app.id === fetched.id);
            if (index === -1) {
              return [...prev, fetched];
            }

            const updated = [...prev];
            updated[index] = fetched;
            return updated;
          });
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
  }, [appId, apps, fetchAttempted, setApps]);

  useEffect(() => {
    if (!currentApp) {
      if (!hasCustomPreviewUrl) {
        setPreviewUrl(null);
        setPreviewUrlInput('');
        setHistory([]);
        setHistoryIndex(-1);
        initialPreviewUrlRef.current = null;
      }
      return;
    }

    if (currentApp.is_partial) {
      setStatusMessage('Loading application details...');
      setLoading(true);
      if (!hasCustomPreviewUrl) {
        setPreviewUrl(null);
        setPreviewUrlInput('');
        setHistory([]);
        setHistoryIndex(-1);
        initialPreviewUrlRef.current = null;
      }
      return;
    }

    setLoading(false);

    if (!isRunningStatus(currentApp.status) || isStoppedStatus(currentApp.status)) {
      if (!hasCustomPreviewUrl) {
        setPreviewUrl(null);
        setPreviewUrlInput('');
        setHistory([]);
        setHistoryIndex(-1);
        initialPreviewUrlRef.current = null;
      }
      setStatusMessage('Application is not running. Start it from the Applications view to access the UI preview.');
      return;
    }

    const url = buildPreviewUrl(currentApp);
    if (!url) {
      if (!hasCustomPreviewUrl) {
        setPreviewUrl(null);
        setPreviewUrlInput('');
        setHistory([]);
        setHistoryIndex(-1);
        initialPreviewUrlRef.current = null;
      }
      setStatusMessage('This application does not expose a UI endpoint to preview.');
      return;
    }

    setStatusMessage(null);
    if (!hasCustomPreviewUrl) {
      initialPreviewUrlRef.current = url;
      setPreviewUrl(url);
      setPreviewUrlInput(url);
      setHistory([url]);
      setHistoryIndex(0);
    } else if (previewUrl === null) {
      initialPreviewUrlRef.current = url;
      setPreviewUrl(url);
    }
  }, [currentApp, hasCustomPreviewUrl, previewUrl]);

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

  const handleAppAction = useCallback(async (appToControl: string, action: 'start' | 'stop' | 'restart') => {
    setPendingAction(action);
    const actionInProgressMessage = action === 'stop'
      ? 'Stopping application...'
      : action === 'start'
        ? 'Starting application...'
        : 'Restarting application...';
    setStatusMessage(actionInProgressMessage);

    try {
      const success = await appService.controlApp(appToControl, action);
      if (success) {
        setApps(prev => prev.map(app => {
          if (app.id === appToControl) {
            return {
              ...app,
              status: action === 'stop' ? 'stopped' : 'running',
              updated_at: new Date().toISOString(),
            };
          }
          return app;
        }));
        setCurrentApp(prev => {
          if (prev && prev.id === appToControl) {
            return {
              ...prev,
              status: action === 'stop' ? 'stopped' : 'running',
              updated_at: new Date().toISOString(),
            };
          }
          return prev;
        });
        setStatusMessage(action === 'stop'
          ? 'Application stopped. Start it again to relaunch the UI preview.'
          : action === 'start'
            ? 'Application started. Preview will refresh automatically.'
            : 'Application restarted. Preview will refresh automatically.');
      } else {
        setStatusMessage(`Unable to ${action} the application. Check logs for details.`);
      }
    } catch (error) {
      logger.error(`Failed to ${action} app ${appToControl}`, error);
      setStatusMessage(`Unable to ${action} the application. Check logs for details.`);
    } finally {
      setPendingAction(null);
    }
  }, [setApps]);

  const handleToggleApp = useCallback(() => {
    if (!currentApp || pendingAction) {
      return;
    }

    const action: 'start' | 'stop' = isRunningStatus(currentApp.status) ? 'stop' : 'start';
    handleAppAction(currentApp.id, action);
  }, [currentApp, handleAppAction, pendingAction]);

  const handleRestartApp = useCallback(() => {
    if (!currentApp || pendingAction || !isRunningStatus(currentApp.status)) {
      return;
    }

    handleAppAction(currentApp.id, 'restart');
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

    setLoading(true);
    setStatusMessage('Refreshing application status...');

    appService.getApp(appId)
      .then(fetched => {
        if (fetched) {
          setCurrentApp(fetched);
          setApps(prev => {
            const index = prev.findIndex(app => app.id === fetched.id);
            if (index === -1) {
              return [...prev, fetched];
            }

            const updated = [...prev];
            updated[index] = fetched;
            return updated;
          });
          if (!hasCustomPreviewUrl) {
            setStatusMessage(null);
          }
        } else {
          setStatusMessage('Application not found.');
        }
      })
      .catch(error => {
        logger.error('Failed to refresh application preview', error);
        setStatusMessage('Failed to refresh application preview.');
      })
      .finally(() => {
        setLoading(false);
      });
  }, [appId, hasCustomPreviewUrl, setApps]);

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
    setReportIncludeScreenshot(canCaptureScreenshot);
    if (canCaptureScreenshot) {
      void captureIframeScreenshot();
    }
  }, [canCaptureScreenshot, captureIframeScreenshot]);

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
    if (!reportScreenshotData || reportScreenshotLoading || reportScreenshotClip || !reportScreenshotOriginalDimensions) {
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
  }, [clampValue, reportScreenshotClip, reportScreenshotData, reportScreenshotOriginalDimensions, reportScreenshotLoading]);

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

    const scaleX = reportScreenshotOriginalDimensions.width / dragState.containerWidth;
    const scaleY = reportScreenshotOriginalDimensions.height / dragState.containerHeight;

    const rawClip = {
      x: Math.max(0, Math.round(selection.x * scaleX)),
      y: Math.max(0, Math.round(selection.y * scaleY)),
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
  }, [cropScreenshot, reportScreenshotOriginalDimensions, reportSelectionRect]);

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

      const response = await appService.reportAppIssue(targetAppId, {
        message: trimmed,
        includeScreenshot,
        previewUrl: previewContextUrl,
        appName: currentApp?.name ?? null,
        scenarioName: currentApp?.scenario_name ?? null,
        source: 'app-monitor',
        screenshotData: includeScreenshot ? reportScreenshotData ?? null : null,
      });

      const issueId = response.data?.issue_id;
      setReportResult({
        issueId,
        message: response.message ?? 'Issue report sent successfully.',
      });
      setReportMessage('');
    } catch (error: unknown) {
      const fallbackMessage = (error as { message?: string })?.message ?? 'Failed to send issue report.';
      setReportError(fallbackMessage);
    } finally {
      setReportSubmitting(false);
    }
  }, [activePreviewUrl, appId, canCaptureScreenshot, currentApp, reportIncludeScreenshot, reportMessage, reportScreenshotData]);

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
            src={previewUrl}
            title={`${currentApp?.name ?? 'Application'} preview`}
            className="preview-iframe"
            loading="lazy"
            ref={iframeRef}
          />
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
              <h2 id="app-report-dialog-title">Report an Issue</h2>
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
                            reportScreenshotClip === null && 'report-dialog__preview-image--selectable',
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
                              Selection locked. Reset to choose another area.
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
                              : 'Crop saved. Reset the area if you need a different view.'}
                          </p>
                        </div>
                      </>
                    )}

                    {!reportScreenshotLoading && !reportScreenshotError && !reportScreenshotData && (
                      <p className="report-dialog__preview-hint">Ready to capture the current preview.</p>
                    )}
                  </div>
                )}

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
