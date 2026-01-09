import { useEffect, useState } from 'react';
import { PREVIEW_TIMEOUTS, PREVIEW_MESSAGES } from '@/components/views/previewConstants';
import type { PreviewOverlayState } from '@/types/preview';

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
  previewOverlay: PreviewOverlayState;
  setPreviewOverlay: React.Dispatch<React.SetStateAction<PreviewOverlayState>>;
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
  const [previewOverlay, setPreviewOverlay] = useState<PreviewOverlayState>(null);

  useEffect(() => {
    // Check if we should preserve the current overlay state
    setPreviewOverlay(current => {
      const preserveErrorOverlay = current?.type === 'error' && (
        current.message === previewNoUiMessage ||
        current.message === scenarioStoppedMessage
      );

      if (preserveErrorOverlay) {
        return current;
      }

      if (!previewUrl) {
        if (!current) {
          return current;
        }
        if (
          current.type === 'waiting' && current.message === PREVIEW_MESSAGES.CONNECTING
        ) {
          return null;
        }
        if (
          current.type === 'error' &&
          (current.message === PREVIEW_MESSAGES.TIMEOUT || current.message === PREVIEW_MESSAGES.MIXED_CONTENT)
        ) {
          return null;
        }
        return current;
      }

      if (bridgeIsReady || iframeLoadedAt) {
        if (current && current.type === 'waiting' && current.message === PREVIEW_MESSAGES.CONNECTING) {
          return null;
        }
        return current;
      }

      // If we reach here, we need to show loading state
      return current;
    });

    // Early return if no preview URL or already loaded
    if (!previewUrl || bridgeIsReady || iframeLoadedAt) {
      return;
    }

    let cancelled = false;
    let waitingApplied = false;
    const waitingTimeoutId = window.setTimeout(() => {
      if (cancelled) {
        return;
      }
      setPreviewOverlay(prev => {
        if (bridgeIsReady || iframeLoadedAt) {
          return prev;
        }
        if (prev && prev.type === 'restart') {
          return prev;
        }
        waitingApplied = true;
        return { type: 'waiting', message: PREVIEW_MESSAGES.CONNECTING };
      });
    }, PREVIEW_TIMEOUTS.WAITING_DELAY);
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
          ? PREVIEW_MESSAGES.MIXED_CONTENT
          : PREVIEW_MESSAGES.TIMEOUT;

      setPreviewOverlay(current => {
        if (current && current.type === 'restart') {
          return current;
        }
        return { type: 'error', message };
      });
    }, PREVIEW_TIMEOUTS.LOAD);

    return () => {
      cancelled = true;
      window.clearTimeout(waitingTimeoutId);
      window.clearTimeout(timeoutId);
      if (waitingApplied) {
        setPreviewOverlay(prev => {
          if (prev && prev.type === 'waiting' && prev.message === PREVIEW_MESSAGES.CONNECTING) {
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
    scenarioStoppedMessage,
    previewNoUiMessage,
  ]);

  return {
    previewOverlay,
    setPreviewOverlay,
  };
};
