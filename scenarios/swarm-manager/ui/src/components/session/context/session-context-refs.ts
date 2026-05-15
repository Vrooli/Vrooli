import type {
  AgentActivity,
  AgentSession,
  AgentSessionContextRef,
  AgentSessionContextType,
  BacklogItem,
  Capture,
  ExecutionRecord,
  InitiativeWithRollup,
  Scenario,
} from "../../../types";

export interface SessionContextOption extends AgentSessionContextRef {
  title: string;
  subtitle?: string;
  nodeId?: string;
}

interface OperatingModeOption {
  mode?: string;
  id?: string;
  label?: string;
  title?: string;
  description?: string;
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

export function operatingModeOption(mode: OperatingModeOption): SessionContextOption {
  const ref = mode.mode || mode.id || mode.label || "";
  return {
    type: "operating_mode",
    ref,
    title: mode.label || mode.title || ref,
    subtitle: mode.description,
    nodeId: `operatingMode/${ref}`,
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

export function contextKey(type: AgentSessionContextType, ref: string): string {
  return `${type}:${ref}`;
}
