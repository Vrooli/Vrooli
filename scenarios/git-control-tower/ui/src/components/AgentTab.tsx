import { useState, useEffect, useCallback, useRef } from "react";
import { Loader2, Play, Square, CheckCircle2, XCircle, AlertTriangle, ChevronDown, ChevronRight, X, Send, Eye, MessageSquare, Bot } from "lucide-react";
import { Button } from "./ui/button";
import {
  useAgentProfiles,
  useAgentRuns,
  useAgentRun,
  useAgentRunEvents,
  useAgentRunDiff,
  useCreateAgentRun,
  useContinueAgentRun,
  useApproveAgentRun,
  useRejectAgentRun,
  useStopAgentRun,
} from "../lib/hooks";
import { composePrompt } from "../lib/agentContext";
import type { AgentContextItem, AgentRunEvent, AgentRunStatus, AgentRunDiffFile } from "../lib/api";

const AGENT_PROFILE_KEY = "gct.agent.defaultProfileId";

interface AgentTabProps {
  scenarioSlug: string;
  repoId?: string | null;
  agentManagerAvailable: boolean;
  contextItems: AgentContextItem[];
  onRemoveContext: (id: string) => void;
  onClearContext: () => void;
}

const STATUS_COLORS: Record<AgentRunStatus, string> = {
  pending: "bg-slate-500",
  starting: "bg-blue-500",
  running: "bg-blue-500 animate-pulse",
  needs_review: "bg-amber-500",
  complete: "bg-emerald-500",
  failed: "bg-red-500",
  cancelled: "bg-slate-600",
};

const STATUS_LABELS: Record<AgentRunStatus, string> = {
  pending: "Pending",
  starting: "Starting",
  running: "Running",
  needs_review: "Needs Review",
  complete: "Complete",
  failed: "Failed",
  cancelled: "Cancelled",
};

function StatusBadge({ status }: { status: AgentRunStatus }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className={`h-2 w-2 rounded-full ${STATUS_COLORS[status]}`} />
      <span className="text-xs text-slate-300">{STATUS_LABELS[status]}</span>
    </span>
  );
}

export function AgentTab({
  scenarioSlug,
  repoId,
  agentManagerAvailable,
  contextItems,
  onRemoveContext,
  onClearContext,
}: AgentTabProps) {
  const [message, setMessage] = useState("");
  const [selectedProfileId, setSelectedProfileId] = useState<string>(() => {
    try { return localStorage.getItem(AGENT_PROFILE_KEY) ?? ""; } catch { return ""; }
  });
  const [activeRunId, setActiveRunId] = useState<string | null>(null);
  const [events, setEvents] = useState<AgentRunEvent[]>([]);
  const [lastEventSequence, setLastEventSequence] = useState(0);
  const [showDiff, setShowDiff] = useState(false);
  const [showHistory, setShowHistory] = useState(false);
  const [followUpMessage, setFollowUpMessage] = useState("");
  const eventsEndRef = useRef<HTMLDivElement>(null);

  const profiles = useAgentProfiles(agentManagerAvailable);
  const runs = useAgentRuns(scenarioSlug, agentManagerAvailable, repoId);
  const activeRun = useAgentRun(activeRunId, agentManagerAvailable, repoId);
  const runEvents = useAgentRunEvents(activeRunId, lastEventSequence, agentManagerAvailable && !!activeRunId, repoId);
  const runDiff = useAgentRunDiff(activeRunId, showDiff && agentManagerAvailable, repoId);

  const createRun = useCreateAgentRun(repoId);
  const continueRun = useContinueAgentRun(repoId);
  const approve = useApproveAgentRun(repoId);
  const reject = useRejectAgentRun(repoId);
  const stop = useStopAgentRun(repoId);

  // Auto-detect active run on mount
  useEffect(() => {
    if (!activeRunId && runs.data?.runs?.length) {
      const active = runs.data.runs.find((r) =>
        ["pending", "starting", "running", "needs_review"].includes(r.status)
      );
      if (active) setActiveRunId(active.id);
    }
  }, [activeRunId, runs.data]);

  // Accumulate events
  useEffect(() => {
    if (runEvents.data?.events?.length) {
      setEvents((prev) => {
        const incoming = runEvents.data?.events ?? [];
        const newEvents = incoming.filter(
          (e) => !prev.some((p) => p.id === e.id)
        );
        if (newEvents.length === 0) return prev;
        const merged = [...prev, ...newEvents].sort((a, b) => a.sequence - b.sequence);
        const lastSeq = merged[merged.length - 1]?.sequence ?? 0;
        setLastEventSequence(lastSeq);
        return merged;
      });
    }
  }, [runEvents.data]);

  // Auto-scroll events
  useEffect(() => {
    eventsEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [events.length]);

  const handleSend = useCallback(() => {
    const prompt = composePrompt(message, contextItems);
    if (!prompt.trim()) return;

    createRun.mutate(
      {
        scenarioSlug,
        prompt,
        profileId: selectedProfileId || undefined,
      },
      {
        onSuccess: (resp) => {
          setActiveRunId(resp.runId);
          setEvents([]);
          setLastEventSequence(0);
          setShowDiff(false);
          setMessage("");
          onClearContext();
        },
      }
    );
  }, [message, contextItems, scenarioSlug, selectedProfileId, createRun, onClearContext]);

  const handleContinue = useCallback(() => {
    if (!activeRunId || !followUpMessage.trim()) return;
    continueRun.mutate(
      { runId: activeRunId, request: { message: followUpMessage } },
      {
        onSuccess: () => {
          setFollowUpMessage("");
          setShowDiff(false);
        },
      }
    );
  }, [activeRunId, followUpMessage, continueRun]);

  const handleSetDefaultProfile = useCallback((id: string) => {
    setSelectedProfileId(id);
    try { localStorage.setItem(AGENT_PROFILE_KEY, id); } catch { /* ignore */ }
  }, []);

  if (!agentManagerAvailable) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <Bot className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm">Agent Manager is not available</p>
        <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">
          Start the agent-manager scenario to use AI agents for fixing issues
        </p>
      </div>
    );
  }

  const isActive = activeRun.data && ["pending", "starting", "running"].includes(activeRun.data.status);
  const isCreating = createRun.isPending;

  return (
    <div className="space-y-4">
      {/* Context chips */}
      {contextItems.length > 0 && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <h3 className="text-xs font-medium text-slate-400">Attached Context</h3>
            <button
              type="button"
              className="text-[11px] text-slate-500 hover:text-slate-300"
              onClick={onClearContext}
            >
              Clear all
            </button>
          </div>
          <div className="flex flex-wrap gap-1.5">
            {contextItems.map((item) => (
              <span
                key={item.id}
                className="inline-flex items-center gap-1 px-2 py-1 rounded-md bg-slate-800 border border-slate-700 text-[11px] text-slate-300"
              >
                {item.label}
                <button
                  type="button"
                  className="text-slate-500 hover:text-slate-300"
                  onClick={() => onRemoveContext(item.id)}
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Message input + send */}
      {!activeRunId && (
        <div className="space-y-2">
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSend();
              }
            }}
            placeholder="Describe the issue or what you'd like the agent to fix..."
            className="w-full h-24 px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-200 placeholder-slate-500 resize-none focus:outline-none focus:border-blue-500"
            disabled={isCreating}
          />
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              {profiles.data?.profiles && profiles.data.profiles.length > 0 && (
                <select
                  value={selectedProfileId}
                  onChange={(e) => handleSetDefaultProfile(e.target.value)}
                  className="h-7 px-2 bg-slate-800 border border-slate-700 rounded text-xs text-slate-300 focus:outline-none focus:border-blue-500"
                >
                  <option value="">Default profile</option>
                  {profiles.data.profiles.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </select>
              )}
            </div>
            <Button
              variant="default"
              size="sm"
              onClick={handleSend}
              disabled={isCreating || (!message.trim() && contextItems.length === 0)}
              className="h-7 text-xs gap-1"
            >
              {isCreating ? (
                <Loader2 className="h-3 w-3 animate-spin" />
              ) : (
                <Send className="h-3 w-3" />
              )}
              Send
            </Button>
          </div>
        </div>
      )}

      {/* Active run display */}
      {activeRunId && activeRun.data && (
        <div className="space-y-3">
          {/* Status header */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <StatusBadge status={activeRun.data.status} />
              {activeRun.data.phase && (
                <span className="text-[11px] text-slate-500">{activeRun.data.phase}</span>
              )}
            </div>
            {activeRun.data.actions?.canStop && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => stop.mutate(activeRunId)}
                disabled={stop.isPending}
                className="h-6 text-[11px] gap-1 text-red-400 border-red-800 hover:bg-red-950/50"
              >
                <Square className="h-3 w-3" />
                Stop
              </Button>
            )}
          </div>

          {/* Progress bar */}
          {activeRun.data.progressPercent != null && activeRun.data.progressPercent > 0 && (
            <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-blue-500 rounded-full transition-all"
                style={{ width: `${activeRun.data.progressPercent}%` }}
              />
            </div>
          )}

          {/* Error display */}
          {activeRun.data.errorMsg && (
            <div className="flex items-start gap-2 p-3 rounded-lg bg-red-950/30 border border-red-900/40">
              <AlertTriangle className="h-4 w-4 text-red-400 mt-0.5 shrink-0" />
              <p className="text-xs text-red-300">{activeRun.data.errorMsg}</p>
            </div>
          )}

          {/* Event stream */}
          {events.length > 0 && (
            <div className="rounded-lg border border-slate-800 bg-slate-900/50 max-h-64 overflow-y-auto">
              <div className="p-3 space-y-2">
                {events.map((evt) => (
                  <EventItem key={evt.id} event={evt} />
                ))}
                <div ref={eventsEndRef} />
              </div>
            </div>
          )}

          {/* Summary */}
          {activeRun.data.summary && (
            <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-3">
              <h4 className="text-xs font-medium text-slate-400 mb-2">Summary</h4>
              {activeRun.data.summary.description && (
                <p className="text-xs text-slate-300 mb-2">{activeRun.data.summary.description}</p>
              )}
              <div className="flex gap-4 text-[11px] text-slate-400">
                <span>Modified: {activeRun.data.summary.filesModified}</span>
                <span>Created: {activeRun.data.summary.filesCreated}</span>
                <span>Deleted: {activeRun.data.summary.filesDeleted}</span>
                {activeRun.data.summary.tokensUsed != null && (
                  <span>Tokens: {activeRun.data.summary.tokensUsed.toLocaleString()}</span>
                )}
              </div>
            </div>
          )}

          {/* Run actions */}
          {activeRun.data.actions && (
            <div className="flex items-center gap-2 flex-wrap">
              {activeRun.data.actions.canApprove && (
                <>
                  <Button
                    variant="default"
                    size="sm"
                    onClick={() => setShowDiff(!showDiff)}
                    className="h-7 text-xs gap-1"
                  >
                    <Eye className="h-3 w-3" />
                    {showDiff ? "Hide Diff" : "View Diff"}
                  </Button>
                  <Button
                    variant="default"
                    size="sm"
                    onClick={() => approve.mutate({ runId: activeRunId, request: {} })}
                    disabled={approve.isPending}
                    className="h-7 text-xs gap-1 bg-emerald-700 hover:bg-emerald-600"
                  >
                    {approve.isPending ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <CheckCircle2 className="h-3 w-3" />
                    )}
                    Approve
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => reject.mutate({ runId: activeRunId, request: {} })}
                    disabled={reject.isPending}
                    className="h-7 text-xs gap-1 text-red-400 border-red-800 hover:bg-red-950/50"
                  >
                    {reject.isPending ? (
                      <Loader2 className="h-3 w-3 animate-spin" />
                    ) : (
                      <XCircle className="h-3 w-3" />
                    )}
                    Reject
                  </Button>
                </>
              )}
            </div>
          )}

          {/* Diff viewer */}
          {showDiff && runDiff.data && (
            <DiffViewer files={runDiff.data.files} isLoading={runDiff.isLoading} />
          )}

          {/* Follow-up input */}
          {activeRun.data.actions?.canContinue && (
            <div className="flex items-center gap-2">
              <input
                type="text"
                value={followUpMessage}
                onChange={(e) => setFollowUpMessage(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    handleContinue();
                  }
                }}
                placeholder="Send a follow-up message..."
                className="flex-1 h-8 px-3 bg-slate-900 border border-slate-700 rounded text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-blue-500"
                disabled={continueRun.isPending}
              />
              <Button
                variant="default"
                size="sm"
                onClick={handleContinue}
                disabled={continueRun.isPending || !followUpMessage.trim()}
                className="h-8 text-xs gap-1"
              >
                {continueRun.isPending ? (
                  <Loader2 className="h-3 w-3 animate-spin" />
                ) : (
                  <MessageSquare className="h-3 w-3" />
                )}
              </Button>
            </div>
          )}

          {/* New run button (for terminal states) */}
          {!isActive && activeRun.data.status !== "needs_review" && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setActiveRunId(null);
                setEvents([]);
                setLastEventSequence(0);
                setShowDiff(false);
              }}
              className="h-7 text-xs gap-1"
            >
              <Play className="h-3 w-3" />
              New Run
            </Button>
          )}
        </div>
      )}

      {/* Recent runs */}
      {runs.data && runs.data.runs.length > 0 && (
        <div>
          <button
            type="button"
            className="flex items-center gap-1 text-xs text-slate-400 hover:text-slate-200"
            onClick={() => setShowHistory(!showHistory)}
          >
            {showHistory ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
            Recent runs ({runs.data.runs.length})
          </button>
          {showHistory && (
            <div className="mt-2 space-y-1">
              {runs.data.runs.map((run) => (
                <button
                  key={run.id}
                  type="button"
                  onClick={() => {
                    setActiveRunId(run.id);
                    setEvents([]);
                    setLastEventSequence(0);
                    setShowDiff(false);
                  }}
                  className={`w-full flex items-center justify-between px-3 py-2 rounded text-xs hover:bg-slate-800/60 ${
                    run.id === activeRunId ? "bg-slate-800 border border-slate-700" : ""
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <StatusBadge status={run.status} />
                    <span className="text-slate-500 text-[11px]">
                      {run.id.slice(0, 8)}
                    </span>
                  </div>
                  <span className="text-[11px] text-slate-600">
                    {new Date(run.createdAt).toLocaleString()}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ── Sub-components ───────────────────────────────────────────────────

function EventItem({ event }: { event: AgentRunEvent }) {
  const data = event.data as Record<string, unknown> | undefined;
  const text = (data?.message ?? data?.content ?? data?.name ?? "") as string;

  const iconMap: Record<string, JSX.Element> = {
    message: <MessageSquare className="h-3 w-3 text-blue-400" />,
    tool_call: <Play className="h-3 w-3 text-amber-400" />,
    tool_result: <CheckCircle2 className="h-3 w-3 text-emerald-400" />,
    error: <XCircle className="h-3 w-3 text-red-400" />,
    status_change: <Bot className="h-3 w-3 text-slate-400" />,
  };

  return (
    <div className="flex items-start gap-2 text-[11px]">
      <span className="mt-0.5 shrink-0">{iconMap[event.eventType] ?? <Bot className="h-3 w-3 text-slate-500" />}</span>
      <div className="min-w-0 flex-1">
        <span className="text-slate-500">{event.eventType}</span>
        {text && <span className="text-slate-300 ml-1.5">{text}</span>}
      </div>
      <span className="text-slate-600 shrink-0">
        {new Date(event.timestamp).toLocaleTimeString()}
      </span>
    </div>
  );
}

function DiffViewer({ files, isLoading }: { files: AgentRunDiffFile[]; isLoading: boolean }) {
  if (isLoading) {
    return (
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <div className="flex items-center gap-2 text-xs text-slate-400">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Loading diff...
        </div>
      </div>
    );
  }

  if (!files.length) {
    return (
      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
        <p className="text-xs text-slate-500">No file changes</p>
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/50 divide-y divide-slate-800">
      {files.map((file) => (
        <div key={file.path} className="p-3">
          <div className="flex items-center justify-between mb-1">
            <code className="text-xs text-slate-300">{file.path}</code>
            <div className="flex items-center gap-2 text-[11px]">
              <span className={`${file.status === "added" ? "text-emerald-400" : file.status === "deleted" ? "text-red-400" : "text-blue-400"}`}>
                {file.status}
              </span>
              <span className="text-emerald-500">+{file.additions}</span>
              <span className="text-red-500">-{file.deletions}</span>
            </div>
          </div>
          {file.patch && (
            <pre className="mt-2 p-2 bg-slate-950 rounded text-[11px] text-slate-400 overflow-x-auto max-h-48">{file.patch}</pre>
          )}
        </div>
      ))}
    </div>
  );
}

/** Small inline button used in other tabs to attach context. */
export function AttachToAgentButton({ onClick, label }: { onClick: () => void; label?: string }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded border border-blue-800/50 text-[11px] text-blue-400 hover:text-blue-300 hover:border-blue-700 transition-colors"
    >
      <Bot className="h-3 w-3" />
      {label ?? "+ Agent"}
    </button>
  );
}
