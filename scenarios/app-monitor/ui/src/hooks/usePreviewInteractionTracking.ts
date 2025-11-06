import { useEffect, useState } from 'react';
import type { RefObject } from 'react';

/**
 * Tracks user interactions with the preview iframe and browser window.
 * Increments a signal counter when user interacts (blur, pointer events).
 * Used to close menus/overlays when user interacts with preview.
 */
interface UsePreviewInteractionTrackingOptions {
  iframeRef: RefObject<HTMLIFrameElement | null>;
  previewUrl: string | null;
  previewReloadToken: number;
}

export interface UsePreviewInteractionTrackingReturn {
  /** Signal counter that increments on user interaction */
  previewInteractionSignal: number;
}

export const usePreviewInteractionTracking = ({
  iframeRef,
  previewUrl,
  previewReloadToken,
}: UsePreviewInteractionTrackingOptions): UsePreviewInteractionTrackingReturn => {
  const [previewInteractionSignal, setPreviewInteractionSignal] = useState(0);

  useEffect(() => {
    if (typeof window === 'undefined') {
      return () => {};
    }

    const handleWindowBlur = () => {
      setPreviewInteractionSignal(value => value + 1);
    };
    window.addEventListener('blur', handleWindowBlur);

    const iframe = iframeRef.current;
    const handlePointerDown = iframe ? () => {
      setPreviewInteractionSignal(value => value + 1);
    } : null;

    if (iframe && handlePointerDown) {
      iframe.addEventListener('pointerdown', handlePointerDown);
    }

    return () => {
      window.removeEventListener('blur', handleWindowBlur);
      if (iframe && handlePointerDown) {
        iframe.removeEventListener('pointerdown', handlePointerDown);
      }
    };
  }, [iframeRef, previewReloadToken, previewUrl]);

  return { previewInteractionSignal };
};
