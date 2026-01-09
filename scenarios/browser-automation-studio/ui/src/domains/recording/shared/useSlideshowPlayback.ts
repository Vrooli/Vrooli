/**
 * useSlideshowPlayback - Hook for managing slideshow playback state
 *
 * Manages current index, playing/paused state, and auto-advance behavior.
 * Used by PlaybackControls to control screenshot slideshows.
 */

import { useCallback, useEffect, useRef, useState } from 'react';

export interface SlideshowPlaybackState {
  /** Current index in the slideshow */
  currentIndex: number;
  /** Whether the slideshow is auto-playing */
  isPlaying: boolean;
  /** Total number of frames */
  totalFrames: number;
  /** Progress as a value between 0 and 1 */
  progress: number;
}

export interface SlideshowPlaybackActions {
  /** Start auto-advancing */
  play: () => void;
  /** Stop auto-advancing */
  pause: () => void;
  /** Toggle play/pause */
  toggle: () => void;
  /** Go to next frame */
  next: () => void;
  /** Go to previous frame */
  previous: () => void;
  /** Jump to a specific frame */
  seekTo: (index: number) => void;
}

export interface UseSlideshowPlaybackOptions {
  /** Auto-advance interval in milliseconds (default: 1000ms) */
  interval?: number;
  /** Loop back to start when reaching the end (default: true) */
  loop?: boolean;
  /** Initial index (default: 0) */
  initialIndex?: number;
}

export function useSlideshowPlayback(
  totalFrames: number,
  options: UseSlideshowPlaybackOptions = {}
): SlideshowPlaybackState & SlideshowPlaybackActions {
  const { interval = 1000, loop = true, initialIndex = 0 } = options;

  const [currentIndex, setCurrentIndex] = useState(initialIndex);
  const [isPlaying, setIsPlaying] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Calculate progress
  const progress = totalFrames > 1 ? currentIndex / (totalFrames - 1) : 0;

  // Clamp index to valid range
  const clampIndex = useCallback(
    (index: number) => Math.max(0, Math.min(totalFrames - 1, index)),
    [totalFrames]
  );

  // Clear interval on unmount
  useEffect(() => {
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  // Reset index if totalFrames changes and current index is out of bounds
  useEffect(() => {
    if (currentIndex >= totalFrames) {
      setCurrentIndex(Math.max(0, totalFrames - 1));
    }
  }, [currentIndex, totalFrames]);

  // Auto-advance when playing
  useEffect(() => {
    if (!isPlaying || totalFrames <= 1) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
      return;
    }

    intervalRef.current = setInterval(() => {
      setCurrentIndex((prev) => {
        const next = prev + 1;
        if (next >= totalFrames) {
          if (loop) {
            return 0;
          } else {
            setIsPlaying(false);
            return prev;
          }
        }
        return next;
      });
    }, interval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [isPlaying, totalFrames, interval, loop]);

  const play = useCallback(() => {
    if (totalFrames <= 1) return;
    setIsPlaying(true);
  }, [totalFrames]);

  const pause = useCallback(() => {
    setIsPlaying(false);
  }, []);

  const toggle = useCallback(() => {
    if (isPlaying) {
      pause();
    } else {
      play();
    }
  }, [isPlaying, play, pause]);

  const next = useCallback(() => {
    setCurrentIndex((prev) => {
      const next = prev + 1;
      if (next >= totalFrames) {
        return loop ? 0 : prev;
      }
      return next;
    });
  }, [totalFrames, loop]);

  const previous = useCallback(() => {
    setCurrentIndex((prev) => {
      const next = prev - 1;
      if (next < 0) {
        return loop ? totalFrames - 1 : 0;
      }
      return next;
    });
  }, [totalFrames, loop]);

  const seekTo = useCallback(
    (index: number) => {
      setCurrentIndex(clampIndex(index));
    },
    [clampIndex]
  );

  return {
    currentIndex,
    isPlaying,
    totalFrames,
    progress,
    play,
    pause,
    toggle,
    next,
    previous,
    seekTo,
  };
}
