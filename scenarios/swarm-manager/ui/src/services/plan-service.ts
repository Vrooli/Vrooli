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

export interface CanonicalPlanSummary {
  id: string;
  slug: string;
  title: string;
  status: string;
  updatedAt?: string;
  createdAt?: string;
  phaseCount: number;
}

export interface PlanImportContainerInput {
  type: "items" | "initiative";
  name?: string;
  title?: string;
  description?: string;
  mode?: string;
}

export interface PlanImportInput {
  planId?: string;
  sourcePath?: string;
  markdown?: string;
  title?: string;
  slug?: string;
  container?: PlanImportContainerInput;
}

export interface PlanImportItemResult {
  kind: string;
  name: string;
  title: string;
  action: string;
}

export interface PlanImportInitiativeResult {
  name: string;
  title: string;
  mode?: string;
  action: string;
}

export interface PlanImportResult {
  slug: string;
  planId: string;
  container: "items" | "initiative";
  items: PlanImportItemResult[];
  initiative?: PlanImportInitiativeResult;
  count: number;
  created: number;
  linked: number;
  updated: number;
}

export interface IPlanService {
  getBoard(options?: PlanRequestOptions): Promise<PlanBoardData>;
  listCanonicalPlans(options?: { signal?: AbortSignal }): Promise<CanonicalPlanSummary[]>;
  importPlan(input: PlanImportInput, options?: { signal?: AbortSignal }): Promise<PlanImportResult>;
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

function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function asNumber(value: unknown): number {
  return typeof value === "number" ? value : Number(value ?? 0);
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
    async listCanonicalPlans(options?: { signal?: AbortSignal }): Promise<CanonicalPlanSummary[]> {
      const data = await apiClient.get<{ plans?: Array<Record<string, unknown>> }>(
        API_ENDPOINTS.planImportPlans,
        { signal: options?.signal },
      );
      return (data.plans ?? []).map((plan) => ({
        id: asString(plan.id),
        slug: asString(plan.slug),
        title: asString(plan.title),
        status: asString(plan.status),
        updatedAt: asString(plan.updated_at) || undefined,
        createdAt: asString(plan.created_at) || undefined,
        phaseCount: asNumber(plan.phase_count),
      }));
    },
    async importPlan(input: PlanImportInput, options?: { signal?: AbortSignal }): Promise<PlanImportResult> {
      const body = {
        plan_id: input.planId,
        source_path: input.sourcePath,
        markdown: input.markdown,
        title: input.title,
        slug: input.slug,
        container: input.container,
      };
      const data = await apiClient.post<Record<string, unknown>>(API_ENDPOINTS.planImport, body, {
        signal: options?.signal,
      });
      return {
        slug: asString(data.slug),
        planId: asString(data.plan_id),
        container: data.container === "initiative" ? "initiative" : "items",
        items: Array.isArray(data.items)
          ? data.items.map((item) => {
              const raw = item as Record<string, unknown>;
              return {
                kind: asString(raw.kind),
                name: asString(raw.name),
                title: asString(raw.title),
                action: asString(raw.action),
              };
            })
          : [],
        initiative: data.initiative && typeof data.initiative === "object"
          ? {
              name: asString((data.initiative as Record<string, unknown>).name),
              title: asString((data.initiative as Record<string, unknown>).title),
              mode: asString((data.initiative as Record<string, unknown>).mode) || undefined,
              action: asString((data.initiative as Record<string, unknown>).action),
            }
          : undefined,
        count: asNumber(data.count),
        created: asNumber(data.created),
        linked: asNumber(data.linked),
        updated: asNumber(data.updated),
      };
    },
  };
}

export const planService = createPlanService();
