import { useState } from "react";
import { HelpCircle, X } from "lucide-react";
import { cn } from "../lib/utils";

interface ScoreLegendProps {
  className?: string;
}

const SCORE_RANGES = [
  {
    range: "90-100",
    name: "Production Ready",
    emoji: "🟢",
    color: "text-emerald-300",
    description: "Fully validated, documented, tested - ready for deployment",
  },
  {
    range: "80-89",
    name: "Nearly Ready",
    emoji: "🟢",
    color: "text-emerald-400",
    description: "Minor gaps in tests or documentation",
  },
  {
    range: "65-79",
    name: "Mostly Complete",
    emoji: "🟡",
    color: "text-amber-300",
    description: "Core functionality works but needs more coverage",
  },
  {
    range: "50-64",
    name: "Functional Incomplete",
    emoji: "🟡",
    color: "text-amber-400",
    description: "Basic features work, significant gaps remain",
  },
  {
    range: "25-49",
    name: "Foundation Laid",
    emoji: "🟠",
    color: "text-orange-400",
    description: "Structure exists but much work needed",
  },
  {
    range: "0-24",
    name: "Early Stage",
    emoji: "🔴",
    color: "text-red-400",
    description: "Just getting started",
  },
];

/**
 * A help button that shows a popover explaining score classifications.
 * Helps first-time users understand what the numbers mean.
 */
export function ScoreLegend({ className }: ScoreLegendProps) {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <div className={cn("relative", className)}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="inline-flex items-center gap-1 text-xs text-slate-500 hover:text-slate-300 transition-colors"
        title="What do the scores mean?"
        type="button"
        data-testid="score-legend-button"
      >
        <HelpCircle className="h-3.5 w-3.5" />
        <span>What do scores mean?</span>
      </button>

      {isOpen && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />

          {/* Popover */}
          <div
            className="absolute top-6 left-0 z-50 w-80 p-4 rounded-xl border border-white/10 bg-slate-900 shadow-xl"
            data-testid="score-legend-popover"
          >
            <div className="flex items-center justify-between mb-3">
              <h3 className="font-medium text-slate-200">Score Classifications</h3>
              <button
                onClick={() => setIsOpen(false)}
                className="p-1 rounded hover:bg-white/10"
                type="button"
              >
                <X className="h-4 w-4 text-slate-400" />
              </button>
            </div>

            <p className="text-xs text-slate-400 mb-3">
              Scores are calculated from 4 dimensions: Quality (50%), UI (25%),
              Coverage (15%), and Quantity (10%).
            </p>

            <div className="space-y-2">
              {SCORE_RANGES.map((range) => (
                <div
                  key={range.name}
                  className="flex items-start gap-2 text-xs"
                >
                  <span className="w-12 text-slate-500 shrink-0">
                    {range.range}
                  </span>
                  <span className="shrink-0">{range.emoji}</span>
                  <div>
                    <span className={cn("font-medium", range.color)}>
                      {range.name}
                    </span>
                    <p className="text-slate-500 mt-0.5">{range.description}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
