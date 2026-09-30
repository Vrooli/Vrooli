import type { Reading } from "./api";

/** Enabling (unranked) work. `direct` marks an explicit enables edge into the rung. */
export interface LadderWork { name: string; status: string; finishBar?: string; urgency: number; direct?: boolean }
export interface LadderGoal { name: string; title: string; priority: number }
export interface LadderRung {
  rank: number;
  id: string;
  name: string;
  status: string;
  finishBar?: string;
  ramps: string[];
  streams: string[];
  audiences: string[];
  blockers: LadderWork[];
  goals: LadderGoal[];
  readiness: { reported: boolean; goalClosed?: boolean; approvedCommit?: string };
}
export type UnlockKind = "ramp" | "stream" | "audience";
/** The first rank at which a ramp, stream or audience opens; 0 when no scheduled rung opens it. */
export interface LadderUnlock { kind: UnlockKind; name: string; opensAt: number }
export interface LadderReading {
  rungs: LadderRung[];
  nextRank: number;
  enabling: LadderWork[];
  unscheduled: Array<{ name: string; status: string; finishBar?: string }>;
  reach: LadderUnlock[];
  unavailable?: Array<{ source: string; reason: string }>;
}

/** The ladder a room can draw: the measured one, else the authored sample, marked as sampled. */
export function ladderOf(reading: Reading): { ladder: LadderReading; sampled: boolean } | null {
  if (reading.ladder?.rungs.length) return { ladder: reading.ladder, sampled: false };
  if (reading.sample?.ladder?.rungs.length) return { ladder: reading.sample.ladder, sampled: true };
  return null;
}

const STATUS_LABELS: Record<string, string> = {
  IDEA: "Idea", CANDIDATE: "Candidate", TRIGGER_MET: "Trigger met", ACTIVE: "Active", SHIPPED: "Shipped", PROPOSED: "Proposed", RETIRED: "Retired",
};
export const statusLabel = (status: string): string => STATUS_LABELS[status] ?? status.toLowerCase().replace(/_/g, " ");

const FINISH_LABELS: Record<string, string> = { CUSTOMER_FACING: "customer-facing", OPERATOR_FACING: "operator-facing", INTERNAL: "internal" };
export const finishLabel = (finishBar?: string): string => (finishBar ? FINISH_LABELS[finishBar] ?? finishBar.toLowerCase().replace(/_/g, " ") : "");

export const nextRung = (ladder: LadderReading): LadderRung | null => ladder.rungs.find((rung) => rung.rank === ladder.nextRank) ?? null;

/** Rungs after the next one, in schedule order. */
export const rungsAfterNext = (ladder: LadderReading): LadderRung[] => (ladder.nextRank > 0 ? ladder.rungs.filter((rung) => rung.rank > ladder.nextRank) : []);

export const stillIdeas = (ladder: LadderReading): number => ladder.rungs.filter((rung) => rung.status === "IDEA").length;

/** How many of each unlock kind are open once the schedule reaches `rank`. */
export function openedBy(ladder: LadderReading, rank: number): Record<UnlockKind, { open: number; total: number }> {
  const out: Record<UnlockKind, { open: number; total: number }> = { ramp: { open: 0, total: 0 }, stream: { open: 0, total: 0 }, audience: { open: 0, total: 0 } };
  for (const unlock of ladder.reach) {
    const bucket = out[unlock.kind];
    bucket.total += 1;
    if (unlock.opensAt > 0 && unlock.opensAt <= rank) bucket.open += 1;
  }
  return out;
}

/** The latest rank at which anything opens: the rung after which reach stops growing. */
export const lastOpening = (ladder: LadderReading): LadderUnlock[] => {
  const latest = Math.max(0, ...ladder.reach.map((unlock) => unlock.opensAt));
  return latest > 0 ? ladder.reach.filter((unlock) => unlock.opensAt === latest) : [];
};
