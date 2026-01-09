import { useCallback } from 'react';
import type { RefObject } from 'react';
import { resolvePreviewBackgroundColor } from '@/utils/previewBackgroundColor';

/**
 * Hook to get a callback that resolves the preview background color
 *
 * @param iframeRef - Reference to the iframe element containing the preview
 * @param containerRef - Reference to the container element wrapping the preview
 * @returns A callback that resolves the background color when called
 */
export const usePreviewBackgroundColor = (
  iframeRef: RefObject<HTMLIFrameElement | null>,
  containerRef: RefObject<HTMLElement | null>,
) => {
  return useCallback(
    () => resolvePreviewBackgroundColor(iframeRef, containerRef),
    [iframeRef, containerRef],
  );
};
