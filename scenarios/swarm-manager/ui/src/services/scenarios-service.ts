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
import type {
  Scenario,
  ScenarioFile,
  UpdateScenarioMetadataRequest,
  DeleteScenarioResponse,
  PreserveFilesRequest,
} from "../types";

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
 * Interface for the scenarios service.
 * This is the seam - implementations can be swapped for testing.
 * [REQ:REQ-P0-007] Includes updateMetadata for scenario metadata management
 * [REQ:REQ-P0-008] Includes delete for scenario deletion with archive option
 */
export interface IScenariosService {
  list(): Promise<Scenario[]>;
  get(name: string): Promise<Scenario>;
  getFiles(name: string): Promise<ScenarioFile[]>;
  updateMetadata(name: string, request: UpdateScenarioMetadataRequest): Promise<Scenario>;
  delete(name: string, options?: DeleteScenarioOptions): Promise<DeleteScenarioResponse>;
  start(name: string): Promise<Scenario>;
  stop(name: string): Promise<Scenario>;
  restart(name: string): Promise<Scenario>;
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
      const endpoint = archive
        ? `${API_ENDPOINTS.scenarioByName(name)}?archive=true`
        : API_ENDPOINTS.scenarioByName(name);

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
  };
}

/**
 * Default scenarios service instance.
 * Uses the default API client for production use.
 */
export const scenariosService = createScenariosService();
