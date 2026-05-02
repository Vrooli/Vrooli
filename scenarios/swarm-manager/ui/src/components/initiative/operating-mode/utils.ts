import { formatDisplayText } from "../../../lib/format-utils";
import type { OperatingModeRound } from "../../../types/operating-mode";

export function activeRound(rounds: OperatingModeRound[]): OperatingModeRound | undefined {
  return rounds.find((round) => round.status === "reserved" || round.status === "agent_running");
}

export function modeLabel(mode: string, label?: string): string {
  return label?.trim() || formatDisplayText(mode.replace(/-/g, " "));
}

export function phaseLabel(phase: string): string {
  return formatDisplayText(phase.replace(/_/g, " "));
}

// Stable DOM id used by the operating-mode details page to anchor scroll +
// highlight when a phase is selected from the graph view.
export function phaseCardDomId(phase: string): string {
  return `phase-row-${phase}`;
}

// humanizeScopeKind / humanizeRunStrategy use explicit switches over the known
// enum values from api/internal/operatingmode/registry.go. If a new value is
// added server-side and not mirrored here, the unknown branch surfaces the raw
// token so the page does not silently render garbage.
export function humanizeScopeKind(kind: string): string {
  switch (kind) {
    case "backlog_item":
      return "Backlog item";
    case "initiative":
      return "Initiative";
    default:
      return kind || "—";
  }
}

export function humanizeRunStrategy(strategy: string): string {
  switch (strategy) {
    case "existing_item_flow":
      return "Existing item flow";
    case "single_phase_run":
      return "Single phase run";
    case "sequential_handoff":
      return "Sequential handoff";
    case "operator_gated_loop":
      return "Operator-gated loop";
    default:
      return strategy || "—";
  }
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
