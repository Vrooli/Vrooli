import { useMemo } from "react";
import { ChatComposer } from "../chat/ChatComposer";
import { ChatThread } from "../chat/ChatThread";
import { formatRelativeTime } from "../../lib/format-utils";
import { cn } from "../../lib/utils";
import type { AgentSessionMessage } from "../../types";
import type { ChatMessageView } from "../chat/chat-types";

interface SessionConversationProps {
  messages: AgentSessionMessage[];
  draft: string;
  onDraftChange: (value: string) => void;
  onSend: () => void;
  isMutating: boolean;
  isWaitingForAgent: boolean;
  variant?: "desktop" | "mobile";
}

export function SessionConversation({
  messages,
  draft,
  onDraftChange,
  onSend,
  isMutating,
  isWaitingForAgent,
  variant = "desktop",
}: SessionConversationProps) {
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
        isWaiting={isWaitingForAgent}
        emptyLabel="No messages recorded yet."
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
          placeholder="Continue this session..."
          submitLabel="Send"
          testId="agent-session-composer"
        />
      </div>
    </section>
  );
}
