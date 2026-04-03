/**
 * Backlog Workshop Service — workshop rounds and clarification operations
 */

import type { IApiClient } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import type { BacklogKind, ClarificationThread } from "../../types";
import type { WorkshopSaveResponse, WorkshopDeleteRoundResponse, WorkshopResetResponse } from "./types";

export function createWorkshopMethods(apiClient: IApiClient) {
  return {
    async workshopSave(
      kind: BacklogKind,
      name: string,
      roundNumber: number,
      content: string,
    ): Promise<WorkshopSaveResponse> {
      const body: Record<string, unknown> = {
        round_number: roundNumber,
        content,
      };
      const data = await apiClient.post<{
        file: Record<string, unknown>;
        auto_advance: { triggered: boolean; run_id?: string; task_id?: string; reason: string; next_mode?: "workshop" | "finalize" };
      }>(API_ENDPOINTS.backlogWorkshopSave(kind, name), body);
      return {
        file: {
          name: String(data.file?.name ?? ""),
          path: String(data.file?.path ?? ""),
          type: String(data.file?.type ?? "file") as "file" | "directory",
          size: Number(data.file?.size ?? 0),
        },
        autoAdvance: {
          triggered: data.auto_advance?.triggered ?? false,
          runId: data.auto_advance?.run_id,
          taskId: data.auto_advance?.task_id,
          reason: data.auto_advance?.reason ?? "",
          nextMode: data.auto_advance?.next_mode,
        },
      };
    },

    async workshopDeleteRound(
      kind: BacklogKind,
      name: string,
      roundNumber: number,
    ): Promise<WorkshopDeleteRoundResponse> {
      const data = await apiClient.delete<{
        deleted_round: number;
        remaining_rounds: number;
      }>(API_ENDPOINTS.backlogWorkshopDeleteRound(kind, name), {
        round_number: roundNumber,
      });
      return {
        deletedRound: data.deleted_round,
        remainingRounds: data.remaining_rounds,
      };
    },

    async workshopReset(
      kind: BacklogKind,
      name: string,
    ): Promise<WorkshopResetResponse> {
      const data = await apiClient.post<{
        deleted_rounds?: number;
        status_reverted?: boolean;
      }>(API_ENDPOINTS.backlogWorkshopReset(kind, name), {});
      return {
        deletedRounds: data.deleted_rounds ?? 0,
        statusReverted: data.status_reverted ?? false,
      };
    },

    async createClarification(
      kind: BacklogKind,
      name: string,
      roundNumber: number,
      itemId: string,
      message?: string,
      files?: File[],
    ): Promise<{ thread: ClarificationThread }> {
      const formData = new FormData();
      formData.append("round_number", String(roundNumber));
      formData.append("item_id", itemId);
      if (message) formData.append("message", message);
      if (files) {
        for (const file of files) {
          formData.append("files", file);
        }
      }
      return apiClient.post<{ thread: ClarificationThread }>(
        `/backlog/${kind}/${name}/workshop/clarification`,
        formData,
      );
    },

    async getClarification(
      kind: BacklogKind,
      name: string,
      threadId: string,
    ): Promise<{ thread: ClarificationThread }> {
      return apiClient.get<{ thread: ClarificationThread }>(
        `/backlog/${kind}/${name}/workshop/clarification/${threadId}`,
      );
    },

    async continueClarification(
      kind: BacklogKind,
      name: string,
      threadId: string,
      message: string,
      files?: File[],
    ): Promise<{ thread: ClarificationThread }> {
      const formData = new FormData();
      formData.append("message", message);
      if (files) {
        for (const file of files) {
          formData.append("files", file);
        }
      }
      return apiClient.post<{ thread: ClarificationThread }>(
        `/backlog/${kind}/${name}/workshop/clarification/${threadId}/continue`,
        formData,
      );
    },

    async clarificationAction(
      kind: BacklogKind,
      name: string,
      threadId: string,
      action: string,
      updatedItemJson?: string,
    ): Promise<{ action: string; success: boolean; message: string; run_id?: string; task_id?: string }> {
      return apiClient.post(
        `/backlog/${kind}/${name}/workshop/clarification/${threadId}/action`,
        { action, updated_item_json: updatedItemJson },
      );
    },
  };
}
