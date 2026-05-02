/**
 * SessionsTab - Lists durable Agent Manager conversations owned by Swarm Manager.
 */

import { Bot, GitPullRequestArrow, Layers3, MessageSquareMore, Workflow } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { cn } from "../../../../lib/utils";
import { formatRelativeTime } from "../../../../lib/format-utils";
import { isActiveAgentSession, useAgentSessionStore } from "../../../../stores";
import { sessionDetailPath } from "../../../../app/routes/route-paths";
import { matchesSearch } from "./useSidebarSearch";
import { applySessionFilters, applySessionSort } from "./session-list-utils";
import type { AgentSession } from "../../../../types";
import type { SessionFilters, SortConfig } from "./types";
import { SidebarEmptyState } from "./SidebarEmptyState";

interface SessionsTabProps {
  searchQuery: string;
  filters: SessionFilters;
  sort: SortConfig;
  onOpenSession?: (sessionId: string) => void;
  onClearSearch?: () => void;
}

const STATUS_COLORS: Record<AgentSession["status"], string> = {
  draft: "bg-slate-700/60 text-slate-300",
  starting: "bg-blue-500/20 text-blue-300",
  running: "bg-cyan-500/20 text-cyan-300",
  waiting_for_user: "bg-amber-500/20 text-amber-300",
  proposal_ready: "bg-violet-500/20 text-violet-300",
  applying: "bg-blue-500/20 text-blue-300",
  complete: "bg-green-500/20 text-green-300",
  failed: "bg-red-500/20 text-red-300",
  canceled: "bg-slate-700/40 text-slate-500",
};

const KIND_LABELS: Record<AgentSession["kind"], string> = {
  meta_orchestration: "Plan work",
  operating_mode_authoring: "Author mode",
};

const KIND_ICONS = {
  meta_orchestration: Workflow,
  operating_mode_authoring: GitPullRequestArrow,
};

export function SessionsTab({ searchQuery, filters, sort, onOpenSession, onClearSearch }: SessionsTabProps) {
  const sessions = useAgentSessionStore((s) => s.sessions);
  const status = useAgentSessionStore((s) => s.status);
  const error = useAgentSessionStore((s) => s.error);
  const fetchSessions = useAgentSessionStore((s) => s.fetchSessions);
  const navigate = useNavigate();

  let filtered = applySessionFilters(sessions, filters);
  if (searchQuery) {
    filtered = filtered.filter((session) =>
      matchesSearch(searchQuery, session.title, session.kind, session.status, session.skillId, session.runId),
    );
  }
  const sorted = applySessionSort(filtered, sort);
  const hasFilters = filters.statuses.length > 0 || filters.kinds.length > 0 || filters.activeOnly || filters.hasProposals || filters.hasAppliedArtifacts;

  const handleOpen = (sessionId: string) => {
    navigate(sessionDetailPath(sessionId));
    onOpenSession?.(sessionId);
  };

  if (status === "error" && sessions.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-12 text-center text-slate-500" data-testid="sidebar-sessions-error">
        <Bot className="h-8 w-8 text-red-300" />
        <p className="text-sm text-slate-300">Unable to load sessions.</p>
        {error?.message && <p className="max-w-56 break-words text-xs text-slate-500">{error.message}</p>}
        <button
          type="button"
          onClick={() => void fetchSessions(undefined, { force: true })}
          className="rounded-md border border-slate-700 px-2.5 py-1 text-xs font-medium text-slate-300 hover:border-slate-600 hover:bg-slate-800"
        >
          Retry
        </button>
      </div>
    );
  }

  if (sorted.length === 0) {
    const title = hasFilters ? "No sessions match your filters." : "No agent sessions yet.";
    return (
      <SidebarEmptyState
        icon={MessageSquareMore}
        title={title}
        hint={hasFilters ? undefined : "Plan-work and authoring conversations show up here once started."}
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
    );
  }

  return (
    <div className="space-y-1.5" data-testid="sidebar-sessions-list">
      {sorted.map((session) => {
        const Icon = KIND_ICONS[session.kind];
        const artifactCount = session.artifacts.length;
        const proposalCount = session.proposals.length;
        const active = isActiveAgentSession(session);

        return (
          <button
            key={session.id}
            type="button"
            onClick={() => handleOpen(session.id)}
            className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
            data-testid="sidebar-session-item"
          >
            <div className="flex items-start justify-between gap-2">
              <div className="flex min-w-0 items-start gap-2">
                <Icon className="mt-0.5 h-3.5 w-3.5 shrink-0 text-cyan-300" />
                <p className="line-clamp-2 min-w-0 text-[13px] font-medium leading-snug text-slate-100">
                  {session.title}
                </p>
              </div>
              <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_COLORS[session.status])}>
                {session.status.replace(/_/g, " ")}
              </span>
            </div>

            <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-slate-500">
              <span>{KIND_LABELS[session.kind]}</span>
              <span>{formatRelativeTime(session.updatedAt)}</span>
              {active && (
                <span className="inline-flex items-center gap-1 text-cyan-300">
                  <span className="h-1.5 w-1.5 rounded-full bg-cyan-300" />
                  active
                </span>
              )}
            </div>

            <div className="mt-1.5 flex flex-wrap items-center gap-1.5 text-[10px] text-slate-400">
              <span className="inline-flex items-center gap-1 rounded-full bg-slate-800/70 px-1.5 py-0.5">
                <MessageSquareMore className="h-3 w-3" />
                {session.messages.length}
              </span>
              <span className="inline-flex items-center gap-1 rounded-full bg-slate-800/70 px-1.5 py-0.5">
                <GitPullRequestArrow className="h-3 w-3" />
                {proposalCount}
              </span>
              <span className="inline-flex items-center gap-1 rounded-full bg-slate-800/70 px-1.5 py-0.5">
                <Layers3 className="h-3 w-3" />
                {artifactCount}
              </span>
            </div>
          </button>
        );
      })}
    </div>
  );
}
