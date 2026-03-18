import { useConversationStore, getSessionConversationEvents } from "../stores/useConversationStore";

interface MessagesPaneProps {
  sessionId: string;
}

export default function MessagesPane({ sessionId }: MessagesPaneProps) {
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
          events.map((event) => (
            <article
              key={event.id}
              className="rounded-2xl border border-wc-default bg-wc-surface px-4 py-3 shadow-sm"
            >
              <div className="mb-2 flex items-center justify-between gap-3 text-[11px] uppercase tracking-[0.12em] text-wc-text-faint">
                <span>{event.source === "claude_hook" ? "Claude Code" : "Codex"}</span>
                <span>#{event.sequence}</span>
              </div>
              <div className="whitespace-pre-wrap text-sm text-wc-text-primary">{event.text}</div>
            </article>
          ))
        )}
      </div>
    </div>
  );
}
