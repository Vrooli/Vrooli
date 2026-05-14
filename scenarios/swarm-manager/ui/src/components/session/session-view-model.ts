import { GitPullRequestArrow, Workflow } from "lucide-react";
import type { AgentSession, AgentSessionArtifact, AgentSessionProposal } from "../../types";

export type SessionInspectorSection = "events" | "proposals" | "artifacts" | "details";

export const SESSION_KIND_LABELS: Record<AgentSession["kind"], string> = {
  meta_orchestration: "Plan work",
  operating_mode_authoring: "Author operating mode",
};

export const SESSION_KIND_ICONS = {
  meta_orchestration: Workflow,
  operating_mode_authoring: GitPullRequestArrow,
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
