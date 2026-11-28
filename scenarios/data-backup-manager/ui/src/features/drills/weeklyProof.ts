import { DrillStatus, type RecoveryDrill } from "../../api/drills";
import type { Plan } from "../../api/plans";
import { tsToDate } from "../../lib/proto";

export const WEEKLY_PROOF_WINDOW_MS = 7 * 24 * 60 * 60 * 1000;

export type WeeklyProofState = "verified" | "missing" | "pending" | "failed" | "stale" | "retired" | "manual" | "disabled";

export interface WeeklyProofUnit {
  planId: string;
  targetId: string;
  destinationId: string;
  state: WeeklyProofState;
  receipt?: RecoveryDrill;
}

const scheduledForUnit = (receipts: RecoveryDrill[], planId: string, targetId: string, destinationId: string) =>
  receipts
    .filter((receipt) => receipt.scheduled && receipt.planId === planId && receipt.targetId === targetId && receipt.destinationId === destinationId)
    .sort((a, b) => {
      const aTime = tsToDate(a.requestedAt)?.getTime() ?? Number.NEGATIVE_INFINITY;
      const bTime = tsToDate(b.requestedAt)?.getTime() ?? Number.NEGATIVE_INFINITY;
      return bTime - aTime || b.id.localeCompare(a.id);
    });

export function evaluateWeeklyProof(plan: Plan, targetId: string, destinationId: string, receipts: RecoveryDrill[], now = new Date()): WeeklyProofUnit {
  const base = { planId: plan.id, targetId, destinationId };
  if (!plan.enabled) return { ...base, state: "disabled" };
  const schedule = plan.recoveryDrillSchedule.trim().toLowerCase();
  if (!schedule || schedule === "manual") return { ...base, state: "manual" };

  const latest = scheduledForUnit(receipts, plan.id, targetId, destinationId)[0];
  if (!latest) return { ...base, state: "missing" };
  if (latest.retiredAt) return { ...base, state: "retired", receipt: latest };
  if (latest.status === DrillStatus.REQUESTED || latest.status === DrillStatus.RUNNING) return { ...base, state: "pending", receipt: latest };
  if (latest.status === DrillStatus.FAILED) return { ...base, state: "failed", receipt: latest };
  if (latest.status !== DrillStatus.VERIFIED) return { ...base, state: "missing", receipt: latest };

  const completedAt = tsToDate(latest.finishedAt)?.getTime();
  const age = completedAt === undefined ? Number.POSITIVE_INFINITY : now.getTime() - completedAt;
  if (age < 0 || age > WEEKLY_PROOF_WINDOW_MS) return { ...base, state: "stale", receipt: latest };
  return { ...base, state: "verified", receipt: latest };
}

export function weeklyProofUnits(plans: Plan[], receipts: RecoveryDrill[], now = new Date()): WeeklyProofUnit[] {
  return plans.flatMap((plan) => plan.targetIds.flatMap((targetId) => plan.destinationIds.map((destinationId) =>
    evaluateWeeklyProof(plan, targetId, destinationId, receipts, now),
  )));
}
