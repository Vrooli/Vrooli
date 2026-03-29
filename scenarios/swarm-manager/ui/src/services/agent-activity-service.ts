import type { AgentActivity } from "../types";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import {
  agentActivityResponseSchema,
  listAgentActivitiesResponseSchema,
  mapProtoAgentActivity,
  parseProtoResponse,
  requireProtoField,
} from "./proto-contracts";

export interface ListAgentActivitiesFilters {
  ownerType?: "backlog" | "capture" | "scenario";
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

function buildQuery(filters?: ListAgentActivitiesFilters): string {
  if (!filters) return "";
  const params = new URLSearchParams();
  if (filters.ownerType) params.set("owner_type", filters.ownerType);
  if (filters.ownerKind) params.set("owner_kind", filters.ownerKind);
  if (filters.ownerName) params.set("owner_name", filters.ownerName);
  if (filters.executionId) params.set("execution_id", filters.executionId);
  if (filters.purpose) params.set("purpose", filters.purpose);
  if (filters.status) params.set("status", filters.status);
  if (filters.runId) params.set("run_id", filters.runId);
  if (typeof filters.active === "boolean") params.set("active", String(filters.active));
  const encoded = params.toString();
  return encoded ? `?${encoded}` : "";
}

export function createAgentActivityService(apiClient: IApiClient = defaultApiClient): IAgentActivityService {
  return {
    async list(filters?: ListAgentActivitiesFilters): Promise<AgentActivity[]> {
      const data = await apiClient.get<unknown>(
        `${API_ENDPOINTS.agentActivities}${buildQuery(filters)}`
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
