/**
 * PlanHelpPanel - FloatingPanel explaining the Plan board's visual language.
 *
 * The board-side counterpart to GraphHelpPanel: same shell and placement, but
 * it documents what the board actually shows — the four columns, card status
 * dots, gate / effort / wave badges, outcome glyphs, and the ETA strip.
 *
 * Sections: columns, status dots, badges, ETA, how you act.
 */

import { useEffect, useRef } from "react";
import { Clock, HelpCircle, MousePointerClick, X } from "lucide-react";
import { cn } from "../../../lib/utils";
import { useSpatialNavContext } from "../../../hooks/SpatialNavContext";

interface PlanHelpPanelProps {
  isOpen: boolean;
  onClose: () => void;
}

const COLUMNS: { name: string; blurb: string }[] = [
  { name: "Now", blurb: "In-flight work — live agents, lane utilization, and the queue." },
  { name: "Next", blurb: "Actionable right now — ready to run or waiting on your answer." },
  { name: "Later", blurb: "Blocked work, grouped into dependency waves." },
  { name: "Done", blurb: "Recent outcomes in the selected time window." },
];

const DOTS: { tone: string; label: string; className: string }[] = [
  { tone: "active", label: "Active — running or ready to run", className: "bg-cyan-400" },
  { tone: "attention", label: "Attention — a gate or a question is waiting", className: "bg-amber-400" },
  { tone: "positive", label: "Succeeded", className: "bg-emerald-400" },
  { tone: "negative", label: "Failed", className: "bg-rose-400" },
  { tone: "neutral", label: "Pending / blocked", className: "bg-slate-500" },
];

const OUTCOMES: { glyph: string; label: string; className: string }[] = [
  { glyph: "✓", label: "ok", className: "text-emerald-400" },
  { glyph: "✗", label: "failed", className: "text-rose-400" },
  { glyph: "◆", label: "needs review", className: "text-amber-400" },
  { glyph: "⚠", label: "needs follow-up", className: "text-amber-400" },
];

export function PlanHelpPanel({ isOpen, onClose }: PlanHelpPanelProps) {
  // Push a spatial nav modal scope so D-pad navigation is trapped inside.
  const spatialNavRef = useSpatialNavContext();
  const panelRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const ctrl = spatialNavRef?.current;
    const el = panelRef.current;
    if (!isOpen || !ctrl || !el) return;
    ctrl.pushScope(el);
    return () => { ctrl.popScope(); };
  }, [isOpen, spatialNavRef]);

  if (!isOpen) return null;

  return (
    <div
      ref={panelRef}
      className="absolute right-14 top-14 z-40 max-h-[70vh] w-80 overflow-y-auto rounded-lg border border-slate-700/60 bg-slate-900/95 shadow-xl backdrop-blur-sm"
      data-testid="plan-help-panel"
    >
      {/* Header */}
      <div className="sticky top-0 z-10 flex items-center justify-between border-b border-slate-700/60 bg-slate-900/95 px-3 py-2">
        <div className="flex items-center gap-2">
          <HelpCircle className="h-4 w-4 text-slate-400" />
          <span className="text-sm font-semibold text-slate-100">Plan Guide</span>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="rounded p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
          aria-label="Close help"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      <div className="space-y-4 p-3">
        {/* Columns */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Columns</h3>
          <p className="mb-2 text-[11px] text-slate-500">
            Derived from dependency waves and gates — not drag-and-drop.
          </p>
          <div className="space-y-1.5">
            {COLUMNS.map((col) => (
              <div key={col.name} className="text-[11px] text-slate-400">
                <strong className="text-slate-200">{col.name}</strong> — {col.blurb}
              </div>
            ))}
          </div>
        </section>

        {/* Status dots */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Card status dot</h3>
          <div className="space-y-1">
            {DOTS.map((dot) => (
              <div key={dot.tone} className="flex items-center gap-2">
                <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", dot.className)} />
                <span className="text-[11px] text-slate-300">{dot.label}</span>
              </div>
            ))}
          </div>
        </section>

        {/* Badges */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Badges</h3>
          <div className="space-y-1.5 text-[11px] text-slate-400">
            <div className="flex items-center gap-2">
              <span className="rounded bg-amber-500/15 px-1.5 py-0.5 text-[9px] font-medium uppercase tracking-wide text-amber-300">
                gate
              </span>
              <span>A gate blocks the card — a decision or review you must clear.</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded bg-indigo-500/15 px-1.5 py-0.5 text-[9px] font-medium uppercase tracking-wide text-indigo-300">
                M
              </span>
              <span>Effort estimate (S / M / L).</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded bg-slate-700/60 px-1.5 py-0.5 text-[9px] font-medium uppercase tracking-wide text-slate-400">
                w3
              </span>
              <span>Wave — how many dependency hops from actionable. Not clock time.</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="rounded bg-rose-500/15 px-1.5 py-0.5 text-[9px] font-medium uppercase tracking-wide text-rose-300">
                cycle
              </span>
              <span>Trapped in a dependency loop — can&apos;t become ready until it&apos;s broken.</span>
            </div>
          </div>
          <div className="mt-2 flex flex-wrap gap-x-3 gap-y-1">
            {OUTCOMES.map((o) => (
              <span key={o.label} className="text-[11px] text-slate-400">
                <span className={cn("font-mono", o.className)}>{o.glyph}</span> {o.label}
              </span>
            ))}
          </div>
        </section>

        {/* ETA */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">ETA strip</h3>
          <div className="flex items-start gap-2 text-[11px] text-slate-400">
            <Clock className="mt-0.5 h-3 w-3 shrink-0 text-slate-500" />
            <span>
              A p50–p80 completion band for the remaining work. The basis label is colour-coded by
              confidence —
              <span className="text-emerald-300"> emerald</span> (many samples),
              <span className="text-amber-300"> amber</span> (some), and
              <span className="text-slate-400"> slate</span> (priors only). Hidden when nothing is left
              to estimate.
            </span>
          </div>
        </section>

        {/* How you act */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">How you act</h3>
          <div className="space-y-1.5 text-[11px] text-slate-400">
            <div className="flex items-start gap-2">
              <MousePointerClick className="mt-0.5 h-3 w-3 shrink-0 text-slate-500" />
              <span>
                Cards act through explicit levers — <strong className="text-slate-300">Run</strong>,{" "}
                <strong className="text-slate-300">Answer</strong>, and{" "}
                <strong className="text-slate-300">Snooze</strong> — never by dragging.
              </span>
            </div>
            <p>
              The <strong className="text-slate-300">goal picker</strong> scopes the whole board — cards,
              waves, and ETA — to a goal&apos;s transitive closure.
            </p>
          </div>
        </section>
      </div>
    </div>
  );
}
