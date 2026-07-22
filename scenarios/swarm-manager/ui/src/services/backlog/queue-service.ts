/**
 * Backlog Queue Service — queue and research operations
 */

import { QueueBacklogItemRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import {
  mapProtoBacklogItem,
  parseProtoResponse,
  queueBacklogResponseSchema,
  buildMessage,
  toProtoJson,
} from "../proto-contracts";
import type { IApiClient } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import type { BacklogKind } from "../../types";
import type { QueueResponse } from "./types";

export interface RetryBacklogResponse {
  newExecutionId: string;
  parentExecutionId: string;
  status: string;
}

export function createQueueMethods(apiClient: IApiClient) {
  return {
    async queue(
      kind: BacklogKind,
      name: string,
      options?: {
        operation?: "generator" | "improver";
        mode?: "manual" | "yolo";
        startedBy?: string;
        confirm?: boolean;
        force?: boolean;
      }
    ): Promise<QueueResponse> {
      const msg = buildMessage(QueueBacklogItemRequestSchema, {
        operation: options?.operation ?? "generator",
        mode: options?.mode ?? "yolo",
        ...(options?.startedBy ? { startedBy: options.startedBy } : {}),
        ...(options?.confirm !== undefined ? { confirm: options.confirm } : {}),
        ...(options?.force !== undefined ? { force: options.force } : {}),
      });
      const data = await apiClient.post<unknown>(
        API_ENDPOINTS.backlogQueue(kind, name),
        toProtoJson(QueueBacklogItemRequestSchema, msg),
      );
      const parsed = parseProtoResponse(queueBacklogResponseSchema, data, "backlog queue");
      return {
        item: parsed.item ? mapProtoBacklogItem(parsed.item) : undefined,
        taskId: parsed.taskId ?? "",
        runId: parsed.runId ?? "",
        baseUrl: parsed.baseUrl ?? "",
        created: parsed.created ?? "",
        dryRun: parsed.dryRun ?? false,
        queued: parsed.queued ?? false,
        message: parsed.message ?? "",
        blockingReasons: (parsed.blockingReasons ?? []).map((r) => ({
          message: typeof r === "string" ? r : (r.message ?? ""),
          forceable: typeof r === "string" ? false : (r.forceable ?? false),
        })),
        pendingDecisions: parsed.unansweredQuestions ?? 0,
        pendingSuggestions: parsed.pendingSuggestions ?? 0,
      };
    },

    async retry(kind: BacklogKind, name: string, note?: string): Promise<RetryBacklogResponse> {
      const data = await apiClient.post<{
        new_execution_id: string;
        parent_execution_id: string;
        status: string;
      }>(API_ENDPOINTS.backlogRetry(kind, name), note ? { note } : {});
      return {
        newExecutionId: data.new_execution_id,
        parentExecutionId: data.parent_execution_id,
        status: data.status,
      };
    },
  };
}
