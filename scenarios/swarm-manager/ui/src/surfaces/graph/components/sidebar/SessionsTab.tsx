/**
 * SessionsTab - Lists durable Agent Manager conversations owned by Swarm Manager.
 */

import { memo, useEffect, useMemo, useState } from "react";
import { Bot, Loader2, MessageSquareMore, RefreshCw, Trash2 } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { isActiveAgentSession, useAgentSessionStore } from "../../../../stores";
import { SessionSummaryCard } from "../../../../components/session/session-summary-card";
import { sessionDetailPath } from "../../../../app/routes/route-paths";
import { matchesSearch } from "./useSidebarSearch";
import { applySessionFilters, applySessionSort } from "./session-list-utils";
import type { AgentSession } from "../../../../types";
import type { SessionFilters, SortConfig } from "./types";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { ConfirmDialog } from "../../../../components/ui/confirm-dialog";
import { useDeleteConfirm } from "../../../../hooks/useDeleteConfirm";
import { runBulkAction, summarizeBulkOutcomes, type BulkOutcome } from "./bulk-actions";

interface SessionsTabProps {
  searchQuery: string;
  filters: SessionFilters;
  sort: SortConfig;
  onOpenSession?: (sessionId: string) => void;
  onClearSearch?: () => void;
  selectionMode?: boolean;
  selectedIds?: Set<string>;
  onToggleSelection?: (id: string) => void;
  onVisibleIdsChange?: (ids: string[]) => void;
}

function sessionSelectionId(session: AgentSession): string {
  return `session:${session.id}`;
}

function SessionsTabImpl({
  searchQuery,
  filters,
  sort,
  onOpenSession,
  onClearSearch,
  selectionMode = false,
  selectedIds = new Set<string>(),
  onToggleSelection,
  onVisibleIdsChange,
}: SessionsTabProps) {
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
  useEffect(() => {
    onVisibleIdsChange?.(sorted.map(sessionSelectionId));
  }, [onVisibleIdsChange, sorted]);
  const selectedSessions = useMemo(
    () => sorted.filter((session) => selectedIds.has(sessionSelectionId(session))),
    [selectedIds, sorted],
  );
  const hasFilters = filters.statuses.length > 0 || filters.kinds.length > 0 || filters.activeOnly || filters.hasProposals || filters.hasAppliedArtifacts;

  const handleOpen = (sessionId: string) => {
    navigate(sessionDetailPath(sessionId));
    onOpenSession?.(sessionId);
  };

  if (sorted.length === 0) {
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
    const title = hasFilters ? "No sessions match your filters." : "No agent sessions yet.";
    return (
      <SidebarEmptyState
        icon={MessageSquareMore}
        title={title}
        hint={hasFilters ? undefined : "Plan-work, operations, and authoring conversations show up here once started."}
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
    );
  }

  return (
    <>
      {selectionMode && <SessionBulkActions selectedSessions={selectedSessions} />}
      <div className="space-y-1.5" data-testid="sidebar-sessions-list">
        {sorted.map((session) => {
          const selectionId = sessionSelectionId(session);
          return (
            <SessionSummaryCard
              key={session.id}
              session={session}
              onOpen={handleOpen}
              batchMode={selectionMode}
              batchSelected={selectedIds.has(selectionId)}
              onBatchToggle={() => onToggleSelection?.(selectionId)}
            />
          );
        })}
      </div>
    </>
  );
}

export const SessionsTab = memo(SessionsTabImpl);

function SessionBulkActions({ selectedSessions }: { selectedSessions: AgentSession[] }) {
  const refreshSession = useAgentSessionStore((s) => s.refreshSession);
  const cancelSession = useAgentSessionStore((s) => s.cancelSession);
  const deleteSession = useAgentSessionStore((s) => s.deleteSession);
  const fetchSessions = useAgentSessionStore((s) => s.fetchSessions);
  const [action, setAction] = useState<"refresh" | "cancel" | "delete">("refresh");
  // `confirm` now only gates the non-delete (cancel) action; delete routes
  // through the shared useDeleteConfirm hook so it honors the session level.
  const [confirm, setConfirm] = useState<"cancel" | null>(null);
  const { requestDelete, dialogProps: deleteDialogProps } = useDeleteConfirm("session");
  const [running, setRunning] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const [outcomes, setOutcomes] = useState<BulkOutcome[]>([]);

  const eligible = selectedSessions.filter((session) => {
    if (action === "cancel") return isActiveAgentSession(session);
    if (action === "delete") return session.status === "draft" || session.status === "complete" || session.status === "failed" || session.status === "canceled";
    return true;
  });

  const execute = async () => {
    setRunning(true);
    setSummary(null);
    setOutcomes([]);
    try {
      const next = await runBulkAction(eligible, {
        getId: sessionSelectionId,
        getLabel: (session) => session.title,
        run: (session) => {
          if (action === "cancel") return cancelSession(session.id);
          if (action === "delete") return deleteSession(session.id);
          return refreshSession(session.id);
        },
      });
      setOutcomes(next);
      setSummary(summarizeBulkOutcomes(next));
      await fetchSessions(undefined, { force: true });
    } finally {
      setRunning(false);
      setConfirm(null);
    }
  };

  const failed = outcomes.filter((outcome) => outcome.status === "failed");

  return (
    <div className="mb-2 rounded-lg border border-slate-800 bg-slate-900/70 p-2" data-testid="sidebar-session-bulk-actions">
      <div className="flex flex-wrap items-center gap-2">
        <select value={action} onChange={(event) => setAction(event.target.value as "refresh" | "cancel" | "delete")} className="h-8 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200" aria-label="Session bulk action">
          <option value="refresh">Refresh selected</option>
          <option value="cancel">Cancel active</option>
          <option value="delete">Delete terminal/draft</option>
        </select>
        <button
          type="button"
          disabled={selectedSessions.length === 0 || eligible.length === 0 || running}
          onClick={() => {
            if (action === "delete") {
              requestDelete({
                entityName: eligible.length === 1 ? (eligible[0]?.title ?? "session") : `${eligible.length} sessions`,
                count: eligible.length,
                description: `This permanently deletes ${eligible.length} session${eligible.length === 1 ? "" : "s"}. This action cannot be undone.`,
                confirmLabel: "Delete selected",
                onConfirm: execute,
              });
            } else if (action === "cancel") {
              setConfirm("cancel");
            } else {
              void execute();
            }
          }}
          className="inline-flex h-8 items-center gap-1.5 rounded border border-cyan-500/40 bg-cyan-500/10 px-2 text-xs font-medium text-cyan-200 hover:bg-cyan-500/20 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {running ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : action === "delete" ? <Trash2 className="h-3.5 w-3.5" /> : <RefreshCw className="h-3.5 w-3.5" />}
          Apply
        </button>
      </div>
      <div className="mt-1.5 text-[11px] text-slate-500">{eligible.length} eligible{summary ? ` - ${summary}` : ""}</div>
      {failed.length > 0 && <div className="mt-1 text-[11px] text-red-300">{failed.map((outcome) => <div key={outcome.id}>{outcome.label}: {outcome.message}</div>)}</div>}
      <ConfirmDialog
        isOpen={confirm === "cancel"}
        onClose={() => setConfirm(null)}
        onConfirm={() => void execute()}
        title="Cancel selected sessions"
        description={`Cancel ${eligible.length} selected session${eligible.length === 1 ? "" : "s"}?`}
        confirmLabel="Cancel selected"
        isLoading={running}
      />
      <ConfirmDialog {...deleteDialogProps} />
    </div>
  );
}
