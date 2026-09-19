import type { AgentActivity } from "../types";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import { buildQueryString } from "../lib/query-utils";
import {
  agentActivityResponseSchema,
  listAgentActivitiesResponseSchema,
  mapProtoAgentActivity,
  parseProtoResponse,
  requireProtoField,
} from "./proto-contracts";

export interface ListAgentActivitiesFilters {
  ownerType?: AgentActivity["ownerType"];
  ownerKind?: string;
  ownerName?: string;
  executionId?: string;
  purpose?: AgentActivity["purpose"];
  status?: AgentActivity["status"];
  runId?: string;
  active?: boolean;
}

export interface IAgentActivityService {
  list(filters?: ListAgentActivitiesFilters): Promise<AgentActivity[]>;
  get(activityId: string): Promise<AgentActivity>;
  stopRun(runId: string): Promise<void>;
}

export function createAgentActivityService(apiClient: IApiClient = defaultApiClient): IAgentActivityService {
  return {
    async list(filters?: ListAgentActivitiesFilters): Promise<AgentActivity[]> {
      const suffix = buildQueryString({
        owner_type: filters?.ownerType,
        owner_kind: filters?.ownerKind,
        owner_name: filters?.ownerName,
        execution_id: filters?.executionId,
        purpose: filters?.purpose,
        status: filters?.status,
        run_id: filters?.runId,
        active: filters?.active,
      });
      const data = await apiClient.get<unknown>(
        `${API_ENDPOINTS.agentActivities}${suffix}`
      );
      const parsed = parseProtoResponse(listAgentActivitiesResponseSchema, data, "agent activities");
      return parsed.items.map(mapProtoAgentActivity);
    },

    async get(activityId: string): Promise<AgentActivity> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.agentActivityById(activityId));
      const parsed = parseProtoResponse(agentActivityResponseSchema, data, "agent activity");
      return mapProtoAgentActivity(requireProtoField(parsed.activity, "agent activity"));
    },

    async stopRun(runId: string): Promise<void> {
      await apiClient.post(API_ENDPOINTS.agentManagerStopRun(runId), {});
    },
  };
}

export const agentActivityService = createAgentActivityService();
