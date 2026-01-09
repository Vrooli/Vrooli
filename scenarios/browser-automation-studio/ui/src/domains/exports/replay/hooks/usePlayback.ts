/**
 * usePlayback hook
 *
 * Manages playback state machine for the replay player including:
 * - Current frame index tracking
 * - Play/pause state
 * - Frame progress (0-1 animation progress within current frame)
 * - Playback phase (intro/frames/outro)
 * - Animation frame loop for auto-progression
 */

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import type { ReplayFrame, IntroCardSettings, OutroCardSettings } from '../types';
import { clampDuration } from '../utils/formatting';
import { DEFAULT_DURATION } from '../constants';

export type PlaybackPhase = 'intro' | 'frames' | 'outro';

export interface UsePlaybackOptions {
  frames: ReplayFrame[];
  autoPlay?: boolean;
  loop?: boolean;
  introCard?: IntroCardSettings;
  outroCard?: OutroCardSettings;
  isExternallyControlled?: boolean;
}

export interface UsePlaybackResult {
  currentIndex: number;
  setCurrentIndex: React.Dispatch<React.SetStateAction<number>>;
  isPlaying: boolean;
  setIsPlaying: React.Dispatch<React.SetStateAction<boolean>>;
  frameProgress: number;
  setFrameProgress: React.Dispatch<React.SetStateAction<number>>;
  playbackPhase: PlaybackPhase;
  setPlaybackPhase: React.Dispatch<React.SetStateAction<PlaybackPhase>>;
  seekToFrame: (frameIndex: number, progress?: number) => void;
  handleNext: () => void;
  handlePrevious: () => void;
  changeFrame: (index: number, progress?: number) => void;
  normalizedFramesRef: React.MutableRefObject<ReplayFrame[]>;
}

export function usePlayback({
  frames,
  autoPlay = true,
  loop = true,
  introCard,
  outroCard,
  isExternallyControlled = false,
}: UsePlaybackOptions): UsePlaybackResult {
  const normalizedFrames = useMemo(() => {
    return frames
      .filter((frame): frame is ReplayFrame => Boolean(frame))
      .map((frame, index) => ({
        ...frame,
        id: frame.id || `${index}`,
      }));
  }, [frames]);

  const hasIntroCard = Boolean(introCard?.enabled);
  const hasOutroCard = Boolean(outroCard?.enabled);

  const getInitialPhase = (): PlaybackPhase => {
    if (hasIntroCard) return 'intro';
    return 'frames';
  };

  const [currentIndex, setCurrentIndex] = useState(0);
  const [isPlaying, setIsPlaying] = useState(
    () => !isExternallyControlled && autoPlay && normalizedFrames.length > 1,
  );
  const [frameProgress, setFrameProgress] = useState(0);
  const [playbackPhase, setPlaybackPhase] = useState<PlaybackPhase>(getInitialPhase);
  const [_cardProgress, setCardProgress] = useState(0);
  void _cardProgress; // Suppress unused variable warning

  const rafRef = useRef<number | null>(null);
  const durationRef = useRef<number>(DEFAULT_DURATION);
  const normalizedFramesRef = useRef(normalizedFrames);

  useEffect(() => {
    normalizedFramesRef.current = normalizedFrames;
  }, [normalizedFrames]);

  // Reset state when frames change or external control changes
  useEffect(() => {
    if (isExternallyControlled) {
      setCurrentIndex(0);
      setFrameProgress(0);
      setCardProgress(0);
      setPlaybackPhase(hasIntroCard ? 'intro' : 'frames');
      setIsPlaying(false);
      return;
    }
    setCurrentIndex(0);
    setPlaybackPhase(hasIntroCard ? 'intro' : 'frames');
    setCardProgress(0);
    setIsPlaying(autoPlay && normalizedFrames.length > 1);
  }, [autoPlay, normalizedFrames.length, isExternallyControlled, hasIntroCard]);

  // Main playback animation loop
  useEffect(() => {
    if (!isPlaying || normalizedFrames.length <= 1 || isExternallyControlled) {
      setFrameProgress(0);
      setCardProgress(0);
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
      return;
    }

    const playPhase = () => {
      let phaseDuration: number;

      if (playbackPhase === 'intro' && introCard) {
        phaseDuration = introCard.duration;
        setCardProgress(0);
      } else if (playbackPhase === 'outro' && outroCard) {
        phaseDuration = outroCard.duration;
        setCardProgress(0);
      } else {
        // Frames phase
        const frame = normalizedFrames[currentIndex];
        const frameDuration = frame?.totalDurationMs ?? frame?.durationMs;
        phaseDuration = clampDuration(frameDuration);
        setFrameProgress(0);
      }

      durationRef.current = phaseDuration;
      const start = performance.now();

      const step = (now: number) => {
        const elapsed = now - start;
        const progress = Math.min(1, elapsed / durationRef.current);

        if (playbackPhase === 'intro' || playbackPhase === 'outro') {
          setCardProgress(progress);
        } else {
          setFrameProgress(progress);
        }

        if (elapsed >= durationRef.current) {
          // Phase complete, determine next phase
          if (playbackPhase === 'intro') {
            setPlaybackPhase('frames');
            setCardProgress(0);
          } else if (playbackPhase === 'frames') {
            const atLastFrame = currentIndex >= normalizedFrames.length - 1;
            if (atLastFrame) {
              if (hasOutroCard) {
                setPlaybackPhase('outro');
              } else if (loop) {
                setCurrentIndex(0);
                if (hasIntroCard) {
                  setPlaybackPhase('intro');
                }
              } else {
                setIsPlaying(false);
              }
            } else {
              setCurrentIndex((prev) => Math.min(prev + 1, normalizedFrames.length - 1));
            }
          } else if (playbackPhase === 'outro') {
            if (loop) {
              setCurrentIndex(0);
              if (hasIntroCard) {
                setPlaybackPhase('intro');
              } else {
                setPlaybackPhase('frames');
              }
              setCardProgress(0);
            } else {
              setIsPlaying(false);
            }
          }
          return;
        }
        rafRef.current = requestAnimationFrame(step);
      };

      rafRef.current = requestAnimationFrame(step);
    };

    playPhase();

    return () => {
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, [
    isPlaying,
    currentIndex,
    normalizedFrames,
    loop,
    isExternallyControlled,
    playbackPhase,
    hasIntroCard,
    hasOutroCard,
    introCard,
    outroCard,
  ]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (rafRef.current) {
        cancelAnimationFrame(rafRef.current);
      }
    };
  }, []);

  const seekToFrame = useCallback((frameIndex: number, progress: number | undefined) => {
    const framesRef = normalizedFramesRef.current;
    if (framesRef.length === 0) {
      return;
    }
    const clampedIndex = Math.min(Math.max(frameIndex, 0), framesRef.length - 1);
    const clampedProgress = Number.isFinite(progress) ? Math.min(Math.max(progress ?? 0, 0), 1) : 0;
    setIsPlaying(false);
    setCurrentIndex(clampedIndex);
    setFrameProgress(clampedProgress);
  }, []);

  const changeFrame = useCallback(
    (index: number, progress?: number) => {
      seekToFrame(index, progress);
    },
    [seekToFrame],
  );

  const handleNext = useCallback(() => {
    const framesRef = normalizedFramesRef.current;
    if (framesRef.length === 0) {
      return;
    }
    if (currentIndex >= framesRef.length - 1) {
      if (loop) {
        changeFrame(0);
      }
      return;
    }
    changeFrame(currentIndex + 1);
  }, [changeFrame, currentIndex, loop]);

  const handlePrevious = useCallback(() => {
    const framesRef = normalizedFramesRef.current;
    if (framesRef.length === 0) {
      return;
    }
    if (currentIndex === 0) {
      if (loop) {
        changeFrame(framesRef.length - 1);
      }
      return;
    }
    changeFrame(currentIndex - 1);
  }, [changeFrame, currentIndex, loop]);

  return {
    currentIndex,
    setCurrentIndex,
    isPlaying,
    setIsPlaying,
    frameProgress,
    setFrameProgress,
    playbackPhase,
    setPlaybackPhase,
    seekToFrame,
    handleNext,
    handlePrevious,
    changeFrame,
    normalizedFramesRef,
  };
}
