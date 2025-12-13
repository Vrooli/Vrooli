/**
 * FloatingMiniPreview Component
 *
 * A floating, draggable mini preview of the live Playwright session.
 * Used when the timeline is in the main view, allowing users to still
 * see what's happening in the browser.
 *
 * Features:
 * - Draggable via FloatingPanel
 * - Resizable (golden ratio based on width)
 * - Click to swap to full live view
 * - Shows connection status
 */

import { useCallback, useMemo, useRef, useState, useEffect } from 'react';
import { FloatingPanel } from '@/components/FloatingPanel';
import { PlaywrightView, type FrameStats } from './PlaywrightView';

interface FloatingMiniPreviewProps {
  /** Session ID for the live preview */
  sessionId: string;
  /** Stream quality */
  quality?: number;
  /** Stream FPS */
  fps?: number;
  /** Whether WebSocket frames are being used */
  isWsConnected?: boolean;
  /** Callback to expand to full view */
  onExpandToFull: () => void;
  /** Callback when frame stats update */
  onStatsUpdate?: (stats: FrameStats) => void;
  /** Initial width (default: 320) */
  initialWidth?: number;
  /** Minimum width (default: 200) */
  minWidth?: number;
  /** Maximum width (default: 480) */
  maxWidth?: number;
}

/** Golden ratio for aesthetically pleasing dimensions */
const GOLDEN_RATIO = 1.618;

export function FloatingMiniPreview({
  sessionId,
  quality = 50,
  fps = 10,
  isWsConnected = false,
  onExpandToFull,
  onStatsUpdate,
  initialWidth = 320,
  minWidth = 200,
  maxWidth = 480,
}: FloatingMiniPreviewProps) {
  const [width, setWidth] = useState(initialWidth);
  const [isResizing, setIsResizing] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const resizeStartRef = useRef({ x: 0, width: initialWidth });

  // Calculate height based on golden ratio
  const height = Math.round(width / GOLDEN_RATIO);

  // Calculate safe default position (top-left corner, within viewport)
  const safeDefaultPosition = useMemo(() => {
    const margin = 24;
    const topOffset = 80; // Space from top for action bar

    return {
      x: margin,
      y: topOffset,
    };
  }, []);

  // Handle resize
  useEffect(() => {
    if (!isResizing) return;

    const handleMouseMove = (event: MouseEvent) => {
      const delta = event.clientX - resizeStartRef.current.x;
      const newWidth = Math.min(maxWidth, Math.max(minWidth, resizeStartRef.current.width + delta));
      setWidth(newWidth);
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      document.body.style.userSelect = '';
      document.body.style.cursor = '';
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    document.body.style.userSelect = 'none';
    document.body.style.cursor = 'ew-resize';

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      document.body.style.userSelect = '';
      document.body.style.cursor = '';
    };
  }, [isResizing, minWidth, maxWidth]);

  const handleResizeStart = useCallback((event: React.MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    resizeStartRef.current = { x: event.clientX, width };
    setIsResizing(true);
  }, [width]);

  return (
    <FloatingPanel
      id="record-mode-mini-preview"
      defaultPosition={safeDefaultPosition}
      zIndex={90}
      className="select-none"
    >
      <div
        ref={containerRef}
        className="relative bg-white dark:bg-gray-900 rounded-xl shadow-2xl border border-gray-200 dark:border-gray-700 overflow-hidden"
        style={{ width, height }}
      >
        {/* Header */}
        <div className="absolute top-0 left-0 right-0 z-10 flex items-center justify-between px-2 py-1.5 bg-gradient-to-b from-black/60 to-transparent">
          <div className="flex items-center gap-1.5">
            {/* Connection status */}
            <span
              className={`w-2 h-2 rounded-full ${isWsConnected ? 'bg-green-400' : 'bg-yellow-400'}`}
              title={isWsConnected ? 'Connected' : 'Reconnecting...'}
            />
            <span className="text-[10px] font-medium text-white/90">Live Preview</span>
          </div>

          {/* Expand button */}
          <button
            type="button"
            onClick={onExpandToFull}
            className="p-1 text-white/80 hover:text-white hover:bg-white/20 rounded transition-colors"
            title="Expand to full view"
          >
            <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
            </svg>
          </button>
        </div>

        {/* Live preview (no interaction - click expands) */}
        <div
          className="w-full h-full cursor-pointer"
          onClick={onExpandToFull}
          title="Click to expand"
        >
          <PlaywrightView
            sessionId={sessionId}
            quality={quality}
            fps={fps}
            viewport={{ width, height }}
            onStatsUpdate={onStatsUpdate}
          />
        </div>

        {/* Resize handle (bottom-right corner) */}
        <div
          className="absolute bottom-0 right-0 w-4 h-4 cursor-se-resize group"
          onMouseDown={handleResizeStart}
        >
          <svg
            className="absolute bottom-0.5 right-0.5 w-3 h-3 text-gray-400 dark:text-gray-500 group-hover:text-gray-600 dark:group-hover:text-gray-300 transition-colors"
            fill="currentColor"
            viewBox="0 0 24 24"
          >
            <path d="M22 22H20V20H22V22ZM22 18H20V16H22V18ZM18 22H16V20H18V22ZM22 14H20V12H22V14ZM18 18H16V16H18V18ZM14 22H12V20H14V22Z" />
          </svg>
        </div>

        {/* Resize indicator when resizing */}
        {isResizing && (
          <div className="absolute inset-0 border-2 border-blue-500 rounded-xl pointer-events-none">
            <div className="absolute bottom-2 right-2 px-2 py-1 bg-blue-500 text-white text-xs rounded">
              {width} × {height}
            </div>
          </div>
        )}
      </div>
    </FloatingPanel>
  );
}
