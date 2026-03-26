/**
 * Clickable readiness dots with popover detail.
 *
 * Renders 5 small labeled circles (P, S, A, T, R) colored by readiness score.
 * Clicking opens a popover with full dimension names, scores, and delta arrows
 * compared to a previous round.
 *
 * Score-0 uses an outline ring instead of a filled dot so it reads as
 * "intentionally scored zero" rather than "missing data".
 */
import { useState, useRef, useCallback } from "react";
import type { KeyboardEvent, MouseEvent } from "react";
import { cn } from "../../lib";
import {
  READINESS_DIMENSIONS,
  DIMENSION_LABELS,
  DIMENSION_SHORT_LABELS,
} from "../../lib/maturity";
import { useModalBehavior } from "../../hooks/useModalBehavior";
import type { ReadinessDimension, WorkshopRound } from "../../types/domain";

interface ReadinessDotsProps {
  /** The round whose readiness to display */
  round: WorkshopRound;
  /** Optional previous round for delta comparison */
  prevRound?: WorkshopRound | null;
  className?: string;
}

/** Filled-dot classes for scores 1-3 */
const FILLED_CLASSES: Record<number, string> = {
  1: "bg-rose-500 text-white",
  2: "bg-amber-500 text-white",
  3: "bg-emerald-500 text-white",
};

/** Score-0 gets a ring outline instead of a fill */
const ZERO_CLASS = "ring-1 ring-slate-500 text-slate-500";

/** Popover row score badge */
const BADGE_CLASSES: Record<number, string> = {
  0: "bg-slate-700/50 text-slate-400",
  1: "bg-rose-500/20 text-rose-400",
  2: "bg-amber-500/20 text-amber-400",
  3: "bg-emerald-500/20 text-emerald-400",
};

const SCORE_LABELS: Record<number, string> = {
  0: "Not started",
  1: "Rough",
  2: "Draft",
  3: "Solid",
};

function deltaIcon(current: number, previous: number): string | null {
  if (current > previous) return "\u25B2"; // ▲
  if (current < previous) return "\u25BC"; // ▼
  return null; // same
}

function deltaColor(current: number, previous: number): string {
  if (current > previous) return "text-emerald-400";
  if (current < previous) return "text-rose-400";
  return "text-slate-500";
}

export function ReadinessDots({ round, prevRound, className }: ReadinessDotsProps) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const popoverRef = useRef<HTMLDivElement>(null);

  useModalBehavior({
    isOpen,
    onClose: () => setIsOpen(false),
    ref: popoverRef,
    delayClickOutside: true,
  });

  const handleClick = useCallback((e: MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    setIsOpen((prev) => !prev);
  }, []);

  const handleKeyDown = useCallback((e: KeyboardEvent<HTMLElement>) => {
    if (e.key !== "Enter" && e.key !== " ") {
      return;
    }
    e.preventDefault();
    e.stopPropagation();
    setIsOpen((prev) => !prev);
  }, []);

  return (
    <div ref={containerRef} className={cn("relative inline-flex", className)}>
      {/* Dot row */}
      <div
        role="button"
        tabIndex={0}
        onClick={handleClick}
        onKeyDown={handleKeyDown}
        className="flex gap-0.5 rounded px-0.5 py-0.5 hover:bg-slate-700/50 transition-colors"
        aria-label="View readiness details"
        data-testid="readiness-dots-trigger"
      >
        {READINESS_DIMENSIONS.map((dim) => {
          const score = round.readiness?.[dim] ?? 0;
          return (
            <div
              key={dim}
              className={cn(
                "flex h-4 w-4 items-center justify-center rounded-full text-[8px] font-bold leading-none",
                score === 0 ? ZERO_CLASS : FILLED_CLASSES[score],
              )}
            >
              {DIMENSION_SHORT_LABELS[dim]}
            </div>
          );
        })}
      </div>

      {/* Popover */}
      {isOpen && (
        <div
          ref={popoverRef}
          className={cn(
            "absolute left-0 top-full z-50 mt-1 w-64 rounded-md",
            "border border-slate-600 bg-slate-900 shadow-lg",
            "animate-in fade-in-0 zoom-in-95 duration-100",
          )}
          data-testid="readiness-popover"
        >
          <div className="px-3 py-2 border-b border-slate-700">
            <p className="text-xs font-medium text-slate-300">
              Readiness — Round {round.round}
            </p>
          </div>
          <div className="px-3 py-2 space-y-1.5">
            {READINESS_DIMENSIONS.map((dim) => {
              const score = round.readiness?.[dim] ?? 0;
              const prevScore = prevRound?.readiness?.[dim];
              const hasPrev = prevScore !== undefined && prevScore !== null;
              const arrow = hasPrev ? deltaIcon(score, prevScore) : null;
              const arrowColor = hasPrev ? deltaColor(score, prevScore) : "";

              return (
                <DimensionRow
                  key={dim}
                  dim={dim}
                  score={score}
                  arrow={arrow}
                  arrowColor={arrowColor}
                />
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

interface DimensionRowProps {
  dim: ReadinessDimension;
  score: number;
  arrow: string | null;
  arrowColor: string;
}

function DimensionRow({ dim, score, arrow, arrowColor }: DimensionRowProps) {
  return (
    <div className="flex items-center justify-between gap-2" data-testid={`readiness-row-${dim}`}>
      <span className="text-xs text-slate-400 truncate">{DIMENSION_LABELS[dim]}</span>
      <div className="flex items-center gap-1.5 shrink-0">
        {arrow && (
          <span className={cn("text-[10px]", arrowColor)} data-testid={`readiness-delta-${dim}`}>
            {arrow}
          </span>
        )}
        <span className={cn(
          "rounded px-1.5 py-0.5 text-[10px] font-medium",
          BADGE_CLASSES[score] ?? BADGE_CLASSES[0],
        )}>
          {score}/3
        </span>
        <span className="text-[10px] text-slate-500 w-14 text-right">{SCORE_LABELS[score]}</span>
      </div>
    </div>
  );
}
