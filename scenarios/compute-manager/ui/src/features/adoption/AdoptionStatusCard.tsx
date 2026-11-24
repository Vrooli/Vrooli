import type { AdoptionViewModel } from "./model";
import type { AdoptionState } from "./model";
import { selectors } from "../../consts/selectors";
import { formatMinorMoney } from "./formatMoney";

export interface AdoptionStatusCardCopy {
  stateLabel: string;
  states: Record<AdoptionState, string>;
  host: string;
  user: string;
  monthlyCost: string;
  taxBasis: string;
  releaseSemantics: string;
  verificationPending: string;
  minorUnitsLabel: string;
  locale?: string;
}

export function AdoptionStatusCard({ adoption, copy }: { adoption: AdoptionViewModel; copy: AdoptionStatusCardCopy }) {
  const isReady = adoption.state === "ready";
  return (
    <article
      aria-labelledby={`adoption-${adoption.id}-heading`}
      className="grid max-w-2xl gap-space-sm rounded-card border border-border bg-surface p-space-md"
      data-testid={selectors.adoption.statusCard}
    >
      <header>
        <h2 id={`adoption-${adoption.id}-heading`} className="text-lg font-semibold">
          {adoption.displayName || adoption.host}
        </h2>
        <p role={isReady ? "status" : "group"} aria-label={copy.stateLabel} data-testid={selectors.adoption.status} className="text-muted">
          {copy.states[adoption.state]}
        </p>
      </header>
      <dl className="grid grid-cols-[minmax(0,1fr)_auto] gap-x-4 gap-y-1 text-sm">
        <dt>{copy.host}</dt><dd>{adoption.host}</dd>
        <dt>{copy.user}</dt><dd>{adoption.user}</dd>
        <dt>{copy.monthlyCost}</dt><dd>{formatMinorMoney(adoption.monthlyCostMinor, adoption.currency, copy.locale, copy.minorUnitsLabel)}</dd>
        <dt>{copy.taxBasis}</dt><dd>{adoption.taxBasis}</dd>
      </dl>
      {adoption.statusReason === "verification_pending_without_evidence" && <p role="status">{copy.verificationPending}</p>}
      <p className="text-sm text-muted">{copy.releaseSemantics}</p>
    </article>
  );
}
