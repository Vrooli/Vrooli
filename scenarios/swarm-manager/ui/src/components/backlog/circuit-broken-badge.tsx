/**
 * CircuitBrokenBadge — Small inline indicator for circuit-broken backlog items.
 *
 * Shows a red "Circuit broken" pill when the item's circuit breaker is tripped.
 */

import { ShieldOff } from "lucide-react";
import { useGovernanceStore, isCircuitBroken } from "../../stores/governance-store";
import type { BacklogKind } from "../../types";

interface CircuitBrokenBadgeProps {
  backlogKind: BacklogKind;
  backlogName: string;
}

export function CircuitBrokenBadge({ backlogKind, backlogName }: CircuitBrokenBadgeProps) {
  const broken = useGovernanceStore((s) => isCircuitBroken(s, backlogKind, backlogName));
  if (!broken) return null;

  return (
    <span
      className="inline-flex items-center gap-1 rounded-full bg-red-500/15 px-1.5 py-0.5 text-[10px] font-medium text-red-400"
      title="Circuit breaker tripped — repeated failures. Reset via settings or CLI."
    >
      <ShieldOff className="h-3 w-3" />
      Circuit broken
    </span>
  );
}
