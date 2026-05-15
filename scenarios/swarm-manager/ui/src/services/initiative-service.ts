/**
 * Initiative Service - Data access layer for initiative operations
 *
 * Encapsulates initiative-related API operations behind a clean seam.
 * Accepts an API client as a dependency for test injection.
 *
 * The backend returns JSON with proto field names (snake_case). This service
 * normalizes responses to match the generated TS types (camelCase).
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { AgentSessionAttribution, InitiativeWithRollup } from "../types";
import type { TreeFile } from "../components/ui/file-tree";

/** Raw API rollup shape (snake_case from proto JSON). */
interface RawRollup {
  total?: number;
  completed?: number;
  in_progress?: number;
  inProgress?: number;
  failed?: number;
  pending?: number;
  archived?: number;
}

/** Raw initiative shape (snake_case from proto JSON). */
interface RawInitiative {
  archived_at?: string;
  archivedAt?: string;
  acceptance_criteria?: string[];
  acceptanceCriteria?: string[];
  created_by?: RawAttribution;
  createdBy?: RawAttribution;
  depends_on?: string[];
  dependsOn?: string[];
  mode?: string;
  priority?: number;
  [key: string]: unknown;
}

interface RawAttribution {
  type?: string;
  run_id?: string;
  runId?: string;
  task_id?: string;
  taskId?: string;
  profile_key?: string;
  profileKey?: string;
  session_id?: string;
  sessionId?: string;
  session_kind?: string;
  sessionKind?: string;
  source?: string;
}

/** Normalize a single initiative-with-rollup from the API's snake_case to camelCase. */
function normalizeItem(
  raw: {
    initiative?: RawInitiative;
    rollup?: RawRollup;
    target_scenarios?: string[];
    targetScenarios?: string[];
  },
): InitiativeWithRollup {
  const rollup = raw.rollup ?? {};
  const initiative = raw.initiative ?? {};
  // Normalize snake_case fields from API to camelCase expected by TS types.
  const archivedAt = initiative.archivedAt ?? initiative.archived_at;
  const acceptanceCriteria = initiative.acceptanceCriteria ?? initiative.acceptance_criteria ?? [];
  const dependsOn = initiative.dependsOn ?? initiative.depends_on ?? [];
  const mode = initiative.mode ?? "item-level";
  const priority = initiative.priority ?? 0;
  const createdBy = normalizeAttribution(initiative.createdBy ?? initiative.created_by);
  const targetScenarios = raw.targetScenarios ?? raw.target_scenarios;
  return {
    ...raw,
    initiative: {
      ...initiative,
      acceptanceCriteria,
      mode,
      priority,
      dependsOn,
      ...(archivedAt ? { archivedAt } : {}),
      ...(createdBy ? { createdBy } : {}),
    },
    rollup: {
      ...rollup,
      // Accept both snake_case (API) and camelCase (already normalized)
      inProgress: rollup.inProgress ?? rollup.in_progress ?? 0,
    },
    ...(targetScenarios ? { targetScenarios } : {}),
  } as InitiativeWithRollup;
}

function normalizeAttribution(raw?: RawAttribution): AgentSessionAttribution | undefined {
  if (!raw) return undefined;
  const type = raw.type === "agent" ? "agent" : "operator";
  const sessionKind = raw.sessionKind ?? raw.session_kind;
  return {
    type,
    ...(raw.runId ?? raw.run_id ? { runId: raw.runId ?? raw.run_id } : {}),
    ...(raw.taskId ?? raw.task_id ? { taskId: raw.taskId ?? raw.task_id } : {}),
    ...(raw.profileKey ?? raw.profile_key ? { profileKey: raw.profileKey ?? raw.profile_key } : {}),
    ...(raw.sessionId ?? raw.session_id ? { sessionId: raw.sessionId ?? raw.session_id } : {}),
    ...(sessionKind === "meta_orchestration" || sessionKind === "operating_mode_authoring" || sessionKind === "swarm_operations" ? { sessionKind } : {}),
    ...(raw.source ? { source: raw.source } : {}),
  };
}

function normalizeItems(raw: unknown[]): InitiativeWithRollup[] {
  return raw.map((item) => normalizeItem(item as {
    initiative?: Record<string, unknown>;
    rollup?: RawRollup;
    target_scenarios?: string[];
    targetScenarios?: string[];
  }));
}

export interface IInitiativeService {
  list(): Promise<InitiativeWithRollup[]>;
  get(name: string): Promise<InitiativeWithRollup>;
  listFiles(name: string): Promise<TreeFile[]>;
  getFileContent(name: string, path: string): Promise<string>;
  updateNote(name: string, note: string): Promise<InitiativeWithRollup>;
  updateMetadata(name: string, patch: {
    acceptanceCriteria?: string[];
  }): Promise<InitiativeWithRollup>;
}

export function createInitiativeService(
  apiClient: IApiClient = defaultApiClient,
): IInitiativeService {
  return {
    async list(): Promise<InitiativeWithRollup[]> {
      const resp = await apiClient.get<{ items?: unknown[] } | unknown[]>(API_ENDPOINTS.initiatives);
      const raw = Array.isArray(resp) ? resp : (resp.items ?? []);
      return normalizeItems(raw);
    },

    async get(name: string): Promise<InitiativeWithRollup> {
      const raw = await apiClient.get<Record<string, unknown>>(API_ENDPOINTS.initiativeByName(name));
      return normalizeItem(raw as { initiative?: Record<string, unknown>; rollup?: RawRollup });
    },

    async listFiles(name: string): Promise<TreeFile[]> {
      const resp = await apiClient.get<{ files: TreeFile[] }>(API_ENDPOINTS.initiativeFiles(name));
      return resp.files ?? [];
    },

    async getFileContent(name: string, path: string): Promise<string> {
      return apiClient.get<string>(API_ENDPOINTS.initiativeFileContent(name, path), { responseType: "text" });
    },

    async updateNote(name: string, note: string): Promise<InitiativeWithRollup> {
      const raw = await apiClient.put<Record<string, unknown>>(
        API_ENDPOINTS.initiativeByName(name),
        { note },
      );
      return normalizeItem(raw as { initiative?: Record<string, unknown>; rollup?: RawRollup });
    },

    async updateMetadata(name: string, patch: {
      acceptanceCriteria?: string[];
    }): Promise<InitiativeWithRollup> {
      const body: Record<string, unknown> = {};
      if (patch.acceptanceCriteria !== undefined) {
        body.acceptance_criteria = patch.acceptanceCriteria;
      }
      const raw = await apiClient.put<Record<string, unknown>>(
        API_ENDPOINTS.initiativeByName(name),
        body,
      );
      return normalizeItem(raw as { initiative?: Record<string, unknown>; rollup?: RawRollup });
    },
  };
}

export const initiativeService = createInitiativeService();
