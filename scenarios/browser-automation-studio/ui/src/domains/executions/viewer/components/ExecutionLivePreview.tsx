/**
 * ExecutionLivePreview Component
 *
 * Displays a live browser preview during workflow execution.
 * Uses WebSocket to receive real-time frames from the playwright-driver.
 *
 * Usage:
 *   <ExecutionLivePreview executionId={executionId} />
 */

import { useState, useEffect } from 'react';
import clsx from 'clsx';
import { Video, VideoOff, Loader2, AlertCircle, Maximize2, Minimize2 } from 'lucide-react';
import { useExecutionFrameStream } from '../../hooks';

interface ExecutionLivePreviewProps {
  /** The execution ID to stream frames from */
  executionId: string;
  /** Whether streaming should be enabled */
  enabled?: boolean;
  /** Additional CSS classes */
  className?: string;
  /** Callback when streaming status changes */
  onStreamingChange?: (isStreaming: boolean) => void;
}

export function ExecutionLivePreview({
  executionId,
  enabled = true,
  className,
  onStreamingChange,
}: ExecutionLivePreviewProps) {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [showStats, setShowStats] = useState(false);

  const { frameUrl, frame, isStreaming, isSubscribed, frameCount, error } = useExecutionFrameStream(
    executionId,
    { enabled }
  );

  // Notify parent of streaming status changes
  useEffect(() => {
    onStreamingChange?.(isStreaming);
  }, [isStreaming, onStreamingChange]);

  // Calculate FPS (rough estimate based on frame count over time)
  const [fps, setFps] = useState(0);
  useEffect(() => {
    if (!isStreaming) {
      setFps(0);
      return;
    }

    const startTime = Date.now();
    const startCount = frameCount;

    const interval = setInterval(() => {
      const elapsed = (Date.now() - startTime) / 1000;
      if (elapsed > 0) {
        setFps(Math.round((frameCount - startCount) / elapsed));
      }
    }, 1000);

    return () => clearInterval(interval);
  }, [isStreaming, frameCount]);

  // Render loading state
  if (!isSubscribed && enabled) {
    return (
      <div className={clsx('flex flex-col items-center justify-center h-full min-h-[300px] bg-flow-node/70 rounded-lg border border-flow-border', className)}>
        <Loader2 className="w-8 h-8 text-flow-text-secondary animate-spin mb-3" />
        <p className="text-sm text-flow-text-secondary">Connecting to live preview...</p>
      </div>
    );
  }

  // Render disabled state
  if (!enabled) {
    return (
      <div className={clsx('flex flex-col items-center justify-center h-full min-h-[300px] bg-flow-node/70 rounded-lg border border-flow-border', className)}>
        <VideoOff className="w-8 h-8 text-flow-text-secondary mb-3" />
        <p className="text-sm text-flow-text-secondary">Live preview disabled</p>
        <p className="text-xs text-flow-text-tertiary mt-1">
          Enable frame streaming when starting the execution
        </p>
      </div>
    );
  }

  // Render error state
  if (error) {
    return (
      <div className={clsx('flex flex-col items-center justify-center h-full min-h-[300px] bg-flow-node/70 rounded-lg border border-rose-500/30', className)}>
        <AlertCircle className="w-8 h-8 text-rose-400 mb-3" />
        <p className="text-sm text-rose-200">Failed to connect to live preview</p>
        <p className="text-xs text-rose-100/70 mt-1">{error}</p>
      </div>
    );
  }

  // Render waiting for frames state
  if (isSubscribed && !frameUrl) {
    return (
      <div className={clsx('flex flex-col items-center justify-center h-full min-h-[300px] bg-flow-node/70 rounded-lg border border-flow-border', className)}>
        <Video className="w-8 h-8 text-flow-text-secondary animate-pulse mb-3" />
        <p className="text-sm text-flow-text-secondary">Waiting for frames...</p>
        <p className="text-xs text-flow-text-tertiary mt-1">
          Frames will appear once the browser starts
        </p>
      </div>
    );
  }

  return (
    <div className={clsx('relative group', isFullscreen && 'fixed inset-0 z-50 bg-black', className)}>
      {/* Frame display */}
      <div className={clsx(
        'relative overflow-hidden rounded-lg border border-flow-border bg-black',
        isFullscreen && 'h-full flex items-center justify-center'
      )}>
        {frameUrl && (
          <img
            src={frameUrl}
            alt="Live browser preview"
            className={clsx(
              'w-full h-auto object-contain',
              isFullscreen && 'max-h-full max-w-full w-auto'
            )}
          />
        )}

        {/* Streaming indicator */}
        <div className="absolute top-2 left-2 flex items-center gap-2 px-2 py-1 rounded-md bg-black/70 text-xs">
          <span className={clsx(
            'w-2 h-2 rounded-full',
            isStreaming ? 'bg-green-500 animate-pulse' : 'bg-gray-500'
          )} />
          <span className="text-white/80">
            {isStreaming ? 'LIVE' : 'Connecting...'}
          </span>
        </div>

        {/* Stats overlay (toggle with click) */}
        {showStats && (
          <div className="absolute top-2 right-12 px-2 py-1 rounded-md bg-black/70 text-xs text-white/80">
            <div>Frames: {frameCount}</div>
            <div>FPS: ~{fps}</div>
            {frame && (
              <div>
                {frame.width}x{frame.height}
              </div>
            )}
          </div>
        )}

        {/* Controls overlay */}
        <div className="absolute top-2 right-2 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={() => setShowStats(!showStats)}
            className="p-1.5 rounded-md bg-black/70 text-white/80 hover:bg-black/90 hover:text-white transition-colors"
            title={showStats ? 'Hide stats' : 'Show stats'}
          >
            <Video className="w-4 h-4" />
          </button>
          <button
            onClick={() => setIsFullscreen(!isFullscreen)}
            className="p-1.5 rounded-md bg-black/70 text-white/80 hover:bg-black/90 hover:text-white transition-colors"
            title={isFullscreen ? 'Exit fullscreen' : 'Fullscreen'}
          >
            {isFullscreen ? (
              <Minimize2 className="w-4 h-4" />
            ) : (
              <Maximize2 className="w-4 h-4" />
            )}
          </button>
        </div>
      </div>

      {/* Info bar */}
      {!isFullscreen && (
        <div className="flex items-center justify-between mt-2 px-1 text-xs text-flow-text-tertiary">
          <span>
            {isStreaming ? (
              <>Live preview - {frameCount} frames received</>
            ) : (
              'Connecting to live preview...'
            )}
          </span>
          {frame && (
            <span>
              {frame.width}x{frame.height} @ ~{fps} FPS
            </span>
          )}
        </div>
      )}
    </div>
  );
}

export default ExecutionLivePreview;
