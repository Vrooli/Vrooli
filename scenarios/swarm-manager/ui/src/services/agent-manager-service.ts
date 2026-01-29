/**
 * Agent Manager Service - Data access layer for agent-manager status
 */

import { z } from "zod";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { AgentManagerStatus } from "../types";

const statusSchema = z.object({
  enabled: z.boolean(),
  available: z.boolean().optional().default(false),
  url: z.string().optional(),
  profileId: z.string().optional(),
});

export interface IAgentManagerService {
  getStatus(): Promise<AgentManagerStatus>;
}

export function createAgentManagerService(apiClient: IApiClient = defaultApiClient): IAgentManagerService {
  return {
    async getStatus(): Promise<AgentManagerStatus> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.agentManagerStatus);
      const parsed = statusSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error("Invalid agent-manager status response");
      }
      return parsed.data;
    },
  };
}

export const agentManagerService = createAgentManagerService();
