import { memo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Info, Paperclip } from "lucide-react";
import type { ConversationEvent } from "../../api/conversation";
import { strings } from "../../consts/strings";
import { cn } from "../../lib/classnames";
import { eventTime, speakerKey, speakerParams } from "./speaker";
import { noticeSummary, noticeTitleKey } from "./noticeSummary";
import { useTranslate } from "./useTranslate";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#notices

interface NoticeRowProps {
  event: ConversationEvent;
  isDimmed?: boolean;
  isSearchFocused?: boolean;
}

/**
 * Something the harness or a program put into the conversation — a task
 * update, an interruption, a slash command, an image the operator attached —
 * as one quiet line in secondary text. It is never shown as the operator's
 * own message. Tapping it shows the full text.
 */
function NoticeRowImpl({ event, isDimmed = false, isSearchFocused = false }: NoticeRowProps) {
  const t = useTranslate();
  const { i18n } = useTranslation();
  const [open, setOpen] = useState(false);
  const attachment = event.kind === "attachment";
  const Icon = attachment ? Paperclip : Info;
  const title = t(noticeTitleKey(event));
  // An attachment is the operator's; a notice names its writer only when it
  // is a person or another agent (a harness line speaks for itself).
  const writer = attachment || event.author === "peer" || event.author === "human"
    ? t(speakerKey(event), speakerParams(event))
    : null;
  const clock = eventTime(event, new Date(), i18n.language);
  return (
    <div
      data-testid={`msg-card-${event.id}`}
      data-notice={event.label ?? event.kind}
      tabIndex={-1}
      className={cn("border-b border-wc-default/60 px-3", isDimmed && "opacity-40", isSearchFocused && "ring-1 ring-wc-accent/50")}
    >
      <button
        type="button"
        data-testid={`notice-toggle-${event.id}`}
        aria-expanded={open}
        aria-label={`${title}. ${t(open ? strings.messagesPane.notice.hideAria : strings.messagesPane.notice.showAria)}`}
        onClick={() => { setOpen((value) => !value); }}
        className="flex min-h-9 w-full min-w-0 items-center gap-1.5 py-1 text-start text-[12px] text-wc-text-faint transition-colors hover:text-wc-text-secondary"
      >
        <Icon aria-hidden="true" className="h-3.5 w-3.5 shrink-0" />
        {writer && <span data-testid={`notice-writer-${event.id}`} className="shrink-0 font-medium text-wc-text-muted">{writer}</span>}
        <span className="shrink-0 font-medium">{title}</span>
        <span data-testid={`notice-summary-${event.id}`} className="min-w-0 truncate">{noticeSummary(event)}</span>
        <time dateTime={event.occurredAt || event.createdAt} title={`#${String(event.sequence)}`} className="ms-auto shrink-0 font-mono text-[10px]">
          {clock.timestamp}
        </time>
      </button>
      {open && (
        <pre data-testid={`notice-text-${event.id}`} className="mb-2 max-h-64 overflow-auto whitespace-pre-wrap break-words rounded bg-wc-surface-input/40 p-2 font-mono text-[11px] text-wc-text-secondary [overflow-wrap:anywhere]">
          {event.text}
        </pre>
      )}
    </div>
  );
}

export const NoticeRow = memo(NoticeRowImpl);
