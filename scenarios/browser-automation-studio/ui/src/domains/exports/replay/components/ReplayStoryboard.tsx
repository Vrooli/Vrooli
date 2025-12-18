/**
 * ReplayStoryboard component
 *
 * Renders the thumbnail grid navigation for browsing frames.
 */

import clsx from 'clsx';
import type { ReplayFrame } from '../types';
import { clampDuration, formatDurationSeconds } from '../utils/formatting';

export interface ReplayStoryboardProps {
  frames: ReplayFrame[];
  currentIndex: number;
  aspectRatio: number;
  onSelectFrame: (index: number) => void;
}

export function ReplayStoryboard({
  frames,
  currentIndex,
  aspectRatio,
  onSelectFrame,
}: ReplayStoryboardProps) {
  if (frames.length <= 1) {
    return null;
  }

  return (
    <div className="rounded-3xl border border-white/10 bg-slate-950/70 p-3">
      <div className="mb-2 text-xs uppercase tracking-[0.2em] text-slate-500">Storyboard</div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8">
        {frames.map((frame, index) => (
          <button
            key={frame.id}
            className={clsx(
              'group relative overflow-hidden rounded-2xl border border-white/5 bg-white/5 text-left transition-all duration-200',
              currentIndex === index
                ? 'border-flow-accent shadow-[0_12px_40px_rgba(37,99,235,0.35)]'
                : 'hover:border-flow-accent/60 hover:shadow-[0_10px_30px_rgba(59,130,246,0.18)]'
            )}
            onClick={() => onSelectFrame(index)}
          >
            <div className="relative" style={{ paddingTop: `${aspectRatio}%` }}>
              <img
                src={frame.screenshot?.thumbnailUrl || frame.screenshot?.url}
                alt={`Timeline frame ${index + 1}`}
                loading="lazy"
                className="absolute inset-0 h-full w-full object-cover"
              />
              <div className={clsx('absolute inset-0 bg-black/40 transition-opacity duration-200', {
                'opacity-0': currentIndex === index,
                'opacity-70 group-hover:opacity-40': currentIndex !== index,
              })}
              />
            </div>
            <div className="px-3 py-2 text-[11px] text-slate-300">
              <div className="truncate font-medium text-slate-100">
                {frame.nodeId || frame.stepType || `Step ${frame.stepIndex + 1}`}
              </div>
              <div className="text-[10px] uppercase tracking-[0.2em] text-slate-500">
                {formatDurationSeconds(clampDuration(frame.durationMs))}
              </div>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
