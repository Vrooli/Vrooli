import type {
  PlanCard as ProtoPlanCard,
  PlanCardGroup as ProtoPlanCardGroup,
  PlanColumn as ProtoPlanColumn,
} from "@vrooli/proto-types/swarm-manager/v1/domain/plan_pb";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { parseProtoResponse, planBoardResponseSchema } from "./proto-contracts";
import type {
  PlanBlockerKind,
  PlanBoardData,
  PlanCardAction,
  PlanCardData,
  PlanCardGroupData,
  PlanCardType,
  PlanColumnData,
  PlanGateData,
  PlanGateKind,
  PlanOutcome,
} from "../surfaces/plan/types";

export interface PlanRequestOptions {
  signal?: AbortSignal;
  windowSeconds?: number;
  /** When set, scopes the board to this goal's transitive prerequisite closure. */
  goal?: string;
}

export interface IPlanService {
  getBoard(options?: PlanRequestOptions): Promise<PlanBoardData>;
}

const CARD_TYPES = new Set<PlanCardType>(["item", "gate", "outcome"]);
const CARD_ACTIONS = new Set<PlanCardAction>([
  "run",
  "workshop",
  "finalize",
  "decide",
  "review",
  "classify",
  "none",
]);
const GATE_KINDS = new Set<PlanGateKind>(["decide", "review", "classify", "workshop"]);
const BLOCKER_KINDS = new Set<PlanBlockerKind>(["none", "gate", "items", "cycle"]);
const OUTCOMES = new Set<PlanOutcome>(["ok", "failed", "needs_review", "needs_followup"]);

function mapCardType(value: string): PlanCardType {
  return CARD_TYPES.has(value as PlanCardType) ? (value as PlanCardType) : "item";
}

function mapAction(value: string): PlanCardAction {
  return CARD_ACTIONS.has(value as PlanCardAction) ? (value as PlanCardAction) : "none";
}

function mapBlockerKind(value: string): PlanBlockerKind {
  return BLOCKER_KINDS.has(value as PlanBlockerKind) ? (value as PlanBlockerKind) : "none";
}

function mapOutcome(value: string): PlanOutcome | "" {
  return OUTCOMES.has(value as PlanOutcome) ? (value as PlanOutcome) : "";
}

function mapGate(card: ProtoPlanCard): PlanGateData | null {
  const gate = card.gate;
  if (!gate) return null;
  const ownerType =
    gate.ownerType === "execution" || gate.ownerType === "capture"
      ? gate.ownerType
      : "backlog";
  return {
    id: gate.id,
    kind: GATE_KINDS.has(gate.kind as PlanGateKind) ? (gate.kind as PlanGateKind) : "decide",
    ownerType,
    ownerKind: gate.ownerKind,
    ownerName: gate.ownerName,
    ownerTitle: gate.ownerTitle,
    count: gate.count,
    blocks: gate.blocks ?? [],
    decidableSince: gate.decidableSince,
    suggested: gate.suggested,
  };
}

function mapCard(card: ProtoPlanCard): PlanCardData {
  return {
    id: card.id,
    cardType: mapCardType(card.cardType),
    action: mapAction(card.action),
    itemKind: card.itemKind,
    itemName: card.itemName,
    title: card.title,
    status: card.status,
    priority: card.priority,
    wave: card.wave,
    initiative: card.initiative,
    effort: card.effort,
    gate: mapGate(card),
    outcome: mapOutcome(card.outcome),
    finishedAt: card.finishedAt,
    executionId: card.executionId,
    unblocks: card.unblocks,
  };
}

function mapGroup(group: ProtoPlanCardGroup): PlanCardGroupData {
  return {
    id: group.id,
    label: group.label,
    blockerKind: mapBlockerKind(group.blockerKind),
    gateId: group.gateId,
    blockerKeys: group.blockerKeys ?? [],
    cards: (group.cards ?? []).map(mapCard),
  };
}

function mapColumn(column: ProtoPlanColumn | undefined): PlanColumnData {
  return {
    groups: (column?.groups ?? []).map(mapGroup),
    cardCount: column?.cardCount ?? 0,
  };
}

export function createPlanService(apiClient: IApiClient = defaultApiClient): IPlanService {
  return {
    async getBoard(options?: PlanRequestOptions): Promise<PlanBoardData> {
      const params = new URLSearchParams();
      if (options?.windowSeconds && options.windowSeconds > 0) {
        params.set("window_seconds", String(options.windowSeconds));
      }
      if (options?.goal) {
        params.set("goal", options.goal);
      }
      const query = params.toString();
      const endpoint = query ? `${API_ENDPOINTS.plan}?${query}` : API_ENDPOINTS.plan;
      const data = await apiClient.get<unknown>(endpoint, { signal: options?.signal });
      const proto = parseProtoResponse(planBoardResponseSchema, data, "plan board");

      return {
        now: {
          activeCount: proto.now?.activeCount ?? 0,
          queueDepth: proto.now?.queueDepth ?? 0,
          maxQueueDepth: proto.now?.maxQueueDepth ?? 0,
          lanes: (proto.now?.lanes ?? []).map((lane) => ({
            lane: lane.lane,
            active: lane.active,
            capacity: lane.capacity,
          })),
        },
        next: mapColumn(proto.next),
        later: mapColumn(proto.later),
        done: mapColumn(proto.done),
        meta: {
          generatedAt: proto.meta?.generatedAt ?? "",
          windowSeconds: proto.meta?.windowSeconds ?? 0,
          maxWave: proto.meta?.maxWave ?? 0,
          cycles: proto.meta?.cycles ?? [],
          eta: proto.meta?.eta
            ? {
                p50Hours: proto.meta.eta.p50Hours,
                p80Hours: proto.meta.eta.p80Hours,
                p50Label: proto.meta.eta.p50Label,
                p80Label: proto.meta.eta.p80Label,
                basis: proto.meta.eta.basis,
                basisLabel: proto.meta.eta.basisLabel,
                confidence: proto.meta.eta.confidence,
                remainingItems: proto.meta.eta.remainingItems,
                laneCapacity: proto.meta.eta.laneCapacity,
              }
            : null,
        },
      };
    },
  };
}

export const planService = createPlanService();
