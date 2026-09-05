// ============================================================================
// Baselines feature — shared display model (Plan B §4.2/§4.3)
// ============================================================================
//
// Surface ordering, human labels, capture-cost estimates, and the verdict
// vocabulary shared by every baseline view. The verdict strings mirror
// test-genie's RunsService classifier verbatim (clean / regression /
// new-failure / preexisting / not-comparable) — see baselines.proto.

import type { BadgeProps } from "../../components/ui/badge";
import { PhaseComparisonReasonCode, type PhaseDiff } from "@vrooli/proto-types/test-genie/v1/runs/runs_pb";
import type { RepoStatus } from "../../lib/api";

// ── Verdicts ──────────────────────────────────────────────────────────────

export type Verdict =
  | "clean"
  | "changed"
  | "regression"
  | "new-failure"
  | "preexisting"
  | "not-comparable";

export interface VerdictMeta {
  label: string;
  variant: NonNullable<BadgeProps["variant"]>;
  // True when this verdict means "your change broke something" — the only
  // verdict that should read as a hard failure in the UI.
  isRegression: boolean;
}

const VERDICT_META: Record<string, VerdictMeta> = {
  clean: { label: "Clean", variant: "success", isRegression: false },
  // "changed" is the neutral, advisory visual tier — the UI moved and is worth a
  // look, but it is NOT a failure (it never gates). Rendered as an informational
  // badge, distinct from both clean and the failure verdicts.
  changed: { label: "Changed — review", variant: "info", isRegression: false },
  regression: { label: "Regression", variant: "error", isRegression: true },
  "new-failure": { label: "New failure", variant: "warning", isRegression: false },
  preexisting: { label: "Preexisting", variant: "default", isRegression: false },
  "not-comparable": { label: "Not comparable", variant: "default", isRegression: false },
};

export function verdictMeta(verdict: string): VerdictMeta {
  return VERDICT_META[verdict] ?? { label: verdict || "Unknown", variant: "default", isRegression: false };
}

// ── Diff roll-up helpers ────────────────────────────────────────────────────

export function countFindings(diff: PhaseDiff): {
  regressions: number;
  newFailures: number;
  preexisting: number;
  cleared: number;
} {
  return {
    regressions: diff.regressions.length,
    newFailures: diff.newFailures.length,
    preexisting: diff.preexistingFailures.length,
    cleared: diff.clearedFailures.length,
  };
}

export interface ComparisonSummary {
  regressions: number;
  newFailures: number;
  preexisting: number;
  cleared: number;
  notComparable: number;
  catalogAdded: number;
  catalogRetired: number;
}

export function summarizePhaseDiffs(phases: PhaseDiff[]): ComparisonSummary {
  return phases.reduce<ComparisonSummary>((summary, phase) => {
    summary.regressions += phase.regressions.length;
    summary.newFailures += phase.newFailures.length;
    summary.preexisting += phase.preexistingFailures.length;
    summary.cleared += phase.clearedFailures.length;
    if (phase.verdict === "not-comparable") summary.notComparable += 1;
    if (phase.reasons.some((reason) => reason.code === PhaseComparisonReasonCode.NEW_PHASE)) summary.catalogAdded += 1;
    if (phase.reasons.some((reason) => reason.code === PhaseComparisonReasonCode.RETIRED_PHASE)) summary.catalogRetired += 1;
    return summary;
  }, { regressions: 0, newFailures: 0, preexisting: 0, cleared: 0, notComparable: 0, catalogAdded: 0, catalogRetired: 0 });
}

export function phaseDiffNeedsAttention(diff: PhaseDiff): boolean {
  return diff.verdict !== "clean" || diff.regressions.length > 0 || diff.newFailures.length > 0 || diff.reasons.length > 0;
}

// ── Working-tree dirtiness (drives the SetBaselineModal warning) ────────────

export interface DirtyState {
  dirty: boolean;
  modified: number;
}

export function dirtyStateFromStatus(status?: RepoStatus): DirtyState {
  if (!status?.summary) return { dirty: false, modified: 0 };
  const { staged, unstaged, untracked, conflicts } = status.summary;
  const modified = staged + unstaged + untracked + conflicts;
  return { dirty: modified > 0, modified };
}
