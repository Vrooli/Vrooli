import { KeyRound, Search, Users } from "lucide-react";
import { useMemo, useState } from "react";
import { Link } from "react-router-dom";

import type { Agent, Thread } from "../../api/console";
import { AgentMark } from "../../components/console/AgentMark";
import { budgetPressure } from "../../components/console/BudgetMeter";
import { ChannelChip } from "../../components/console/ChannelChip";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { truncate } from "../../lib/identity";
import { relativeTime } from "../../lib/time";

const SELF_ADDRESSES = new Set(["owner", "browser"]);

interface ThreadListProps {
  threads: Thread[];
  agents: Record<string, Agent>;
  selectedId?: string;
}

/**
 * Every thread across every channel in one list. The leading accent edge names
 * the channel before the text is read; the agent mark names who answers here.
 */
export function ThreadList({ threads, agents, selectedId }: ThreadListProps) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [channel, setChannel] = useState<string>("all");

  const channels = useMemo(() => {
    const seen = new Map<string, { id: string; name?: string; accent?: string }>();
    for (const thread of threads) {
      if (!seen.has(thread.channel_id)) seen.set(thread.channel_id, { id: thread.channel_id, name: thread.channel_display_name, accent: thread.channel_accent });
    }
    return [...seen.values()];
  }, [threads]);

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return threads.filter((thread) => {
      if (channel !== "all" && thread.channel_id !== channel) return false;
      if (!needle) return true;
      const haystack = [thread.thread_key, thread.agent_display_name, thread.agent_id, thread.last_message?.text, thread.last_message?.sender_address].join(" ").toLowerCase();
      return haystack.includes(needle);
    });
  }, [threads, query, channel]);

  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex flex-col gap-2 pb-3">
        <label className="relative block">
          <span className="sr-only">{t(strings.console.conversations.search)}</span>
          <Search aria-hidden="true" className="pointer-events-none absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-app-muted-foreground" />
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={t(strings.console.conversations.search)}
            data-testid="conversations-search"
            className="min-h-11 w-full rounded-control border border-app-border bg-app-surface pl-8 pr-3 text-base text-app-foreground placeholder:text-app-muted-foreground md:min-h-10 md:text-sm"
          />
        </label>
        {channels.length > 1 ? (
          <div role="group" aria-label={t(strings.console.conversations.filterByChannel)} className="flex flex-wrap gap-1.5">
            <FilterChip active={channel === "all"} onClick={() => setChannel("all")}>
              {t(strings.console.conversations.allChannels)}
            </FilterChip>
            {channels.map((entry) => (
              <FilterChip key={entry.id} active={channel === entry.id} onClick={() => setChannel(entry.id)}>
                <span aria-hidden="true" className="h-2 w-2 rounded-full" style={{ background: entry.accent ?? "var(--color-accent)" }} />
                {entry.name ?? entry.id}
              </FilterChip>
            ))}
          </div>
        ) : null}
      </div>
      <ul data-testid="conversations-thread-list" aria-label={t(strings.console.conversations.threads)} className="-mx-1 flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto px-1 pb-2">
        {visible.map((thread) => (
          <ThreadRow key={thread.id} thread={thread} agent={agents[thread.agent_id]} selected={thread.id === selectedId} />
        ))}
        {visible.length === 0 ? (
          <li className="px-3 py-6 text-center text-sm text-app-muted-foreground">{t(strings.console.conversations.noMatches)}</li>
        ) : null}
      </ul>
    </div>
  );
}

function FilterChip({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={[
        "inline-flex min-h-11 items-center gap-1.5 rounded-pill border px-3 text-xs font-medium transition-colors md:min-h-8 md:px-2.5",
        active ? "border-app-primary bg-app-primary/10 text-app-primary" : "border-app-border text-app-muted-foreground hover:bg-app-surface-muted",
      ].join(" ")}
    >
      {children}
    </button>
  );
}

function ThreadRow({ thread, agent, selected }: { thread: Thread; agent?: Agent; selected: boolean }) {
  const { t } = useTranslation();
  const last = thread.last_message;
  const { tone } = budgetPressure(thread.budget);
  const agentName = agent?.display_name || thread.agent_display_name || thread.agent_id;
  const humanIsSelf = last?.author_kind === "human" && SELF_ADDRESSES.has(last.sender_address);
  const human = last?.author_kind === "human" && !humanIsSelf ? last.display_name || last.sender_address : undefined;
  const title = human ?? (thread.is_group ? thread.thread_key : agentName);
  const previewAuthor = last ? (last.author_kind === "agent" ? agentName : humanIsSelf ? t(strings.console.conversations.you) : last.display_name || last.sender_address) : undefined;
  return (
    <li data-testid="conversations-thread-item" data-thread-id={thread.id}>
      <Link
        to={`/conversations/${thread.id}`}
        aria-current={selected ? "page" : undefined}
        className={[
          "relative flex gap-3 rounded-panel border py-2.5 pl-4 pr-3 transition-colors",
          selected ? "border-app-primary/40 bg-app-primary/5" : "border-transparent hover:bg-app-surface-muted",
        ].join(" ")}
      >
        <span aria-hidden="true" className="absolute inset-y-2.5 left-1.5 w-1 rounded-full" style={{ background: thread.channel_accent ?? "var(--color-accent)" }} />
        <AgentMark name={agentName} appearance={agent?.appearance} size="md" />
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between gap-2">
            <span className="truncate text-sm font-semibold text-app-foreground">{title}</span>
            {last ? <span className="shrink-0 text-xs text-app-muted-foreground">{relativeTime(last.received_at)}</span> : null}
          </div>
          <p className="truncate text-sm text-app-muted-foreground">
            {last ? (
              <>
                {previewAuthor && (last.author_kind === "agent" || humanIsSelf || thread.is_group) ? <span className="font-medium text-app-foreground/80">{previewAuthor}: </span> : null}
                {truncate(last.text, 90) || t(strings.console.conversations.attachmentOnly)}
              </>
            ) : (
              <span className="italic">{t(strings.console.conversations.noMessagesYet)}</span>
            )}
          </p>
          <div className="mt-1.5 flex flex-wrap items-center gap-1.5">
            <ChannelChip id={thread.channel_id} name={thread.channel_display_name} accent={thread.channel_accent} testId="conversations-channel-chip" />
            {thread.is_group ? (
              <span className="inline-flex items-center gap-1 text-xs text-app-muted-foreground" title={t(strings.console.conversations.groupRoom)}>
                <Users aria-hidden="true" className="h-3 w-3" />
                {thread.participant_count}
              </span>
            ) : null}
            {thread.pending_gates > 0 ? (
              <span className="inline-flex items-center gap-1 rounded-pill bg-app-warning/15 px-1.5 py-0.5 text-[11px] font-semibold text-app-warning">
                <KeyRound aria-hidden="true" className="h-3 w-3" />
                {t(strings.console.conversations.gateWaiting)}
              </span>
            ) : null}
            {tone === "warning" || tone === "danger" ? (
              <span className={["rounded-pill px-1.5 py-0.5 text-[11px] font-semibold", tone === "danger" ? "bg-app-danger/15 text-app-danger" : "bg-app-warning/15 text-app-warning"].join(" ")}>
                {tone === "danger" ? t(strings.console.budget.exhausted) : t(strings.console.budget.nearLimit)}
              </span>
            ) : null}
          </div>
        </div>
      </Link>
    </li>
  );
}
