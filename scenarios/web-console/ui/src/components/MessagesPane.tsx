import { Play, Square, Volume2 } from "lucide-react";
import { useConversationStore, getSessionConversationEvents } from "../stores/useConversationStore";
import { cn } from "../lib/classnames";

interface MessagesPaneProps {
  sessionId: string;
  /** Speak from this event through all subsequent events. */
  onSpeakFromHere: (eventId: string) => void;
  /** Speak only this event's text. */
  onSpeakOne: (eventId: string, text: string) => void;
  /** Stop any active TTS playback. */
  onTtsStop: () => void;
  /** Event ID currently being spoken, or null. */
  activeSpeakingEventId: string | null;
  /** Whether TTS is active on this pane. */
  isTtsSpeaking: boolean;
}

export default function MessagesPane({
  sessionId,
  onSpeakFromHere,
  onSpeakOne,
  onTtsStop,
  activeSpeakingEventId,
  isTtsSpeaking,
}: MessagesPaneProps) {
  const events = useConversationStore((state) => getSessionConversationEvents(state, sessionId));

  return (
    <div
      data-testid={`messages-pane-${sessionId}`}
      className="h-full overflow-auto bg-wc-surface-base p-4"
    >
      <div className="mx-auto flex max-w-3xl flex-col gap-3">
        {events.length === 0 ? (
          <div className="rounded-xl border border-dashed border-wc-default bg-wc-surface px-4 py-6 text-sm text-wc-text-muted">
            No conversation events yet for this session.
          </div>
        ) : (
          events.map((event) => {
            const isActive = isTtsSpeaking && activeSpeakingEventId === event.id;
            return (
              <article
                key={event.id}
                data-testid={`msg-card-${event.id}`}
                className={cn(
                  "rounded-2xl border border-wc-default bg-wc-surface px-4 py-3 shadow-sm transition-colors",
                  isActive && "border-l-[3px] border-l-wc-accent",
                )}
              >
                <div className="mb-2 flex items-center gap-2 text-[11px] uppercase tracking-[0.12em] text-wc-text-faint">
                  {/* TTS controls */}
                  {isActive ? (
                    <button
                      data-testid={`msg-speak-from-${event.id}`}
                      onClick={() => onTtsStop()}
                      className="rounded p-0.5 text-wc-accent transition hover:bg-wc-accent/10"
                      title="Stop playback"
                    >
                      <Square className="h-3.5 w-3.5 animate-pulse" />
                    </button>
                  ) : (
                    <button
                      data-testid={`msg-speak-from-${event.id}`}
                      onClick={() => onSpeakFromHere(event.id)}
                      className="rounded p-0.5 text-wc-text-muted transition hover:text-wc-text-primary hover:bg-wc-accent/10"
                      title="Read from here"
                    >
                      <Play className="h-3.5 w-3.5" />
                    </button>
                  )}
                  <button
                    data-testid={`msg-speak-one-${event.id}`}
                    onClick={() => onSpeakOne(event.id, event.text)}
                    className="rounded p-0.5 text-wc-text-faint transition hover:text-wc-text-muted hover:bg-wc-accent/10"
                    title="Read this message"
                  >
                    <Volume2 className="h-3 w-3" />
                  </button>
                  <span className="flex-1">{event.source === "claude_hook" ? "Claude Code" : "Codex"}</span>
                  <span>#{event.sequence}</span>
                </div>
                <div className="whitespace-pre-wrap text-sm text-wc-text-primary">{event.text}</div>
              </article>
            );
          })
        )}
      </div>
    </div>
  );
}
