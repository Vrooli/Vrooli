import { useCallback, useEffect, useRef } from 'react';
import { ensureDataUrl } from '@/utils/dataUrl';
import { logger } from '@/services/logger';
import type { UseIframeBridgeReturn } from '@/hooks/useIframeBridge';
import type { SurfaceScreenshotMeta, SurfaceType } from '@/state/surfaceMediaStore';

const PREVIEW_SCREENSHOT_STABILITY_MS = 1000;

type ScheduleOptions = {
  force?: boolean;
};

interface UsePreviewCaptureOptions {
  activeOverlay: string | null;
  bridgeIsReady: boolean;
  bridgeLastReadyAt?: number;
  bridgeSupportsScreenshot: boolean;
  canCaptureScreenshot: boolean;
  currentAppIdentifier: string | null;
  iframeLoadedAt: number | null;
  requestScreenshot: UseIframeBridgeReturn['requestScreenshot'];
  resolvePreviewBackgroundColor: () => string | undefined;
  setSurfaceScreenshot: (
    type: SurfaceType,
    id: string,
    meta: SurfaceScreenshotMeta,
  ) => void;
  surfaceType: SurfaceType;
}

type ScreenshotGuard = {
  surfaceId: string | null;
  capturedAt: number;
};

export const usePreviewCapture = ({
  activeOverlay,
  bridgeIsReady,
  bridgeLastReadyAt,
  bridgeSupportsScreenshot,
  canCaptureScreenshot,
  currentAppIdentifier,
  iframeLoadedAt,
  requestScreenshot,
  resolvePreviewBackgroundColor,
  setSurfaceScreenshot,
  surfaceType,
}: UsePreviewCaptureOptions) => {
  const captureInFlightRef = useRef(false);
  const lastScreenshotRef = useRef<ScreenshotGuard>({ surfaceId: null, capturedAt: 0 });
  const overlayStateRef = useRef<string | null>(null);

  const scheduleScreenshotCapture = useCallback((delayMs = 0, options?: ScheduleOptions) => {
    if (!currentAppIdentifier || !bridgeSupportsScreenshot || !canCaptureScreenshot) {
      return undefined;
    }

    if (!bridgeIsReady || !iframeLoadedAt) {
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

    const stabilityAnchor = bridgeLastReadyAt ?? iframeLoadedAt;
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
      const backgroundColor = resolvePreviewBackgroundColor();

      void (async () => {
        try {
          const result = await requestScreenshot({ mode: 'viewport', scale, backgroundColor });
          if (cancelled) {
            return;
          }
          const dataUrl = ensureDataUrl(result.data);
          if (dataUrl) {
            setSurfaceScreenshot(surfaceType, currentAppIdentifier, {
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
    bridgeIsReady,
    bridgeLastReadyAt,
    bridgeSupportsScreenshot,
    canCaptureScreenshot,
    currentAppIdentifier,
    iframeLoadedAt,
    requestScreenshot,
    resolvePreviewBackgroundColor,
    setSurfaceScreenshot,
    surfaceType,
  ]);

  useEffect(() => {
    if (lastScreenshotRef.current.surfaceId !== currentAppIdentifier) {
      lastScreenshotRef.current = { surfaceId: currentAppIdentifier, capturedAt: 0 };
    }
  }, [currentAppIdentifier]);

  useEffect(() => {
    const previous = overlayStateRef.current;
    overlayStateRef.current = activeOverlay;

    if (activeOverlay !== 'tabs' || previous === 'tabs') {
      return undefined;
    }

    const now = Date.now();
    const delay = iframeLoadedAt ? Math.max(0, 250 - (now - iframeLoadedAt)) : 0;
    return scheduleScreenshotCapture(delay);
  }, [activeOverlay, iframeLoadedAt, scheduleScreenshotCapture]);

  useEffect(() => {
    if (!iframeLoadedAt) {
      return undefined;
    }

    return scheduleScreenshotCapture(200);
  }, [iframeLoadedAt, scheduleScreenshotCapture]);
};

export type { UsePreviewCaptureOptions };
