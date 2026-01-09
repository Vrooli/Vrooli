/**
 * PlaywrightView Component
 *
 * Renders a live view of the Playwright session with user input forwarding.
 * Uses extracted hooks for frame streaming and input handling.
 *
 * ## Coordinate Mapping (IMPORTANT)
 *
 * The live preview displays JPEG screenshots captured by Playwright. These screenshots
 * are captured at the device's pixel ratio (e.g., 2x on HiDPI/Retina displays), meaning
 * a 900x700 viewport produces an 1800x1400 image.
 *
 * When the user clicks on the preview, we must map their click coordinates to the
 * Playwright viewport coordinates, NOT the screenshot bitmap coordinates.
 *
 * Example:
 * - Viewport: 900x700 (logical pixels)
 * - Screenshot: 1800x1400 (physical pixels at 2x DPR)
 * - User clicks at (450, 350) on the displayed preview
 * - Correct: Send (450, 350) to Playwright
 * - Wrong: Send (900, 700) to Playwright (2x scaled - clicks land off-screen!)
 *
 * The `viewport` prop provides the logical viewport dimensions for correct mapping.
 * If not provided, falls back to frame dimensions (which may be wrong on HiDPI).
 */

import { useCallback, useEffect, useRef } from 'react';
import { useFrameStream, type PageMetadata, type StreamConnectionStatus } from './useFrameStream';
import { useInputForwarding } from './useInputForwarding';
import type { FrameStats } from '../hooks/useFrameStats';

// Re-export types for consumers
export type { FrameStats } from '../hooks/useFrameStats';
export type { PageMetadata, StreamConnectionStatus } from './useFrameStream';

interface PlaywrightViewProps {
  sessionId: string;
  /** Optional page ID for multi-tab sessions. When provided, frames are received for this specific page. */
  pageId?: string;
  quality?: number;
  fps?: number;
  onStreamError?: (message: string) => void;
  refreshToken?: number;
  /** Whether to use WebSocket for frame updates (default: true) */
  useWebSocketFrames?: boolean;
  /** Logical viewport dimensions (for coordinate mapping, independent of device pixel ratio) */
  viewport?: { width: number; height: number };
  /** Callback to receive frame statistics updates */
  onStatsUpdate?: (stats: FrameStats) => void;
  /** Callback when page metadata (title, url) changes */
  onPageMetadataChange?: (metadata: PageMetadata) => void;
  /** Callback with the rendered content rect (relative to the container) */
  onContentRectChange?: (rect: { x: number; y: number; width: number; height: number }) => void;
  /** Callback when stream connection status changes */
  onConnectionStatusChange?: (status: StreamConnectionStatus) => void;
  /** Whether to hide the in-preview connection indicator (use when status is shown in header) */
  hideConnectionIndicator?: boolean;
  /** Whether a viewport resize is in progress (from ViewportSyncManager) */
  isResizing?: boolean;
  /** Whether viewport sync is pending (awaiting backend acknowledgment) */
  isViewportSyncing?: boolean;
}

export function PlaywrightView({
  sessionId,
  pageId,
  quality = 65,
  fps = 30,
  onStreamError,
  refreshToken,
  useWebSocketFrames = true,
  viewport,
  onStatsUpdate,
  onPageMetadataChange,
  onContentRectChange,
  onConnectionStatusChange,
  hideConnectionIndicator = false,
  isResizing = false,
  isViewportSyncing = false,
}: PlaywrightViewProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);

  // Frame streaming hook
  const {
    canvasRef,
    hasFrame,
    displayDimensions,
    displayedTimestamp,
    error,
    isFetching,
    isWsFrameActive,
    isPageSwitching,
  } = useFrameStream({
    sessionId,
    pageId,
    quality,
    fps,
    useWebSocketFrames,
    refreshToken,
    onStreamError,
    onStatsUpdate,
    onPageMetadataChange,
    onConnectionStatusChange,
  });

  // Input forwarding hook
  // Pass both viewport (for output coordinates) and displayDimensions (for display calculation)
  // This ensures accurate cursor position on HiDPI displays where frame size differs from viewport
  const {
    handlePointer: handlePointerBase,
    handleWheel: handleWheelBase,
    handleKey: handleKeyBase,
    setWsSubscribed,
  } = useInputForwarding({
    sessionId,
    pageId,
    viewport,
    frameDimensions: displayDimensions,
    onError: onStreamError,
  });

  // Sync WebSocket subscription state
  useEffect(() => {
    setWsSubscribed(isWsFrameActive);
  }, [isWsFrameActive, setWsSubscribed]);

  // Content rect reporting
  const onContentRectChangeRef = useRef(onContentRectChange);
  onContentRectChangeRef.current = onContentRectChange;

  const reportContentRect = useCallback(() => {
    if (!onContentRectChangeRef.current || !hasFrame) {
      return;
    }
    const container = containerRef.current;
    const canvas = canvasRef.current;
    if (!container || !canvas) {
      return;
    }
    const containerRect = container.getBoundingClientRect();
    const canvasRect = canvas.getBoundingClientRect();
    if (containerRect.width <= 0 || containerRect.height <= 0 || canvasRect.width <= 0 || canvasRect.height <= 0) {
      return;
    }
    onContentRectChangeRef.current({
      x: canvasRect.left - containerRect.left,
      y: canvasRect.top - containerRect.top,
      width: canvasRect.width,
      height: canvasRect.height,
    });
  }, [hasFrame, canvasRef]);

  useEffect(() => {
    if (!onContentRectChangeRef.current) {
      return;
    }
    const container = containerRef.current;
    if (!container || typeof ResizeObserver === 'undefined') {
      return;
    }
    const observer = new ResizeObserver(() => {
      reportContentRect();
    });
    observer.observe(container);
    return () => observer.disconnect();
  }, [reportContentRect]);

  useEffect(() => {
    reportContentRect();
  }, [displayDimensions, reportContentRect]);

  // Wrap input handlers to get canvas rect and hasFrame state
  const handlePointer = useCallback(
    (action: 'move' | 'down' | 'up' | 'click', e: React.PointerEvent<HTMLDivElement>) => {
      const canvas = canvasRef.current;
      if (!canvas) return;

      // IMPORTANT: Use canvas rect, not container rect!
      // The canvas uses CSS object-contain which may letterbox the image.
      const rect = canvas.getBoundingClientRect();

      // Get canvas internal dimensions directly from the element.
      // This is more reliable than displayDimensions state which can be stale
      // during the brief window after a frame is drawn but before React state updates.
      const canvasInternalDimensions = {
        width: canvas.width,
        height: canvas.height,
      };

      handlePointerBase(action, e, rect, hasFrame, canvasInternalDimensions);
    },
    [canvasRef, handlePointerBase, hasFrame]
  );

  const handleWheel = useCallback(
    (e: React.WheelEvent<HTMLDivElement>) => {
      handleWheelBase(e, hasFrame);
    },
    [handleWheelBase, hasFrame]
  );

  const handleKey = useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      handleKeyBase(e, hasFrame);
    },
    [handleKeyBase, hasFrame]
  );

  return (
    <div
      ref={containerRef}
      tabIndex={0}
      onPointerMove={(e) => handlePointer('move', e)}
      onPointerDown={(e) => {
        containerRef.current?.focus();
        handlePointer('down', e);
      }}
      onPointerUp={(e) => handlePointer('up', e)}
      onWheel={handleWheel}
      onKeyDown={handleKey}
      className="relative h-full w-full overflow-hidden rounded-md border border-gray-200 dark:border-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
    >
      {/* Canvas for frame rendering
          IMPORTANT: Do NOT set width/height as React props!
          Setting canvas width/height clears all content. The drawFrameToCanvas function
          in useFrameStream handles dimensions directly on the canvas element.
          React re-renders would clear the canvas between frames, causing white flicker. */}
      <canvas
        ref={canvasRef}
        className={`w-full h-full object-contain bg-white pointer-events-none ${hasFrame ? '' : 'hidden'}`}
      />

      {/* Page switching transition overlay */}
      {isPageSwitching && (
        <div className="absolute inset-0 flex items-center justify-center bg-gray-100/80 dark:bg-gray-900/80 transition-opacity duration-200 z-10">
          <div className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
            <svg className="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
            Switching tab…
          </div>
        </div>
      )}

      {/* Resize transition overlay */}
      {isResizing && hasFrame && (
        <div className="absolute inset-0 bg-gray-100/30 dark:bg-gray-900/30 transition-opacity duration-150 z-5 pointer-events-none" />
      )}

      {/* Viewport sync indicator */}
      {isViewportSyncing && hasFrame && !isResizing && (
        <div className="absolute top-2 right-2 px-2 py-1 text-[10px] text-gray-500 dark:text-gray-400 bg-white/80 dark:bg-gray-900/80 rounded shadow-sm z-5">
          Syncing viewport…
        </div>
      )}

      {/* Loading state */}
      {!hasFrame && !isPageSwitching && (
        <div className="absolute inset-0 flex items-center justify-center text-sm text-gray-500 dark:text-gray-400">
          {isFetching ? 'Connecting to live session…' : 'Waiting for first frame…'}
        </div>
      )}

      {/* Connection status indicator */}
      {!hideConnectionIndicator && (
        <div
          className="absolute left-2 top-2 rounded-full bg-white/90 dark:bg-gray-900/80 px-3 py-1 text-[11px] text-gray-700 dark:text-gray-200 border border-gray-200 dark:border-gray-700 shadow-sm flex items-center gap-2 cursor-help group"
          title="Recording session is synced to this URL. Interactions are executed in the Playwright browser session."
        >
          <span className={`w-2 h-2 rounded-full ${isWsFrameActive ? 'bg-green-500' : 'bg-yellow-500'}`} />
          Live {isWsFrameActive ? '(WebSocket)' : '(polling)'}
          {displayedTimestamp && (
            <span className="text-gray-500 dark:text-gray-400">
              {new Date(displayedTimestamp).toLocaleTimeString()}
            </span>
          )}
        </div>
      )}

      {/* Error display */}
      {error && (
        <div className="absolute inset-x-4 bottom-4 rounded-md bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-200 text-xs px-3 py-2 border border-red-100 dark:border-red-800 shadow-sm">
          {error}. Streaming will retry automatically.
        </div>
      )}
    </div>
  );
}
