import { useRef, useCallback, useEffect, useState } from "react";
import {
  Loader2,
  Bot,
  ArrowDown,
} from "lucide-react";
import {
  UserBubble,
  AgentBubble,
  ToolGroupBubble,
  ErrorBubble,
  SummaryCard,
  DiffSection,
  ActionButtons,
} from "./AgentTabBubbles";
import type { ChatMessage } from "./AgentTabTypes";
import type { AgentRun } from "../lib/api";
import type { useApproveAgentRun, useRejectAgentRun } from "../lib/hooks";

interface AgentTabChatThreadProps {
  chatMessages: ChatMessage[];
  activeRunId: string | null | undefined;
  activeRun: AgentRun | null;
  isRunning: boolean;
  runEventsLoading: boolean;
  runEventsError?: Error | null;
  runDiffLoading: boolean;
  workspaceSandboxBaseUrl: string;
  workspaceSandboxAvailable: boolean;
  approve: ReturnType<typeof useApproveAgentRun>;
  reject: ReturnType<typeof useRejectAgentRun>;
}

export function AgentTabChatThread({
  chatMessages,
  activeRunId,
  activeRun,
  isRunning,
  runEventsLoading,
  runEventsError,
  runDiffLoading,
  workspaceSandboxBaseUrl,
  workspaceSandboxAvailable,
  approve,
  reject,
}: AgentTabChatThreadProps) {
  const chatEndRef = useRef<HTMLDivElement>(null);
  const chatContainerRef = useRef<HTMLDivElement>(null);
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [hasNewMessages, setHasNewMessages] = useState(false);

  const handleScroll = useCallback(() => {
    const el = chatContainerRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
    setIsNearBottom(nearBottom);
    if (nearBottom) setHasNewMessages(false);
  }, []);

  useEffect(() => {
    if (isNearBottom) {
      chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
    } else {
      setHasNewMessages(true);
    }
  }, [chatMessages.length, activeRun?.status, isNearBottom]);

  const scrollToBottom = useCallback(() => {
    chatEndRef.current?.scrollIntoView({ behavior: "smooth" });
    setHasNewMessages(false);
  }, []);

  return (
    <>
      <div
        ref={chatContainerRef}
        onScroll={handleScroll}
        className="flex-1 overflow-y-auto px-4 py-4 space-y-3"
      >
        {chatMessages.length === 0 && !runEventsLoading && (
          <div className="flex flex-col items-center justify-center h-full text-slate-500">
            <Bot className="h-8 w-8 mb-3 opacity-30" />
            <p className="text-xs text-center">
              {activeRunId ? "No messages available for this run" : "Send a message to start an agent run"}
            </p>
            {runEventsError && (
              <p className="text-[11px] text-red-400/70 mt-2 text-center max-w-xs">
                Failed to load events: {runEventsError.message}
              </p>
            )}
          </div>
        )}
        {chatMessages.length === 0 && runEventsLoading && activeRunId && (
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
              return <DiffSection key={idx} files={msg.files} isLoading={runDiffLoading} />;
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
        {activeRun?.progressPercent != null && activeRun.progressPercent > 0 && isRunning && (
          <div className="h-1.5 bg-slate-800 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all"
              style={{ width: `${activeRun.progressPercent}%` }}
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
    </>
  );
}
