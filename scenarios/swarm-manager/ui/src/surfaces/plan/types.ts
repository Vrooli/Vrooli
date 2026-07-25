/**
 * Plan surface types — the UI model for the Plan-lens kanban board.
 *
 * Mirrors the server projection (GET /api/v1/plan): four columns
 * (Now/Next/Later/Done) of card groups. Wave membership is ordinal
 * ("dependency layers from runnable"), never clock time.
 */

export type PlanCardType = "item" | "gate" | "outcome";

export type PlanCardAction =
  | "run"
  | "workshop"
  | "finalize"
  | "decide"
  | "proposal"
  | "review"
  | "classify"
  | "none";

export type PlanGateKind = "decide" | "proposal" | "review" | "classify" | "workshop";

export type PlanBlockerKind = "none" | "gate" | "items" | "cycle";

export type PlanOutcome = "ok" | "failed" | "needs_review" | "needs_followup" | "dropped";

/** Wave index marking a dependency-cycle-trapped card. */
export const CYCLE_WAVE = -1;

export interface PlanGateData {
  id: string;
  kind: PlanGateKind;
  ownerType: "backlog" | "execution" | "capture" | "milestone" | "goal";
  ownerKind: string;
  ownerName: string;
  ownerTitle: string;
  count: number;
  blocks: string[];
  decidableSince: string;
  suggested: string;
}

export interface PlanCardData {
  /** Canonical node id (backlog-item/…, capture/…, execution-record/…). */
  id: string;
  cardType: PlanCardType;
  action: PlanCardAction;
  itemKind: string;
  itemName: string;
  title: string;
  status: string;
  priority: number;
  wave: number;
  milestone: string;
  effort: string;
  gate: PlanGateData | null;
  outcome: PlanOutcome | "";
  finishedAt: string;
  executionId: string;
  unblocks: number;
}

export interface PlanCardGroupData {
  id: string;
  label: string;
  blockerKind: PlanBlockerKind;
  gateId: string;
  blockerKeys: string[];
  cards: PlanCardData[];
}

export interface PlanColumnData {
  groups: PlanCardGroupData[];
  cardCount: number;
}

export interface PlanLaneStatusData {
  lane: string;
  active: number;
  capacity: number;
}

export interface PlanNowSummaryData {
  activeCount: number;
  queueDepth: number;
  maxQueueDepth: number;
  lanes: PlanLaneStatusData[];
}

/**
 * Completion estimate for the board's remaining work: a p50/p80 band with an
 * explicit basis/confidence label that honestly degrades to "priors only"
 * under cold start.
 */
export interface PlanEtaBandData {
  p50Hours: number;
  p80Hours: number;
  p50Label: string;
  p80Label: string;
  /** live | backfill | priors | default | mixed */
  basis: string;
  /** "27 samples" vs "priors only" */
  basisLabel: string;
  /** low | medium | high */
  confidence: string;
  remainingItems: number;
  laneCapacity: number;
}

export interface PlanBoardMetaData {
  generatedAt: string;
  windowSeconds: number;
  maxWave: number;
  cycles: string[];
  /** Present when the estimator is wired and there is pending work. */
  eta: PlanEtaBandData | null;
}

export interface PlanBoardData {
  now: PlanNowSummaryData;
  next: PlanColumnData;
  later: PlanColumnData;
  done: PlanColumnData;
  meta: PlanBoardMetaData;
}
