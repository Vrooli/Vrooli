import type { RefObject } from 'react';
import { logger } from '@/services/logger';

/**
 * Resolves the background color of a preview iframe or container element.
 * Attempts to inspect the iframe's body/documentElement first, then falls back to the container.
 *
 * @param iframeRef - Reference to the iframe element containing the preview
 * @param containerRef - Reference to the container element wrapping the preview
 * @returns The resolved background color as a CSS color string, or undefined if none found
 */
export const resolvePreviewBackgroundColor = (
  iframeRef: RefObject<HTMLIFrameElement | null>,
  containerRef: RefObject<HTMLElement | null>,
): string | undefined => {
  if (typeof window === 'undefined') {
    return undefined;
  }

  const iframe = iframeRef.current;
  if (iframe) {
    try {
      const iframeWindow = iframe.contentWindow;
      const iframeDocument = iframe.contentDocument ?? iframeWindow?.document ?? null;
      if (iframeDocument && iframeWindow) {
        const candidates: Element[] = [];
        if (iframeDocument.body) {
          candidates.push(iframeDocument.body);
        }
        if (iframeDocument.documentElement && iframeDocument.documentElement !== iframeDocument.body) {
          candidates.push(iframeDocument.documentElement);
        }
        for (const element of candidates) {
          const style = iframeWindow.getComputedStyle(element);
          if (!style) {
            continue;
          }
          const color = style.backgroundColor;
          if (color && color !== 'rgba(0, 0, 0, 0)' && color.toLowerCase() !== 'transparent') {
            return color;
          }
        }
      }
    } catch (error) {
      logger.debug('Unable to inspect iframe background color', error);
    }
  }

  const container = containerRef.current;
  if (container) {
    try {
      const style = window.getComputedStyle(container);
      const color = style.backgroundColor;
      if (color && color !== 'rgba(0, 0, 0, 0)' && color.toLowerCase() !== 'transparent') {
        return color;
      }
    } catch (error) {
      logger.debug('Unable to inspect preview container background color', error);
    }
  }

  return undefined;
};
