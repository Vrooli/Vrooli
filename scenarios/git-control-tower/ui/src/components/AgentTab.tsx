import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import {
  Loader2,
  Square,
  CheckCircle2,
  XCircle,
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  X,
  Send,
  Bot,
  ArrowDown,
  Wrench,
  Clock,
} from "lucide-react";
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
import { ContextPickerPopover } from "./ContextPickerPopover";
import type {
  AgentContextItem,
  AgentRunEvent,
  AgentRunStatus,
  AgentRunDiffFile,
  AgentRunSummary,
  AgentRunActions,
  RepoFileStats,
} from "../lib/api";

const AGENT_PROFILE_KEY = "gct.agent.defaultProfileId";

// ── Chat message types ──────────────────────────────────────────────

type ChatMessage =
  | { type: "user"; text: string; contextCount: number; timestamp: string }
  | { type: "agent-message"; text: string; timestamp: string }
  | { type: "tool-group"; tools: { name: string; result?: string }[]; timestamp: string }
  | { type: "status"; status: AgentRunStatus; phase?: string; timestamp: string }
  | { type: "error"; text: string; timestamp: string }
  | { type: "summary"; summary: AgentRunSummary; timestamp: string }
  | { type: "diff"; files: AgentRunDiffFile[]; runId: string; actions: AgentRunActions }
  | { type: "action-prompt"; actions: AgentRunActions; runId: string };

interface SentMessage {
  text: string;
  contextCount: number;
  runId: string;
  timestamp: string;
}

// ── Status helpers ──────────────────────────────────────────────────

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

// ── Props ───────────────────────────────────────────────────────────

interface AgentTabProps {
  scenarioSlug: string;
  repoId?: string | null;
  agentManagerAvailable: boolean;
  contextItems: AgentContextItem[];
  onAddContext: (item: AgentContextItem) => void;
  onRemoveContext: (id: string) => void;
  onClearContext: () => void;
  testGenieAvailable: boolean;
  tidinessAvailable: boolean;
  fileStats?: RepoFileStats;
}

// ── Build chat messages from events ─────────────────────────────────

function buildChatMessages(
  sentMessages: SentMessage[],
  events: AgentRunEvent[],
  activeRun: { status: AgentRunStatus; phase?: string; summary?: AgentRunSummary; errorMsg?: string; actions?: AgentRunActions } | null,
  diffFiles: AgentRunDiffFile[] | null,
  runId: string | null,
): ChatMessage[] {
  const messages: ChatMessage[] = [];

  // Add sent user messages
  for (const sent of sentMessages) {
    if (sent.runId === runId) {
      messages.push({
        type: "user",
        text: sent.text,
        contextCount: sent.contextCount,
        timestamp: sent.timestamp,
      });
    }
  }

  // Walk events
  let i = 0;
  while (i < events.length) {
    const evt = events[i]!;
    const data = evt.data as Record<string, unknown> | undefined;

    if (evt.eventType === "message") {
      const text = (data?.message ?? data?.content ?? "") as string;
      if (text) {
        messages.push({ type: "agent-message", text, timestamp: evt.timestamp });
      }
    } else if (evt.eventType === "tool_call") {
      // Group consecutive tool_call/tool_result pairs
      const tools: { name: string; result?: string }[] = [];
      while (i < events.length) {
        const e = events[i]!;
        if (e.eventType !== "tool_call" && e.eventType !== "tool_result") break;
        const d = e.data as Record<string, unknown> | undefined;
        if (e.eventType === "tool_call") {
          tools.push({ name: (d?.name ?? "tool") as string });
        } else if (e.eventType === "tool_result" && tools.length > 0) {
          const last = tools[tools.length - 1]!;
          if (!last.result) {
            last.result = ((d?.result ?? d?.content ?? "") as string).slice(0, 500);
          }
        }
        i++;
      }
      if (tools.length > 0) {
        messages.push({ type: "tool-group", tools, timestamp: evt.timestamp });
      }
      continue; // already advanced i
    } else if (evt.eventType === "error") {
      const text = (data?.message ?? data?.error ?? "") as string;
      if (text) {
        messages.push({ type: "error", text, timestamp: evt.timestamp });
      }
    } else if (evt.eventType === "status_change") {
      const status = (data?.status ?? data?.newStatus) as AgentRunStatus | undefined;
      if (status) {
        messages.push({ type: "status", status, phase: data?.phase as string | undefined, timestamp: evt.timestamp });
      }
    }
    i++;
  }

  // Append error from run
  if (activeRun?.errorMsg && !messages.some((m) => m.type === "error" && m.text === activeRun.errorMsg)) {
    messages.push({ type: "error", text: activeRun.errorMsg, timestamp: new Date().toISOString() });
  }

  // Append summary
  if (activeRun?.summary) {
    messages.push({ type: "summary", summary: activeRun.summary, timestamp: new Date().toISOString() });
  }

  // Append diff + action-prompt when needs_review
  if (activeRun?.status === "needs_review" && activeRun.actions && runId) {
    if (diffFiles && diffFiles.length > 0) {
      messages.push({ type: "diff", files: diffFiles, runId, actions: activeRun.actions });
    }
    messages.push({ type: "action-prompt", actions: activeRun.actions, runId });
  }

  return messages;
}

// ── Main component ──────────────────────────────────────────────────

export function AgentTab({
  scenarioSlug,
  repoId,
  agentManagerAvailable,
  contextItems,
  onAddContext,
  onRemoveContext,
  onClearContext,
  testGenieAvailable,
  tidinessAvailable,
  fileStats,
}: AgentTabProps) {
  const [message, setMessage] = useState("");
  const [selectedProfileId, setSelectedProfileId] = useState<string>(() => {
    try { return localStorage.getItem(AGENT_PROFILE_KEY) ?? ""; } catch { return ""; }
  });
  const [activeRunId, setActiveRunId] = useState<string | null>(null);
  const [events, setEvents] = useState<AgentRunEvent[]>([]);
  const [lastEventSequence, setLastEventSequence] = useState(0);
  const [sentMessages, setSentMessages] = useState<SentMessage[]>([]);
  const [showRunHistory, setShowRunHistory] = useState(false);

  // Scroll tracking
  const chatEndRef = useRef<HTMLDivElement>(null);
  const chatContainerRef = useRef<HTMLDivElement>(null);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [hasNewMessages, setHasNewMessages] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const profiles = useAgentProfiles(agentManagerAvailable);
  const runs = useAgentRuns(scenarioSlug, agentManagerAvailable, repoId);
  const activeRun = useAgentRun(activeRunId, agentManagerAvailable, repoId);

  const isActiveRun = activeRun.data && ["pending", "starting", "running", "needs_review"].includes(activeRun.data.status);
  const shouldPollDiff = activeRun.data?.status === "needs_review" && agentManagerAvailable;
  const runEvents = useAgentRunEvents(activeRunId, lastEventSequence, agentManagerAvailable && !!activeRunId, repoId);
  const runDiff = useAgentRunDiff(activeRunId, shouldPollDiff, repoId);

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

  // Scroll tracking
  const handleScroll = useCallback(() => {
    const el = chatContainerRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
    setIsNearBottom(nearBottom);
    if (nearBottom) setHasNewMessages(false);
  }, []);

  // Auto-scroll on new messages
  useEffect(() => {
    if (isNearBottom) {
      chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
    } else {
      setHasNewMessages(true);
    }
  }, [events.length, activeRun.data?.status, isNearBottom]);

  const scrollToBottom = useCallback(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
    setHasNewMessages(false);
  }, []);

  // Auto-resize textarea
  const autoResize = useCallback(() => {
    const el = textareaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 144) + "px"; // max 6 lines ~144px
  }, []);

  useEffect(() => {
    autoResize();
  }, [message, autoResize]);

  const handleSend = useCallback(() => {
    const prompt = composePrompt(message, contextItems);
    if (!prompt.trim()) return;

    const profileKey = selectedProfileId ? undefined : "git-control-tower-reviewer";

    createRun.mutate(
      {
        scenarioSlug,
        prompt,
        profileId: selectedProfileId || undefined,
        profileKey,
      },
      {
        onSuccess: (resp) => {
          setSentMessages((prev) => [
            ...prev,
            {
              text: message,
              contextCount: contextItems.length,
              runId: resp.runId,
              timestamp: new Date().toISOString(),
            },
          ]);
          setActiveRunId(resp.runId);
          setEvents([]);
          setLastEventSequence(0);
          setMessage("");
          onClearContext();
        },
      }
    );
  }, [message, contextItems, scenarioSlug, selectedProfileId, createRun, onClearContext]);

  const handleContinue = useCallback(() => {
    if (!activeRunId || !message.trim()) return;
    const followUp = message.trim();
    continueRun.mutate(
      { runId: activeRunId, request: { message: followUp } },
      {
        onSuccess: () => {
          setSentMessages((prev) => [
            ...prev,
            {
              text: followUp,
              contextCount: 0,
              runId: activeRunId,
              timestamp: new Date().toISOString(),
            },
          ]);
          setMessage("");
        },
      }
    );
  }, [activeRunId, message, continueRun]);

  const handleSetDefaultProfile = useCallback((id: string) => {
    setSelectedProfileId(id);
    try { localStorage.setItem(AGENT_PROFILE_KEY, id); } catch { /* ignore */ }
  }, []);

  const chatMessages = useMemo(
    () =>
      buildChatMessages(
        sentMessages,
        events,
        activeRun.data ?? null,
        runDiff.data?.files ?? null,
        activeRunId,
      ),
    [sentMessages, events, activeRun.data, runDiff.data, activeRunId]
  );

  if (!agentManagerAvailable) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500 h-full">
        <Bot className="h-8 w-8 mb-3 opacity-50" />
        <p className="text-sm">Agent Manager is not available</p>
        <p className="text-xs mt-2 text-slate-600 text-center max-w-sm">
          Start the agent-manager scenario to use AI agents for fixing issues
        </p>
      </div>
    );
  }

  const isRunning = activeRun.data && ["pending", "starting", "running"].includes(activeRun.data.status);
  const canContinue = activeRun.data?.actions?.canContinue ?? false;
  const isCreating = createRun.isPending;
  const isContinuing = continueRun.isPending;

  // Determine input behavior
  const canSendNew = !activeRunId || (!isActiveRun && activeRun.data?.status !== "needs_review");
  const inputDisabled = isRunning || isCreating || isContinuing;
  const placeholder = isRunning
    ? "Agent is working..."
    : canContinue
      ? "Send a follow-up..."
      : canSendNew
        ? "Ask the agent to fix something..."
        : "Waiting for review...";

  const handleInputSend = () => {
    if (canContinue && activeRunId) {
      handleContinue();
    } else if (canSendNew) {
      handleSend();
    }
  };

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-slate-800 shrink-0">
        <div className="flex items-center gap-2">
          {activeRun.data && <StatusBadge status={activeRun.data.status} />}
          {activeRun.data?.phase && (
            <span className="text-[11px] text-slate-500">{activeRun.data.phase}</span>
          )}
          {/* Run history dropdown */}
          {runs.data && runs.data.runs.length > 0 && (
            <div className="relative">
              <button
                type="button"
                onClick={() => setShowRunHistory((v) => !v)}
                className="flex items-center gap-1 text-[11px] text-slate-500 hover:text-slate-300"
              >
                <Clock className="h-3 w-3" />
                Runs
                {showRunHistory ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
              </button>
              {showRunHistory && (
                <div className="absolute top-full mt-1 left-0 z-50 w-64 max-h-52 overflow-y-auto rounded-lg border border-slate-700 bg-slate-900 shadow-xl">
                  {runs.data.runs.map((run) => (
                    <button
                      key={run.id}
                      type="button"
                      onClick={() => {
                        setActiveRunId(run.id);
                        setEvents([]);
                        setLastEventSequence(0);
                        setShowRunHistory(false);
                      }}
                      className={`w-full flex items-center justify-between px-3 py-2 text-xs hover:bg-slate-800/60 ${
                        run.id === activeRunId ? "bg-slate-800" : ""
                      }`}
                    >
                      <div className="flex items-center gap-2">
                        <StatusBadge status={run.status} />
                        <span className="text-slate-500 text-[11px]">{run.id.slice(0, 8)}</span>
                      </div>
                      <span className="text-[11px] text-slate-600">
                        {new Date(run.createdAt).toLocaleTimeString()}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
        <div className="flex items-center gap-1">
          {activeRun.data?.actions?.canStop && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => activeRunId && stop.mutate(activeRunId)}
              disabled={stop.isPending}
              className="h-6 text-[11px] gap-1 text-red-400 border-red-800 hover:bg-red-950/50"
            >
              <Square className="h-3 w-3" />
              Stop
            </Button>
          )}
          {!isActiveRun && activeRunId && activeRun.data?.status !== "needs_review" && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setActiveRunId(null);
                setEvents([]);
                setLastEventSequence(0);
              }}
              className="h-6 text-[11px] gap-1"
            >
              New Run
            </Button>
          )}
        </div>
      </div>

      {/* Chat thread */}
      <div
        ref={chatContainerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto px-4 py-4 space-y-3"
      >
        {chatMessages.length === 0 && (
          <div className="flex flex-col items-center justify-center h-full text-slate-500">
            <Bot className="h-8 w-8 mb-3 opacity-30" />
            <p className="text-xs text-center">Send a message to start an agent run</p>
          </div>
        )}

        {chatMessages.map((msg, idx) => {
          switch (msg.type) {
            case "user":
              return <UserBubble key={idx} text={msg.text} contextCount={msg.contextCount} timestamp={msg.timestamp} />;
            case "agent-message":
              return <AgentBubble key={idx} text={msg.text} timestamp={msg.timestamp} />;
            case "tool-group":
              return <ToolGroupBubble key={idx} tools={msg.tools} timestamp={msg.timestamp} />;
            case "status":
              return <StatusPill key={idx} status={msg.status} phase={msg.phase} />;
            case "error":
              return <ErrorBubble key={idx} text={msg.text} />;
            case "summary":
              return <SummaryCard key={idx} summary={msg.summary} />;
            case "diff":
              return <DiffSection key={idx} files={msg.files} isLoading={runDiff.isLoading} />;
            case "action-prompt":
              return (
                <ActionButtons
                  key={idx}
                  runId={msg.runId}
                  actions={msg.actions}
                  approve={approve}
                  reject={reject}
                />
              );
          }
        })}

        {/* Progress bar for running state */}
        {activeRun.data?.progressPercent != null && activeRun.data.progressPercent > 0 && isRunning && (
          <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all"
              style={{ width: `${activeRun.data.progressPercent}%` }}
            />
          </div>
        )}

        <div ref={chatEndRef} />
      </div>

      {/* New messages indicator */}
      {hasNewMessages && (
        <div className="flex justify-center -mt-10 relative z-10 pb-2">
          <button
            type="button"
            onClick={scrollToBottom}
            className="flex items-center gap-1 px-3 py-1.5 rounded-full bg-blue-600 text-white text-xs shadow-lg hover:bg-blue-500 transition-colors"
          >
            <ArrowDown className="h-3 w-3" />
            New messages
          </button>
        </div>
      )}

      {/* Context chips above input */}
      {contextItems.length > 0 && (
        <div className="px-4 pb-1 flex flex-wrap gap-1 border-t border-slate-800/50 pt-2">
          {contextItems.map((item) => (
            <span
              key={item.id}
              className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-slate-800 border border-slate-700 text-[11px] text-slate-300"
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
          <button
            type="button"
            className="text-[11px] text-slate-500 hover:text-slate-300 px-1"
            onClick={onClearContext}
          >
            Clear
          </button>
        </div>
      )}

      {/* Input bar - always visible */}
      <div className="shrink-0 border-t border-slate-800 px-4 py-3 flex items-end gap-2">
        <ContextPickerPopover
          scenarioSlug={scenarioSlug}
          repoId={repoId}
          testGenieAvailable={testGenieAvailable}
          tidinessAvailable={tidinessAvailable}
          fileStats={fileStats}
          contextItems={contextItems}
          onAddContext={onAddContext}
          onRemoveContext={onRemoveContext}
        />
        <textarea
          ref={textareaRef}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              handleInputSend();
            }
          }}
          placeholder={placeholder}
          className="flex-1 min-h-[36px] max-h-[144px] px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-sm text-slate-200 placeholder-slate-500 resize-none focus:outline-none focus:border-blue-500"
          disabled={inputDisabled}
          rows={1}
        />
        <div className="flex items-center gap-1.5 shrink-0">
          {profiles.data?.profiles && profiles.data.profiles.length > 0 && (
            <select
              value={selectedProfileId}
              onChange={(e) => handleSetDefaultProfile(e.target.value)}
              className="h-9 px-2 bg-slate-800 border border-slate-700 rounded text-xs text-slate-300 focus:outline-none focus:border-blue-500"
            >
              <option value="">Default</option>
              {profiles.data.profiles.map((p) => (
                <option key={p.id} value={p.id}>{p.name}</option>
              ))}
            </select>
          )}
          <Button
            variant="default"
            size="sm"
            onClick={handleInputSend}
            disabled={inputDisabled || (!message.trim() && contextItems.length === 0)}
            className="h-9 w-9 p-0"
          >
            {isCreating || isContinuing ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
        </div>
      </div>
    </div>
  );
}

// ── Bubble components ───────────────────────────────────────────────

function UserBubble({ text, contextCount, timestamp }: { text: string; contextCount: number; timestamp: string }) {
  return (
    <div className="flex justify-end">
      <div className="max-w-[80%] rounded-lg px-3 py-2 bg-blue-900/40 border border-blue-800/50">
        <p className="text-sm text-slate-200 whitespace-pre-wrap">{text}</p>
        <div className="flex items-center justify-end gap-2 mt-1">
          {contextCount > 0 && (
            <span className="text-[10px] text-blue-400/70">{contextCount} context item{contextCount > 1 ? "s" : ""}</span>
          )}
          <span className="text-[10px] text-slate-500">{new Date(timestamp).toLocaleTimeString()}</span>
        </div>
      </div>
    </div>
  );
}

function AgentBubble({ text, timestamp }: { text: string; timestamp: string }) {
  return (
    <div className="flex justify-start gap-2">
      <div className="h-6 w-6 rounded-full bg-slate-700 flex items-center justify-center shrink-0 mt-1">
        <Bot className="h-3.5 w-3.5 text-slate-400" />
      </div>
      <div className="max-w-[80%] rounded-lg px-3 py-2 bg-slate-800/60 border border-slate-700">
        <p className="text-sm text-slate-300 whitespace-pre-wrap">{text}</p>
        <span className="text-[10px] text-slate-500 block mt-1">{new Date(timestamp).toLocaleTimeString()}</span>
      </div>
    </div>
  );
}

function ToolGroupBubble({ tools, timestamp }: { tools: { name: string; result?: string }[]; timestamp: string }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="flex justify-start gap-2">
      <div className="h-6 w-6 rounded-full bg-slate-700 flex items-center justify-center shrink-0 mt-1">
        <Wrench className="h-3.5 w-3.5 text-amber-400" />
      </div>
      <div className="max-w-[80%] rounded-lg bg-slate-800/40 border border-slate-700/50">
        <button
          type="button"
          onClick={() => setExpanded((v) => !v)}
          className="w-full flex items-center gap-2 px-3 py-2 text-xs text-slate-400 hover:text-slate-300"
        >
          {expanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          Used {tools.length} tool{tools.length > 1 ? "s" : ""}
          <span className="text-[10px] text-slate-600 ml-auto">{new Date(timestamp).toLocaleTimeString()}</span>
        </button>
        {expanded && (
          <div className="px-3 pb-2 space-y-1.5 border-t border-slate-700/50 pt-2">
            {tools.map((tool, i) => (
              <div key={i} className="text-[11px]">
                <span className="text-amber-400/80 font-mono">{tool.name}</span>
                {tool.result && (
                  <pre className="mt-0.5 text-slate-500 overflow-hidden text-ellipsis whitespace-nowrap max-w-full">
                    {tool.result.slice(0, 200)}
                  </pre>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function StatusPill({ status, phase }: { status: AgentRunStatus; phase?: string }) {
  return (
    <div className="flex justify-center py-1">
      <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full bg-slate-800/50 border border-slate-700/50">
        <StatusBadge status={status} />
        {phase && <span className="text-[11px] text-slate-500">{phase}</span>}
      </div>
    </div>
  );
}

function ErrorBubble({ text }: { text: string }) {
  return (
    <div className="flex justify-start gap-2">
      <div className="h-6 w-6 rounded-full bg-red-950/50 flex items-center justify-center shrink-0 mt-1">
        <AlertTriangle className="h-3.5 w-3.5 text-red-400" />
      </div>
      <div className="max-w-[80%] rounded-lg px-3 py-2 bg-red-950/30 border border-red-900/40">
        <p className="text-xs text-red-300">{text}</p>
      </div>
    </div>
  );
}

function SummaryCard({ summary }: { summary: AgentRunSummary }) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-3">
      <h4 className="text-xs font-medium text-slate-400 mb-2">Summary</h4>
      {summary.description && (
        <p className="text-xs text-slate-300 mb-2">{summary.description}</p>
      )}
      <div className="flex gap-4 text-[11px] text-slate-400">
        <span>Modified: {summary.filesModified}</span>
        <span>Created: {summary.filesCreated}</span>
        <span>Deleted: {summary.filesDeleted}</span>
        {summary.tokensUsed != null && (
          <span>Tokens: {summary.tokensUsed.toLocaleString()}</span>
        )}
      </div>
    </div>
  );
}

function DiffSection({ files, isLoading }: { files: AgentRunDiffFile[]; isLoading: boolean }) {
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

function ActionButtons({
  runId,
  actions,
  approve,
  reject,
}: {
  runId: string;
  actions: AgentRunActions;
  approve: ReturnType<typeof useApproveAgentRun>;
  reject: ReturnType<typeof useRejectAgentRun>;
}) {
  return (
    <div className="flex items-center gap-2 justify-center py-2">
      {actions.canApprove && (
        <>
          <Button
            variant="default"
            size="sm"
            onClick={() => approve.mutate({ runId, request: {} })}
            disabled={approve.isPending}
            className="h-8 text-xs gap-1 bg-emerald-700 hover:bg-emerald-600"
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
            onClick={() => reject.mutate({ runId, request: {} })}
            disabled={reject.isPending}
            className="h-8 text-xs gap-1 text-red-400 border-red-800 hover:bg-red-950/50"
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
