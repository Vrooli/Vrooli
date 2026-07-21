import { Archive, Gauge, Workflow } from "lucide-react";
import type { AgentSession, AgentSessionArtifact, AgentSessionKind, AgentSessionProposal } from "../../types";

export type SessionInspectorSection = "events" | "artifacts" | "details";

export const SESSION_KIND_LABELS: Record<AgentSession["kind"], string> = {
  meta_orchestration: "Plan work",
  operating_mode_authoring: "Archived mode authoring",
  swarm_operations: "Swarm operations",
	workflow_authoring: "Workflow authoring",
};

export const SESSION_KIND_LAUNCHER_LABELS: Record<Exclude<AgentSessionKind, "operating_mode_authoring">, string> = {
  meta_orchestration: "Plan Work With Agent",
  swarm_operations: "Manage Swarm",
	workflow_authoring: "Author Workflow",
};

export const SESSION_KIND_DESCRIPTIONS: Record<Exclude<AgentSessionKind, "operating_mode_authoring">, string> = {
  meta_orchestration: "Draft initiatives, backlog items, and approval-ready work plans.",
  swarm_operations: "Review progress, pending decisions, priorities, and work routing.",
	workflow_authoring: "Turn your agent-working method into a reviewed workflow or a Swarm improvement proposal.",
};

export const SESSION_CREATE_TITLES: Record<Exclude<AgentSessionKind, "operating_mode_authoring">, string> = {
  meta_orchestration: "Plan work with agent",
  swarm_operations: "Manage Swarm operations",
	workflow_authoring: "Author a Swarm workflow",
};

export const SESSION_KIND_ICONS = {
  meta_orchestration: Workflow,
  operating_mode_authoring: Archive,
  swarm_operations: Gauge,
	workflow_authoring: Workflow,
};

export const TERMINAL_SESSION_STATUSES = new Set<AgentSession["status"]>(["complete", "failed", "canceled"]);

export function defaultSessionInspectorSection(
  proposals: AgentSessionProposal[],
  artifacts: AgentSessionArtifact[],
  status?: AgentSession["status"],
): SessionInspectorSection {
  if (proposals.length > 0) return "artifacts";
  if (status === "starting" || status === "running") return "events";
  if (artifacts.length > 0) return "artifacts";
  return "details";
}

export function isSessionWaitingForAgent(session: AgentSession): boolean {
  const statusSuggestsWork = session.status === "starting" || session.status === "running" || session.status === "applying";
  const sortedMessages = [...session.messages].sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime());
  const lastMessage = sortedMessages[sortedMessages.length - 1];
  return statusSuggestsWork || (lastMessage?.role === "user" && session.status !== "waiting_for_user");
}
