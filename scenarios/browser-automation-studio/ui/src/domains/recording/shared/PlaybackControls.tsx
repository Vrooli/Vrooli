/**
 * PlaybackControls - Unified playback controls for video and slideshow content
 *
 * Provides play/pause, previous/next, progress bar, and time/frame indicator.
 * Adapts appearance based on content type (video vs slideshow).
 */

import { useCallback, useEffect, useRef, useState } from 'react';
import { ChevronLeft, ChevronRight, Pause, Play } from 'lucide-react';
import clsx from 'clsx';

export type PlaybackContentType = 'video' | 'slideshow';

interface BaseControlsProps {
  /** Content type determines control behavior and display */
  contentType: PlaybackContentType;
  /** Whether playback is active */
  isPlaying: boolean;
  /** Progress from 0 to 1 */
  progress: number;
  /** Callback when play/pause is clicked */
  onPlayPause: () => void;
  /** Callback when previous is clicked */
  onPrevious: () => void;
  /** Callback when next is clicked */
  onNext: () => void;
  /** Callback when user seeks to a position (0 to 1) */
  onSeek?: (position: number) => void;
  /** Additional class name for the container */
  className?: string;
}

interface VideoControlsProps extends BaseControlsProps {
  contentType: 'video';
  /** Video ref for getting current time and duration */
  videoRef?: React.RefObject<HTMLVideoElement | null>;
  /** Current time in seconds (optional, derived from videoRef if available) */
  currentTime?: number;
  /** Total duration in seconds (optional, derived from videoRef if available) */
  duration?: number;
}

interface SlideshowControlsProps extends BaseControlsProps {
  contentType: 'slideshow';
  /** Current frame index (0-based) */
  currentIndex: number;
  /** Total number of frames */
  totalFrames: number;
}

export type PlaybackControlsProps = VideoControlsProps | SlideshowControlsProps;

/** Format seconds as mm:ss or hh:mm:ss */
function formatTime(seconds: number): string {
  if (!isFinite(seconds) || seconds < 0) return '0:00';

  const hrs = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);

  if (hrs > 0) {
    return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  }
  return `${mins}:${secs.toString().padStart(2, '0')}`;
}

export function PlaybackControls(props: PlaybackControlsProps) {
  const {
    contentType,
    isPlaying,
    progress,
    onPlayPause,
    onPrevious,
    onNext,
    onSeek,
    className,
  } = props;

  const progressBarRef = useRef<HTMLDivElement>(null);
  const [isDragging, setIsDragging] = useState(false);

  // Get display values based on content type
  const displayInfo = (() => {
    if (contentType === 'slideshow') {
      const { currentIndex, totalFrames } = props as SlideshowControlsProps;
      return {
        current: `${currentIndex + 1}`,
        total: `${totalFrames}`,
        separator: '/',
      };
    } else {
      const videoProps = props as VideoControlsProps;
      let currentTime = videoProps.currentTime ?? 0;
      let duration = videoProps.duration ?? 0;

      // Try to get from video ref if available
      if (videoProps.videoRef?.current) {
        const video = videoProps.videoRef.current;
        currentTime = video.currentTime;
        duration = video.duration || 0;
      }

      return {
        current: formatTime(currentTime),
        total: formatTime(duration),
        separator: '/',
      };
    }
  })();

  // Handle click on progress bar
  const handleProgressClick = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (!progressBarRef.current || !onSeek) return;

      const rect = progressBarRef.current.getBoundingClientRect();
      const position = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
      onSeek(position);
    },
    [onSeek]
  );

  // Handle drag start
  const handleMouseDown = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (!onSeek) return;
      setIsDragging(true);
      handleProgressClick(e);
    },
    [onSeek, handleProgressClick]
  );

  // Handle drag and release
  useEffect(() => {
    if (!isDragging) return;

    const handleMouseMove = (e: MouseEvent) => {
      if (!progressBarRef.current || !onSeek) return;

      const rect = progressBarRef.current.getBoundingClientRect();
      const position = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
      onSeek(position);
    };

    const handleMouseUp = () => {
      setIsDragging(false);
    };

    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);

    return () => {
      document.removeEventListener('mousemove', handleMouseMove);
      document.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, onSeek]);

  return (
    <div
      className={clsx(
        'flex items-center gap-3 px-4 py-3 bg-slate-900/95 backdrop-blur-sm border-t border-white/10',
        className
      )}
    >
      {/* Previous button */}
      <button
        className="flex h-8 w-8 items-center justify-center rounded-full border border-white/10 bg-white/5 text-white hover:bg-white/10 transition-colors"
        onClick={onPrevious}
        aria-label={contentType === 'slideshow' ? 'Previous frame' : 'Skip back'}
      >
        <ChevronLeft size={16} />
      </button>

      {/* Play/Pause button */}
      <button
        className="flex h-8 w-8 items-center justify-center rounded-full bg-flow-accent text-white hover:bg-blue-500 transition-colors"
        onClick={onPlayPause}
        aria-label={isPlaying ? 'Pause' : 'Play'}
      >
        {isPlaying ? <Pause size={16} /> : <Play size={16} className="ml-0.5" />}
      </button>

      {/* Next button */}
      <button
        className="flex h-8 w-8 items-center justify-center rounded-full border border-white/10 bg-white/5 text-white hover:bg-white/10 transition-colors"
        onClick={onNext}
        aria-label={contentType === 'slideshow' ? 'Next frame' : 'Skip forward'}
      >
        <ChevronRight size={16} />
      </button>

      {/* Progress bar */}
      <div
        ref={progressBarRef}
        className={clsx(
          'flex-1 h-1.5 rounded-full bg-white/10 cursor-pointer relative',
          onSeek && 'hover:h-2 transition-[height]'
        )}
        onClick={handleProgressClick}
        onMouseDown={handleMouseDown}
      >
        {/* Filled portion */}
        <div
          className="absolute inset-y-0 left-0 rounded-full bg-flow-accent transition-[width] duration-75"
          style={{ width: `${progress * 100}%` }}
        />
        {/* Seek handle (shown on hover/drag) */}
        {onSeek && (
          <div
            className={clsx(
              'absolute top-1/2 -translate-y-1/2 w-3 h-3 rounded-full bg-white shadow-md transition-opacity',
              isDragging ? 'opacity-100' : 'opacity-0 group-hover:opacity-100'
            )}
            style={{ left: `calc(${progress * 100}% - 6px)` }}
          />
        )}
      </div>

      {/* Time/Frame indicator */}
      <div className="text-xs text-slate-300 font-mono min-w-[80px] text-right">
        <span>{displayInfo.current}</span>
        <span className="text-slate-500 mx-1">{displayInfo.separator}</span>
        <span className="text-slate-400">{displayInfo.total}</span>
      </div>
    </div>
  );
}
