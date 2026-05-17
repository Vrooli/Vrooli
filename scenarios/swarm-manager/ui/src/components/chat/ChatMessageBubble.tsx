import { Volume2, VolumeX } from "lucide-react";
import { renderMarkdown } from "../../lib/render-markdown";
import { cn } from "../../lib/utils";
import type { ChatAccent, ChatMessageRenderSlot, ChatMessageView } from "./chat-types";

const assistantAccentClasses: Record<ChatAccent, string> = {
  cyan: "border-cyan-500/20 bg-cyan-500/5",
  violet: "border-violet-500/20 bg-violet-500/5",
  slate: "border-white/10 bg-slate-900/70",
};

export interface ChatMessageSpeakController {
  speakingMessageId: string | null;
  unavailable: boolean;
  speak: (messageId: string, text: string) => void;
  stop: () => void;
}

interface ChatMessageBubbleProps {
  message: ChatMessageView;
  accent?: ChatAccent;
  getMessageMeta?: ChatMessageRenderSlot;
  renderAttachmentPreview?: ChatMessageRenderSlot;
  speak?: ChatMessageSpeakController;
}

export function ChatMessageBubble({
  message,
  accent = "cyan",
  getMessageMeta,
  renderAttachmentPreview,
  speak,
}: ChatMessageBubbleProps) {
  const isUser = message.role === "user";
  const isSystem = message.role === "system";
  const isAssistant = !isUser && !isSystem;
  const isSpeaking = speak?.speakingMessageId === message.id;

  return (
    <article className={cn("flex", isUser ? "justify-end" : "justify-start")} data-role={message.role}>
      <div
        className={cn(
          "max-w-[88%] rounded-lg border px-3 py-2 text-sm text-slate-200",
          isUser && "border-slate-700/70 bg-slate-700/60",
          isAssistant && assistantAccentClasses[accent],
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
        {isAssistant && speak && !speak.unavailable && message.content.trim() && (
          <button
            type="button"
            onClick={() => (isSpeaking ? speak.stop() : speak.speak(message.id, message.content))}
            className={cn(
              "mt-1 inline-flex items-center gap-1 rounded p-1 text-[11px] transition-colors",
              isSpeaking ? "text-cyan-300 hover:bg-cyan-500/10" : "text-slate-500 hover:bg-slate-700 hover:text-slate-300",
            )}
            title={isSpeaking ? "Stop speaking" : "Speak this message"}
            aria-pressed={isSpeaking}
            data-testid={`chat-bubble-speak-${message.id}`}
          >
            {isSpeaking ? <VolumeX className="h-3.5 w-3.5" /> : <Volume2 className="h-3.5 w-3.5" />}
            <span className="sr-only">{isSpeaking ? "Stop speaking" : "Speak message"}</span>
          </button>
        )}
      </div>
    </article>
  );
}
