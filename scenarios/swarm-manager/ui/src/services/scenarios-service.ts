/**
 * Scenarios Service - Data access layer for scenario operations
 *
 * This service encapsulates all scenario-related API operations behind a clean seam.
 * It accepts an API client as a dependency, making it easy to substitute for testing.
 *
 * Responsibilities:
 * - Scenario listing and details
 * - Scenario metadata updates
 * - Scenario lifecycle actions (start/stop/restart)
 * - Request/response transformation if needed
 *
 * NOT responsible for:
 * - HTTP implementation details (delegated to api client)
 * - UI state or caching (delegated to React Query)
 */

import { UpdateScenarioMetadataRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/scenarios_pb";
import {
  buildMessage,
  deleteScenarioResponseSchema,
  DeleteScenarioRequestSchema,
  PreserveFilesRequestSchema,
  SpecSyncArchiveRequestSchema,
  PreviewScenarioRemediationRequestSchema,
  ApplyScenarioRemediationRequestSchema,
  PreviewScenarioMaturityCampaignRequestSchema,
  ApplyScenarioMaturityCampaignRequestSchema,
  previewScenarioRemediationResponseSchema,
  applyScenarioRemediationResponseSchema,
  previewScenarioMaturityCampaignResponseSchema,
  applyScenarioMaturityCampaignResponseSchema,
  specSyncArchiveResponseSchema,
  mapSpecSyncArchiveResponse,
  listScenariosResponseSchema,
  mapDeleteScenarioResponse,
  mapProtoScenario,
  mapProtoScenarioFile,
  parseProtoResponse,
  requireProtoField,
  scenarioFilesResponseSchema,
  scenarioResponseSchema,
  toProtoJson,
} from "./proto-contracts";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { buildQueryString } from "../lib/query-utils";
import type {
  Scenario,
  ScenarioFile,
  UpdateScenarioMetadataRequest,
  DeleteScenarioResponse,
  SpecSyncArchiveResponse,
  PreserveFilesRequest,
} from "../types";

/**
 * Rollup counts for a scenario coverage view (combined across goal scope
 * items and unassigned items targeting the scenario).
 */
export interface ScenarioContextRollup {
  total: number;
  completed: number;
  inProgress: number;
  failed: number;
  pending: number;
  archived: number;
}

/**
 * A goal whose derived scope targets a scenario, as returned by the coverage view.
 * Only the fields needed by the UI are exposed.
 */
export interface ScenarioContextGoal {
  name: string;
  title: string;
  status: string;
  priority: number;
  rollup: ScenarioContextRollup;
}

/**
 * A backlog item targeting a scenario but not in any goal's derived scope.
 */
export interface ScenarioContextOrphanItem {
  kind: string;
  name: string;
  title: string;
  status: string;
  priority: number;
  archivedAt?: string;
}

/**
 * A fix backlog item targeting a scenario, surfaced for the Fix History
 * section. Includes both goal-scope and unassigned fixes.
 */
export interface ScenarioFix {
  name: string;
  title: string;
  status: string;
  priority: number;
  goal?: string;
  updated?: string;
  archivedAt?: string;
  path: string;
}

/**
 * Active vs archived partition of a scenario's fix history.
 */
export interface ScenarioFixHistory {
  active: ScenarioFix[];
  archived: ScenarioFix[];
}

/**
 * Full coverage view for a scenario: every goal whose derived scope
 * target the scenario, every orphan backlog item targeting the scenario,
 * and a combined completion rollup.
 */
export interface ScenarioContext {
  scenarioName: string;
  goals: ScenarioContextGoal[];
  orphanItems: ScenarioContextOrphanItem[];
  rollup: ScenarioContextRollup;
  fixes: ScenarioFixHistory;
}

/**
 * Options for deleting a scenario
 */
export interface DeleteScenarioOptions {
  /** Whether to archive the scenario to backlog (idea) */
  archive?: boolean;
  /** Files to preserve when archiving */
  preserveFiles?: PreserveFilesRequest;
}

/**
 * Options for spec-sync-archive
 */
export interface SpecSyncArchiveOptions {
  /** Files to preserve when archiving */
  preserveFiles?: PreserveFilesRequest;
}

/**
 * Interface for the scenarios service.
 * This is the seam - implementations can be swapped for testing.
 * [REQ:REQ-P0-007] Includes updateMetadata for scenario metadata management
 * [REQ:REQ-P0-008] Includes delete for scenario deletion with archive option
 */
export interface IScenariosService {
  list(): Promise<Scenario[]>;
  get(name: string): Promise<Scenario>;
  getContext(name: string): Promise<ScenarioContext>;
  getFiles(name: string): Promise<ScenarioFile[]>;
  updateMetadata(name: string, request: UpdateScenarioMetadataRequest): Promise<Scenario>;
  delete(name: string, options?: DeleteScenarioOptions): Promise<DeleteScenarioResponse>;
  specSyncArchive(name: string, options?: SpecSyncArchiveOptions): Promise<SpecSyncArchiveResponse>;
  start(name: string): Promise<Scenario>;
  stop(name: string): Promise<Scenario>;
  restart(name: string): Promise<Scenario>;
  previewRemediation(name: string, target: ScenarioRemediationTarget): Promise<ScenarioRemediationPreview>;
  applyRemediation(name: string, target: ScenarioRemediationTarget, fingerprint: string): Promise<ScenarioRemediationApplyResult>;
  previewMaturityCampaign(name: string, target: ScenarioMaturityCampaignTarget): Promise<ScenarioMaturityCampaignPreview>;
  applyMaturityCampaign(name: string, target: ScenarioMaturityCampaignTarget, fingerprint: string): Promise<ScenarioMaturityCampaignApplyResult>;
}

export interface ScenarioRemediationTarget { scenarioName: string; providerPhase: string; capabilityId: string; }
export interface ScenarioRemediationProposal { target: ScenarioRemediationTarget; fingerprint: string; title: string; description: string; acceptanceCriteria: string[]; acceptanceAllow: string[]; recommendedWorkflows: string[]; }
export interface ScenarioRemediationPreview { proposal: ScenarioRemediationProposal; existing?: { state: string; workRef?: string }; }
export interface ScenarioRemediationApplyResult { proposal: ScenarioRemediationProposal; workRef: string; created: boolean; }
export interface ScenarioMaturityCampaignTarget { scenarioName: string; maturityTarget: string; providerPhases: string[]; }
export interface ScenarioMaturityCampaignProposal { target: ScenarioMaturityCampaignTarget; fingerprint: string; title: string; description: string; acceptanceCriteria: string[]; declaredWorkflow: string; trackerAvailability: string; trackerRef?: string; }
export interface ScenarioMaturityCampaignPreview { proposal: ScenarioMaturityCampaignProposal; existingGoalRef?: string; }
export interface ScenarioMaturityCampaignApplyResult { proposal: ScenarioMaturityCampaignProposal; goalRef: string; created: boolean; trackerAvailability: string; trackerRef?: string; }

function mapRemediationProposal(value: { target?: ScenarioRemediationTarget; fingerprint: string; title: string; description: string; acceptanceCriteria: string[]; acceptanceAllow: string[]; recommendedWorkflows: string[] }): ScenarioRemediationProposal { return { ...value, target: requireProtoField(value.target, "scenario remediation target") }; }
function mapCampaignProposal(value: { target?: ScenarioMaturityCampaignTarget; fingerprint: string; title: string; description: string; acceptanceCriteria: string[]; declaredWorkflow: string; trackerAvailability: string; trackerRef?: string }): ScenarioMaturityCampaignProposal { return { ...value, target: requireProtoField(value.target, "scenario maturity campaign target") }; }

interface RawRollup {
  total?: number;
  completed?: number;
  in_progress?: number;
  inProgress?: number;
  failed?: number;
  pending?: number;
  archived?: number;
}

interface RawScenarioContextGoal {
  name?: string;
  title?: string;
  status?: string;
  priority?: number;
  scope?: RawRollup;
}

interface RawScenarioContextOrphan {
  kind?: string;
  name?: string;
  title?: string;
  status?: string;
  priority?: number;
  archived_at?: string;
  archivedAt?: string;
}

interface RawScenarioFix {
  name?: string;
  title?: string;
  status?: string;
  priority?: number;
  goal?: string;
  updated?: string;
  archived_at?: string;
  archivedAt?: string;
  path?: string;
}

interface RawScenarioFixHistory {
  active?: RawScenarioFix[];
  archived?: RawScenarioFix[];
}

interface RawScenarioContext {
  scenario_name?: string;
  scenarioName?: string;
  goals?: RawScenarioContextGoal[];
  orphan_items?: RawScenarioContextOrphan[];
  orphanItems?: RawScenarioContextOrphan[];
  rollup?: RawRollup;
  fixes?: RawScenarioFixHistory;
}

function normalizeFix(raw: RawScenarioFix): ScenarioFix {
  return {
    name: raw.name ?? "",
    title: raw.title ?? "",
    status: raw.status ?? "",
    priority: raw.priority ?? 0,
    goal: raw.goal,
    updated: raw.updated,
    archivedAt: raw.archivedAt ?? raw.archived_at,
    path: raw.path ?? "",
  };
}

function normalizeRollup(raw: RawRollup | undefined): ScenarioContextRollup {
  const r = raw ?? {};
  return {
    total: r.total ?? 0,
    completed: r.completed ?? 0,
    inProgress: r.inProgress ?? r.in_progress ?? 0,
    failed: r.failed ?? 0,
    pending: r.pending ?? 0,
    archived: r.archived ?? 0,
  };
}

function normalizeScenarioContext(raw: RawScenarioContext): ScenarioContext {
  return {
    scenarioName: raw.scenarioName ?? raw.scenario_name ?? "",
    goals: (raw.goals ?? []).map((goal) => ({
      name: goal.name ?? "",
      title: goal.title ?? "",
      status: goal.status ?? "",
      priority: goal.priority ?? 0,
      rollup: normalizeRollup(goal.scope),
    })),
    orphanItems: (raw.orphanItems ?? raw.orphan_items ?? []).map((o) => ({
      kind: o.kind ?? "",
      name: o.name ?? "",
      title: o.title ?? "",
      status: o.status ?? "",
      priority: o.priority ?? 0,
      archivedAt: o.archivedAt ?? o.archived_at,
    })),
    rollup: normalizeRollup(raw.rollup),
    fixes: {
      active: (raw.fixes?.active ?? []).map(normalizeFix),
      archived: (raw.fixes?.archived ?? []).map(normalizeFix),
    },
  };
}

/**
 * Creates a scenarios service with the given API client.
 *
 * @param apiClient - The API client to use for HTTP requests
 * @returns A scenarios service instance
 */
export function createScenariosService(
  apiClient: IApiClient = defaultApiClient
): IScenariosService {
  return {
    async list(): Promise<Scenario[]> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.scenarios);
      const parsed = parseProtoResponse(listScenariosResponseSchema, data, "scenarios list");
      return parsed.scenarios.map(mapProtoScenario);
    },

    async get(name: string): Promise<Scenario> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.scenarioByName(name));
      const parsed = parseProtoResponse(scenarioResponseSchema, data, "scenario");
      return mapProtoScenario(requireProtoField(parsed.scenario, "scenario"));
    },

    async getContext(name: string): Promise<ScenarioContext> {
      const raw = await apiClient.get<RawScenarioContext>(API_ENDPOINTS.scenarioContext(name));
      return normalizeScenarioContext(raw);
    },

    /**
     * Gets the file tree for a scenario
     */
    async getFiles(name: string): Promise<ScenarioFile[]> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.scenarioFiles(name));
      const parsed = parseProtoResponse(scenarioFilesResponseSchema, data, "scenario files");
      return parsed.files.map(mapProtoScenarioFile);
    },

    /**
     * Updates scenario metadata (greenfield toggle)
     * [REQ:REQ-P0-007] PATCH endpoint for scenario metadata management
     */
    async updateMetadata(name: string, request: UpdateScenarioMetadataRequest): Promise<Scenario> {
      const message = buildMessage(UpdateScenarioMetadataRequestSchema, {
        isGreenfield: request.isGreenfield,
      });
      const payload = toProtoJson(UpdateScenarioMetadataRequestSchema, message);
      const data = await apiClient.patch<unknown>(API_ENDPOINTS.scenarioByName(name), payload);
      const parsed = parseProtoResponse(scenarioResponseSchema, data, "scenario");
      return mapProtoScenario(requireProtoField(parsed.scenario, "scenario"));
    },

    /**
     * Deletes a scenario with optional archive to backlog (idea)
     * [REQ:REQ-P0-008] DELETE endpoint for scenario deletion with safeguards
     */
    async delete(name: string, options: DeleteScenarioOptions = {}): Promise<DeleteScenarioResponse> {
      const { archive = false, preserveFiles } = options;
      const qs = buildQueryString({ archive: archive ? "true" : undefined });
      const endpoint = `${API_ENDPOINTS.scenarioByName(name)}${qs}`;

      // Build request body if preserveFiles is specified
      let body: unknown = undefined;
      if (preserveFiles && (preserveFiles.paths?.length || preserveFiles.preset)) {
        const preserveFilesMsg = buildMessage(PreserveFilesRequestSchema, {
          paths: preserveFiles.paths ?? [],
          preset: preserveFiles.preset,
        });
        const requestMsg = buildMessage(DeleteScenarioRequestSchema, {
          preserveFiles: preserveFilesMsg,
        });
        body = toProtoJson(DeleteScenarioRequestSchema, requestMsg);
      }

      const data = body
        ? await apiClient.delete<unknown>(endpoint, body)
        : await apiClient.delete<unknown>(endpoint);
      const parsed = parseProtoResponse(deleteScenarioResponseSchema, data, "scenario delete");
      return mapDeleteScenarioResponse(parsed);
    },

    /**
     * Triggers spec-sync agent followed by archive on completion.
     * Returns an execution ID for polling progress.
     */
    async specSyncArchive(name: string, options: SpecSyncArchiveOptions = {}): Promise<SpecSyncArchiveResponse> {
      const { preserveFiles } = options;
      const endpoint = API_ENDPOINTS.scenarioSpecSyncArchive(name);

      let body: unknown = undefined;
      if (preserveFiles && (preserveFiles.paths?.length || preserveFiles.preset)) {
        const preserveFilesMsg = buildMessage(PreserveFilesRequestSchema, {
          paths: preserveFiles.paths ?? [],
          preset: preserveFiles.preset,
        });
        const requestMsg = buildMessage(SpecSyncArchiveRequestSchema, {
          preserveFiles: preserveFilesMsg,
        });
        body = toProtoJson(SpecSyncArchiveRequestSchema, requestMsg);
      }

      const data = body
        ? await apiClient.post<unknown>(endpoint, body)
        : await apiClient.post<unknown>(endpoint, {});
      const parsed = parseProtoResponse(specSyncArchiveResponseSchema, data, "spec-sync-archive");
      return mapSpecSyncArchiveResponse(parsed);
    },

    async start(name: string): Promise<Scenario> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.scenarioStart(name), {});
      const parsed = parseProtoResponse(scenarioResponseSchema, data, "scenario");
      return mapProtoScenario(requireProtoField(parsed.scenario, "scenario"));
    },

    async stop(name: string): Promise<Scenario> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.scenarioStop(name), {});
      const parsed = parseProtoResponse(scenarioResponseSchema, data, "scenario");
      return mapProtoScenario(requireProtoField(parsed.scenario, "scenario"));
    },

    async restart(name: string): Promise<Scenario> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.scenarioRestart(name), {});
      const parsed = parseProtoResponse(scenarioResponseSchema, data, "scenario");
      return mapProtoScenario(requireProtoField(parsed.scenario, "scenario"));
    },

    async previewRemediation(name, target) {
      const request = buildMessage(PreviewScenarioRemediationRequestSchema, { target });
      const data = await apiClient.post<unknown>(API_ENDPOINTS.scenarioRemediationPreview(name), toProtoJson(PreviewScenarioRemediationRequestSchema, request));
      const parsed = parseProtoResponse(previewScenarioRemediationResponseSchema, data, "scenario remediation preview");
      const proposal = requireProtoField(parsed.proposal, "scenario remediation proposal");
      return { proposal: mapRemediationProposal(proposal), existing: parsed.existing ? { state: parsed.existing.state, workRef: parsed.existing.workRef } : undefined };
    },

    async applyRemediation(name, target, fingerprint) {
      const request = buildMessage(ApplyScenarioRemediationRequestSchema, { target, fingerprint });
      const data = await apiClient.post<unknown>(API_ENDPOINTS.scenarioRemediationApply(name), toProtoJson(ApplyScenarioRemediationRequestSchema, request));
      const parsed = parseProtoResponse(applyScenarioRemediationResponseSchema, data, "scenario remediation apply");
      return { proposal: mapRemediationProposal(requireProtoField(parsed.proposal, "scenario remediation proposal")), workRef: parsed.workRef, created: parsed.created };
    },

    async previewMaturityCampaign(name, target) {
      const request = buildMessage(PreviewScenarioMaturityCampaignRequestSchema, { target });
      const data = await apiClient.post<unknown>(API_ENDPOINTS.scenarioMaturityCampaignPreview(name), toProtoJson(PreviewScenarioMaturityCampaignRequestSchema, request));
      const parsed = parseProtoResponse(previewScenarioMaturityCampaignResponseSchema, data, "scenario maturity campaign preview");
      return { proposal: mapCampaignProposal(requireProtoField(parsed.proposal, "scenario maturity campaign proposal")), existingGoalRef: parsed.existingGoalRef };
    },

    async applyMaturityCampaign(name, target, fingerprint) {
      const request = buildMessage(ApplyScenarioMaturityCampaignRequestSchema, { target, fingerprint });
      const data = await apiClient.post<unknown>(API_ENDPOINTS.scenarioMaturityCampaignApply(name), toProtoJson(ApplyScenarioMaturityCampaignRequestSchema, request));
      const parsed = parseProtoResponse(applyScenarioMaturityCampaignResponseSchema, data, "scenario maturity campaign apply");
      return { proposal: mapCampaignProposal(requireProtoField(parsed.proposal, "scenario maturity campaign proposal")), goalRef: parsed.goalRef, created: parsed.created, trackerAvailability: parsed.trackerAvailability, trackerRef: parsed.trackerRef };
    },
  };
}

/**
 * Default scenarios service instance.
 * Uses the default API client for production use.
 */
export const scenariosService = createScenariosService();
