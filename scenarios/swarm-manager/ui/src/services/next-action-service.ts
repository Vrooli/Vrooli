import { defaultApiClient, type IApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";

export type NextActionID = "decide" | "review" | "accept_plan" | "author_plan" | "repair_plan" | "plan_goal" | "run" | "dispatch_followup" | "author_followup" | "resolve_dependencies" | "accept_suggestion" | "retry" | "archive" | "close_out" | string;

/**
 * What performing the action does to the system, as declared by the server.
 *
 * Optional on the wire so a UI build newer than its API does not crash; an
 * absent value is treated as unknown, never as harmless.
 */
export type NextActionEffect = "none" | "state_change" | "agent_run" | "agent_session";

export interface NextActionFeedEntry {
  entity_kind: "backlog_item" | "goal" | "capture";
  entity_ref: string;
  entity_title: string;
  action: { id: NextActionID; compact_label: string; expanded_label: string; enabled: boolean; reason?: string; target?: string; transition_key?: string; effect?: NextActionEffect; destructive?: boolean; blockers?: Array<{ code: string; message: string }>; follow_up?: { steering: string; disposition: "follow_up_run" | "replan" | "new_items"; items?: Array<{ name: string; title: string }> } };
  tier: number;
  goal_priority?: number;
  backlog_rank?: number;
  chained_ref?: string;
}

export function createNextActionService(apiClient: IApiClient = defaultApiClient) {
  return { getFeed: () => apiClient.get<{ entries: NextActionFeedEntry[] }>(API_ENDPOINTS.nextActionsFeed) };
}

export const nextActionService = createNextActionService();
