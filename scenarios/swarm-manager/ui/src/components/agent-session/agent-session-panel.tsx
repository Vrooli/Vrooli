import { useCallback, useMemo, useRef, useState } from "react";
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
import { FloatingPanel } from "../ui/floating-panel";
import { Button } from "../ui/button";
import { Textarea } from "../ui/textarea";
import { cn } from "../../lib/utils";
import { formatDisplayText, formatRelativeTime } from "../../lib/format-utils";
import { useAutoResizeTextarea } from "../../hooks/useAutoResizeTextarea";
import { useAgentSessionStore } from "../../stores";
import type { AgentSession, AgentSessionArtifact, AgentSessionProposal } from "../../types";

const INITIAL_POSITION = {
  x: Math.max(8, window.innerWidth - 720),
  y: Math.max(8, window.innerHeight * 0.12),
};

const MAX_TEXTAREA_HEIGHT = 104;

const KIND_LABELS: Record<AgentSession["kind"], string> = {
  meta_orchestration: "Plan work",
  operating_mode_authoring: "Author operating mode",
};

const KIND_ICONS = {
  meta_orchestration: Workflow,
  operating_mode_authoring: GitPullRequestArrow,
};

const STATUS_STYLES: Record<AgentSession["status"], string> = {
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

interface AgentSessionPanelProps {
  onOpenArtifact?: (artifact: AgentSessionArtifact) => void;
}

export function AgentSessionPanel({ onOpenArtifact }: AgentSessionPanelProps) {
  const session = useAgentSessionStore((s) => s.activeSession);
  const setActiveSession = useAgentSessionStore((s) => s.setActiveSession);
  const continueSession = useAgentSessionStore((s) => s.continueSession);
  const refreshSession = useAgentSessionStore((s) => s.refreshSession);
  const cancelSession = useAgentSessionStore((s) => s.cancelSession);
  const applyProposal = useAgentSessionStore((s) => s.applyProposal);
  const isMutating = useAgentSessionStore((s) => s.isMutating);
  const isRefreshing = useAgentSessionStore((s) => s.isRefreshing);
  const error = useAgentSessionStore((s) => s.error);
  const [draft, setDraft] = useState("");
  const [localError, setLocalError] = useState<string | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useAutoResizeTextarea(textareaRef, draft, { maxHeight: MAX_TEXTAREA_HEIGHT });

  const isOpen = Boolean(session);
  const canSend = Boolean(session && draft.trim() && !isMutating);
  const Icon = session ? KIND_ICONS[session.kind] : Bot;
  const title = session?.title || "Agent session";

  const sortedMessages = useMemo(
    () => [...(session?.messages ?? [])].sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()),
    [session?.messages],
  );

  const handleClose = useCallback(() => {
    setDraft("");
    setLocalError(null);
    setActiveSession(null);
  }, [setActiveSession]);

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

  if (!session) return null;

  return (
    <FloatingPanel
      isOpen={isOpen}
      onClose={handleClose}
      title={title}
      initialPosition={INITIAL_POSITION}
      className="max-w-3xl"
      testId="agent-session-panel"
    >
      <div className="flex h-[76vh] flex-col gap-3">
        <header className="rounded-lg border border-white/10 bg-slate-950/40 p-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="flex items-center gap-2 text-xs text-slate-400">
                <Icon className="h-4 w-4 text-cyan-300" />
                <span>{KIND_LABELS[session.kind]}</span>
                <span className={cn("rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_STYLES[session.status])}>
                  {formatDisplayText(session.status)}
                </span>
              </div>
              <h3 className="mt-1 truncate text-base font-semibold text-slate-100">{session.title}</h3>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              <Button variant="ghost" size="icon" onClick={handleRefresh} disabled={isMutating || isRefreshing} aria-label="Refresh session">
                <RefreshCw className={cn("h-4 w-4", isRefreshing && "animate-spin")} />
              </Button>
              <Button variant="ghost" size="icon" onClick={handleCancel} disabled={isMutating || session.status === "canceled" || session.status === "complete"} aria-label="Cancel session">
                <Square className="h-4 w-4" />
              </Button>
            </div>
          </div>
          <dl className="mt-3 grid gap-2 text-[11px] text-slate-400 sm:grid-cols-2">
            <RunDetail label="Run" value={session.runId} />
            <RunDetail label="Task" value={session.taskId} />
            <RunDetail label="Profile" value={session.profileKey} />
            <RunDetail label="Updated" value={formatRelativeTime(session.updatedAt)} />
          </dl>
        </header>

        {(localError || error?.message) && (
          <div className="flex items-start gap-2 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200" role="alert">
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <span className="min-w-0 break-words">{localError || error?.message}</span>
          </div>
        )}

        <div className="grid min-h-0 flex-1 gap-3 lg:grid-cols-[minmax(0,1fr)_280px]">
          <section className="flex min-h-0 flex-col rounded-lg border border-white/10 bg-slate-950/30">
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

          <aside className="min-h-0 space-y-3 overflow-y-auto">
            <ProposalList proposals={session.proposals} isMutating={isMutating} onApply={handleApply} />
            <ArtifactList artifacts={session.artifacts} onOpenArtifact={onOpenArtifact} />
            <section className="rounded-lg border border-white/10 bg-slate-950/30 p-3">
              <h4 className="text-xs font-medium text-slate-300">Session details</h4>
              <dl className="mt-2 space-y-1 text-[11px] text-slate-400">
                <RunDetail label="Session ID" value={session.id} />
                <RunDetail label="Skill" value={session.skillId} />
                <RunDetail label="Created" value={formatRelativeTime(session.createdAt)} />
              </dl>
            </section>
          </aside>
        </div>
      </div>
    </FloatingPanel>
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
  onOpenArtifact?: (artifact: AgentSessionArtifact) => void;
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
          artifacts.map((artifact) => (
            <button
              key={artifact.id}
              type="button"
              onClick={() => onOpenArtifact?.(artifact)}
              className="w-full rounded-md border border-white/10 bg-slate-900/70 p-2 text-left transition-colors hover:border-slate-700 hover:bg-slate-800/70 disabled:pointer-events-none"
              disabled={!onOpenArtifact}
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
                {onOpenArtifact && <ExternalLink className="h-3.5 w-3.5 shrink-0 text-slate-500" />}
              </div>
            </button>
          ))
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
