import { Bot, Paperclip } from "lucide-react";
import { Fragment, useEffect, useRef } from "react";

import type { Agent, Message } from "../../api/console";
import { AgentMark } from "../../components/console/AgentMark";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatNumber } from "../../i18n/format";
import { initials } from "../../lib/identity";
import { clockTime, dayLabel, sameDay } from "../../lib/time";

interface TranscriptProps {
  messages: Message[];
  agent?: Agent;
  agentName: string;
  /** Addresses that belong to the viewer, rendered as "You". */
  selfAddresses?: readonly string[];
  pendingIds?: ReadonlySet<string>;
}

/**
 * Non-sided transcript: every author is left-aligned with a mark and a name,
 * because a two-sided layout stops working the moment a room has three
 * authors. Agent-authored messages carry a tinted surface and an explicit
 * marker so they are distinguishable at a glance, without relying on hue.
 */
export function Transcript({ messages, agent, agentName, selfAddresses = ["owner", "browser"], pendingIds }: TranscriptProps) {
  const { t } = useTranslation();
  const end = useRef<HTMLDivElement>(null);
  const count = messages.length;
  useEffect(() => {
    const node = end.current;
    // jsdom has no scrollIntoView; the guard keeps the transcript testable.
    if (node && typeof node.scrollIntoView === "function") node.scrollIntoView({ block: "end" });
  }, [count]);

  return (
    <div data-testid="conversations-transcript" role="log" aria-live="polite" aria-relevant="additions" className="flex flex-col gap-1 px-1 py-2">
      {messages.map((message, index) => {
        const previous = messages[index - 1];
        const newDay = !previous || !sameDay(previous.received_at, message.received_at);
        const grouped = !newDay && previous.author_kind === message.author_kind && previous.sender_address === message.sender_address;
        const isAgent = message.author_kind === "agent";
        const isSelf = !isAgent && selfAddresses.includes(message.sender_address);
        const name = isAgent ? agentName : isSelf ? t(strings.console.conversations.you) : message.display_name || message.sender_address;
        const pending = pendingIds?.has(String(message.id)) ?? false;
        return (
          <Fragment key={message.id}>
            {newDay ? (
              <div role="separator" className="my-3 flex items-center gap-3 text-xs text-app-muted-foreground">
                <span className="h-px flex-1 bg-app-border" />
                {dayLabel(message.received_at)}
                <span className="h-px flex-1 bg-app-border" />
              </div>
            ) : null}
            <article
              data-testid="conversations-message"
              data-author-kind={message.author_kind}
              data-pending={pending || undefined}
              className={["group flex gap-3 rounded-panel px-2", grouped ? "py-0.5" : "mt-2 py-1"].join(" ")}
            >
              <div className="w-8 shrink-0">
                {grouped ? null : isAgent ? (
                  <AgentMark name={agentName} appearance={agent?.appearance} size="sm" />
                ) : isSelf ? (
                  <span aria-hidden="true" className="grid h-7 w-7 place-items-center rounded-full bg-app-primary/15 text-[11px] font-semibold text-app-primary">
                    {initials(name)}
                  </span>
                ) : (
                  <span aria-hidden="true" className="grid h-7 w-7 place-items-center rounded-full bg-app-surface-muted text-[11px] font-semibold text-app-muted-foreground">
                    {initials(name)}
                  </span>
                )}
              </div>
              <div className="min-w-0 flex-1">
                {grouped ? null : (
                  <div className="mb-0.5 flex items-baseline gap-2">
                    <span className="text-sm font-semibold text-app-foreground">{name}</span>
                    {isAgent ? (
                      <span
                        data-testid="conversations-agent-marker"
                        role="note"
                        className="inline-flex items-center gap-1 rounded-sm bg-app-accent/15 px-1.5 py-px text-[10px] font-semibold uppercase tracking-wide text-app-accent"
                      >
                        <Bot aria-hidden="true" className="h-3 w-3" />
                        {t(strings.console.conversations.agentMarker)}
                      </span>
                    ) : null}
                    <time dateTime={message.received_at} className="text-xs text-app-muted-foreground">
                      {clockTime(message.received_at)}
                    </time>
                  </div>
                )}
                <div
                  className={[
                    "max-w-[72ch] whitespace-pre-wrap break-words rounded-panel px-3 py-2 text-sm leading-relaxed",
                    isAgent ? "border-l-2 border-app-accent bg-app-accent/5 text-app-foreground" : "bg-app-surface-muted text-app-foreground",
                    pending ? "opacity-60" : "",
                  ].join(" ")}
                >
                  {message.text}
                  {message.media?.length ? (
                    <ul className="mt-2 flex flex-wrap gap-2">
                      {message.media.map((item) => (
                        <li key={`${item.name}-${item.size}`} className="inline-flex items-center gap-1.5 rounded-control border border-app-border bg-app-surface px-2 py-1 text-xs">
                          <Paperclip aria-hidden="true" className="h-3 w-3 text-app-muted-foreground" />
                          <span className="max-w-[16rem] truncate font-medium">{item.name}</span>
                          <span className="text-app-muted-foreground">{formatBytes(item.size)}</span>
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </div>
                {pending ? <span className="mt-0.5 block text-[11px] text-app-muted-foreground">{t(strings.console.conversations.sending)}</span> : null}
              </div>
            </article>
          </Fragment>
        );
      })}
      <div ref={end} />
    </div>
  );
}

export function formatBytes(size: number): string {
  if (size < 1024) return `${formatNumber(size)} B`;
  if (size < 1024 * 1024) return `${formatNumber(size / 1024, { maximumFractionDigits: 0 })} KB`;
  return `${formatNumber(size / (1024 * 1024), { maximumFractionDigits: 1 })} MB`;
}
