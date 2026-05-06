import { renderMarkdown } from "../../lib/render-markdown";
import { cn } from "../../lib/utils";
import type { ChatAccent, ChatMessageRenderSlot, ChatMessageView } from "./chat-types";

const assistantAccentClasses: Record<ChatAccent, string> = {
  cyan: "border-cyan-500/20 bg-cyan-500/5",
  violet: "border-violet-500/20 bg-violet-500/5",
  slate: "border-white/10 bg-slate-900/70",
};

interface ChatMessageBubbleProps {
  message: ChatMessageView;
  accent?: ChatAccent;
  getMessageMeta?: ChatMessageRenderSlot;
  renderAttachmentPreview?: ChatMessageRenderSlot;
}

export function ChatMessageBubble({
  message,
  accent = "cyan",
  getMessageMeta,
  renderAttachmentPreview,
}: ChatMessageBubbleProps) {
  const isUser = message.role === "user";
  const isSystem = message.role === "system";

  return (
    <article className={cn("flex", isUser ? "justify-end" : "justify-start")} data-role={message.role}>
      <div
        className={cn(
          "max-w-[88%] rounded-lg border px-3 py-2 text-sm text-slate-200",
          isUser && "border-slate-700/70 bg-slate-700/60",
          !isUser && !isSystem && assistantAccentClasses[accent],
          isSystem && "border-amber-500/20 bg-amber-500/10",
        )}
      >
        {getMessageMeta && (
          <div className="mb-1 flex flex-wrap items-center justify-between gap-2 text-[10px] uppercase tracking-wide text-slate-500">
            {getMessageMeta(message)}
          </div>
        )}
        <div className="prose-sm-slate break-words" dangerouslySetInnerHTML={{ __html: renderMarkdown(message.content) }} />
        {renderAttachmentPreview?.(message)}
      </div>
    </article>
  );
}
