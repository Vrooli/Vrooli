import { useEffect, useState } from 'react';

const PREVIEW_LOAD_TIMEOUT_MS = 6000;
const PREVIEW_WAITING_DELAY_MS = 400;
const PREVIEW_CONNECTING_LABEL = 'Connecting to preview...';
const PREVIEW_TIMEOUT_MESSAGE = 'Preview did not respond. Ensure the application UI is running and reachable from App Monitor.';
const PREVIEW_MIXED_CONTENT_MESSAGE = 'Preview blocked: browser refused to load HTTP content inside an HTTPS dashboard. Expose the UI through the tunnel hostname or enable HTTPS for the scenario.';

type PreviewOverlay = {
  type: 'restart' | 'waiting' | 'error';
  message: string;
} | null;

interface UsePreviewOverlayOptions {
  previewUrl: string | null;
  previewReloadToken: number;
  bridgeIsReady: boolean;
  iframeLoadedAt: number | null;
  iframeLoadError: string | null;
  scenarioStoppedMessage: string;
  previewNoUiMessage: string;
}

export interface UsePreviewOverlayReturn {
  previewOverlay: PreviewOverlay;
  setPreviewOverlay: (overlay: PreviewOverlay) => void;
}

export const usePreviewOverlay = ({
  previewUrl,
  previewReloadToken,
  bridgeIsReady,
  iframeLoadedAt,
  iframeLoadError,
  scenarioStoppedMessage,
  previewNoUiMessage,
}: UsePreviewOverlayOptions): UsePreviewOverlayReturn => {
  const [previewOverlay, setPreviewOverlay] = useState<PreviewOverlay>(null);

  useEffect(() => {
    const preserveErrorOverlay = previewOverlay?.type === 'error' && (
      previewOverlay.message === previewNoUiMessage ||
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

    if (bridgeIsReady || iframeLoadedAt) {
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
        if (cancelled || bridgeIsReady || iframeLoadedAt) {
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
      if (cancelled || bridgeIsReady || iframeLoadedAt) {
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
    bridgeIsReady,
    iframeLoadedAt,
    iframeLoadError,
    previewOverlay,
    scenarioStoppedMessage,
    previewNoUiMessage,
  ]);

  return {
    previewOverlay,
    setPreviewOverlay,
  };
};
