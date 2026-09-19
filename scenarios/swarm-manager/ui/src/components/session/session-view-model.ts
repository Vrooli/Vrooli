import { Archive, Gauge, Wrench, Workflow } from "lucide-react";
import type { AgentSession, AgentSessionArtifact, AgentSessionKind, AgentSessionProposal } from "../../types";

export type SessionInspectorSection = "events" | "artifacts" | "details";

export const SESSION_KIND_LABELS: Record<AgentSession["kind"], string> = {
  meta_orchestration: "Plan work",
  operating_mode_authoring: "Archived mode authoring",
  swarm_operations: "Swarm operations",
	workflow_authoring: "Improve the system",
};

// The three launcher entries divide on subject: two act on the work ledger,
// one acts on the machine that runs it. The descriptions say which, because
// choosing the wrong kind is the mistake that wastes a whole conversation.
export const SESSION_KIND_LAUNCHER_LABELS: Record<Exclude<AgentSessionKind, "operating_mode_authoring">, string> = {
  meta_orchestration: "Plan Work With Agent",
  swarm_operations: "Manage Swarm",
	workflow_authoring: "Improve the System",
};

export const SESSION_KIND_DESCRIPTIONS: Record<Exclude<AgentSessionKind, "operating_mode_authoring">, string> = {
  meta_orchestration: "Bring in something new. Shape an idea into goals and backlog items.",
  swarm_operations: "Move what is already here. Progress, blockers, staleness, and what to do next.",
	workflow_authoring: "Change how you and agents work together — skills, prompts, workflows, and briefs.",
};

export const SESSION_CREATE_TITLES: Record<Exclude<AgentSessionKind, "operating_mode_authoring">, string> = {
  meta_orchestration: "Plan work with agent",
  swarm_operations: "Manage Swarm operations",
	workflow_authoring: "Improve the system",
};

export const SESSION_KIND_ICONS = {
  meta_orchestration: Workflow,
  operating_mode_authoring: Archive,
  swarm_operations: Gauge,
  // Deliberately not Workflow: it collided with meta_orchestration, so two of
  // the three launcher entries rendered the same glyph.
	workflow_authoring: Wrench,
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
