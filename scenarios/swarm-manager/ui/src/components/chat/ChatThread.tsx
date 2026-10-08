import { useCallback, useEffect, useRef, useState } from "react";
import { ArrowDown, Loader2 } from "lucide-react";
import { cn } from "../../lib/utils";
import { useAgentMessageTTS } from "../../hooks/useAgentMessageTTS";
import { useAudioPrefs } from "../../hooks/useAudioPrefs";
import type { ChatAccent, ChatDensity, ChatMessageRenderSlot, ChatMessageView } from "./chat-types";
import { ChatMessageBubble } from "./ChatMessageBubble";

const waitingAccentClasses: Record<ChatAccent, string> = {
  cyan: "border-cyan-500/20 bg-cyan-500/5 text-cyan-400",
  violet: "border-violet-500/20 bg-violet-500/5 text-violet-400",
  slate: "border-white/10 bg-slate-900/70 text-slate-400",
};

/**
 * How close to the bottom still counts as "following the conversation". Wide
 * enough to survive a reflow (an image loading, a diagram rendering) without
 * treating it as the reader having scrolled away.
 */
const PINNED_THRESHOLD_PX = 72;

interface ChatThreadProps {
  messages: ChatMessageView[];
  isWaiting?: boolean;
  emptyLabel?: string;
  accent?: ChatAccent;
  density?: ChatDensity;
  getMessageMeta?: ChatMessageRenderSlot;
  renderAttachmentPreview?: ChatMessageRenderSlot;
  /** Per-message controls (retry, edit) rendered under the message body. */
  renderMessageActions?: ChatMessageRenderSlot;
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
  density = "comfortable",
  getMessageMeta,
  renderAttachmentPreview,
  renderMessageActions,
  className,
  testId,
  onReferenceNavigate,
}: ChatThreadProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const tts = useAgentMessageTTS();
  const [audioPrefs] = useAudioPrefs();
  // Whether the reader is following the tail. Starts true so a freshly opened
  // session lands on the newest turn.
  const pinnedRef = useRef(true);
  const [showJumpToLatest, setShowJumpToLatest] = useState(false);

  const scrollToLatest = useCallback(() => {
    const element = scrollRef.current;
    if (!element) return;
    element.scrollTop = element.scrollHeight;
    pinnedRef.current = true;
    setShowJumpToLatest(false);
  }, []);

  const handleScroll = useCallback(() => {
    const element = scrollRef.current;
    if (!element) return;
    const distanceFromBottom = element.scrollHeight - element.scrollTop - element.clientHeight;
    pinnedRef.current = distanceFromBottom <= PINNED_THRESHOLD_PX;
    if (pinnedRef.current) setShowJumpToLatest(false);
  }, []);

  // Scroll the pane, never the page. scrollIntoView on a bottom sentinel walks
  // every scrollable ancestor up to the document, so on any layout where the
  // thread is not height-bounded it drags the whole page down with it. Setting
  // scrollTop on the pane cannot escape the pane.
  //
  // Only follow when the reader is already at the tail: yanking someone out of
  // scrollback to show a new arrival loses their place mid-read.
  useEffect(() => {
    const element = scrollRef.current;
    if (!element) return;
    if (pinnedRef.current) {
      element.scrollTop = element.scrollHeight;
      return;
    }
    if (messages.length > 0) setShowJumpToLatest(true);
  }, [messages.length, isWaiting]);

  // Auto-speak: only fire for assistant messages that arrive AFTER the
  // thread first mounts, and never re-speak the same id on re-render.
  // mountedAt freezes the lower bound; spokenIds.current dedupes across
  // remounts of the same message id.
  const mountedAtRef = useRef(messages.length);
  const spokenIdsRef = useRef<Set<string>>(new Set(messages.slice(0, mountedAtRef.current).map((m) => m.id)));

  // Depends on the two fields it actually reads rather than the whole
  // controller. `tts` used to be a new object every render, so this effect ran
  // on every render of the thread — including each 3s session poll — instead of
  // only when a message arrived.
  const { speak: ttsSpeak, unavailable: ttsUnavailable } = tts;
  useEffect(() => {
    if (!audioPrefs.autoSpeak || ttsUnavailable) return;
    for (let i = mountedAtRef.current; i < messages.length; i++) {
      const m = messages[i];
      if (!m || m.role !== "assistant") continue;
      if (spokenIdsRef.current.has(m.id)) continue;
      if (!m.content.trim()) continue;
      spokenIdsRef.current.add(m.id);
      ttsSpeak(m.id, m.content);
      break; // Speak one new arrival per effect run; older ones queued naturally.
    }
    mountedAtRef.current = messages.length;
  }, [messages, audioPrefs.autoSpeak, ttsSpeak, ttsUnavailable]);

  const isCompact = density === "compact";

  return (
    <div className="relative flex min-h-0 flex-1 flex-col">
      <div
        ref={scrollRef}
        onScroll={handleScroll}
        className={cn(
          "min-h-0 flex-1 overflow-y-auto overscroll-contain",
          isCompact ? "flex flex-col" : "space-y-3",
          className,
        )}
        data-testid={testId}
        data-density={density}
      >
        {messages.length > 0 ? (
          messages.map((message) => (
            <div
              key={message.id}
              className={cn(!isCompact && "mx-auto w-full max-w-5xl")}
            >
              <ChatMessageBubble
                message={message}
                accent={accent}
                density={density}
                getMessageMeta={getMessageMeta}
                renderAttachmentPreview={renderAttachmentPreview}
                renderMessageActions={renderMessageActions}
                speak={tts}
                onReferenceNavigate={onReferenceNavigate}
              />
            </div>
          ))
        ) : (
          <div className="flex h-full min-h-32 items-center justify-center text-sm text-slate-500">{emptyLabel}</div>
        )}

        {isWaiting && (
          <div className={cn("flex justify-start", isCompact ? "px-3 py-3" : "mx-auto w-full max-w-5xl")}>
            <div className={cn("flex items-center gap-2 rounded-lg border px-3 py-2 text-sm text-slate-400", waitingAccentClasses[accent])}>
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Thinking...
            </div>
          </div>
        )}
      </div>

      {showJumpToLatest && (
        <button
          type="button"
          onClick={scrollToLatest}
          className="absolute bottom-3 left-1/2 inline-flex -translate-x-1/2 items-center gap-1.5 rounded-full border border-cyan-400/40 bg-slate-900/95 px-3 py-1.5 text-xs font-medium text-cyan-200 shadow-lg backdrop-blur transition-colors hover:border-cyan-300/60 hover:text-cyan-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/50"
          data-testid="chat-jump-to-latest"
        >
          <ArrowDown className="h-3.5 w-3.5" />
          Jump to latest
        </button>
      )}
    </div>
  );
}
