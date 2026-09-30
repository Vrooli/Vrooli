/**
 * Goals Service — data access for goal operations.
 *
 * The goals backend is mux+JSON (snake_case). This service normalizes responses to the
 * camelCase types the UI consumes and exposes the create / update / target /
 * priority levers the goals UX drives.
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import {
  backlogFileOperationResponseSchema,
  backlogFileResponseSchema,
  backlogFilesResponseSchema,
  buildMessage,
  mapProtoBacklogFile,
  parseProtoResponse,
  requireProtoField,
  toProtoJson,
} from "./proto-contracts";
import { BacklogFileOperationRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import type {
  CreateGoalInput,
  Goal,
  GoalFile,
  GoalMilestone,
  GoalScope,
  GoalScopeEntities,
  GoalScopeSnapshot,
  GoalWithScope,
  UpdateGoalInput,
} from "../types/goal";
import type { BacklogItem } from "../types/backlog";
import type { BacklogFile } from "../types/backlog";
import type { FileOperationResult } from "./file-service-types";
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
  milestones?: RawGoalMilestone[];
  seeded?: boolean;
  scope_history?: RawSnapshot[];
  scopeHistory?: RawSnapshot[];
  created?: string;
  updated?: string;
  archived_at?: string;
  archivedAt?: string;
  verified_delivered_at?: string;
  verifiedDeliveredAt?: string;
}

interface RawGoalMilestone {
  name?: string;
  title?: string;
  description?: string;
  items?: string[];
  acceptance_criteria?: string[];
  acceptanceCriteria?: string[];
  depends_on?: string[];
  dependsOn?: string[];
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

/** Raw backlog item as the goals backend embeds it (Go snake_case JSON). */
interface RawScopeItem {
  name?: string;
  title?: string;
  description?: string;
  status?: string;
  priority?: number;
  tags?: string[];
  created?: string;
  updated?: string;
  kind?: string;
  depends_on?: string[];
  dependsOn?: string[];
	milestone?: string;
  effort?: string;
  note?: string;
  archived_at?: string;
  archivedAt?: string;
}

interface RawScopeEntities {
  items?: Record<string, RawScopeItem>;
}

interface RawGoalWithScope {
  goal?: RawGoal;
  scope?: RawScope;
  eta?: RawEta | null;
  scope_entities?: RawScopeEntities;
  scopeEntities?: RawScopeEntities;
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
  const status = raw.status === "archived" || raw.status === "achieved" ? raw.status : "active";
  const archivedAt = raw.archivedAt ?? raw.archived_at;
  const verifiedDeliveredAt = raw.verifiedDeliveredAt ?? raw.verified_delivered_at;
  return {
    name: raw.name ?? "",
    title: raw.title ?? raw.name ?? "",
    description: raw.description ?? "",
    status,
    priority: raw.priority ?? 0,
    targets: raw.targets ?? [],
    milestones: (raw.milestones ?? []).map(normalizeMilestone),
    seeded: raw.seeded ?? false,
    scopeHistory: history.map(normalizeSnapshot),
    created: raw.created ?? "",
    updated: raw.updated ?? "",
    ...(archivedAt ? { archivedAt } : {}),
    ...(verifiedDeliveredAt ? { verifiedDeliveredAt } : {}),
  };
}

function normalizeMilestone(raw: RawGoalMilestone): GoalMilestone {
  const archivedAt = raw.archivedAt ?? raw.archived_at;
  return {
    name: raw.name ?? "",
    title: raw.title ?? raw.name ?? "",
    description: raw.description ?? "",
    items: raw.items ?? [],
    acceptanceCriteria: raw.acceptanceCriteria ?? raw.acceptance_criteria ?? [],
    dependsOn: raw.dependsOn ?? raw.depends_on ?? [],
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

function normalizeScopeItem(raw: RawScopeItem): BacklogItem {
  const archivedAt = raw.archivedAt ?? raw.archived_at;
  return {
    name: raw.name ?? "",
    title: raw.title ?? raw.name ?? "",
    description: raw.description ?? "",
    status: (raw.status ?? "backlog") as BacklogItem["status"],
    priority: raw.priority ?? 0,
    tags: raw.tags ?? [],
    created: raw.created ?? "",
    updated: raw.updated ?? "",
    kind: (raw.kind ?? "execute") as BacklogItem["kind"],
    dependsOn: raw.dependsOn ?? raw.depends_on ?? [],
	...(raw.milestone ? { milestone: raw.milestone } : {}),
    ...(raw.effort ? { effort: raw.effort } : {}),
    ...(raw.note ? { note: raw.note } : {}),
    ...(archivedAt ? { archivedAt } : {}),
  } as BacklogItem;
}

function normalizeScopeEntities(raw: RawScopeEntities | undefined): GoalScopeEntities | undefined {
  if (!raw) return undefined;
  const items: GoalScopeEntities["items"] = {};
  for (const [ref, item] of Object.entries(raw.items ?? {})) {
    items[ref] = normalizeScopeItem(item);
  }
  if (Object.keys(items).length === 0) return undefined;
  return { items };
}

function normalizeWithScope(raw: RawGoalWithScope): GoalWithScope {
  const scopeEntities = normalizeScopeEntities(raw.scopeEntities ?? raw.scope_entities);
  return {
    goal: normalizeGoal(raw.goal ?? {}),
    scope: normalizeScope(raw.scope),
    eta: normalizeEta(raw.eta),
    ...(scopeEntities ? { scopeEntities } : {}),
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
  createMilestone(name: string, milestone: GoalMilestone): Promise<GoalWithScope>;
  updateMilestone(name: string, milestone: GoalMilestone): Promise<GoalWithScope>;
  archiveMilestone(name: string, milestone: string): Promise<GoalWithScope>;
  assignMilestoneItems(name: string, milestone: string, items: string[]): Promise<GoalWithScope>;
  unassignMilestoneItems(name: string, milestone: string, items: string[]): Promise<GoalWithScope>;
  archive(name: string): Promise<void>;
  closeOut(name: string): Promise<void>;
  remove(name: string): Promise<void>;
  startPlan(name: string): Promise<{ execution_id: string; run_id?: string; definition_digest: string }>;
  startDiscover(name: string): Promise<{ execution_id: string; run_id?: string; definition_digest: string }>;
  startMilestoneReview(name: string, milestone: string): Promise<{ execution_id: string; run_id?: string; definition_digest: string }>;
  getFiles(name: string): Promise<GoalFile[]>;
  getFileContent(name: string, filePath: string): Promise<string>;
  uploadFile(name: string, file: File, path?: string): Promise<BacklogFile>;
  saveFileContent(name: string, filePath: string, content: string, contentType?: string): Promise<BacklogFile>;
  renameFile(name: string, sourcePath: string, destinationPath: string): Promise<FileOperationResult>;
  moveFile(name: string, sourcePath: string, destinationPath: string): Promise<FileOperationResult>;
  copyFile(name: string, sourcePath: string, destinationPath: string): Promise<FileOperationResult>;
  deleteFile(name: string, sourcePath: string): Promise<FileOperationResult>;
}

export function createGoalsService(apiClient: IApiClient = defaultApiClient): IGoalsService {
  const uploadGoalFile = async (name: string, file: File, path?: string): Promise<BacklogFile> => {
    const formData = new FormData();
    formData.append("file", file);
    if (path) formData.append("path", path);
    const data = await apiClient.post<unknown>(API_ENDPOINTS.goalFiles(name), formData, { headers: {} });
    const parsed = parseProtoResponse(backlogFileResponseSchema, data, "goal file");
    return mapProtoBacklogFile(requireProtoField(parsed.file, "goal file"));
  };
  const operateGoalFile = async (
    name: string,
    operation: "rename" | "move" | "copy" | "delete",
    sourcePath: string,
    destinationPath?: string,
  ): Promise<FileOperationResult> => {
    const message = buildMessage(BacklogFileOperationRequestSchema, {
      operation,
      sourcePath,
      ...(destinationPath ? { destinationPath } : {}),
    });
    const data = await apiClient.patch<unknown>(
      API_ENDPOINTS.goalFileOperations(name),
      toProtoJson(BacklogFileOperationRequestSchema, message),
    );
    const parsed = parseProtoResponse(backlogFileOperationResponseSchema, data, "goal file operation");
    return {
      ...(parsed.file ? { file: mapProtoBacklogFile(parsed.file) } : {}),
      ...(parsed.deletedPath ? { deletedPath: parsed.deletedPath } : {}),
    };
  };

  return {
    async list(): Promise<GoalWithScope[]> {
      const resp = await apiClient.get<
        { items?: RawGoalWithScope[]; goals?: RawGoalWithScope[] } | RawGoalWithScope[]
      >(API_ENDPOINTS.goals);
	  // The API envelopes the list under `items`.
      // Keep `goals` + bare-array fallbacks for forward/backward compatibility.
      const raw = Array.isArray(resp) ? resp : (resp.items ?? resp.goals ?? []);
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

    async createMilestone(name: string, milestone: GoalMilestone): Promise<GoalWithScope> {
      return normalizeWithScope(await apiClient.post<RawGoalWithScope>(API_ENDPOINTS.goalMilestones(name), milestone));
    },
    async updateMilestone(name: string, milestone: GoalMilestone): Promise<GoalWithScope> {
      return normalizeWithScope(await apiClient.put<RawGoalWithScope>(API_ENDPOINTS.goalMilestone(name, milestone.name), milestone));
    },
    async archiveMilestone(name: string, milestone: string): Promise<GoalWithScope> {
      return normalizeWithScope(await apiClient.delete<RawGoalWithScope>(API_ENDPOINTS.goalMilestone(name, milestone)));
    },
    async assignMilestoneItems(name: string, milestone: string, items: string[]): Promise<GoalWithScope> {
      return normalizeWithScope(await apiClient.post<RawGoalWithScope>(API_ENDPOINTS.goalMilestoneItems(name, milestone), { targets: items }));
    },
    async unassignMilestoneItems(name: string, milestone: string, items: string[]): Promise<GoalWithScope> {
      return normalizeWithScope(await apiClient.delete<RawGoalWithScope>(API_ENDPOINTS.goalMilestoneItems(name, milestone), { targets: items }));
    },

    async archive(name: string): Promise<void> {
      await apiClient.patch<unknown>(API_ENDPOINTS.goalArchiveItem(name), {});
    },

    async closeOut(name: string): Promise<void> {
      await apiClient.post<unknown>(API_ENDPOINTS.goalCloseOut(name), {});
    },

    async remove(name: string): Promise<void> {
      await apiClient.delete<unknown>(API_ENDPOINTS.goalByName(name));
    },

    startPlan(name: string) {
      return apiClient.post<{ execution_id: string; run_id?: string; definition_digest: string }>(API_ENDPOINTS.goalPlanRun(name), {});
    },

    startDiscover(name: string) {
      return apiClient.post<{ execution_id: string; run_id?: string; definition_digest: string }>(API_ENDPOINTS.goalDiscoverRun(name), {});
    },

    startMilestoneReview(name: string, milestone: string) {
      return apiClient.post<{ execution_id: string; run_id?: string; definition_digest: string }>(API_ENDPOINTS.goalMilestoneReviewRun(name, milestone), {});
    },

    async getFiles(name: string): Promise<GoalFile[]> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.goalFiles(name));
      const parsed = parseProtoResponse(backlogFilesResponseSchema, data, "goal files");
      return parsed.files.map(mapProtoBacklogFile);
    },

    getFileContent(name: string, filePath: string) {
      return apiClient.get<string>(API_ENDPOINTS.goalFileContent(name, filePath), { responseType: "text" });
    },

    uploadFile: uploadGoalFile,

    async saveFileContent(name: string, filePath: string, content: string, contentType = "text/plain") {
      const normalizedPath = filePath.replace(/^[\\/]+/, "");
      const segments = normalizedPath.split("/");
      const fileName = segments.pop() || "notes.txt";
      return uploadGoalFile(name, new File([content], fileName, { type: contentType }), segments.length > 0 ? segments.join("/") : undefined);
    },

    renameFile: (name, sourcePath, destinationPath) => operateGoalFile(name, "rename", sourcePath, destinationPath),
    moveFile: (name, sourcePath, destinationPath) => operateGoalFile(name, "move", sourcePath, destinationPath),
    copyFile: (name, sourcePath, destinationPath) => operateGoalFile(name, "copy", sourcePath, destinationPath),
    deleteFile: (name, sourcePath) => operateGoalFile(name, "delete", sourcePath),
  };
}

export const goalsService = createGoalsService();
