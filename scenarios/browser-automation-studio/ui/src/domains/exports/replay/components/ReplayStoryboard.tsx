/**
 * ReplayStoryboard component
 *
 * Renders the thumbnail grid navigation for browsing frames.
 * Supports multi-page workflows with page color indicators.
 */

import { useMemo } from 'react';
import clsx from 'clsx';
import type { ReplayFrame } from '../types';
import { clampDuration, formatDurationSeconds } from '../utils/formatting';

/** Page definition for storyboard color indicators. */
export interface StoryboardPage {
  id: string;
  title?: string;
  url?: string;
  isInitial?: boolean;
}

/** Color palette for page indicators (consistent colors per page). */
const PAGE_COLORS = [
  { bg: 'bg-blue-500', hex: '#3b82f6' },
  { bg: 'bg-emerald-500', hex: '#10b981' },
  { bg: 'bg-purple-500', hex: '#8b5cf6' },
  { bg: 'bg-orange-500', hex: '#f97316' },
  { bg: 'bg-pink-500', hex: '#ec4899' },
  { bg: 'bg-cyan-500', hex: '#06b6d4' },
  { bg: 'bg-yellow-500', hex: '#eab308' },
  { bg: 'bg-rose-500', hex: '#f43f5e' },
];

/** Get consistent color for a page based on its index. */
function getPageColor(pageIndex: number) {
  return PAGE_COLORS[pageIndex % PAGE_COLORS.length];
}

export interface ReplayStoryboardProps {
  frames: ReplayFrame[];
  currentIndex: number;
  aspectRatio: number;
  onSelectFrame: (index: number) => void;
  /** Optional pages for multi-page color indicators. */
  pages?: StoryboardPage[];
}

export function ReplayStoryboard({
  frames,
  currentIndex,
  aspectRatio,
  onSelectFrame,
  pages = [],
}: ReplayStoryboardProps) {
  // Build page lookup for color indicators
  const pageIndexMap = useMemo(() => {
    const map = new Map<string, number>();
    pages.forEach((page, index) => {
      map.set(page.id, index);
    });
    return map;
  }, [pages]);

  const hasMultiplePages = pages.length > 1;

  if (frames.length <= 1) {
    return null;
  }

  return (
    <div className="rounded-3xl border border-white/10 bg-slate-950/70 p-3">
      <div className="mb-2 flex items-center justify-between">
        <div className="text-xs uppercase tracking-[0.2em] text-slate-500">Storyboard</div>
        {hasMultiplePages && (
          <div className="flex items-center gap-2">
            {pages.map((page, index) => {
              const color = getPageColor(index);
              return (
                <div
                  key={page.id}
                  className="flex items-center gap-1 text-[10px] text-slate-400"
                  title={page.title || page.url}
                >
                  <span className={`w-2 h-2 rounded-full ${color.bg}`} />
                  <span>{page.isInitial ? 'main' : `tab ${index + 1}`}</span>
                </div>
              );
            })}
          </div>
        )}
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8">
        {frames.map((frame, index) => {
          const pageIndex = frame.pageId ? pageIndexMap.get(frame.pageId) : undefined;
          const pageColor = pageIndex !== undefined ? getPageColor(pageIndex) : null;

          return (
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
                <div
                  className={clsx('absolute inset-0 bg-black/40 transition-opacity duration-200', {
                    'opacity-0': currentIndex === index,
                    'opacity-70 group-hover:opacity-40': currentIndex !== index,
                  })}
                />
                {/* Page color indicator */}
                {hasMultiplePages && pageColor && pageIndex !== undefined && (
                  <span
                    className={`absolute top-1.5 left-1.5 w-2.5 h-2.5 rounded-full ${pageColor.bg} shadow-sm`}
                    title={pages[pageIndex]?.title || `Page ${pageIndex + 1}`}
                  />
                )}
              </div>
              <div className="px-3 py-2 text-[11px] text-slate-300">
                <div className="flex items-center gap-1.5">
                  {hasMultiplePages && pageColor && (
                    <span className={`flex-shrink-0 w-1.5 h-1.5 rounded-full ${pageColor.bg}`} />
                  )}
                  <span className="truncate font-medium text-slate-100">
                    {frame.nodeId || frame.stepType || `Step ${frame.stepIndex + 1}`}
                  </span>
                </div>
                <div className="text-[10px] uppercase tracking-[0.2em] text-slate-500">
                  {formatDurationSeconds(clampDuration(frame.durationMs))}
                </div>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
