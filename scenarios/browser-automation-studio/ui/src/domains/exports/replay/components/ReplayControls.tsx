/**
 * ReplayControls component
 *
 * Renders play/pause, previous/next buttons, and progress bar
 * for the replay player.
 */

import { ChevronLeft, ChevronRight, Pause, Play } from 'lucide-react';
import { formatDurationSeconds } from '../utils/formatting';

export interface ReplayControlsProps {
  isPlaying: boolean;
  onPlayPause: () => void;
  onPrevious: () => void;
  onNext: () => void;
  frameProgress: number;
  displayDurationMs: number | null | undefined;
}

export function ReplayControls({
  isPlaying,
  onPlayPause,
  onPrevious,
  onNext,
  frameProgress,
  displayDurationMs,
}: ReplayControlsProps) {
  return (
    <div className="flex items-center gap-4">
      <button
        className="flex h-9 w-9 items-center justify-center rounded-full border border-white/10 bg-white/5 text-white hover:bg-white/10"
        onClick={onPrevious}
        aria-label="Previous frame"
      >
        <ChevronLeft size={16} />
      </button>

      <button
        className="flex h-9 w-9 items-center justify-center rounded-full border border-white/10 bg-flow-accent text-white hover:bg-blue-500"
        onClick={onPlayPause}
        aria-label={isPlaying ? 'Pause replay' : 'Play replay'}
      >
        {isPlaying ? <Pause size={16} /> : <Play size={16} />}
      </button>

      <button
        className="flex h-9 w-9 items-center justify-center rounded-full border border-white/10 bg-white/5 text-white hover:bg-white/10"
        onClick={onNext}
        aria-label="Next frame"
      >
        <ChevronRight size={16} />
      </button>

      <div className="flex-1">
        <div className="relative h-1 overflow-hidden rounded-full bg-white/10">
          <div
            className="absolute inset-y-0 left-0 rounded-full bg-flow-accent transition-[width] duration-75"
            style={{ width: `${frameProgress * 100}%` }}
          />
        </div>
      </div>

      <div className="text-xs uppercase tracking-[0.2em] text-slate-200/80">
        {displayDurationMs ? formatDurationSeconds(displayDurationMs) : 'Auto'}
      </div>
    </div>
  );
}
