/**
 * Goals Service — data access for goal operations.
 *
 * The goals backend is mux+JSON (snake_case), mirroring the initiatives domain
 * rather than the proto surfaces. This service normalizes responses to the
 * camelCase types the UI consumes and exposes the create / update / target /
 * priority levers the goals UX drives.
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type {
  CreateGoalInput,
  Goal,
  GoalScope,
  GoalScopeSnapshot,
  GoalStatus,
  GoalWithScope,
  UpdateGoalInput,
} from "../types/goal";
import type { PlanEtaBandData } from "../surfaces/plan/types";

interface RawSnapshot {
  at?: string;
  target_count?: number;
  targetCount?: number;
  closure_size?: number;
  closureSize?: number;
  completed?: number;
}

interface RawGoal {
  name?: string;
  title?: string;
  description?: string;
  status?: string;
  priority?: number;
  targets?: string[];
  seeded?: boolean;
  scope_history?: RawSnapshot[];
  scopeHistory?: RawSnapshot[];
  created?: string;
  updated?: string;
  archived_at?: string;
  archivedAt?: string;
}

interface RawScope {
  targets?: string[];
  closure?: string[];
  completed?: string[];
  ready?: string[];
  blocked?: string[];
  total?: number;
  completed_count?: number;
  completedCount?: number;
  blocked_count?: number;
  blockedCount?: number;
  progress_pct?: number;
  progressPct?: number;
}

interface RawEta {
  p50_hours?: number;
  p50Hours?: number;
  p80_hours?: number;
  p80Hours?: number;
  p50_label?: string;
  p50Label?: string;
  p80_label?: string;
  p80Label?: string;
  basis?: string;
  basis_label?: string;
  basisLabel?: string;
  confidence?: string;
  remaining_items?: number;
  remainingItems?: number;
  lane_capacity?: number;
  laneCapacity?: number;
}

interface RawGoalWithScope {
  goal?: RawGoal;
  scope?: RawScope;
  eta?: RawEta | null;
}

function normalizeSnapshot(raw: RawSnapshot): GoalScopeSnapshot {
  return {
    at: raw.at ?? "",
    targetCount: raw.targetCount ?? raw.target_count ?? 0,
    closureSize: raw.closureSize ?? raw.closure_size ?? 0,
    completed: raw.completed ?? 0,
  };
}

function normalizeGoal(raw: RawGoal): Goal {
  const history = raw.scopeHistory ?? raw.scope_history ?? [];
  const status = (raw.status === "archived" ? "archived" : "active") as GoalStatus;
  const archivedAt = raw.archivedAt ?? raw.archived_at;
  return {
    name: raw.name ?? "",
    title: raw.title ?? raw.name ?? "",
    description: raw.description ?? "",
    status,
    priority: raw.priority ?? 0,
    targets: raw.targets ?? [],
    seeded: raw.seeded ?? false,
    scopeHistory: history.map(normalizeSnapshot),
    created: raw.created ?? "",
    updated: raw.updated ?? "",
    ...(archivedAt ? { archivedAt } : {}),
  };
}

function normalizeScope(raw: RawScope | undefined): GoalScope {
  const s = raw ?? {};
  return {
    targets: s.targets ?? [],
    closure: s.closure ?? [],
    completed: s.completed ?? [],
    ready: s.ready ?? [],
    blocked: s.blocked ?? [],
    total: s.total ?? 0,
    completedCount: s.completedCount ?? s.completed_count ?? 0,
    blockedCount: s.blockedCount ?? s.blocked_count ?? 0,
    progressPct: s.progressPct ?? s.progress_pct ?? 0,
  };
}

function normalizeEta(raw: RawEta | null | undefined): PlanEtaBandData | null {
  if (!raw) return null;
  return {
    p50Hours: raw.p50Hours ?? raw.p50_hours ?? 0,
    p80Hours: raw.p80Hours ?? raw.p80_hours ?? 0,
    p50Label: raw.p50Label ?? raw.p50_label ?? "",
    p80Label: raw.p80Label ?? raw.p80_label ?? "",
    basis: raw.basis ?? "",
    basisLabel: raw.basisLabel ?? raw.basis_label ?? "",
    confidence: raw.confidence ?? "",
    remainingItems: raw.remainingItems ?? raw.remaining_items ?? 0,
    laneCapacity: raw.laneCapacity ?? raw.lane_capacity ?? 0,
  };
}

function normalizeWithScope(raw: RawGoalWithScope): GoalWithScope {
  return {
    goal: normalizeGoal(raw.goal ?? {}),
    scope: normalizeScope(raw.scope),
    eta: normalizeEta(raw.eta),
  };
}

export interface IGoalsService {
  list(): Promise<GoalWithScope[]>;
  get(name: string): Promise<GoalWithScope>;
  create(input: CreateGoalInput): Promise<GoalWithScope>;
  update(name: string, input: UpdateGoalInput): Promise<GoalWithScope>;
  setPriority(name: string, priority: number): Promise<GoalWithScope>;
  addTargets(name: string, targets: string[]): Promise<GoalWithScope>;
  removeTargets(name: string, targets: string[]): Promise<GoalWithScope>;
  archive(name: string): Promise<void>;
  remove(name: string): Promise<void>;
}

export function createGoalsService(apiClient: IApiClient = defaultApiClient): IGoalsService {
  return {
    async list(): Promise<GoalWithScope[]> {
      const resp = await apiClient.get<{ goals?: RawGoalWithScope[] } | RawGoalWithScope[]>(
        API_ENDPOINTS.goals,
      );
      const raw = Array.isArray(resp) ? resp : (resp.goals ?? []);
      return raw.map(normalizeWithScope);
    },

    async get(name: string): Promise<GoalWithScope> {
      const raw = await apiClient.get<RawGoalWithScope>(API_ENDPOINTS.goalByName(name));
      return normalizeWithScope(raw);
    },

    async create(input: CreateGoalInput): Promise<GoalWithScope> {
      const body: Record<string, unknown> = { title: input.title };
      if (input.name) body.name = input.name;
      if (input.description) body.description = input.description;
      if (input.priority !== undefined) body.priority = input.priority;
      if (input.targets) body.targets = input.targets;
      const raw = await apiClient.post<RawGoalWithScope>(API_ENDPOINTS.goals, body);
      return normalizeWithScope(raw);
    },

    async update(name: string, input: UpdateGoalInput): Promise<GoalWithScope> {
      const body: Record<string, unknown> = {};
      if (input.title !== undefined) body.title = input.title;
      if (input.description !== undefined) body.description = input.description;
      if (input.priority !== undefined) body.priority = input.priority;
      if (input.targets !== undefined) body.targets = input.targets;
      const raw = await apiClient.put<RawGoalWithScope>(API_ENDPOINTS.goalByName(name), body);
      return normalizeWithScope(raw);
    },

    async setPriority(name: string, priority: number): Promise<GoalWithScope> {
      return this.update(name, { priority });
    },

    async addTargets(name: string, targets: string[]): Promise<GoalWithScope> {
      const raw = await apiClient.post<RawGoalWithScope>(API_ENDPOINTS.goalTargets(name), { targets });
      return normalizeWithScope(raw);
    },

    async removeTargets(name: string, targets: string[]): Promise<GoalWithScope> {
      const raw = await apiClient.delete<RawGoalWithScope>(API_ENDPOINTS.goalTargets(name), { targets });
      return normalizeWithScope(raw);
    },

    async archive(name: string): Promise<void> {
      await apiClient.patch<unknown>(API_ENDPOINTS.goalArchiveItem(name), {});
    },

    async remove(name: string): Promise<void> {
      await apiClient.delete<unknown>(API_ENDPOINTS.goalByName(name));
    },
  };
}

export const goalsService = createGoalsService();
