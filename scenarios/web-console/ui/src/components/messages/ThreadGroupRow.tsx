import { memo } from "react";
import { Bot, ChevronRight, ShieldCheck, Users } from "lucide-react";
import type { ConversationThreadSummary } from "../../api/conversation";
import { strings } from "../../consts/strings";
import { useTranslate } from "./useTranslate";
import { threadDuration } from "./workSummary";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#agent-threads

const KIND_LABEL: Readonly<Record<ConversationThreadSummary["kind"], string>> = {
  subagent: strings.messagesPane.thread.subagent,
  teammate: strings.messagesPane.thread.teammate,
  review: strings.messagesPane.thread.review,
};

const KIND_ICON = { subagent: Bot, teammate: Users, review: ShieldCheck } as const;

/**
 * An agent the session started or talked to — a subagent, a teammate, the
 * approval reviewer — collapsed to one group in the conversation at the point
 * it began: its name, what it was asked, how many messages and how long. Its
 * own conversation opens in place of the list.
 */
function ThreadGroupRowImpl({ thread, onOpen }: { thread: ConversationThreadSummary; onOpen: (threadId: string) => void }) {
  const t = useTranslate();
  const name = thread.label || t(strings.messagesPane.thread.unnamed);
  const Icon = KIND_ICON[thread.kind];
  const duration = threadDuration(thread, t);
  const facts = [
    t(KIND_LABEL[thread.kind]),
    t(strings.messagesPane.thread.messageCount, { count: thread.messageCount }),
    ...(duration ? [duration] : []),
  ];
  return (
    <div className="px-3 py-1.5">
      <button
        type="button"
        data-testid={`thread-group-${thread.threadId}`}
        data-thread-kind={thread.kind}
        data-message-count={thread.messageCount}
        aria-label={t(strings.messagesPane.thread.openAria, { name })}
        onClick={() => { onOpen(thread.threadId); }}
        className="flex w-full min-w-0 items-start gap-2 rounded-lg border border-wc-default bg-wc-surface-input/30 px-3 py-2 text-start transition-colors hover:bg-wc-surface-input/60"
      >
        <Icon aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-wc-text-muted" />
        <span className="min-w-0 flex-1">
          <span className="flex min-w-0 items-baseline gap-1.5">
            <b data-testid={`thread-name-${thread.threadId}`} className="truncate text-[12.5px] font-semibold text-wc-text-secondary">{name}</b>
            <span data-testid={`thread-facts-${thread.threadId}`} className="truncate text-[11px] text-wc-text-faint">{facts.join(" · ")}</span>
          </span>
          {thread.description && (
            <span className="mt-0.5 line-clamp-1 block text-[12px] text-wc-text-muted">{thread.description}</span>
          )}
        </span>
        <ChevronRight aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0 text-wc-text-faint rtl:-scale-x-100" />
      </button>
    </div>
  );
}

export const ThreadGroupRow = memo(ThreadGroupRowImpl);
