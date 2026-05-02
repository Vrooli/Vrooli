/**
 * CapabilityList
 *
 * Canonical capability-rendering component for an operating mode. Used in
 * both the details page (`variant="full"`) and the picker compare panel
 * (`variant="compact"`). Replaces what used to be duplicated rendering in
 * mode-compare-panel.tsx.
 */

import { Check } from "lucide-react";
import type { OperatingModeCapabilities } from "../../../types/operating-mode";
import { capabilityLabel } from "./utils";

const CAPABILITY_FLAGS: ReadonlyArray<keyof OperatingModeCapabilities> = [
  "supportsPhases",
  "canStartPhases",
  "canCompleteItems",
  "canApplyBacklogSyncProposals",
  "requiresAcceptanceCriteria",
  "supportsArtifacts",
  "supportsHandoffs",
  "usesItemExecutionFlow",
];

export interface CapabilityListProps {
  capabilities: OperatingModeCapabilities;
  /**
   * compact: tighter type, used in the picker compare panel column.
   * full: roomy two-line rows used on the details page Capabilities section.
   */
  variant?: "compact" | "full";
  emptyMessage?: string;
  testId?: string;
}

export function CapabilityList({
  capabilities,
  variant = "compact",
  emptyMessage = "No capabilities enabled.",
  testId,
}: CapabilityListProps) {
  const enabled = CAPABILITY_FLAGS.filter((flag) => capabilities[flag]);
  if (enabled.length === 0) {
    return <p className="text-xs italic text-slate-500" data-testid={testId}>{emptyMessage}</p>;
  }
  if (variant === "compact") {
    return (
      <ul className="mt-2 space-y-1" data-testid={testId}>
        {enabled.map((flag) => (
          <li key={flag} className="flex items-center gap-1.5 text-xs text-slate-300">
            <Check className="h-3 w-3 text-emerald-400" aria-hidden="true" />
            {capabilityLabel(flag)}
          </li>
        ))}
      </ul>
    );
  }
  return (
    <ul className="space-y-2" data-testid={testId}>
      {enabled.map((flag) => (
        <li key={flag} className="flex items-start gap-2 text-sm text-slate-200">
          <Check className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" aria-hidden="true" />
          <span>{capabilityLabel(flag)}</span>
        </li>
      ))}
    </ul>
  );
}
