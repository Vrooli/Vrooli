import { useEffect, useRef } from "react";
import { Loader2 } from "lucide-react";
import { cn } from "../../lib/utils";
import { useAgentMessageTTS } from "../../hooks/useAgentMessageTTS";
import { useAudioPrefs } from "../../hooks/useAudioPrefs";
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
  /** Forwarded to each bubble; intercepts clicks on linkified entity
   * references for client-side routing. */
  onReferenceNavigate?: (href: string) => void;
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
  onReferenceNavigate,
}: ChatThreadProps) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const tts = useAgentMessageTTS();
  const [audioPrefs] = useAudioPrefs();

  useEffect(() => {
    bottomRef.current?.scrollIntoView?.({ behavior: "smooth" });
  }, [messages.length, isWaiting]);

  // Auto-speak: only fire for assistant messages that arrive AFTER the
  // thread first mounts, and never re-speak the same id on re-render.
  // mountedAt freezes the lower bound; spokenIds.current dedupes across
  // remounts of the same message id.
  const mountedAtRef = useRef(messages.length);
  const spokenIdsRef = useRef<Set<string>>(new Set(messages.slice(0, mountedAtRef.current).map((m) => m.id)));

  useEffect(() => {
    if (!audioPrefs.autoSpeak || tts.unavailable) return;
    for (let i = mountedAtRef.current; i < messages.length; i++) {
      const m = messages[i];
      if (!m || m.role !== "assistant") continue;
      if (spokenIdsRef.current.has(m.id)) continue;
      if (!m.content.trim()) continue;
      spokenIdsRef.current.add(m.id);
      tts.speak(m.id, m.content);
      break; // Speak one new arrival per effect run; older ones queued naturally.
    }
    mountedAtRef.current = messages.length;
  }, [messages, audioPrefs.autoSpeak, tts]);

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
            speak={tts}
            onReferenceNavigate={onReferenceNavigate}
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
