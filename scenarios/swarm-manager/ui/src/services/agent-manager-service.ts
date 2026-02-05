/**
 * Agent Manager Service - Data access layer for agent-manager status
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { AgentManagerStatus } from "../types";
import { agentManagerStatusResponseSchema, parseProtoResponse } from "./proto-contracts";

export interface IAgentManagerService {
  getStatus(): Promise<AgentManagerStatus>;
}

export function createAgentManagerService(apiClient: IApiClient = defaultApiClient): IAgentManagerService {
  return {
    async getStatus(): Promise<AgentManagerStatus> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.agentManagerStatus);
      return parseProtoResponse(agentManagerStatusResponseSchema, data, "agent-manager status");
    },
  };
}

export const agentManagerService = createAgentManagerService();
