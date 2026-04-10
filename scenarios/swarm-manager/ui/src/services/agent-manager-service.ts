/**
 * Agent Manager Service - Data access layer for agent-manager status and runs
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { AgentManagerStatus, AgentRunState, AgentRunStatus } from "../types";
import { agentManagerStatusResponseSchema, parseProtoResponse } from "./proto-contracts";

interface AgentRunStateResponse {
  run_id?: string;
  task_id?: string;
  status?: string;
  started_at?: string;
  finished_at?: string;
  error_message?: string;
  duration_seconds?: number;
  active?: boolean;
  tokens_used?: number;
  turns_used?: number;
  cost_estimate?: number;
  changed_files?: number;
  context_tokens?: number;
}

interface AgentRunDetailsResponse {
  run?: {
    sandboxId?: string;
  };
}

export interface AgentRunDetails {
  sandboxId?: string;
}

export interface IAgentManagerService {
  getStatus(): Promise<AgentManagerStatus>;
  getRunState(runId: string): Promise<AgentRunState>;
  getRunDetails(runId: string): Promise<AgentRunDetails>;
  stopRun(runId: string): Promise<void>;
}

const validRunStatuses = new Set<AgentRunStatus>([
  "pending",
  "starting",
  "running",
  "needs_review",
  "complete",
  "failed",
  "cancelled",
  "unspecified",
]);

const normalizeRunStatus = (value: string | undefined): AgentRunStatus => {
  const normalized = (value ?? "unspecified").trim().toLowerCase() as AgentRunStatus;
  return validRunStatuses.has(normalized) ? normalized : "unspecified";
};

export function createAgentManagerService(apiClient: IApiClient = defaultApiClient): IAgentManagerService {
  return {
    async getStatus(): Promise<AgentManagerStatus> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.agentManagerStatus);
      return parseProtoResponse(agentManagerStatusResponseSchema, data, "agent-manager status");
    },
    async getRunState(runId: string): Promise<AgentRunState> {
      const data = await apiClient.get<AgentRunStateResponse>(API_ENDPOINTS.agentManagerRun(runId));
      return {
        runId: data.run_id ?? runId,
        taskId: data.task_id,
        status: normalizeRunStatus(data.status),
        startedAt: data.started_at,
        finishedAt: data.finished_at,
        errorMessage: data.error_message,
        durationSeconds:
          typeof data.duration_seconds === "number" && Number.isFinite(data.duration_seconds)
            ? data.duration_seconds
            : undefined,
        active: Boolean(data.active),
        tokensUsed:
          typeof data.tokens_used === "number" && data.tokens_used > 0
            ? data.tokens_used
            : undefined,
        turnsUsed:
          typeof data.turns_used === "number" && data.turns_used > 0
            ? data.turns_used
            : undefined,
        costEstimate:
          typeof data.cost_estimate === "number" && data.cost_estimate > 0
            ? data.cost_estimate
            : undefined,
        changedFiles:
          typeof data.changed_files === "number" && data.changed_files > 0
            ? data.changed_files
            : undefined,
        contextTokens:
          typeof data.context_tokens === "number" && data.context_tokens > 0
            ? data.context_tokens
            : undefined,
      };
    },
    async getRunDetails(runId: string): Promise<AgentRunDetails> {
      const data = await apiClient.get<AgentRunDetailsResponse>(API_ENDPOINTS.agentManagerRun(runId));
      return {
        sandboxId: typeof data?.run?.sandboxId === "string" ? data.run.sandboxId : undefined,
      };
    },
    async stopRun(runId: string): Promise<void> {
      await apiClient.post(API_ENDPOINTS.agentManagerStopRun(runId), {});
    },
  };
}

export const agentManagerService = createAgentManagerService();
