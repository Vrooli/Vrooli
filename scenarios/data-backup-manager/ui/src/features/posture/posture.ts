import type { TargetStatus } from "../../api/runs";
import { isOlderThan } from "../../lib/format";
import { tsToDate } from "../../lib/proto";
import { STALE_VERIFY_MS } from "../../lib/status";

export interface PostureSummary {
  /** Distinct targets that are overdue/failed OR not recently verified. */
  attention: number;
  overdue: number;
  unverified: number;
  /** True only when there is at least one target and none need attention. */
  allClear: boolean;
}

/**
 * Summarize backup posture from the per-target status rollup.
 *
 * The headline must count the *union* of affected targets. A target can be both
 * overdue and unverified, so `overdue + unverified` can exceed the number of
 * registered targets and overstate how much is wrong.
 */
export function summarizePosture(
  statuses: readonly TargetStatus[],
  now: Date = new Date(),
): PostureSummary {
  const overdueTargets = statuses.filter((s) => s.overdue);
  const unverifiedTargets = statuses.filter((s) =>
    isOlderThan(tsToDate(s.lastVerifiedAt), STALE_VERIFY_MS, now),
  );

  const attention = new Set<string>();
  for (const s of overdueTargets) attention.add(s.targetId);
  for (const s of unverifiedTargets) attention.add(s.targetId);

  return {
    attention: attention.size,
    overdue: overdueTargets.length,
    unverified: unverifiedTargets.length,
    allClear: statuses.length > 0 && attention.size === 0,
  };
}
