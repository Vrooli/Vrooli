import { formatDisplayText } from "../../../lib/format-utils";
import type { OperatingModeRound } from "../../../types/operating-mode";
import type { InitiativeOperatingMode } from "../../../types";

export const OPERATING_MODES: Array<{ value: InitiativeOperatingMode; label: string }> = [
  { value: "item-level", label: "Item Level" },
  { value: "holistic-loop", label: "Holistic Loop" },
  { value: "phased-plan-drain", label: "Phased Plan Drain" },
];

export function activeRound(rounds: OperatingModeRound[]): OperatingModeRound | undefined {
  return rounds.find((round) => round.status === "reserved" || round.status === "agent_running");
}

export function phaseLabel(phase: string): string {
  return formatDisplayText(phase.replace(/_/g, " "));
}

export function statusClasses(status: string): string {
  switch (status) {
    case "completed":
      return "border-emerald-500/30 bg-emerald-500/10 text-emerald-300";
    case "failed":
      return "border-red-500/30 bg-red-500/10 text-red-300";
    case "canceled":
      return "border-slate-600 bg-slate-800/70 text-slate-300";
    case "agent_running":
      return "border-cyan-500/30 bg-cyan-500/10 text-cyan-300";
    default:
      return "border-amber-500/30 bg-amber-500/10 text-amber-300";
  }
}
