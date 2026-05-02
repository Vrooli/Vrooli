import { useState, useEffect, useCallback, useRef, useMemo } from "react";
import {
  Loader2,
  Square,
  ChevronDown,
  ChevronRight,
  Send,
  Bot,
  Clock,
  ExternalLink,
  Image,
  ArrowDown,
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
  useScenarioEnvelope,
} from "../lib/hooks";
import { buildScenarioEnvelope, composePrompt, resolveScreenshotPaths } from "../lib/agentContext";
import { ContextPickerPopover } from "./ContextPickerPopover";
import { ContextPreviewPopover } from "./ContextPreviewPopover";
import { AttachmentPreview } from "./AttachmentPreview";
import { useAttachments } from "../hooks/useAttachments";
import {
  StatusBadge,
  formatDuration,
  UserBubble,
  AgentBubble,
  ToolGroupBubble,
  ErrorBubble,
  SummaryCard,
  DiffSection,
  ActionButtons,
} from "./AgentTabBubbles";
import {
  RUN_STATUS,
  ACTIVE_STATUSES,
  uploadAgentAttachment,
  type AgentContextItem,
  type AgentRunEvent,
  type RepoFileStats,
} from "../lib/api";
import { fetchExternalUrl } from "../lib/api-internals";
import { buildChatMessages } from "./AgentTabTypes";

const AGENT_PROFILE_KEY = "gct.agent.defaultProfileId";

// ── Props ───────────────────────────────────────────────────────────

interface AgentTabProps {
  scenarioSlug: string;
  repoId?: string | null;
  agentManagerAvailable: boolean;
  workspaceSandboxAvailable: boolean;
  contextItems: AgentContextItem[];
  onAddContext: (item: AgentContextItem) => void;
  onRemoveContext: (id: string) => void;
  onClearContext: () => void;
  testGenieAvailable: boolean;
  tidinessAvailable: boolean;
  auditorAvailable: boolean;
  visualCaptureAvailable: boolean;
  fileStats?: RepoFileStats;
  activeRunId?: string | null;
  onActiveRunIdChange?: (id: string | null) => void;
}

// ── Main component ──────────────────────────────────────────────────

export function AgentTab({
  scenarioSlug,
  repoId,
  agentManagerAvailable,
  workspaceSandboxAvailable,
  contextItems,
  onAddContext,
  onRemoveContext,
  onClearContext,
  testGenieAvailable,
  tidinessAvailable,
  auditorAvailable,
  visualCaptureAvailable,
  fileStats,
  activeRunId: controlledRunId,
  onActiveRunIdChange,
}: AgentTabProps) {
  const [message, setMessage] = useState("");
  const [selectedProfileId, setSelectedProfileId] = useState<string>(() => {
    try { return localStorage.getItem(AGENT_PROFILE_KEY) ?? ""; } catch { return ""; }
  });
  const [internalRunId, setInternalRunId] = useState<string | null>(null);
  const activeRunId = controlledRunId !== undefined ? controlledRunId : internalRunId;
  const setActiveRunId = useCallback((id: string | null) => {
    if (onActiveRunIdChange) {
      onActiveRunIdChange(id);
    } else {
      setInternalRunId(id);
    }
  }, [onActiveRunIdChange]);
  const [events, setEvents] = useState<AgentRunEvent[]>([]);
  const [lastEventSequence, setLastEventSequence] = useState(-1);
  const [showRunHistory, setShowRunHistory] = useState(false);

  // Scroll tracking
  const chatEndRef = useRef<HTMLDivElement>(null);
  const chatContainerRef = useRef<HTMLDivElement>(null);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [hasNewMessages, setHasNewMessages] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Resolve agent-manager base URL (same pattern as ScenarioReviewPanel)
  const [agentManagerBaseUrl, setAgentManagerBaseUrl] = useState(`/embedded/agent-manager/`);
  useEffect(() => {
    if (!agentManagerAvailable) return;
    let cancelled = false;
    fetchExternalUrl(`/embedded/agent-manager/external-url`)
      .then((url) => {
        if (!cancelled && url) {
          setAgentManagerBaseUrl(url);
        }
      })
      .catch(() => { /* keep fallback */ });
    return () => { cancelled = true; };
  }, [agentManagerAvailable]);

  // Resolve workspace-sandbox base URL for "Review in Sandbox" deep-links
  const [workspaceSandboxBaseUrl, setWorkspaceSandboxBaseUrl] = useState(`/embedded/workspace-sandbox/`);
  useEffect(() => {
    if (!workspaceSandboxAvailable) return;
    let cancelled = false;
    fetchExternalUrl(`/embedded/workspace-sandbox/external-url`)
      .then((url) => {
        if (!cancelled && url) {
          setWorkspaceSandboxBaseUrl(url);
        }
      })
      .catch(() => { /* keep fallback */ });
    return () => { cancelled = true; };
  }, [workspaceSandboxAvailable]);

  // Attachment upload
  const fileInputRef = useRef<HTMLInputElement>(null);
  const uploadAttachment = useCallback(
    async (file: File) => uploadAgentAttachment(file, repoId ?? undefined),
    [repoId]
  );
  const { attachments, addAttachment, removeAttachment, clearAttachments, isUploading, getUploadedIds } = useAttachments(uploadAttachment);
  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) addAttachment(file);
    e.target.value = "";
  }, [addAttachment]);

  const profiles = useAgentProfiles(agentManagerAvailable);
  const envelopeQuery = useScenarioEnvelope(scenarioSlug, agentManagerAvailable);
  const runs = useAgentRuns(scenarioSlug, agentManagerAvailable, repoId);
  const activeRun = useAgentRun(activeRunId, agentManagerAvailable, repoId);

  const isActiveRun = activeRun.data && ([...ACTIVE_STATUSES, RUN_STATUS.NEEDS_REVIEW] as string[]).includes(activeRun.data.status);
  const shouldPollDiff = activeRun.data?.status === RUN_STATUS.NEEDS_REVIEW && agentManagerAvailable;
  const runEvents = useAgentRunEvents(activeRunId, lastEventSequence, agentManagerAvailable && !!activeRunId, repoId, activeRun.data?.status);
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
        ([...ACTIVE_STATUSES, RUN_STATUS.NEEDS_REVIEW] as string[]).includes(r.status)
      );
      if (active) setActiveRunId(active.id);
    }
  }, [activeRunId, runs.data, setActiveRunId]);

  // Reset events when switching runs (handles all paths: auto-detect, history, parent change)
  const prevRunIdRef = useRef<string | null | undefined>(activeRunId);
  useEffect(() => {
    if (activeRunId !== prevRunIdRef.current) {
      prevRunIdRef.current = activeRunId;
      setEvents([]);
      setLastEventSequence(-1);
    }
  }, [activeRunId]);

  // Accumulate events
  useEffect(() => {
    if (runEvents.data?.events?.length) {
      const incoming = runEvents.data.events;
      // Guard: ignore events from a different run
      if (incoming[0]?.runId && incoming[0].runId !== activeRunId) return;
      setEvents((prev) => {
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
  }, [runEvents.data, activeRunId]);

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

  const handleSend = useCallback(async () => {
    const resolvedItems = await resolveScreenshotPaths(contextItems, scenarioSlug, repoId ?? undefined);

    // Prepend the scenario envelope on the first message of a conversation
    // so the agent has orientation (name, path, lifecycle commands, etc.).
    const isFirstMessage = events.length === 0;
    const envelope = isFirstMessage && envelopeQuery.data
      ? buildScenarioEnvelope(envelopeQuery.data)
      : undefined;

    const prompt = composePrompt(message, resolvedItems, envelope);
    if (!prompt.trim()) return;

    const profileKey = selectedProfileId ? undefined : "git-control-tower-reviewer";
    const uploadedIds = getUploadedIds();

    createRun.mutate(
      {
        scenarioSlug,
        prompt,
        profileId: selectedProfileId || undefined,
        profileKey,
        ...(uploadedIds.length > 0 ? { attachmentIds: uploadedIds } : {}),
      },
      {
        onSuccess: (resp) => {
          setActiveRunId(resp.runId);
          setEvents([]);
          setLastEventSequence(-1);
          setMessage("");
          onClearContext();
          clearAttachments();
        },
      }
    );
  }, [message, contextItems, scenarioSlug, repoId, selectedProfileId, events.length, envelopeQuery.data, createRun, onClearContext, setActiveRunId, getUploadedIds, clearAttachments]);

  const handleContinue = useCallback(() => {
    if (!activeRunId || !message.trim()) return;
    const followUp = message.trim();
    const uploadedIds = getUploadedIds();
    continueRun.mutate(
      { runId: activeRunId, request: { message: followUp, ...(uploadedIds.length > 0 ? { attachment_ids: uploadedIds } : {}) } },
      {
        onSuccess: () => {
          setMessage("");
          clearAttachments();
        },
      }
    );
  }, [activeRunId, message, continueRun, getUploadedIds, clearAttachments]);

  const handleSetDefaultProfile = useCallback((id: string) => {
    setSelectedProfileId(id);
    try { localStorage.setItem(AGENT_PROFILE_KEY, id); } catch { /* ignore */ }
  }, []);

  // Look up promptPreview from runs list (Get() single-run endpoint doesn't populate it)
  const promptPreview = useMemo(() => {
    if (activeRunId && runs.data?.runs) {
      return runs.data.runs.find((r) => r.id === activeRunId)?.promptPreview;
    }
    return undefined;
  }, [activeRunId, runs.data?.runs]);

  const chatMessages = useMemo(
    () =>
      buildChatMessages(
        events,
        activeRun.data ?? null,
        runDiff.data?.files ?? null,
        activeRunId,
        promptPreview,
      ),
    [events, activeRun.data, runDiff.data, activeRunId, promptPreview]
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

  const isRunning = activeRun.data && (ACTIVE_STATUSES as readonly string[]).includes(activeRun.data.status);
  const canContinue = activeRun.data?.actions?.canContinue ?? false;
  const isCreating = createRun.isPending;
  const isContinuing = continueRun.isPending;

  // Determine input behavior
  const canSendNew = !activeRunId || (!isActiveRun && activeRun.data?.status !== RUN_STATUS.NEEDS_REVIEW);
  const showInputBar = !activeRunId || !!isActiveRun || canContinue;
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

  const duration = activeRun.data ? formatDuration(activeRun.data.startedAt, activeRun.data.endedAt) : null;

  return (
    <div className="flex flex-col h-full">
      {/* Header bar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-slate-800 shrink-0 gap-2 flex-wrap min-w-0">
        <div className="flex items-center gap-2 min-w-0">
          {activeRun.data && <StatusBadge status={activeRun.data.status} />}
          {activeRun.data?.phase && isActiveRun && (
            <span className="text-[11px] text-slate-500">{activeRun.data.phase}</span>
          )}
          {duration && (
            <span className="text-[11px] text-slate-500">{duration}</span>
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
                <div className="absolute top-full mt-1 left-0 z-50 w-80 max-h-52 overflow-y-auto rounded-lg border border-slate-700 bg-slate-900 shadow-xl">
                  {runs.data.runs.map((run) => (
                    <button
                      key={run.id}
                      type="button"
                      onClick={() => {
                        setActiveRunId(run.id);
                        setEvents([]);
                        setLastEventSequence(-1);
                        setShowRunHistory(false);
                      }}
                      className={`w-full flex items-center gap-2 px-3 py-2 text-xs hover:bg-slate-800/60 ${
                        run.id === activeRunId ? "bg-slate-800" : ""
                      }`}
                    >
                      <StatusBadge status={run.status} />
                      <span className="text-slate-400 text-[11px] truncate min-w-0 flex-1">
                        {run.promptPreview || run.id.slice(0, 8)}
                      </span>
                      <span className="text-[11px] text-slate-600 shrink-0">
                        {new Date(run.createdAt).toLocaleTimeString()}
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {activeRunId && (
            <a
              href={`${agentManagerBaseUrl.replace(/\/$/, "")}/runs/${activeRunId}`}
              target="_blank"
              rel="noopener"
              className="inline-flex items-center gap-1 text-[11px] px-2 py-1 rounded border transition-colors text-blue-400 border-blue-800 hover:bg-blue-950/50"
              aria-label="Open run in Agent Manager"
            >
              <ExternalLink className="h-3 w-3" />
              Open
            </a>
          )}
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
          {!isActiveRun && activeRunId && activeRun.data?.status !== RUN_STATUS.NEEDS_REVIEW && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setActiveRunId(null);
                setEvents([]);
                setLastEventSequence(-1);
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
        {chatMessages.length === 0 && !runEvents.isLoading && (
          <div className="flex flex-col items-center justify-center h-full text-slate-500">
            <Bot className="h-8 w-8 mb-3 opacity-30" />
            <p className="text-xs text-center">
              {activeRunId ? "No messages available for this run" : "Send a message to start an agent run"}
            </p>
            {runEvents.error && (
              <p className="text-[11px] text-red-400/70 mt-2 text-center max-w-xs">
                Failed to load events: {runEvents.error.message}
              </p>
            )}
          </div>
        )}
        {chatMessages.length === 0 && runEvents.isLoading && activeRunId && (
          <div className="flex flex-col items-center justify-center h-full text-slate-500">
            <Loader2 className="h-6 w-6 animate-spin mb-3 opacity-50" />
            <p className="text-xs">Loading messages...</p>
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
              return null;
            case "error":
              return <ErrorBubble key={idx} text={msg.text} />;
            case "summary":
              return (
                <SummaryCard
                  key={idx}
                  summary={msg.summary}
                  run={msg.run}
                  sandboxReviewUrl={
                    msg.run?.sandboxId && workspaceSandboxAvailable && msg.run.approvalState === "pending"
                      ? `${workspaceSandboxBaseUrl.replace(/\/$/, "")}?sandbox=${msg.run.sandboxId}&review=true`
                      : undefined
                  }
                />
              );
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
      {contextItems.length > 0 && showInputBar && (
        <div className="px-4 pb-1 flex flex-wrap gap-1 border-t border-slate-800/50 pt-2">
          {contextItems.map((item) => (
            <ContextPreviewPopover key={item.id} item={item} scenarioSlug={scenarioSlug} onRemove={onRemoveContext} />
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

      {/* Attachment previews above input */}
      {attachments.length > 0 && showInputBar && (
        <div className="border-t border-slate-800/50">
          <AttachmentPreview attachments={attachments} onRemove={removeAttachment} />
        </div>
      )}

      {/* Input bar */}
      {showInputBar ? (
        <div className="shrink-0 border-t border-slate-800">
          {/* Profile selector row */}
          {canSendNew && !canContinue && profiles.data?.profiles && profiles.data.profiles.length > 0 && (
            <div className="px-4 pt-2 flex items-center gap-2">
              <span className="text-[11px] text-slate-500">Profile:</span>
              <select
                value={selectedProfileId}
                onChange={(e) => handleSetDefaultProfile(e.target.value)}
                className="h-7 px-2 bg-slate-800 border border-slate-700 rounded text-xs text-slate-300 focus:outline-none focus:border-blue-500"
              >
                <option value="">Default</option>
                {profiles.data.profiles.map((p) => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>
          )}
          {/* Hidden file input for image uploads */}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            className="hidden"
            onChange={handleFileSelect}
          />
          {/* Message input row */}
          <div className="px-4 py-3 flex items-end gap-2">
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="h-8 w-8 flex items-center justify-center rounded border border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600 transition-colors"
              title="Attach image"
            >
              <Image className="h-4 w-4" />
            </button>
            <ContextPickerPopover
              scenarioSlug={scenarioSlug}
              repoId={repoId}
              testGenieAvailable={testGenieAvailable}
              tidinessAvailable={tidinessAvailable}
              auditorAvailable={auditorAvailable}
              visualCaptureAvailable={visualCaptureAvailable}
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
            <Button
              variant="default"
              size="sm"
              onClick={handleInputSend}
              aria-label={canContinue ? "Send follow-up" : "Send message"}
              disabled={inputDisabled || isUploading || (!message.trim() && contextItems.length === 0 && attachments.length === 0)}
              className="h-9 w-9 p-0 shrink-0"
            >
              {isCreating || isContinuing ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
      ) : (
        <div className="shrink-0 border-t border-slate-800 px-4 py-2 flex justify-center">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setActiveRunId(null);
              setEvents([]);
              setLastEventSequence(-1);
            }}
            className="h-8 text-xs gap-1"
          >
            Start New Run
          </Button>
        </div>
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
