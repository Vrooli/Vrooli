import type { AgentSessionContextType, AgentSessionKind } from "../../../types";

export const CONTEXT_TYPE_LABELS: Record<AgentSessionContextType, string> = {
  backlog_item: "Backlog",
  initiative: "Initiatives",
  capture: "Captures",
  execution: "Executions",
  agent_activity: "Activity",
  scenario: "Scenarios",
  operating_mode: "Modes",
  session: "Sessions",
  operations_briefing: "Briefing",
};

export const CONTEXT_TYPE_CAPS: Record<AgentSessionContextType, number> = {
  backlog_item: 8,
  initiative: 4,
  capture: 4,
  execution: 6,
  agent_activity: 6,
  scenario: 3,
  operating_mode: 3,
  session: 2,
  operations_briefing: 1,
};

export function allowedContextTypesForKind(kind: AgentSessionKind): AgentSessionContextType[] {
  switch (kind) {
    case "operating_mode_authoring":
      return ["operating_mode", "initiative", "backlog_item", "execution", "agent_activity", "capture"];
    case "swarm_operations":
      return ["operations_briefing", "initiative", "backlog_item", "execution", "agent_activity", "capture", "session"];
    case "meta_orchestration":
    default:
      return ["initiative", "backlog_item", "capture", "scenario", "session"];
  }
}

export function totalContextCapForKind(kind: AgentSessionKind): number {
  return kind === "operating_mode_authoring" ? 8 : 12;
}
