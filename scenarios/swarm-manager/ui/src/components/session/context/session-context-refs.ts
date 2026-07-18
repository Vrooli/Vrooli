import type {
  AgentActivity,
  AgentSession,
  AgentSessionContextRef,
  AgentSessionContextType,
  BacklogItem,
  Capture,
  ExecutionRecord,
  InitiativeWithRollup,
  GoalWithScope,
  Scenario,
} from "../../../types";

export interface SessionContextOption extends AgentSessionContextRef {
  title: string;
  subtitle?: string;
  nodeId?: string;
}

export function backlogOption(item: BacklogItem): SessionContextOption {
  const ref = `${item.kind}/${item.name}`;
  return {
    type: "backlog_item",
    ref,
    title: item.title || item.name,
    subtitle: `${item.kind} · ${item.status}`,
    nodeId: `backlog-item/${ref}`,
  };
}

export function initiativeOption(item: InitiativeWithRollup): SessionContextOption {
  const initiative = item.initiative;
  return {
    type: "initiative",
    ref: initiative.name,
    title: initiative.title || initiative.name,
    subtitle: initiative.status,
    nodeId: `initiative/${initiative.name}`,
  };
}

export function goalOption(item: GoalWithScope): SessionContextOption {
  return {
    type: "goal",
    ref: item.goal.name,
    title: item.goal.title || item.goal.name,
    subtitle: `${item.goal.status} · ${Math.round(item.scope.progressPct)}% complete`,
    nodeId: `goal/${item.goal.name}`,
  };
}

export function captureOption(capture: Capture): SessionContextOption {
  return {
    type: "capture",
    ref: capture.id,
    title: capture.text.slice(0, 80) || capture.id,
    subtitle: capture.status,
    nodeId: `capture/${capture.id}`,
  };
}

export function executionOption(execution: ExecutionRecord): SessionContextOption {
  return {
    type: "execution",
    ref: execution.executionId,
    title: `${execution.backlogKind}/${execution.backlogName}`,
    subtitle: execution.status,
    nodeId: `execution-record/${execution.executionId}`,
  };
}

export function activityOption(activity: AgentActivity): SessionContextOption {
  const ref = activity.activityId || activity.runId || activity.taskId || activity.ownerName || "activity";
  return {
    type: "agent_activity",
    ref,
    title: activity.ownerName || activity.runId || ref,
    subtitle: activity.status,
    nodeId: `agent-activity/${ref}`,
  };
}

export function scenarioOption(scenario: Scenario): SessionContextOption {
  return {
    type: "scenario",
    ref: scenario.name,
    title: scenario.name,
    subtitle: scenario.status,
    nodeId: `scenario/${scenario.name}`,
  };
}

export function sessionOption(session: AgentSession): SessionContextOption {
  return {
    type: "session",
    ref: session.id,
    title: session.title || session.id,
    subtitle: `${session.kind} · ${session.status}`,
    nodeId: `/sessions/${session.id}`,
  };
}

export function operationsBriefingOption(): SessionContextOption {
  return {
    type: "operations_briefing",
    ref: "operations_briefing/latest",
    title: "Current operations briefing",
    subtitle: "Active work, attention items, handoffs, and drill-down commands",
    nodeId: "/operations",
  };
}

export function startupBriefOption(kind: string, title = "Startup brief", generatedAt?: string): SessionContextOption {
  return {
    type: "startup_brief",
    ref: `startup_brief/${kind}`,
    title,
    subtitle: generatedAt ? `Generated ${generatedAt}` : "Brief-first context and drill-down commands",
    nodeId: kind === "swarm_operations" ? "/operations" : "/initiatives",
  };
}

/**
 * PlanEtaBand — the subset of the plan board's ETA band that is serialized into
 * a plan_eta context ref. Kept structurally compatible with PlanEtaBandData so
 * the board can pass its `eta` object straight through.
 */
export interface PlanEtaBand {
  p50Label: string;
  p80Label: string;
  basisLabel: string;
  confidence: string;
  remainingItems: number;
  laneCapacity: number;
}

/**
 * Virtual context option carrying the plan's dependency cycles. The chains are
 * JSON-serialized into the ref; the server resolver formats them into a summary.
 */
export function planDependencyCyclesOption(cycles: string[]): SessionContextOption {
  const count = cycles.length;
  return {
    type: "plan_dependency_cycles",
    ref: JSON.stringify(cycles),
    title: `Dependency cycles (${count})`,
    subtitle: `${count} ${count === 1 ? "cycle" : "cycles"} blocking a clean execution order`,
    nodeId: "/plan",
  };
}

/** Virtual context option carrying the plan board's ETA band. */
export function planEtaOption(eta: PlanEtaBand): SessionContextOption {
  const payload: PlanEtaBand = {
    p50Label: eta.p50Label,
    p80Label: eta.p80Label,
    basisLabel: eta.basisLabel,
    confidence: eta.confidence,
    remainingItems: eta.remainingItems,
    laneCapacity: eta.laneCapacity,
  };
  return {
    type: "plan_eta",
    ref: JSON.stringify(payload),
    title: `Plan ETA ${eta.p50Label}–${eta.p80Label}`,
    subtitle: `${eta.remainingItems} items · ${eta.confidence} confidence · ${eta.basisLabel}`,
    nodeId: "/plan",
  };
}

export function contextKey(type: AgentSessionContextType, ref: string): string {
  return `${type}:${ref}`;
}
