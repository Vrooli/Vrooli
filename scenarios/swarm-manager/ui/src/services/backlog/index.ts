/**
 * Backlog Service — Barrel re-export
 *
 * Composes domain-specific service modules into a unified IBacklogService
 * for backwards compatibility. All existing import sites continue to work.
 *
 * DOC: docs/internal/SEAMS.md#ui-to-api-seam-improved-in-phase-3
 * DOC: docs/internal/INTENT.md#key-flows
 */

import type { IApiClient } from "../../lib/api-client";
import { defaultApiClient } from "../../lib/api-client";
import type { IBacklogService } from "./types";
import { createCrudMethods } from "./crud-service";
import { createFileMethods } from "./file-service";
import { createQueueMethods } from "./queue-service";
import { createArchiveMethods } from "./archive-service";
import { createBulkMethods } from "./bulk-service";

// Re-export all types so consumers can import from the same path
export type {
  IBacklogService,
  QueueResponse,
  BacklogFileOperationResult,
  BacklogUpdatePatch,
  ImportBacklogResponse,
  RenderedBacklogPlan,
} from "./types";

export function createBacklogService(apiClient: IApiClient = defaultApiClient): IBacklogService {
  return {
    ...createCrudMethods(apiClient),
    ...createFileMethods(apiClient),
    ...createQueueMethods(apiClient),
    ...createArchiveMethods(apiClient),
    ...createBulkMethods(apiClient),
  };
}

/**
 * Default backlog service instance.
 * Uses the default API client for production use.
 */
export const backlogService = createBacklogService();
