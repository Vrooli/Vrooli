import { Gauge, GitPullRequestArrow, Workflow } from "lucide-react";
import type { AgentSession, AgentSessionArtifact, AgentSessionKind, AgentSessionProposal } from "../../types";

export type SessionInspectorSection = "events" | "proposals" | "artifacts" | "details";

export const SESSION_KIND_LABELS: Record<AgentSession["kind"], string> = {
  meta_orchestration: "Plan work",
  operating_mode_authoring: "Author operating mode",
  swarm_operations: "Swarm operations",
};

export const SESSION_KIND_LAUNCHER_LABELS: Record<AgentSessionKind, string> = {
  meta_orchestration: "Plan Work With Agent",
  operating_mode_authoring: "Author Operating Mode",
  swarm_operations: "Manage Swarm",
};

export const SESSION_KIND_DESCRIPTIONS: Record<AgentSessionKind, string> = {
  meta_orchestration: "Draft initiatives, backlog items, and approval-ready work plans.",
  operating_mode_authoring: "Create or refine the operating-mode loop that guides agentic work.",
  swarm_operations: "Review progress, pending decisions, priorities, and work routing.",
};

export const SESSION_CREATE_TITLES: Record<AgentSessionKind, string> = {
  meta_orchestration: "Plan work with agent",
  operating_mode_authoring: "Author operating mode",
  swarm_operations: "Manage Swarm operations",
};

export const SESSION_KIND_ICONS = {
  meta_orchestration: Workflow,
  operating_mode_authoring: GitPullRequestArrow,
  swarm_operations: Gauge,
};

export const TERMINAL_SESSION_STATUSES = new Set<AgentSession["status"]>(["complete", "failed", "canceled"]);

export function defaultSessionInspectorSection(
  proposals: AgentSessionProposal[],
  artifacts: AgentSessionArtifact[],
  status?: AgentSession["status"],
): SessionInspectorSection {
  if (proposals.some((proposal) => proposal.status === "ready")) return "proposals";
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
