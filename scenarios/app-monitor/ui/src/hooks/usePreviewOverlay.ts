import { useEffect, useMemo, useState } from 'react';
import { PREVIEW_TIMEOUTS, PREVIEW_MESSAGES } from '@/components/views/previewConstants';
import type { PreviewOverlayState } from '@/types/preview';

interface UsePreviewOverlayOptions {
  previewUrl: string | null;
  previewReloadToken: number;
  loading: boolean;
  statusMessage: string | null;
  defaultEmptyMessage?: string;
  bridgeIsReady: boolean;
  iframeLoadedAt: number | null;
  iframeLoadError: string | null;
  preserveErrorMessages?: readonly string[];
}

export type PreviewFallbackState = {
  type: 'loading' | 'waiting' | 'restart' | 'error' | 'empty';
  message: string;
  showSkeleton: boolean;
  showSpinner: boolean;
  isBlocking: boolean;
};

export interface UsePreviewOverlayReturn {
  previewOverlay: PreviewOverlayState;
  setPreviewOverlay: React.Dispatch<React.SetStateAction<PreviewOverlayState>>;
  fallbackState: PreviewFallbackState | null;
}

export const usePreviewOverlay = ({
  previewUrl,
  previewReloadToken,
  loading,
  statusMessage,
  defaultEmptyMessage,
  bridgeIsReady,
  iframeLoadedAt,
  iframeLoadError,
  preserveErrorMessages = [],
}: UsePreviewOverlayOptions): UsePreviewOverlayReturn => {
  const [previewOverlay, setPreviewOverlay] = useState<PreviewOverlayState>(null);
  const persistentMessages = useMemo(() => new Set(preserveErrorMessages), [preserveErrorMessages]);

  useEffect(() => {
    setPreviewOverlay(current => {
      const preserveErrorOverlay = current?.type === 'error' && persistentMessages.has(current.message);

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

      return current;
    });

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
    persistentMessages,
  ]);

  const fallbackState = (() => {
    if (loading) {
      return {
        type: 'loading',
        message: statusMessage ?? 'Loading preview...',
        showSkeleton: true,
        showSpinner: false,
        isBlocking: true,
      } satisfies PreviewFallbackState;
    }

    if (!previewUrl) {
      return {
        type: 'empty',
        message: statusMessage ?? defaultEmptyMessage ?? 'Preview unavailable.',
        showSkeleton: false,
        showSpinner: false,
        isBlocking: true,
      } satisfies PreviewFallbackState;
    }

    if (previewOverlay) {
      if (previewOverlay.type === 'error') {
        return {
          type: 'error',
          message: previewOverlay.message,
          showSkeleton: false,
          showSpinner: false,
          isBlocking: true,
        } satisfies PreviewFallbackState;
      }

      return {
        type: previewOverlay.type,
        message: previewOverlay.message,
        showSkeleton: true,
        showSpinner: true,
        isBlocking: true,
      } satisfies PreviewFallbackState;
    }

    if (iframeLoadError) {
      return {
        type: 'error',
        message: iframeLoadError,
        showSkeleton: false,
        showSpinner: false,
        isBlocking: true,
      } satisfies PreviewFallbackState;
    }

    if (!iframeLoadedAt) {
      return {
        type: 'loading',
        message: 'Loading preview...',
        showSkeleton: true,
        showSpinner: false,
        isBlocking: true,
      } satisfies PreviewFallbackState;
    }

    return null;
  })();

  return {
    previewOverlay,
    setPreviewOverlay,
    fallbackState,
  };
};
