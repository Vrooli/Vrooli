/**
 * GraphHelpPanel - guide explaining the graph's visual language, rendered in
 * the shared HelpPanel shell (popover anchored to the header's help button on
 * desktop, bottom sheet on mobile).
 *
 * Sections: node shapes, status colors, edge types, interactions, lenses.
 */

import type { RefObject } from "react";
import { Mouse, MousePointerClick } from "lucide-react";
import { ENTITY_SHAPE_INFO } from "../lib/entity-shapes";
import { STATUS_GROUP_INFO } from "../lib/status-colors";
import { EDGE_STYLES } from "../lib/edge-styles";
import { cn } from "../../../lib/utils";
import { HelpPanel } from "../../../components/ui/help-panel";

interface GraphHelpPanelProps {
  isOpen: boolean;
  onClose: () => void;
  /** Header help button the desktop popover anchors to */
  triggerRef?: RefObject<HTMLElement | null>;
}

function ShapePreview({ cssClass, clipPath }: { cssClass: string; clipPath: string | null }) {
  return (
    <div
      className={cn(
        "w-8 h-4 border-2 border-slate-400/70 bg-slate-500/20 shrink-0",
        cssClass,
      )}
      style={clipPath ? { clipPath: `polygon(${clipPath})` } : undefined}
    />
  );
}

export function GraphHelpPanel({ isOpen, onClose, triggerRef }: GraphHelpPanelProps) {
  return (
    <HelpPanel
      isOpen={isOpen}
      onClose={onClose}
      title="Graph Guide"
      triggerRef={triggerRef}
      testId="graph-help-panel"
    >
      <div className="space-y-4">
        {/* Node Shapes */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Node Shapes</h3>
          <p className="mb-2 text-[11px] text-slate-500">Each entity type has a unique shape.</p>
          <div className="grid grid-cols-2 gap-1.5">
            {ENTITY_SHAPE_INFO.map((info) => (
              <div key={info.entityType} className="flex items-center gap-2 rounded px-1.5 py-1">
                <ShapePreview cssClass={info.cssClass} clipPath={info.clipPath} />
                <span className="text-[11px] text-slate-300">{info.label}</span>
              </div>
            ))}
          </div>
        </section>

        {/* Status Colors */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Status Colors</h3>
          <p className="mb-2 text-[11px] text-slate-500">Both fill and border encode the node&apos;s status.</p>
          <div className="space-y-1">
            {STATUS_GROUP_INFO.map((info) => (
              <div key={info.group} className="flex items-center gap-2">
                <div className={cn("w-5 h-3 rounded-sm border", info.classes.background, info.classes.border)} />
                <span className="text-[11px] font-medium text-slate-300">{info.label}</span>
                <span className="text-[10px] text-slate-500">{info.exampleStatuses.join(", ")}</span>
              </div>
            ))}
          </div>
          <div className="mt-2 space-y-1 text-[11px] text-slate-400">
            <div className="flex items-center gap-2">
              <div className="h-2.5 w-2.5 shrink-0 rounded-full bg-cyan-400" />
              <span><strong className="text-slate-300">Top-right dot</strong> — active execution (pulsing = running)</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-2.5 w-2.5 shrink-0 rounded-full border-2 border-slate-200 bg-slate-300/60" />
              <span><strong className="text-slate-300">Top-left dot</strong> — actionable (appears in Operations)</span>
            </div>
          </div>
        </section>

        {/* Edge Types */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Edge Types</h3>
          <div className="space-y-1">
            {Object.entries(EDGE_STYLES).map(([type, config]) => (
              <div key={type} className="flex items-center gap-2">
                <svg width="24" height="8" className="shrink-0">
                  <line
                    x1="0" y1="4" x2="24" y2="4"
                    stroke={config.stroke}
                    strokeWidth="2.5"
                    strokeDasharray={config.strokeDasharray === "none" ? undefined : config.strokeDasharray}
                  />
                </svg>
                <span className="text-[11px] text-slate-300">{config.label}</span>
              </div>
            ))}
          </div>
        </section>

        {/* Interactions */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Interactions</h3>
          <div className="space-y-1.5 text-[11px] text-slate-400">
            <div className="flex items-start gap-2">
              <MousePointerClick className="mt-0.5 h-3 w-3 shrink-0 text-slate-500" />
              <span><strong className="text-slate-300">Click</strong> a node to highlight its neighborhood</span>
            </div>
            <div className="flex items-start gap-2">
              <Mouse className="mt-0.5 h-3 w-3 shrink-0 text-slate-500" />
              <span><strong className="text-slate-300">Double-click</strong> a node to view its details</span>
            </div>
            <div className="flex items-start gap-2">
              <span className="mt-0.5 shrink-0 rounded border border-slate-600 px-1 text-[9px] text-slate-500">1</span>
              <span><strong className="text-slate-300">1 / 2 / 3</strong> to switch lenses</span>
            </div>
            <div className="flex items-start gap-2">
              <span className="mt-0.5 shrink-0 rounded border border-slate-600 px-1 text-[9px] text-slate-500">Esc</span>
              <span><strong className="text-slate-300">Escape</strong> to close panels</span>
            </div>
          </div>
        </section>

        {/* Lenses */}
        <section>
          <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-slate-400">Lenses</h3>
          <div className="space-y-1.5 text-[11px] text-slate-400">
            <p><strong className="text-slate-300">Plan</strong> — Now / Next / Later / Done board: what is running, what is actionable, and what each blocker unblocks</p>
            <p><strong className="text-slate-300">Focus</strong> — Attention-filtered graph neighborhood: items needing input plus their structural context</p>
          </div>
        </section>
      </div>
    </HelpPanel>
  );
}
