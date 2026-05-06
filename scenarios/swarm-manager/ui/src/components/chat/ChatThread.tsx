import { useEffect, useRef } from "react";
import { Loader2 } from "lucide-react";
import { cn } from "../../lib/utils";
import type { ChatAccent, ChatMessageRenderSlot, ChatMessageView } from "./chat-types";
import { ChatMessageBubble } from "./ChatMessageBubble";

const waitingAccentClasses: Record<ChatAccent, string> = {
  cyan: "border-cyan-500/20 bg-cyan-500/5 text-cyan-400",
  violet: "border-violet-500/20 bg-violet-500/5 text-violet-400",
  slate: "border-white/10 bg-slate-900/70 text-slate-400",
};

interface ChatThreadProps {
  messages: ChatMessageView[];
  isWaiting?: boolean;
  emptyLabel?: string;
  accent?: ChatAccent;
  getMessageMeta?: ChatMessageRenderSlot;
  renderAttachmentPreview?: ChatMessageRenderSlot;
  className?: string;
  testId?: string;
}

export function ChatThread({
  messages,
  isWaiting = false,
  emptyLabel = "No messages yet.",
  accent = "cyan",
  getMessageMeta,
  renderAttachmentPreview,
  className,
  testId,
}: ChatThreadProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: "smooth" });
  }, [messages.length, isWaiting]);

  return (
    <div className={cn("min-h-0 flex-1 space-y-3 overflow-y-auto", className)} data-testid={testId}>
      {messages.length > 0 ? (
        messages.map((message) => (
          <ChatMessageBubble
            key={message.id}
            message={message}
            accent={accent}
            getMessageMeta={getMessageMeta}
            renderAttachmentPreview={renderAttachmentPreview}
          />
        ))
      ) : (
        <div className="flex h-full min-h-32 items-center justify-center text-sm text-slate-500">{emptyLabel}</div>
      )}

      {isWaiting && (
        <div className="flex justify-start">
          <div className={cn("flex items-center gap-2 rounded-lg border px-3 py-2 text-sm text-slate-400", waitingAccentClasses[accent])}>
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Thinking...
          </div>
        </div>
      )}

      <div ref={bottomRef} />
    </div>
  );
}
