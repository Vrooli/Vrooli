import { ArrowRightLeft, Workflow } from "lucide-react";
import { Button } from "../../ui/button";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCatalogEntry,
  OperatingModeRound,
} from "../../../types/operating-mode";
import type { InitiativeOperatingMode } from "../../../types";
import { humanizeRunStrategy, humanizeScopeKind, modeLabel } from "./utils";

export interface OperatingModeHeroProps {
  currentMode: InitiativeOperatingMode;
  catalogEntry?: OperatingModeCatalogEntry;
  runningRound?: OperatingModeRound;
  onSwitchClick: () => void;
}

export function OperatingModeHero({
  currentMode,
  catalogEntry,
  runningRound,
  onSwitchClick,
}: OperatingModeHeroProps) {
  const label = modeLabel(currentMode, catalogEntry?.label);

  return (
    <div
      className="rounded-xl border border-white/10 bg-slate-800/30 p-4"
      data-testid={selectors.initiativeDetails.modeHero}
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <Workflow className="h-4 w-4 text-cyan-400" aria-hidden="true" />
            <h2
              className="text-base font-semibold text-slate-100"
              data-testid={selectors.initiativeDetails.modeHeroLabel}
            >
              {label}
            </h2>
            <code className="rounded bg-slate-900/70 px-1.5 py-0.5 text-[11px] text-slate-400">
              {currentMode}
            </code>
            {runningRound && (
              <span className="rounded-full border border-cyan-500/40 bg-cyan-500/10 px-2.5 py-0.5 text-[11px] font-medium text-cyan-200">
                Round {runningRound.round} running
              </span>
            )}
          </div>
          {catalogEntry && (
            <div className="mt-2 flex flex-wrap items-center gap-1.5 text-[11px]">
              <Chip>Scope: {humanizeScopeKind(catalogEntry.scopeKind)}</Chip>
              <Chip>Run strategy: {humanizeRunStrategy(catalogEntry.runStrategy)}</Chip>
              {catalogEntry.default && <Chip>Default</Chip>}
            </div>
          )}
          {catalogEntry?.description && (
            <p className="mt-2 line-clamp-3 text-sm text-slate-400">{catalogEntry.description}</p>
          )}
        </div>
        <Button
          size="sm"
          onClick={onSwitchClick}
          data-testid={selectors.initiativeDetails.modeHeroSwitchButton}
        >
          <ArrowRightLeft className="mr-1.5 h-4 w-4" />
          Switch Mode
        </Button>
      </div>
    </div>
  );
}

function Chip({ children }: { children: React.ReactNode }) {
  return (
    <span className="rounded-full border border-slate-700/80 bg-slate-900/60 px-2 py-0.5 text-slate-300">
      {children}
    </span>
  );
}
