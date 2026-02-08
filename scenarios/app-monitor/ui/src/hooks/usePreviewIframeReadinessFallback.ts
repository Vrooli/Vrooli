import { useEffect } from 'react';
import type { RefObject } from 'react';

interface UsePreviewIframeReadinessFallbackOptions {
  iframeRef: RefObject<HTMLIFrameElement>;
  enabled: boolean;
  onReady: () => void;
  maxAttempts?: number;
  intervalMs?: number;
}

export function usePreviewIframeReadinessFallback({
  iframeRef,
  enabled,
  onReady,
  maxAttempts = 40,
  intervalMs = 250,
}: UsePreviewIframeReadinessFallbackOptions) {
  useEffect(() => {
    if (!enabled) {
      return;
    }

    let cancelled = false;
    let attempts = 0;
    const intervalId = window.setInterval(() => {
      attempts += 1;
      const iframe = iframeRef.current;
      if (!iframe) {
        if (attempts >= maxAttempts) {
          window.clearInterval(intervalId);
        }
        return;
      }

      try {
        const readyState = iframe.contentDocument?.readyState;
        if (readyState === 'interactive' || readyState === 'complete') {
          if (!cancelled) {
            onReady();
          }
          window.clearInterval(intervalId);
          return;
        }
      } catch {
        // Cross-origin reads can fail; keep waiting for onLoad.
      }

      if (attempts >= maxAttempts) {
        window.clearInterval(intervalId);
      }
    }, intervalMs);

    return () => {
      cancelled = true;
      window.clearInterval(intervalId);
    };
  }, [enabled, iframeRef, intervalMs, maxAttempts, onReady]);
}

export default usePreviewIframeReadinessFallback;
