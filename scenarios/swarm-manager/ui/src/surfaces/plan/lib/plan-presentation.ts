/**
 * Pure presentation helpers for the Plan board. No React — unit-testable
 * grouping/sorting/labeling logic only.
 */

import { CYCLE_WAVE, type PlanCardData, type PlanCardGroupData, type PlanGateData } from "../types";

/**
 * Cards deeper than this wave collapse into the "beyond horizon" rollup
 * (plan decision D1: waves 3–5 stay visible, deeper is noise).
 */
export const HORIZON_WAVE = 5;

export interface HorizonSplit {
  /** Groups with beyond-horizon cards removed (empty groups dropped). */
  visible: PlanCardGroupData[];
  /** Cards past the horizon, in their original order. */
  beyond: PlanCardData[];
}

/**
 * Split Later-column groups into the visible board and the beyond-horizon
 * rollup. Cycle cards are never rolled up — the cycle group is diagnostics
 * and must stay visible.
 */
export function splitBeyondHorizon(
  groups: PlanCardGroupData[],
  horizonWave: number = HORIZON_WAVE,
): HorizonSplit {
  const visible: PlanCardGroupData[] = [];
  const beyond: PlanCardData[] = [];
  for (const group of groups) {
    if (group.blockerKind === "cycle") {
      visible.push(group);
      continue;
    }
    const kept = group.cards.filter((card) => {
      if (card.wave !== CYCLE_WAVE && card.wave > horizonWave) {
        beyond.push(card);
        return false;
      }
      return true;
    });
    if (kept.length > 0) {
      visible.push(kept.length === group.cards.length ? group : { ...group, cards: kept });
    }
  }
  return { visible, beyond };
}

/** Wave badge label: "now" at wave 0, "w3" deeper, "cycle" when trapped. */
export function waveBadgeLabel(wave: number): string {
  if (wave === CYCLE_WAVE) return "cycle";
  if (wave === 0) return "now";
  return `w${wave}`;
}

/** Human label for a gate card's primary action. */
export function gateActionLabel(gate: PlanGateData): string {
  switch (gate.kind) {
    case "decide":
      return gate.count === 1 ? "Answer 1 question" : `Answer ${gate.count} questions`;
    case "review":
      return "Review";
    case "classify":
      return gate.count > 0 ? `Classify (${gate.count})` : "Classify";
    case "workshop":
      return gate.suggested === "finalize" ? "Finalize" : "Workshop";
  }
}

/** Short glyph key for a Done-card outcome. */
export function outcomeGlyph(outcome: PlanCardData["outcome"]): string {
  switch (outcome) {
    case "ok":
      return "✓";
    case "failed":
      return "✗";
    case "needs_review":
      return "◆";
    case "needs_followup":
      return "⚠";
    default:
      return "•";
  }
}

/** Total card count across groups (client-side sanity over cardCount). */
export function countCards(groups: PlanCardGroupData[]): number {
  return groups.reduce((sum, group) => sum + group.cards.length, 0);
}

/**
 * Snooze key for a plan card, matching the Command Post's snooze-store
 * key shapes (backlog:kind/name, execution:id, capture:id).
 */
export function cardSnoozeKey(card: PlanCardData): string | null {
  if (card.executionId) return `execution:${card.executionId}`;
  if (card.id.startsWith("capture/")) return `capture:${card.id.slice("capture/".length)}`;
  if (card.itemKind && card.itemName) return `backlog:${card.itemKind}/${card.itemName}`;
  return null;
}

export interface SnoozeFilterResult {
  groups: PlanCardGroupData[];
  /** Snoozed card ids retained (dimmed) when showSnoozed is on. */
  snoozedIds: Set<string>;
  /** Count of cards hidden (showSnoozed off). */
  hiddenCount: number;
}

/**
 * Apply client-side snooze filtering to a column's groups. Hidden by
 * default; with showSnoozed the cards stay in place and callers dim the
 * ids in `snoozedIds`. Empty groups are dropped.
 */
export function applySnoozeFilter(
  groups: PlanCardGroupData[],
  snoozedKeys: Set<string>,
  showSnoozed: boolean,
): SnoozeFilterResult {
  const snoozedIds = new Set<string>();
  let hiddenCount = 0;
  const out: PlanCardGroupData[] = [];
  for (const group of groups) {
    const kept = group.cards.filter((card) => {
      const key = cardSnoozeKey(card);
      if (!key || !snoozedKeys.has(key)) return true;
      if (showSnoozed) {
        snoozedIds.add(card.id);
        return true;
      }
      hiddenCount += 1;
      return false;
    });
    if (kept.length > 0) {
      out.push(kept.length === group.cards.length ? group : { ...group, cards: kept });
    }
  }
  return { groups: out, snoozedIds, hiddenCount };
}

/**
 * Column-level "N in M waves" subtitle for Later: distinct non-cycle wave
 * count among the given groups.
 */
export function laterWaveSummary(groups: PlanCardGroupData[]): string {
  const waves = new Set<number>();
  let count = 0;
  for (const group of groups) {
    for (const card of group.cards) {
      count += 1;
      if (card.wave !== CYCLE_WAVE) waves.add(card.wave);
    }
  }
  if (count === 0) return "nothing blocked";
  const waveCount = Math.max(waves.size, 1);
  return `${count} in ${waveCount} ${waveCount === 1 ? "wave" : "waves"}`;
}
