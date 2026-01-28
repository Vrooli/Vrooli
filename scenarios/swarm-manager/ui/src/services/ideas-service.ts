/**
 * Ideas Service - Data access layer for idea operations
 *
 * This service encapsulates all idea-related API operations behind a clean seam.
 * It accepts an API client as a dependency, making it easy to substitute for testing.
 *
 * Responsibilities:
 * - Idea CRUD operations
 * - Request/response transformation if needed
 *
 * NOT responsible for:
 * - HTTP implementation details (delegated to api client)
 * - UI state or caching (delegated to React Query)
 * - Domain validation (delegated to API)
 *
 * DOC: docs/internal/SEAMS.md#ui-to-api-seam-improved-in-phase-3
 * DOC: docs/internal/INTENT.md#key-flows
 */

import { CreateIdeaRequestSchema, QueueIdeaRequestSchema, UpdateIdeaRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/ideas_pb";
import { buildMessage, ideaFilesResponseSchema, ideaFileResponseSchema, ideaResponseSchema, listIdeasResponseSchema, mapProtoIdea, mapProtoIdeaFile, parseProtoResponse, queueIdeaResponseSchema, requireProtoField, toProtoJson } from "./proto-contracts";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Idea, IdeaFile, ResearchResponse } from "../types";

/**
 * Response from queueing an idea for processing.
 */
export interface QueueResponse {
  idea: Idea;
  taskId: string;
}

/**
 * Interface for the ideas service.
 * This is the seam - implementations can be swapped for testing.
 */
export interface IIdeasService {
  list(): Promise<Idea[]>;
  get(name: string): Promise<Idea>;
  create(idea: Omit<Idea, "created" | "updated">): Promise<Idea>;
  update(
    name: string,
    idea: Pick<Idea, "title" | "description" | "status" | "priority" | "tags">
  ): Promise<Idea>;
  delete(name: string): Promise<void>;
  getFiles(name: string): Promise<IdeaFile[]>;
  getFileContent(name: string, filePath: string): Promise<string>;
  uploadFile(name: string, file: File, path?: string): Promise<IdeaFile>;
  queue(name: string, operation?: "generator" | "improver"): Promise<QueueResponse>;
  research(
    name: string,
    payload?: { prompt?: string; scopePath?: string; projectRoot?: string }
  ): Promise<ResearchResponse>;
}

/**
 * Creates an ideas service with the given API client.
 *
 * @param apiClient - The API client to use for HTTP requests
 * @returns An ideas service instance
 *
 * @example
 * // Production usage (uses default client)
 * const service = createIdeasService();
 *
 * // Testing usage (uses mock client)
 * const mockClient = { get: vi.fn(), post: vi.fn(), ... };
 * const service = createIdeasService(mockClient);
 */
export function createIdeasService(
  apiClient: IApiClient = defaultApiClient
): IIdeasService {
  return {
    async list(): Promise<Idea[]> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.ideas);
      const parsed = parseProtoResponse(listIdeasResponseSchema, data, "ideas list");
      return parsed.ideas.map(mapProtoIdea);
    },

    async get(name: string): Promise<Idea> {
      const data = await apiClient.get<unknown>(`${API_ENDPOINTS.ideas}/${name}`);
      const parsed = parseProtoResponse(ideaResponseSchema, data, "idea");
      return mapProtoIdea(requireProtoField(parsed.idea, "idea"));
    },

    async create(idea: Omit<Idea, "created" | "updated">): Promise<Idea> {
      const message = buildMessage(CreateIdeaRequestSchema, {
        name: idea.name,
        title: idea.title,
        description: idea.description || undefined,
        priority: idea.priority || undefined,
        tags: idea.tags,
      });
      const payload = toProtoJson(CreateIdeaRequestSchema, message);
      const data = await apiClient.post<unknown>(API_ENDPOINTS.ideas, payload);
      const parsed = parseProtoResponse(ideaResponseSchema, data, "idea");
      return mapProtoIdea(requireProtoField(parsed.idea, "idea"));
    },

    async update(
      name: string,
      idea: Pick<Idea, "title" | "description" | "status" | "priority" | "tags">
    ): Promise<Idea> {
      const message = buildMessage(UpdateIdeaRequestSchema, {
        title: idea.title,
        description: idea.description,
        status: idea.status,
        priority: idea.priority,
        tags: idea.tags,
      });
      const payload = toProtoJson(UpdateIdeaRequestSchema, message);
      const data = await apiClient.put<unknown>(`${API_ENDPOINTS.ideas}/${name}`, payload);
      const parsed = parseProtoResponse(ideaResponseSchema, data, "idea");
      return mapProtoIdea(requireProtoField(parsed.idea, "idea"));
    },

    async delete(name: string): Promise<void> {
      return apiClient.delete<void>(`${API_ENDPOINTS.ideas}/${name}`);
    },

    async getFiles(name: string): Promise<IdeaFile[]> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.ideaFiles(name));
      const parsed = parseProtoResponse(ideaFilesResponseSchema, data, "idea files");
      return parsed.files.map(mapProtoIdeaFile);
    },

    async getFileContent(name: string, filePath: string): Promise<string> {
      return apiClient.get<string>(API_ENDPOINTS.ideaFileContent(name, filePath), {
        responseType: "text",
      });
    },

    async uploadFile(name: string, file: File, path?: string): Promise<IdeaFile> {
      const formData = new FormData();
      formData.append("file", file);
      if (path) {
        formData.append("path", path);
      }
      const data = await apiClient.post<unknown>(API_ENDPOINTS.ideaFiles(name), formData, {
        headers: {},
      });
      const parsed = parseProtoResponse(ideaFileResponseSchema, data, "idea file");
      return mapProtoIdeaFile(requireProtoField(parsed.file, "idea file"));
    },

    async queue(
      name: string,
      operation: "generator" | "improver" = "generator"
    ): Promise<QueueResponse> {
      const message = buildMessage(QueueIdeaRequestSchema, { operation });
      const payload = toProtoJson(QueueIdeaRequestSchema, message);
      const data = await apiClient.post<unknown>(API_ENDPOINTS.ideaQueue(name), payload);
      const parsed = parseProtoResponse(queueIdeaResponseSchema, data, "idea queue");
      const idea = requireProtoField(parsed.idea, "idea queue");
      return {
        idea: mapProtoIdea(idea),
        taskId: parsed.taskId ?? "",
      };
    },

    async research(
      name: string,
      payload?: { prompt?: string; scopePath?: string; projectRoot?: string }
    ): Promise<ResearchResponse> {
      const data = await apiClient.post<unknown>(
        API_ENDPOINTS.ideaResearch(name),
        payload ?? {}
      );
      const parsed = data as ResearchResponse;
      if (!parsed || typeof parsed.runId !== "string") {
        throw new Error("Invalid research response");
      }
      return parsed;
    },
  };
}

/**
 * Default ideas service instance.
 * Uses the default API client for production use.
 */
export const ideasService = createIdeasService();
