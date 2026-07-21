import { useMemo } from "react";
import { Loader2, Volume2, VolumeX } from "lucide-react";
import { detailPathFromNodeId } from "../../app/routes/route-paths";
import { MarkdownRenderer, type InlineTokenResolution } from "../markdown/MarkdownRenderer";
import { cn } from "../../lib/utils";
import type { ChatAccent, ChatMessageRenderSlot, ChatMessageView } from "./chat-types";

// REFERENCE_MARKERS maps the stored context-item type to the marker the agent
// emits in `type:name` spans, so a resolved Context entry can be matched back
// to the exact inline-code token in the message text. Mirrors the server's
// markerContextType (api/internal/sessioncontext/bulk.go).
const REFERENCE_MARKERS: Record<string, string> = {
  initiative: "initiative",
  backlog_item: "backlog",
  execution: "execution",
  capture: "capture",
  session: "session",
  scenario: "scenario",
};

interface InlineReferenceLink extends InlineTokenResolution {
  token: string;
}

/** Build the navigable-reference list from a message's resolved context. Each
 * entry pairs the `type:name` token with its detail path; only entries whose
 * node id resolves to a route are included. */
function referencesFromContext(context: ChatMessageView["context"]): InlineReferenceLink[] {
  if (!context || context.length === 0) return [];
  const out: InlineReferenceLink[] = [];
  for (const item of context) {
    const marker = REFERENCE_MARKERS[item.type];
    if (!marker || !item.ref) continue;
    const nodeId = item.nodeId;
    const href = nodeId?.startsWith("/") ? nodeId : nodeId ? detailPathFromNodeId(nodeId) : null;
    if (href) out.push({ token: `${marker}:${item.ref}`, href });
  }
  return out;
}

const assistantAccentClasses: Record<ChatAccent, string> = {
  cyan: "border-cyan-500/20 bg-cyan-500/5",
  violet: "border-violet-500/20 bg-violet-500/5",
  slate: "border-white/10 bg-slate-900/70",
};

export interface ChatMessageSpeakController {
  speakingMessageId: string | null;
  loadingMessageId: string | null;
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
  /** Called when a linkified entity reference is clicked. When provided, the
   * click is intercepted for client-side routing; when absent, the anchor's
   * href performs a normal navigation. */
  onReferenceNavigate?: (href: string) => void;
}

export function ChatMessageBubble({
  message,
  accent = "cyan",
  getMessageMeta,
  renderAttachmentPreview,
  speak,
  onReferenceNavigate,
}: ChatMessageBubbleProps) {
  const isUser = message.role === "user";
  const isSystem = message.role === "system";
  const isAssistant = !isUser && !isSystem;
  const isSpeaking = speak?.speakingMessageId === message.id;
  const isLoadingAudio = speak?.loadingMessageId === message.id;

  const references = useMemo(() => referencesFromContext(message.context), [message.context]);

  const resolveInlineToken = (text: string): InlineTokenResolution | null => {
    const reference = references.find((candidate) => candidate.token === text);
    return reference ? { ...reference, kind: "entity" } : null;
  };

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
        <MarkdownRenderer
          content={message.content}
          className="prose-sm-slate break-words"
          resolveInlineToken={resolveInlineToken}
          onLinkClick={(href, event) => {
            if (!references.some((reference) => reference.href === href) || !onReferenceNavigate) return;
            event.preventDefault();
            onReferenceNavigate(href);
          }}
        />
        {renderAttachmentPreview?.(message)}
        {isAssistant && speak && !speak.unavailable && message.content.trim() && (
          <button
            type="button"
            onClick={() => (isSpeaking ? speak.stop() : speak.speak(message.id, message.content))}
            disabled={isLoadingAudio}
            className={cn(
              "mt-1 inline-flex items-center gap-1 rounded p-1 text-[11px] transition-colors",
              isLoadingAudio && "cursor-wait text-cyan-300",
              isSpeaking ? "text-cyan-300 hover:bg-cyan-500/10" : "text-slate-500 hover:bg-slate-700 hover:text-slate-300",
            )}
            title={isLoadingAudio ? "Preparing audio" : isSpeaking ? "Stop speaking" : "Speak this message"}
            aria-pressed={isSpeaking}
            aria-busy={isLoadingAudio}
            data-loading={isLoadingAudio ? "true" : "false"}
            data-testid={`chat-bubble-speak-${message.id}`}
          >
            {isLoadingAudio
              ? <Loader2 data-testid={`chat-bubble-audio-loading-${message.id}`} className="h-3.5 w-3.5 animate-spin" />
              : isSpeaking ? <VolumeX className="h-3.5 w-3.5" /> : <Volume2 className="h-3.5 w-3.5" />}
            <span className="sr-only">{isLoadingAudio ? "Preparing audio" : isSpeaking ? "Stop speaking" : "Speak message"}</span>
          </button>
        )}
      </div>
    </article>
  );
}
