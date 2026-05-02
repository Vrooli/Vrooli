/**
 * SessionDetailsPage — Routed agent-session detail page.
 *
 * Conversation transcript, composer, proposals, artifacts, and run/session
 * metadata for a meta-orchestration or operating-mode-authoring session.
 */

import { useCallback, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import {
  AlertCircle,
  Bot,
  CheckCircle2,
  ExternalLink,
  FileText,
  GitPullRequestArrow,
  Layers3,
  Loader2,
  RefreshCw,
  SendHorizontal,
  Square,
  Workflow,
} from "lucide-react";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { Button } from "../components/ui/button";
import { Textarea } from "../components/ui/textarea";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { cn } from "../lib/utils";
import { formatDisplayText, formatRelativeTime } from "../lib/format-utils";
import { useAutoResizeTextarea } from "../hooks/useAutoResizeTextarea";
import { useAgentSessionStore } from "../stores";
import { useAgentSessionPolling } from "../hooks/useAgentSessionPolling";
import { useAppBack } from "../app/routes/useAppBack";
import { detailPathFromNodeId } from "../app/routes/route-paths";
import { buildActivityNodeId, buildBacklogNodeId } from "../surfaces/graph/lib/node-id-parser";
import type { AgentSession, AgentSessionArtifact, AgentSessionProposal } from "../types";

const MAX_TEXTAREA_HEIGHT = 104;

const KIND_LABELS: Record<AgentSession["kind"], string> = {
  meta_orchestration: "Plan work",
  operating_mode_authoring: "Author operating mode",
};

const KIND_ICONS = {
  meta_orchestration: Workflow,
  operating_mode_authoring: GitPullRequestArrow,
};

const TERMINAL_STATUSES = new Set<AgentSession["status"]>(["complete", "failed", "canceled"]);

export function SessionDetailsPage() {
  const { sessionId } = useParams<{ sessionId: string }>();
  const navigate = useNavigate();
  const closeDetail = useAppBack();

  const storeSession = useAgentSessionStore((s) =>
    s.sessions.find((session) => session.id === sessionId),
  );
  const loadSession = useAgentSessionStore((s) => s.loadSession);
  const continueSession = useAgentSessionStore((s) => s.continueSession);
  const refreshSession = useAgentSessionStore((s) => s.refreshSession);
  const cancelSession = useAgentSessionStore((s) => s.cancelSession);
  const applyProposal = useAgentSessionStore((s) => s.applyProposal);
  const isMutating = useAgentSessionStore((s) => s.isMutating);
  const isRefreshing = useAgentSessionStore((s) => s.isRefreshing);
  const error = useAgentSessionStore((s) => s.error);

  useAgentSessionPolling(sessionId);

  const { data: fetchedSession, isLoading, error: queryError } = useQuery({
    queryKey: ["session", sessionId],
    queryFn: () => loadSession(sessionId ?? ""),
    enabled: !!sessionId && !storeSession,
  });

  const session: AgentSession | undefined = storeSession ?? fetchedSession;

  const [draft, setDraft] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useAutoResizeTextarea(textareaRef, draft, { maxHeight: MAX_TEXTAREA_HEIGHT });

  const sortedMessages = useMemo(
    () => [...(session?.messages ?? [])].sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()),
    [session?.messages],
  );

  const handleSend = useCallback(async () => {
    if (!session || !draft.trim()) return;
    const message = draft.trim();
    setDraft("");
    setLocalError(null);
    try {
      await continueSession({ sessionId: session.id, message });
    } catch (err) {
      setDraft(message);
      setLocalError(err instanceof Error ? err.message : "Unable to continue session.");
    }
  }, [continueSession, draft, session]);

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
      event.preventDefault();
      void handleSend();
    }
  };

  const handleRefresh = useCallback(async () => {
    if (!session) return;
    setLocalError(null);
    try {
      await refreshSession(session.id);
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to refresh session.");
    }
  }, [refreshSession, session]);

  const handleCancel = useCallback(async () => {
    if (!session) return;
    setLocalError(null);
    try {
      await cancelSession(session.id);
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : "Unable to cancel session.");
    }
  }, [cancelSession, session]);

  const handleApply = useCallback(
    async (proposalId: string) => {
      if (!session) return;
      setLocalError(null);
      try {
        await applyProposal(session.id, proposalId);
      } catch (err) {
        setLocalError(err instanceof Error ? err.message : "Unable to apply proposal.");
      }
    },
    [applyProposal, session],
  );

  const handleOpenArtifact = useCallback(
    (artifact: AgentSessionArtifact) => {
      const nodeId = nodeIdForArtifact(artifact);
      if (!nodeId) return;
      const path = detailPathFromNodeId(nodeId);
      if (path) navigate(path);
    },
    [navigate],
  );

  if (!sessionId) {
    return (
      <DetailPageLayout header={<DetailPageHeader entityType="Session" title="Not found" nodeId={null} lenses={[]} />}>
        <ErrorState message="No session selected." onRetry={closeDetail} />
      </DetailPageLayout>
    );
  }

  if (isLoading && !session) {
    return <PageLoadingState label="Loading session..." />;
  }

  if ((queryError && !session) || (!session && !isLoading)) {
    return (
      <DetailPageLayout header={<DetailPageHeader entityType="Session" title="Not found" nodeId={null} lenses={[]} />}>
        <ErrorState message="Session not found." onRetry={closeDetail} />
      </DetailPageLayout>
    );
  }

  if (!session) return <PageLoadingState label="Loading session..." />;

  const Icon = KIND_ICONS[session.kind] ?? Bot;
  const canSend = Boolean(draft.trim() && !isMutating);
  const cancelDisabled = isMutating || TERMINAL_STATUSES.has(session.status);

  const headerActions = (
    <>
      <Button variant="ghost" size="sm" onClick={() => void handleRefresh()} disabled={isMutating || isRefreshing} data-testid="session-refresh">
        <RefreshCw className={cn("mr-1.5 h-3.5 w-3.5", isRefreshing && "animate-spin")} />
        Refresh
      </Button>
      <Button variant="ghost" size="sm" onClick={() => void handleCancel()} disabled={cancelDisabled} data-testid="session-cancel">
        <Square className="mr-1.5 h-3.5 w-3.5" />
        Cancel
      </Button>
    </>
  );

  const mobileActions = (
    <div className="flex flex-col gap-2 p-2">
      <Button variant="ghost" onClick={() => void handleRefresh()} disabled={isMutating || isRefreshing}>
        <RefreshCw className={cn("mr-2 h-4 w-4", isRefreshing && "animate-spin")} />
        Refresh
      </Button>
      <Button variant="ghost" onClick={() => void handleCancel()} disabled={cancelDisabled}>
        <Square className="mr-2 h-4 w-4" />
        Cancel
      </Button>
    </div>
  );

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="Session"
          entityIcon={Icon}
          title={session.title || "Agent session"}
          subtitle={`${KIND_LABELS[session.kind]} · ${formatRelativeTime(session.updatedAt)}`}
          status={formatDisplayText(session.status)}
          nodeId={null}
          lenses={[]}
          actions={headerActions}
        />
      }
      mobileActions={mobileActions}
      mobileActionsTitle="Session actions"
    >
      <div className="mx-auto w-full max-w-6xl space-y-3">
        {(localError || error?.message) && (
          <div className="flex items-start gap-2 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200" role="alert">
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <span className="min-w-0 break-words">{localError || error?.message}</span>
          </div>
        )}

        <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_320px]">
          <section className="flex min-h-[60vh] flex-col rounded-lg border border-white/10 bg-slate-950/30">
            <div className="border-b border-white/10 px-3 py-2 text-xs font-medium text-slate-300">Conversation</div>
            <div className="min-h-0 flex-1 space-y-2 overflow-y-auto p-3" data-testid="agent-session-messages">
              {sortedMessages.length > 0 ? (
                sortedMessages.map((message) => (
                  <article
                    key={message.id}
                    className={cn(
                      "rounded-lg border px-3 py-2",
                      message.role === "user"
                        ? "ml-auto max-w-[88%] border-cyan-500/20 bg-cyan-500/10"
                        : "mr-auto max-w-[92%] border-white/10 bg-slate-900/70",
                    )}
                  >
                    <div className="mb-1 flex items-center justify-between gap-2 text-[10px] uppercase tracking-wide text-slate-500">
                      <span>{message.role}</span>
                      <span>{formatRelativeTime(message.createdAt)}</span>
                    </div>
                    <p className="whitespace-pre-wrap break-words text-sm leading-6 text-slate-100">{message.content}</p>
                  </article>
                ))
              ) : (
                <div className="flex h-full items-center justify-center text-sm text-slate-500">
                  No messages recorded yet.
                </div>
              )}
            </div>
            <div className="border-t border-white/10 p-3">
              <Textarea
                ref={textareaRef}
                value={draft}
                onChange={(event) => setDraft(event.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Continue this session..."
                rows={2}
                disabled={isMutating}
                data-testid="agent-session-composer"
              />
              <div className="mt-2 flex justify-end">
                <Button size="sm" onClick={() => void handleSend()} disabled={!canSend} data-testid="agent-session-send">
                  {isMutating ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <SendHorizontal className="mr-2 h-4 w-4" />}
                  Send
                </Button>
              </div>
            </div>
          </section>

          <aside className="space-y-3">
            <ProposalList proposals={session.proposals} isMutating={isMutating} onApply={handleApply} />
            <ArtifactList artifacts={session.artifacts} onOpenArtifact={handleOpenArtifact} />
            <section className="rounded-lg border border-white/10 bg-slate-950/30 p-3">
              <h4 className="text-xs font-medium text-slate-300">Session details</h4>
              <dl className="mt-2 space-y-1 text-[11px] text-slate-400">
                <RunDetail label="Session ID" value={session.id} />
                <RunDetail label="Skill" value={session.skillId} />
                <RunDetail label="Run" value={session.runId} />
                <RunDetail label="Task" value={session.taskId} />
                <RunDetail label="Profile" value={session.profileKey} />
                <RunDetail label="Created" value={formatRelativeTime(session.createdAt)} />
              </dl>
            </section>
          </aside>
        </div>
      </div>
    </DetailPageLayout>
  );
}

function ProposalList({
  proposals,
  isMutating,
  onApply,
}: {
  proposals: AgentSessionProposal[];
  isMutating: boolean;
  onApply: (proposalId: string) => Promise<void>;
}) {
  return (
    <section className="rounded-lg border border-white/10 bg-slate-950/30 p-3" data-testid="agent-session-proposals">
      <div className="flex items-center gap-2">
        <GitPullRequestArrow className="h-4 w-4 text-violet-300" />
        <h4 className="text-xs font-medium text-slate-300">Proposals</h4>
        <span className="ml-auto text-[11px] text-slate-500">{proposals.length}</span>
      </div>
      <div className="mt-2 space-y-2">
        {proposals.length > 0 ? (
          proposals.map((proposal) => (
            <article key={proposal.id} className="rounded-md border border-white/10 bg-slate-900/70 p-2">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate text-xs font-medium text-slate-100">{formatDisplayText(proposal.kind)}</p>
                  <p className="mt-1 line-clamp-3 text-[11px] leading-5 text-slate-400">{proposal.summary}</p>
                </div>
                <span className="shrink-0 rounded-full bg-slate-800 px-2 py-0.5 text-[10px] text-slate-300">
                  {formatDisplayText(proposal.status)}
                </span>
              </div>
              {proposal.status === "ready" && (
                <Button
                  className="mt-2 w-full"
                  size="sm"
                  onClick={() => void onApply(proposal.id)}
                  disabled={isMutating}
                  data-testid="agent-session-apply-proposal"
                >
                  <CheckCircle2 className="mr-2 h-4 w-4" />
                  Apply
                </Button>
              )}
            </article>
          ))
        ) : (
          <p className="py-4 text-center text-xs text-slate-500">No proposals yet.</p>
        )}
      </div>
    </section>
  );
}

function ArtifactList({
  artifacts,
  onOpenArtifact,
}: {
  artifacts: AgentSessionArtifact[];
  onOpenArtifact: (artifact: AgentSessionArtifact) => void;
}) {
  return (
    <section className="rounded-lg border border-white/10 bg-slate-950/30 p-3" data-testid="agent-session-artifacts">
      <div className="flex items-center gap-2">
        <Layers3 className="h-4 w-4 text-cyan-300" />
        <h4 className="text-xs font-medium text-slate-300">Artifacts</h4>
        <span className="ml-auto text-[11px] text-slate-500">{artifacts.length}</span>
      </div>
      <div className="mt-2 space-y-2">
        {artifacts.length > 0 ? (
          artifacts.map((artifact) => {
            const canOpen = nodeIdForArtifact(artifact) !== null;
            return (
              <button
                key={artifact.id}
                type="button"
                onClick={() => onOpenArtifact(artifact)}
                className="w-full rounded-md border border-white/10 bg-slate-900/70 p-2 text-left transition-colors hover:border-slate-700 hover:bg-slate-800/70 disabled:pointer-events-none disabled:opacity-60"
                disabled={!canOpen}
                data-testid="agent-session-artifact"
              >
                <div className="flex items-start gap-2">
                  <FileText className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-400" />
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-xs font-medium text-slate-100">{artifact.title || artifact.entityRef}</p>
                    <p className="mt-0.5 text-[11px] text-slate-500">
                      {formatDisplayText(artifact.action)} {formatDisplayText(artifact.artifactType)}
                    </p>
                  </div>
                  {canOpen && <ExternalLink className="h-3.5 w-3.5 shrink-0 text-slate-500" />}
                </div>
              </button>
            );
          })
        ) : (
          <p className="py-4 text-center text-xs text-slate-500">No artifacts linked yet.</p>
        )}
      </div>
    </section>
  );
}

function RunDetail({ label, value }: { label: string; value?: string }) {
  if (!value) return null;
  return (
    <div className="flex min-w-0 justify-between gap-2">
      <dt className="shrink-0 text-slate-500">{label}</dt>
      <dd className="min-w-0 truncate text-right text-slate-300">{value}</dd>
    </div>
  );
}

function nodeIdForArtifact(artifact: AgentSessionArtifact): string | null {
  const ref = artifact.entityRef?.trim();
  if (!ref) return null;

  switch (artifact.artifactType) {
    case "backlog_item": {
      const slashIndex = ref.indexOf("/");
      if (slashIndex <= 0) return null;
      return buildBacklogNodeId(ref.slice(0, slashIndex), ref.slice(slashIndex + 1));
    }
    case "initiative":
      return `initiative/${ref}`;
    case "capture":
      return `capture/${ref}`;
    case "agent_activity":
      return buildActivityNodeId(ref);
    default:
      return null;
  }
}
