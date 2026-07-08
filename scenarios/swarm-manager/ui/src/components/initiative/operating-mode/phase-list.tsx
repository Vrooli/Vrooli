/**
 * PhaseList
 *
 * Vertical stack of PhaseCards. Used in both list view (cards only) and graph
 * view (graph above the cards) on the Operating Mode Details Page. The page
 * owns selection state and propagates it via `highlightedPhaseId`.
 */

import type {
  OperatingModeCatalogPhase,
  OperatingModePhaseTransition,
} from "../../../types/operating-mode";
import { PhaseCard } from "./phase-card";

interface PhaseListProps {
  phases: OperatingModeCatalogPhase[];
  transitions?: OperatingModePhaseTransition[];
  highlightedPhaseId?: string | null;
}

export function PhaseList({ phases, transitions = [], highlightedPhaseId }: PhaseListProps) {
  if (phases.length === 0) {
    return <p className="text-sm italic text-slate-500">This mode has no phases.</p>;
  }
  return (
    <div className="space-y-3">
      {phases.map((phase) => (
        <PhaseCard
          key={phase.phase}
          phase={phase}
          transitions={transitions.filter((transition) => transition.from === phase.phase)}
          highlighted={highlightedPhaseId === phase.phase}
        />
      ))}
    </div>
  );
}
