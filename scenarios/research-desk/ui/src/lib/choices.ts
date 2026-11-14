import type { Blocker, BlockerChoice } from "../api/launch";

/**
 * The operator's choices with the desk's recommendation selected for every
 * blocker not yet seen ("" when the desk recommends nothing, as for a live
 * supervisor). A choice the operator already made is never overwritten.
 * Returns null when nothing is new, so a caller can skip the update.
 */
export function withRecommendations(blockers: Blocker[], choices: Record<string, BlockerChoice | "">): Record<string, BlockerChoice | ""> | null {
  const fresh = blockers.filter((b) => !(b.commissionId in choices));
  if (fresh.length === 0) return null;
  return { ...choices, ...Object.fromEntries(fresh.map((b) => [b.commissionId, b.recommended ?? ""])) };
}
