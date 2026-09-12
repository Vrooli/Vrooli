/**
 * EtaExplainer — a shared "how was this computed" surface for the completion
 * ETA band shown on the Plan board and the Stats views.
 *
 * The band the UI receives is only a summary (p50/p80 + basis/confidence). The
 * canonical estimator (api/internal/eta) runs a Monte-Carlo simulation over the
 * remaining work and execute-lane capacity; this component makes that model and
 * its inputs legible so the number is trustable rather than magic.
 *
 * `EtaExplainerContent` is presentational. `EtaExplainerTrigger` wraps it in a
 * small "How is this estimated?" button + Popover for surfaces (e.g. Stats
 * cards) that don't already own a popover.
 */

import { useRef, useState, type ReactNode } from "react";
import { Info } from "lucide-react";
import { Popover } from "../ui/popover";
import { cn } from "../../lib/utils";

/** Normalized band shape — both the Plan (camelCase) and Stats (snake_case)
 *  payloads map onto this. */
export interface EtaExplainerBand {
  p50Label: string;
  p80Label: string;
  remainingItems: number;
  laneCapacity: number;
  /** live | backfill | priors | default | mixed */
  basis: string;
  /** e.g. "27 samples" or "priors only" */
  basisLabel: string;
  /** low | medium | high */
  confidence: string;
}

const BASIS_DESCRIPTIONS: Record<string, string> = {
  live: "Measured from recent run durations — the strongest signal.",
  backfill: "Reconstructed from historical event logs when live samples are thin.",
  priors: "Operator priors only — no durations have been recorded yet, so treat the band as a rough guide.",
  default: "Built-in default distribution — a conservative fallback.",
  mixed: "A mix of the above across effort classes (some classes have live samples, others fall back to priors).",
};

const CONFIDENCE_TONE: Record<string, string> = {
  high: "text-emerald-300",
  medium: "text-amber-300",
  low: "text-rose-300",
};

function InputCell({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="rounded bg-slate-950/70 p-2">
      <dt className="text-slate-500">{label}</dt>
      <dd className="font-medium text-slate-100">{value}</dd>
    </div>
  );
}

export function EtaExplainerContent({ band }: { band: EtaExplainerBand }) {
  const basisCopy = BASIS_DESCRIPTIONS[band.basis] ?? BASIS_DESCRIPTIONS.mixed;
  const confidenceTone = CONFIDENCE_TONE[band.confidence] ?? "text-slate-400";
  return (
    <div className="space-y-3" data-testid="eta-explainer">
      <p className="text-slate-400">
        The completion band comes from a Monte-Carlo simulation over the remaining
        work. Each trial draws a duration per pending item from its effort-class
        distribution, then takes
        {" "}
        <span className="font-medium text-slate-200">
          max(longest dependency chain, total work ÷ execute lanes)
        </span>
        . <span className="font-medium text-slate-200">p50</span> and{" "}
        <span className="font-medium text-slate-200">p80</span> are the 50th and
        80th percentiles across those trials.
      </p>
      <dl className="grid grid-cols-2 gap-2">
        <InputCell label="p50" value={band.p50Label} />
        <InputCell label="p80" value={band.p80Label} />
        <InputCell label="Remaining" value={`${band.remainingItems.toLocaleString()} items`} />
        <InputCell label="Execute lanes" value={band.laneCapacity} />
      </dl>
      <div className="rounded border border-slate-800 bg-slate-950/40 p-2">
        <p className="text-slate-300">
          Basis: <span className="font-medium text-slate-100">{band.basisLabel}</span>
          {" · "}
          <span className={cn("font-medium capitalize", confidenceTone)}>
            {band.confidence} confidence
          </span>
        </p>
        <p className="mt-1 text-slate-400">{basisCopy}</p>
      </div>
    </div>
  );
}

/** A compact trigger + popover for surfaces without their own popover. */
export function EtaExplainerTrigger({
  band,
  label = "How is this estimated?",
  testId,
}: {
  band: EtaExplainerBand;
  label?: string;
  testId?: string;
}) {
  const triggerRef = useRef<HTMLButtonElement | null>(null);
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        ref={triggerRef}
        type="button"
        onClick={() => setOpen((prev) => !prev)}
        aria-expanded={open}
        className="mt-2 inline-flex items-center gap-1 text-xs text-cyan-300 transition-colors hover:text-cyan-200"
        data-testid={testId}
      >
        <Info className="h-3 w-3" aria-hidden />
        {label}
      </button>
      <Popover
        isOpen={open}
        onClose={() => setOpen(false)}
        triggerRef={triggerRef}
        placement="bottom-start"
        className="w-72 p-3 text-xs text-slate-200"
        testId={testId ? `${testId}-popover` : undefined}
      >
        <h3 className="mb-2 text-sm font-semibold text-slate-100">How the ETA is computed</h3>
        <EtaExplainerContent band={band} />
      </Popover>
    </>
  );
}
