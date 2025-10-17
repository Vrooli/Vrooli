import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { MutableRefObject } from 'react';
import type {
  ChangeEvent,
  PointerEvent as ReactPointerEvent,
  RefObject,
  SyntheticEvent,
} from 'react';
import type { BridgeScreenshotMode, BridgeScreenshotOptions } from '@vrooli/iframe-bridge';
import type Logger from '@/services/logger';

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

interface ScreenshotHookOptions {
  reportDialogOpen: boolean;
  canCaptureScreenshot: boolean;
  activePreviewUrl: string;
  previewContainerRef: RefObject<HTMLDivElement | null>;
  iframeRef: RefObject<HTMLIFrameElement | null>;
  isPreviewSameOrigin: boolean;
  bridgeSupportsScreenshot: boolean;
  requestScreenshot: (options?: BridgeScreenshotOptions) => Promise<{
    data: string;
    width: number;
    height: number;
    note?: string | null;
    mode?: BridgeScreenshotMode;
    clip?: { x: number; y: number; width: number; height: number };
  }>;
  logger: Logger;
}

interface ScreenshotHookResult {
  reportIncludeScreenshot: boolean;
  reportScreenshotData: string | null;
  reportScreenshotLoading: boolean;
  reportScreenshotError: string | null;
  reportScreenshotOriginalDimensions: { width: number; height: number } | null;
  reportSelectionRect: { x: number; y: number; width: number; height: number } | null;
  reportScreenshotClip: { x: number; y: number; width: number; height: number } | null;
  reportScreenshotInfo: string | null;
  reportScreenshotCountdown: number | null;
  reportScreenshotContainerRef: MutableRefObject<HTMLDivElement | null>;
  reportScreenshotContainerHandlers: {
    onPointerDown: (event: ReactPointerEvent<HTMLDivElement>) => void;
    onPointerMove: (event: ReactPointerEvent<HTMLDivElement>) => void;
    onPointerUp: (event: ReactPointerEvent<HTMLDivElement>) => void;
    onPointerLeave: (event: ReactPointerEvent<HTMLDivElement>) => void;
    onPointerCancel: (event: ReactPointerEvent<HTMLDivElement>) => void;
  };
  selectionDimensionLabel: string | null;
  handleReportIncludeScreenshotChange: (event: ChangeEvent<HTMLInputElement>) => void;
  handleRetryScreenshotCapture: () => void;
  handleResetScreenshotSelection: () => void;
  handleDelayedScreenshotCapture: () => void;
  handleScreenshotImageLoad: (event: SyntheticEvent<HTMLImageElement>) => void;
  prepareForDialogOpen: (canCapture: boolean) => void;
  cleanupAfterDialogClose: (canCapture: boolean) => void;
}

export function useReportScreenshot({
  reportDialogOpen,
  canCaptureScreenshot,
  activePreviewUrl,
  previewContainerRef,
  iframeRef,
  isPreviewSameOrigin,
  bridgeSupportsScreenshot,
  requestScreenshot,
  logger,
}: ScreenshotHookOptions): ScreenshotHookResult {
  const [reportIncludeScreenshot, setReportIncludeScreenshot] = useState(false);
  const [reportScreenshotData, setReportScreenshotData] = useState<string | null>(null);
  const [reportScreenshotOriginalData, setReportScreenshotOriginalData] = useState<string | null>(null);
  const [reportScreenshotLoading, setReportScreenshotLoading] = useState(false);
  const [reportScreenshotError, setReportScreenshotError] = useState<string | null>(null);
  const [reportScreenshotOriginalDimensions, setReportScreenshotOriginalDimensions] = useState<{ width: number; height: number } | null>(null);
  const [reportSelectionRect, setReportSelectionRect] = useState<{ x: number; y: number; width: number; height: number } | null>(null);
  const [reportScreenshotClip, setReportScreenshotClip] = useState<{ x: number; y: number; width: number; height: number } | null>(null);
  const [reportScreenshotInfo, setReportScreenshotInfo] = useState<string | null>(null);
  const [reportScreenshotCountdown, setReportScreenshotCountdown] = useState<number | null>(null);

  const reportScreenshotContainerRef = useRef<HTMLDivElement | null>(null);
  const reportScreenshotContainerSizeRef = useRef<{ width: number; height: number } | null>(null);
  const reportDragStateRef = useRef<{ pointerId: number; startX: number; startY: number; containerWidth: number; containerHeight: number } | null>(null);
  const reportScreenshotCountdownTimerRef = useRef<number | null>(null);

  const cancelScreenshotCountdown = useCallback(() => {
    if (reportScreenshotCountdownTimerRef.current !== null) {
      window.clearTimeout(reportScreenshotCountdownTimerRef.current);
      reportScreenshotCountdownTimerRef.current = null;
    }
    setReportScreenshotCountdown(null);
  }, []);

  const resetScreenshotState = useCallback(() => {
    setReportScreenshotData(null);
    setReportScreenshotOriginalData(null);
    setReportScreenshotError(null);
    setReportScreenshotInfo(null);
    setReportScreenshotOriginalDimensions(null);
    setReportSelectionRect(null);
    setReportScreenshotClip(null);
    reportScreenshotContainerSizeRef.current = null;
    reportDragStateRef.current = null;
  }, []);

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

    return canvas.toDataURL('image/png').replace(/^data:image\/png;base64,/, '');
  }, [reportScreenshotOriginalData]);

  const captureIframeScreenshot = useCallback(async () => {
    if (!reportDialogOpen || !reportIncludeScreenshot || !canCaptureScreenshot) {
      return;
    }

    cancelScreenshotCountdown();
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
        logger.error('html2canvas fallback capture failed', error);
        const placeholder = createPlaceholderCanvas(container.clientWidth, container.clientHeight, reason === 'cross-origin' ? 'cross-origin' : 'fallback');
        if (placeholder) {
          produceFromCanvas(placeholder, 'Attached a placeholder when the live capture was unavailable.');
          return true;
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
        logger.warn('Restricted from accessing iframe document (likely cross-origin).', error);
        return 'cross-origin';
      }

      if (!iframeDocument) {
        try {
          iframeDocument = iframe.contentWindow ? iframe.contentWindow.document : null;
        } catch (error) {
          logger.warn('Restricted from accessing iframe window document (cross-origin).', error);
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
        logger.error('Failed to capture iframe screenshot', error);
        return 'failed';
      }
    };

    let handled = false;

    if (bridgeSupportsScreenshot) {
      try {
        const screenshotOptions: BridgeScreenshotOptions = {
          scale: window.devicePixelRatio || 1,
          mode: 'viewport',
        };
        const result = await requestScreenshot(screenshotOptions);
        setReportScreenshotOriginalData(result.data);
        setReportScreenshotData(result.data);
        setReportScreenshotOriginalDimensions({ width: result.width, height: result.height });
        setReportScreenshotInfo(result.note ?? (result.mode === 'viewport' ? 'Captured the currently visible viewport.' : null));
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
    cancelScreenshotCountdown,
    iframeRef,
    isPreviewSameOrigin,
    logger,
    previewContainerRef,
    reportDialogOpen,
    reportIncludeScreenshot,
    requestScreenshot,
  ]);

  useEffect(() => {
    if (reportScreenshotCountdown === null) {
      if (reportScreenshotCountdownTimerRef.current !== null) {
        window.clearTimeout(reportScreenshotCountdownTimerRef.current);
        reportScreenshotCountdownTimerRef.current = null;
      }
      return;
    }

    if (reportScreenshotCountdown === 0) {
      reportScreenshotCountdownTimerRef.current = null;
      setReportScreenshotCountdown(null);
      void captureIframeScreenshot();
      return;
    }

    const timer = window.setTimeout(() => {
      setReportScreenshotCountdown(prev => (prev === null ? null : Math.max(0, prev - 1)));
    }, 1000);

    reportScreenshotCountdownTimerRef.current = timer;
    return () => {
      window.clearTimeout(timer);
      if (reportScreenshotCountdownTimerRef.current === timer) {
        reportScreenshotCountdownTimerRef.current = null;
      }
    };
  }, [captureIframeScreenshot, reportScreenshotCountdown]);

  useEffect(() => () => {
    cancelScreenshotCountdown();
  }, [cancelScreenshotCountdown]);

  const handleScreenshotPointerDown = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    if (!reportScreenshotData || reportScreenshotLoading || !reportScreenshotOriginalDimensions) {
      return;
    }

    const container = reportScreenshotContainerRef.current;
    if (!container) {
      return;
    }

    const rect = container.getBoundingClientRect();
    container.setPointerCapture(event.pointerId);
    reportScreenshotContainerSizeRef.current = { width: rect.width, height: rect.height };
    const startX = clampValue(event.clientX - rect.left, 0, rect.width);
    const startY = clampValue(event.clientY - rect.top, 0, rect.height);

    reportDragStateRef.current = {
      pointerId: event.pointerId,
      startX,
      startY,
      containerWidth: rect.width,
      containerHeight: rect.height,
    };
    setReportSelectionRect({ x: startX, y: startY, width: 0, height: 0 });
    setReportScreenshotError(null);
    setReportScreenshotInfo(null);
  }, [clampValue, reportScreenshotData, reportScreenshotLoading, reportScreenshotOriginalDimensions]);

  const handleScreenshotPointerMove = useCallback((event: ReactPointerEvent<HTMLDivElement>) => {
    const dragState = reportDragStateRef.current;
    if (!dragState || !reportScreenshotData) {
      return;
    }

    const container = reportScreenshotContainerRef.current;
    if (!container) {
      return;
    }

    const rect = container.getBoundingClientRect();
    const currentX = clampValue(event.clientX - rect.left, 0, rect.width);
    const currentY = clampValue(event.clientY - rect.top, 0, rect.height);

    const left = Math.min(dragState.startX, currentX);
    const top = Math.min(dragState.startY, currentY);
    const width = Math.abs(currentX - dragState.startX);
    const height = Math.abs(currentY - dragState.startY);

    setReportSelectionRect({ x: left, y: top, width, height });
  }, [clampValue, reportScreenshotData]);

  const finalizeScreenshotSelection = useCallback(async (cancelledDrag: boolean) => {
    const dragState = reportDragStateRef.current;
    const selection = reportSelectionRect;
    if (cancelledDrag || !selection || !dragState || !reportScreenshotOriginalDimensions) {
      setReportSelectionRect(null);
      reportDragStateRef.current = null;
      return;
    }

    const containerSize = reportScreenshotContainerSizeRef.current;
    if (!containerSize || containerSize.width <= 0 || containerSize.height <= 0) {
      setReportScreenshotError('Preview container is not available for capture.');
      reportDragStateRef.current = null;
      return;
    }

    try {
      const baseClip = reportScreenshotClip ?? {
        x: 0,
        y: 0,
        width: reportScreenshotOriginalDimensions.width,
        height: reportScreenshotOriginalDimensions.height,
      };

      const widthRatio = baseClip.width / containerSize.width;
      const heightRatio = baseClip.height / containerSize.height;

      const rawClip = {
        x: Math.max(0, Math.round(baseClip.x + selection.x * widthRatio)),
        y: Math.max(0, Math.round(baseClip.y + selection.y * heightRatio)),
        width: Math.max(1, Math.round(selection.width * widthRatio)),
        height: Math.max(1, Math.round(selection.height * heightRatio)),
      };

      rawClip.width = Math.min(rawClip.width, Math.max(1, reportScreenshotOriginalDimensions.width - rawClip.x));
      rawClip.height = Math.min(rawClip.height, Math.max(1, reportScreenshotOriginalDimensions.height - rawClip.y));

      const cropped = await cropScreenshot(rawClip);
      if (!cropped) {
        setReportScreenshotError('Unable to crop the screenshot. Try again.');
        return;
      }

      setReportScreenshotData(cropped);
      setReportScreenshotClip(rawClip);
      setReportSelectionRect(null);
      setReportScreenshotError(null);
      reportScreenshotContainerSizeRef.current = null;
    } catch (error) {
      logger.error('Failed to crop screenshot', error);
      setReportScreenshotError('Failed to crop screenshot.');
    } finally {
      reportDragStateRef.current = null;
    }
  }, [cropScreenshot, logger, reportScreenshotClip, reportScreenshotOriginalDimensions, reportSelectionRect]);

  const handleScreenshotPointerUp = useCallback(() => {
    if (!reportDragStateRef.current) {
      return;
    }

    const container = reportScreenshotContainerRef.current;
    if (container && container.hasPointerCapture(reportDragStateRef.current.pointerId)) {
      container.releasePointerCapture(reportDragStateRef.current.pointerId);
    }

    void finalizeScreenshotSelection(false);
  }, [finalizeScreenshotSelection]);

  const handleScreenshotPointerLeave = useCallback(() => {
    if (!reportDragStateRef.current) {
      return;
    }

    const container = reportScreenshotContainerRef.current;
    if (container && container.hasPointerCapture(reportDragStateRef.current.pointerId)) {
      container.releasePointerCapture(reportDragStateRef.current.pointerId);
    }

    void finalizeScreenshotSelection(true);
  }, [finalizeScreenshotSelection]);

  const reportScreenshotContainerHandlers = useMemo(() => ({
    onPointerDown: handleScreenshotPointerDown,
    onPointerMove: handleScreenshotPointerMove,
    onPointerUp: handleScreenshotPointerUp,
    onPointerLeave: handleScreenshotPointerLeave,
    onPointerCancel: handleScreenshotPointerLeave,
  }), [
    handleScreenshotPointerDown,
    handleScreenshotPointerLeave,
    handleScreenshotPointerMove,
    handleScreenshotPointerUp,
  ]);

  const handleScreenshotImageLoad = useCallback((event: SyntheticEvent<HTMLImageElement>) => {
    const image = event.currentTarget;
    reportScreenshotContainerSizeRef.current = {
      width: image.clientWidth,
      height: image.clientHeight,
    };
  }, []);

  const handleReportIncludeScreenshotChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const nextValue = event.target.checked;
    cancelScreenshotCountdown();
    setReportIncludeScreenshot(nextValue);
    if (nextValue) {
      resetScreenshotState();
      if (!canCaptureScreenshot) {
        setReportScreenshotError('Load the preview to capture a screenshot.');
      }
    } else {
      resetScreenshotState();
      setReportScreenshotLoading(false);
    }
  }, [canCaptureScreenshot, cancelScreenshotCountdown, resetScreenshotState]);

  const handleRetryScreenshotCapture = useCallback(() => {
    if (!reportScreenshotLoading) {
      cancelScreenshotCountdown();
      resetScreenshotState();
      void captureIframeScreenshot();
    }
  }, [cancelScreenshotCountdown, captureIframeScreenshot, reportScreenshotLoading, resetScreenshotState]);

  const handleResetScreenshotSelection = useCallback(() => {
    if (
      reportScreenshotLoading
      || !reportScreenshotOriginalData
      || !reportScreenshotOriginalDimensions
    ) {
      return;
    }
    setReportScreenshotData(reportScreenshotOriginalData);
    setReportSelectionRect(null);
    setReportScreenshotClip(null);
    setReportScreenshotError(null);
    setReportScreenshotInfo(null);
    reportScreenshotContainerSizeRef.current = null;
  }, [reportScreenshotLoading, reportScreenshotOriginalData, reportScreenshotOriginalDimensions]);

  const handleDelayedScreenshotCapture = useCallback(() => {
    if (!reportIncludeScreenshot || !canCaptureScreenshot || reportScreenshotLoading) {
      return;
    }

    if (reportScreenshotCountdown !== null) {
      cancelScreenshotCountdown();
      return;
    }

    resetScreenshotState();
    setReportScreenshotCountdown(5);
  }, [
    cancelScreenshotCountdown,
    canCaptureScreenshot,
    reportIncludeScreenshot,
    reportScreenshotCountdown,
    reportScreenshotLoading,
    resetScreenshotState,
  ]);

  const selectionDimensionLabel = useMemo(() => {
    if (!reportSelectionRect || !reportScreenshotOriginalDimensions) {
      return null;
    }
    const containerSize = reportScreenshotContainerSizeRef.current;
    if (!containerSize || containerSize.width <= 0 || containerSize.height <= 0) {
      return null;
    }

    const baseClip = reportScreenshotClip ?? {
      x: 0,
      y: 0,
      width: reportScreenshotOriginalDimensions.width,
      height: reportScreenshotOriginalDimensions.height,
    };

    const widthRatio = baseClip.width / containerSize.width;
    const heightRatio = baseClip.height / containerSize.height;

    const width = Math.round(reportSelectionRect.width * widthRatio);
    const height = Math.round(reportSelectionRect.height * heightRatio);
    if (width <= 0 || height <= 0) {
      return null;
    }

    return `${width} × ${height}px`;
  }, [reportSelectionRect, reportScreenshotClip, reportScreenshotOriginalDimensions]);

  const prepareForDialogOpen = useCallback((canCapture: boolean) => {
    cancelScreenshotCountdown();
    setReportIncludeScreenshot(canCapture);
    resetScreenshotState();
    setReportScreenshotLoading(false);
    if (!canCapture) {
      setReportScreenshotError('Load the preview to capture a screenshot.');
    }
  }, [cancelScreenshotCountdown, resetScreenshotState]);

  const cleanupAfterDialogClose = useCallback((canCapture: boolean) => {
    cancelScreenshotCountdown();
    setReportIncludeScreenshot(canCapture);
    resetScreenshotState();
    setReportScreenshotLoading(false);
  }, [cancelScreenshotCountdown, resetScreenshotState]);

  useEffect(() => {
    if (!reportDialogOpen || !reportIncludeScreenshot) {
      return;
    }

    if (!canCaptureScreenshot) {
      setReportScreenshotError('Load the preview to capture a screenshot.');
      return;
    }

    if (reportScreenshotLoading || reportScreenshotData) {
      return;
    }

    void captureIframeScreenshot();
  }, [
    canCaptureScreenshot,
    captureIframeScreenshot,
    reportDialogOpen,
    reportIncludeScreenshot,
    reportScreenshotData,
    reportScreenshotLoading,
    setReportScreenshotError,
  ]);

  return {
    reportIncludeScreenshot,
    reportScreenshotData,
    reportScreenshotLoading,
    reportScreenshotError,
    reportScreenshotOriginalDimensions,
    reportSelectionRect,
    reportScreenshotClip,
    reportScreenshotInfo,
    reportScreenshotCountdown,
    reportScreenshotContainerRef,
    reportScreenshotContainerHandlers,
    selectionDimensionLabel,
    handleReportIncludeScreenshotChange,
    handleRetryScreenshotCapture,
    handleResetScreenshotSelection,
    handleDelayedScreenshotCapture,
    handleScreenshotImageLoad,
    prepareForDialogOpen,
    cleanupAfterDialogClose,
  };
}
