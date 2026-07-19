import type { AgentSessionContextType, AgentSessionKind, CreatableAgentSessionKind } from "../../../types";

export const CONTEXT_TYPE_LABELS: Record<AgentSessionContextType, string> = {
  backlog_item: "Backlog",
  initiative: "Initiatives",
  capture: "Captures",
  execution: "Executions",
  agent_activity: "Activity",
  scenario: "Scenarios",
  operating_mode: "Archived workflows",
  session: "Sessions",
  operations_briefing: "Briefing",
  startup_brief: "Startup",
  plan_dependency_cycles: "Dependency cycles",
  plan_eta: "Plan ETA",
  goal: "Goals",
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
  startup_brief: 1,
  plan_dependency_cycles: 1,
  plan_eta: 1,
  goal: 1,
};

export function allowedContextTypesForKind(kind: AgentSessionKind): AgentSessionContextType[] {
  switch (kind) {
    case "operating_mode_authoring":
      return [];
    case "swarm_operations":
      return ["startup_brief", "operations_briefing", "initiative", "backlog_item", "execution", "agent_activity", "capture", "session", "plan_dependency_cycles", "plan_eta", "goal"];
	case "workflow_authoring":
		return ["startup_brief", "initiative", "backlog_item", "scenario", "session", "goal"];
    case "meta_orchestration":
    default:
      return ["startup_brief", "initiative", "backlog_item", "capture", "scenario", "session", "plan_dependency_cycles", "plan_eta", "goal"];
  }
}

export function compatibleSessionKindsForContextType(type: AgentSessionContextType): CreatableAgentSessionKind[] {
	const kinds: CreatableAgentSessionKind[] = ["meta_orchestration", "swarm_operations", "workflow_authoring"];
  return kinds.filter((kind) => allowedContextTypesForKind(kind).includes(type));
}

export function sessionKindAllowsContextType(kind: AgentSessionKind, type: AgentSessionContextType): boolean {
  return allowedContextTypesForKind(kind).includes(type);
}

export function totalContextCapForKind(_kind: AgentSessionKind): number {
  return 12;
}
