// ============================================================================
// Baselines feature — shared display model (Plan B §4.2/§4.3)
// ============================================================================
//
// Surface ordering, human labels, capture-cost estimates, and the verdict
// vocabulary shared by every baseline view. The verdict strings mirror
// test-genie's RunsService classifier verbatim (clean / regression /
// new-failure / preexisting / not-comparable) — see baselines.proto.

import type { BadgeProps } from "../../components/ui/badge";
import type {
  BaselineManifest,
  SurfaceDiff,
} from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";
import type { RepoStatus } from "../../lib/api";

export const BASELINE_SURFACES = ["workflows", "tests", "structure", "visuals", "rules"] as const;
export type BaselineSurface = (typeof BASELINE_SURFACES)[number];

export interface SurfaceMeta {
  id: BaselineSurface;
  label: string;
  // What capturing this surface does, shown next to the checkbox in the modal.
  captureNote: string;
}

export const SURFACE_META: Record<BaselineSurface, SurfaceMeta> = {
  workflows: { id: "workflows", label: "Workflows", captureNote: "runs BAS workflows" },
  tests: { id: "tests", label: "Tests", captureNote: "runs test-genie unit/integration" },
  structure: { id: "structure", label: "Structure", captureNote: "file-tree + structure scan" },
  visuals: { id: "visuals", label: "Visuals", captureNote: "captures page screenshots" },
  rules: { id: "rules", label: "Rules", captureNote: "runs scenario-auditor rules" },
};

export function surfaceLabel(id: string): string {
  return SURFACE_META[id as BaselineSurface]?.label ?? id;
}

// ── Verdicts ──────────────────────────────────────────────────────────────

export type Verdict = "clean" | "regression" | "new-failure" | "preexisting" | "not-comparable";

export interface VerdictMeta {
  label: string;
  variant: NonNullable<BadgeProps["variant"]>;
  // True when this verdict means "your change broke something" — the only
  // verdict that should read as a hard failure in the UI.
  isRegression: boolean;
}

const VERDICT_META: Record<string, VerdictMeta> = {
  clean: { label: "Clean", variant: "success", isRegression: false },
  regression: { label: "Regression", variant: "error", isRegression: true },
  "new-failure": { label: "New failure", variant: "warning", isRegression: false },
  preexisting: { label: "Preexisting", variant: "default", isRegression: false },
  "not-comparable": { label: "Not comparable", variant: "default", isRegression: false },
};

export function verdictMeta(verdict: string): VerdictMeta {
  return VERDICT_META[verdict] ?? { label: verdict || "Unknown", variant: "default", isRegression: false };
}

// ── Surface presence in a stored manifest ──────────────────────────────────

export type SurfacePresence = "captured" | "skipped" | "absent";

export function surfacePresence(manifest: BaselineManifest, surfaceId: string): SurfacePresence {
  if (manifest.surfaces[surfaceId]) return "captured";
  if (manifest.skipped[surfaceId]) return "skipped";
  return "absent";
}

// ── Diff roll-up helpers ────────────────────────────────────────────────────

export function countFindings(diff: SurfaceDiff): {
  regressions: number;
  newFailures: number;
  preexisting: number;
  cleared: number;
} {
  return {
    regressions: diff.regressions.length,
    newFailures: diff.newFailures.length,
    preexisting: diff.preexisting.length,
    cleared: diff.cleared.length,
  };
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
