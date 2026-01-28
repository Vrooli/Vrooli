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

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Idea, IdeaFile } from "../types";

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
  update(name: string, idea: Partial<Idea>): Promise<Idea>;
  delete(name: string): Promise<void>;
  getFiles(name: string): Promise<IdeaFile[]>;
  getFileContent(name: string, filePath: string): Promise<string>;
  uploadFile(name: string, file: File, path?: string): Promise<IdeaFile>;
  queue(name: string, operation?: "generator" | "improver"): Promise<QueueResponse>;
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
      return apiClient.get<Idea[]>(API_ENDPOINTS.ideas);
    },

    async get(name: string): Promise<Idea> {
      return apiClient.get<Idea>(`${API_ENDPOINTS.ideas}/${name}`);
    },

    async create(idea: Omit<Idea, "created" | "updated">): Promise<Idea> {
      return apiClient.post<Idea>(API_ENDPOINTS.ideas, idea);
    },

    async update(name: string, idea: Partial<Idea>): Promise<Idea> {
      return apiClient.put<Idea>(`${API_ENDPOINTS.ideas}/${name}`, idea);
    },

    async delete(name: string): Promise<void> {
      return apiClient.delete<void>(`${API_ENDPOINTS.ideas}/${name}`);
    },

    async getFiles(name: string): Promise<IdeaFile[]> {
      return apiClient.get<IdeaFile[]>(API_ENDPOINTS.ideaFiles(name));
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
      return apiClient.post<IdeaFile>(API_ENDPOINTS.ideaFiles(name), formData, {
        headers: {},
      });
    },

    async queue(
      name: string,
      operation: "generator" | "improver" = "generator"
    ): Promise<QueueResponse> {
      return apiClient.post<QueueResponse>(API_ENDPOINTS.ideaQueue(name), {
        operation,
      });
    },
  };
}

/**
 * Default ideas service instance.
 * Uses the default API client for production use.
 */
export const ideasService = createIdeasService();
