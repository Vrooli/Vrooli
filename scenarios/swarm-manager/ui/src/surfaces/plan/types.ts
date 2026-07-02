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
  | "review"
  | "classify"
  | "none";

export type PlanGateKind = "decide" | "review" | "classify" | "workshop";

export type PlanBlockerKind = "none" | "gate" | "items" | "cycle";

export type PlanOutcome = "ok" | "failed" | "needs_review" | "needs_followup";

/** Wave index marking a dependency-cycle-trapped card. */
export const CYCLE_WAVE = -1;

export interface PlanGateData {
  id: string;
  kind: PlanGateKind;
  ownerType: "backlog" | "execution" | "capture";
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
  initiative: string;
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

export interface PlanBoardMetaData {
  generatedAt: string;
  windowSeconds: number;
  maxWave: number;
  cycles: string[];
}

export interface PlanBoardData {
  now: PlanNowSummaryData;
  next: PlanColumnData;
  later: PlanColumnData;
  done: PlanColumnData;
  meta: PlanBoardMetaData;
}
