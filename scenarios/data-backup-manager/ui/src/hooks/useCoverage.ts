/**
 * Joins the target catalog with the posture rollup into per-target coverage
 * rows. This is the shared substrate behind the Overview coverage grid and the
 * Targets table: a registered target with no run yet still appears (its backup
 * and verify ages are simply "never"), so coverage reflects the catalog, not
 * just run history.
 */
import { RunStatus } from "../api/runs";
import { useTargets } from "./useTargets";
import { useTargetStatus } from "./useTargetStatus";
import { tsToDate } from "../lib/proto";
import type { Target } from "../api/targets";

export interface CoverageRow {
  target: Target;
  lastSuccessAt: Date | undefined;
  lastVerifiedAt: Date | undefined;
  lastVerifiedSnapshotId: string;
  lastRunStatus: RunStatus;
  lastRunAt: Date | undefined;
}

export interface CoverageResult {
  rows: CoverageRow[];
  isLoading: boolean;
  isError: boolean;
  refetch: () => void;
}

export function useCoverage(owner = ""): CoverageResult {
  const targets = useTargets(owner);
  const statuses = useTargetStatus(owner);

  const byId = new Map((statuses.data ?? []).map((s) => [s.targetId, s]));
  const rows: CoverageRow[] = (targets.data ?? []).map((target) => {
    const s = byId.get(target.id);
    return {
      target,
      lastSuccessAt: tsToDate(s?.lastSuccessAt),
      lastVerifiedAt: tsToDate(s?.lastVerifiedAt),
      lastVerifiedSnapshotId: s?.lastVerifiedSnapshotId ?? "",
      lastRunStatus: s?.lastRunStatus ?? RunStatus.UNSPECIFIED,
      lastRunAt: tsToDate(s?.lastRunAt),
    };
  });

  return {
    rows,
    isLoading: targets.isLoading || statuses.isLoading,
    isError: targets.isError || statuses.isError,
    refetch: () => {
      void targets.refetch();
      void statuses.refetch();
    },
  };
}
