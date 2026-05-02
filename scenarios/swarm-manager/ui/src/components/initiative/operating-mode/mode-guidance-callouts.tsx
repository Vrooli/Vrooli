/**
 * ModeGuidanceCallouts
 *
 * Three-column callouts rendering a mode's `bestFor`, `notFor`, and
 * `tradeoffs` decision metadata. Used both in the picker (selected card) and
 * on `OperatingModeDetailsPage`. Reads directly from the catalog entry; the
 * registry validator guarantees each list is non-empty.
 */

import { CheckCircle2, Scale, XCircle } from "lucide-react";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeCatalogEntry } from "../../../types/operating-mode";

export interface ModeGuidanceCalloutsProps {
  mode: OperatingModeCatalogEntry;
  /** Optional override for the testId. Defaults to the picker selector. */
  testId?: string;
}

export function ModeGuidanceCallouts({
  mode,
  testId = selectors.initiativeDetails.modePickerGuidanceCallouts,
}: ModeGuidanceCalloutsProps) {
  return (
    <div
      data-testid={testId}
      className="mt-3 grid grid-cols-1 gap-3 md:grid-cols-3"
    >
      <CalloutColumn
        title="Best for"
        items={mode.bestFor}
        icon={<CheckCircle2 className="h-3.5 w-3.5 text-emerald-400" aria-hidden="true" />}
        accent="text-emerald-300"
      />
      <CalloutColumn
        title="Not for"
        items={mode.notFor}
        icon={<XCircle className="h-3.5 w-3.5 text-amber-400" aria-hidden="true" />}
        accent="text-amber-300"
      />
      <CalloutColumn
        title="Tradeoffs"
        items={mode.tradeoffs}
        icon={<Scale className="h-3.5 w-3.5 text-cyan-400" aria-hidden="true" />}
        accent="text-cyan-300"
      />
    </div>
  );
}

function CalloutColumn({
  title,
  items,
  icon,
  accent,
}: {
  title: string;
  items: string[];
  icon: React.ReactNode;
  accent: string;
}) {
  return (
    <div className="rounded-md border border-slate-800/80 bg-slate-900/40 p-3">
      <p className={`text-[10px] font-semibold uppercase tracking-wide ${accent}`}>{title}</p>
      <ul className="mt-1.5 space-y-1">
        {items.map((entry, idx) => (
          <li key={`${title}-${idx}`} className="flex items-start gap-1.5 text-xs text-slate-300">
            <span className="mt-0.5">{icon}</span>
            <span>{entry}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
