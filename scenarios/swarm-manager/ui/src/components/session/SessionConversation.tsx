import { useMemo } from "react";
import { ChatComposer } from "../chat/ChatComposer";
import { ChatThread } from "../chat/ChatThread";
import { formatRelativeTime } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import type { AgentSessionKind, AgentSessionMessage, AgentSessionStatus } from "../../types";
import type { ChatMessageView } from "../chat/chat-types";

interface SessionConversationProps {
  messages: AgentSessionMessage[];
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
  isMutating: boolean;
  isWaitingForAgent: boolean;
  sessionKind: AgentSessionKind;
  sessionStatus: AgentSessionStatus;
  variant?: "desktop" | "mobile";
}

export function SessionConversation({
  messages,
  draft,
  onDraftChange,
  onSend,
  isMutating,
  isWaitingForAgent,
  sessionKind,
  sessionStatus,
  variant = "desktop",
}: SessionConversationProps) {
  const isDraft = sessionStatus === "draft";
  const placeholder = isDraft
    ? draftPlaceholderForKind(sessionKind)
    : "Continue this session...";

  const sortedMessages = useMemo(
    () => [...messages].sort((a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime()),
    [messages],
  );

  const chatMessages: ChatMessageView[] = useMemo(
    () =>
      sortedMessages.map((message) => ({
        id: message.id,
        role: message.role,
        content: message.content,
        createdAt: message.createdAt,
        attachmentIds: message.attachmentIds,
      })),
    [sortedMessages],
  );

  return (
    <section
      className={cn(
        "flex min-h-0 flex-1 flex-col",
        variant === "desktop" && "rounded-lg border border-white/10 bg-slate-950/30",
      )}
      data-testid="agent-session-conversation"
    >
      {variant === "desktop" && <div className="border-b border-white/10 px-3 py-2 text-xs font-medium text-slate-300">Conversation</div>}
      <ChatThread
        messages={chatMessages}
        isWaiting={!isDraft && isWaitingForAgent}
        emptyLabel={isDraft ? "Start with the real context you want the agent to use." : "No messages recorded yet."}
        accent="cyan"
        className={cn("p-3", variant === "mobile" && "px-1 pb-32")}
        testId="agent-session-messages"
        getMessageMeta={(message) => (
          <>
            <span>{message.role}</span>
            {message.createdAt && <span>{formatRelativeTime(message.createdAt)}</span>}
          </>
        )}
      />
      <div
        className={cn(
          "border-t border-white/10 p-3",
          variant === "mobile" && "fixed inset-x-0 bottom-0 z-40 bg-slate-950/95 px-2 pb-[calc(0.5rem+env(safe-area-inset-bottom))] pt-2 backdrop-blur",
        )}
      >
        <ChatComposer
          value={draft}
          onChange={onDraftChange}
          onSubmit={onSend}
          disabled={isMutating}
          isSubmitting={isMutating}
          placeholder={placeholder}
          submitLabel="Send"
          testId="agent-session-composer"
        />
      </div>
    </section>
  );
}

function draftPlaceholderForKind(kind: AgentSessionKind): string {
  switch (kind) {
    case "operating_mode_authoring":
      return "Describe the recurring agent workflow you want to author...";
    case "swarm_operations":
      return "Ask what to review, unblock, decide, or move forward in Swarm Manager...";
    case "meta_orchestration":
    default:
      return "Describe what you want to plan...";
  }
}
