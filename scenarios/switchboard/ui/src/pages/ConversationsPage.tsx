import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, MessagesSquare, Plus, Users, VolumeX } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import { ConsoleApiError, consoleApi, consoleKeys, type Agent, type Message, type Thread, type ThreadDetail } from "../api/console";
import { AgentMark } from "../components/console/AgentMark";
import { BudgetMeter, budgetPressure } from "../components/console/BudgetMeter";
import { ChannelChip } from "../components/console/ChannelChip";
import { GateCard } from "../components/console/GateCard";
import { Page } from "../components/console/Page";
import { Quiet, Region } from "../components/console/Region";
import { TierBadge } from "../components/console/TierBadge";
import { strings } from "../consts/strings";
import { Composer } from "../features/conversations/Composer";
import { StartConversationDialog } from "../features/conversations/StartConversationDialog";
import { ThreadList } from "../features/conversations/ThreadList";
import { Transcript } from "../features/conversations/Transcript";
import { useInAppSocket } from "../features/conversations/useInAppSocket";
import { useTranslation } from "../i18n";
import { initials } from "../lib/identity";

const SELF_ADDRESS = "owner";

/**
 * Conversations: every thread across every channel in one list, with one
 * thread open beside it. On mobile the list and the thread are two screens.
 */
export function ConversationsPage() {
  const { t } = useTranslation();
  const { threadId } = useParams<{ threadId: string }>();
  const [starting, setStarting] = useState(false);

  const threads = useQuery({ queryKey: consoleKeys.threads, queryFn: ({ signal }) => consoleApi.threads(signal), refetchInterval: 15_000 });
  const roster = useQuery({ queryKey: consoleKeys.agents, queryFn: ({ signal }) => consoleApi.agents(signal), staleTime: 60_000 });
  const rosterAgents = roster.data?.agents;
  const agents = useMemo(() => Object.fromEntries((rosterAgents ?? []).map((agent) => [agent.id, agent])), [rosterAgents]);

  const listState = threads.isPending ? "loading" : threads.isError ? "error" : threads.data.length === 0 ? "empty" : "ready";
  const showListOnMobile = !threadId;

  return (
    <Page
      headingId="conversations-heading"
      title={t(strings.console.conversations.title)}
      description={t(strings.console.conversations.description)}
      layout="fill"
      actions={
        <Button type="button" size="sm" data-testid="conversations-new" onClick={() => setStarting(true)}>
          <Plus aria-hidden="true" className="h-4 w-4" />
          {t(strings.console.conversations.startCta)}
        </Button>
      }
    >
      <div className="grid min-h-0 flex-1 gap-4 md:grid-cols-[minmax(280px,340px)_minmax(0,1fr)]">
        <Region
          surfaceId="thread-list-region"
          testId="conversations-thread-list-region"
          state={listState}
          className="min-h-0"
          errorDetail={threads.error instanceof Error ? threads.error.message : undefined}
          onRetry={() => void threads.refetch()}
          skeletonRows={6}
          empty={
            <Quiet
              icon={<MessagesSquare className="h-6 w-6" />}
              title={t(strings.console.conversations.emptyTitle)}
              description={t(strings.console.conversations.emptyDescription)}
              action={
                <Button type="button" size="sm" data-testid="conversations-empty-cta" onClick={() => setStarting(true)}>
                  {t(strings.console.conversations.startCta)}
                </Button>
              }
            />
          }
        >
          <div className={showListOnMobile ? "flex min-h-0 flex-1 flex-col" : "hidden min-h-0 flex-1 md:flex md:flex-col"}>
            <ThreadList threads={threads.data ?? []} agents={agents} selectedId={threadId} />
          </div>
          {showListOnMobile ? null : (
            <ThreadStrip threads={threads.data ?? []} agents={agents} selectedId={threadId} />
          )}
        </Region>

        <div className="flex min-h-0 min-w-0 flex-col">
          {threadId ? (
            <ThreadPane key={threadId} threadId={threadId} agents={agents} />
          ) : (
            <Region surfaceId="transcript-region" testId="conversations-transcript-region" state="empty" className="md:h-full">
              <div className="flex items-center justify-center rounded-panel border border-dashed border-app-border px-4 py-3 text-sm text-app-muted-foreground md:h-full md:min-h-[16rem]">
                {t(strings.console.conversations.pickThread)}
              </div>
            </Region>
          )}
        </div>
      </div>
      {starting ? <StartConversationDialog open onClose={() => setStarting(false)} /> : null}
    </Page>
  );
}

/** Phone-width thread switcher shown above an open thread, so the list region stays present without a second screen. */
function ThreadStrip({ threads, agents, selectedId }: { threads: Thread[]; agents: Record<string, Agent>; selectedId?: string }) {
  const { t } = useTranslation();
  return (
    <div role="list" aria-label={t(strings.console.conversations.threads)} data-testid="conversations-thread-strip" className="flex gap-2 overflow-x-auto pb-1 md:hidden">
      {threads.map((thread) => {
        const agent = agents[thread.agent_id];
        const name = agent?.display_name || thread.agent_display_name || thread.agent_id;
        const selected = thread.id === selectedId;
        return (
          <Link
            key={thread.id}
            role="listitem"
            to={`/conversations/${thread.id}`}
            aria-current={selected ? "page" : undefined}
            className={["inline-flex min-h-11 shrink-0 items-center gap-2 rounded-pill border py-1 pl-1 pr-3 text-xs font-medium", selected ? "border-app-primary bg-app-primary/10 text-app-primary" : "border-app-border text-app-foreground"].join(" ")}
          >
            <AgentMark name={name} appearance={agent?.appearance} size="sm" />
            <span className="max-w-[9rem] truncate">{thread.is_group ? thread.thread_key : name}</span>
            <span aria-hidden="true" className="h-2 w-2 rounded-full" style={{ background: thread.channel_accent ?? "var(--color-accent)" }} />
          </Link>
        );
      })}
    </div>
  );
}

function ThreadPane({ threadId, agents }: { threadId: string; agents: Record<string, Agent> }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [pending, setPending] = useState<Message[]>([]);
  const [busy, setBusy] = useState(false);

  const detail = useQuery({
    queryKey: consoleKeys.thread(threadId),
    queryFn: ({ signal }) => consoleApi.thread(threadId, signal),
    refetchInterval: (query) => (query.state.data?.thread.channel_id === "in-app" ? 20_000 : 10_000),
  });
  const data = detail.data;
  const thread = data?.thread;
  const agent = thread ? agents[thread.agent_id] : undefined;
  const agentName = agent?.display_name ?? thread?.agent_display_name ?? thread?.agent_id ?? "";
  const isInApp = thread?.channel_id === "in-app";

  const onSocketMessage = useCallback(
    (payload: { text?: string; error?: string }) => {
      if (payload.error) return;
      setPending([]);
      void queryClient.invalidateQueries({ queryKey: consoleKeys.thread(threadId) });
      void queryClient.invalidateQueries({ queryKey: consoleKeys.threads });
    },
    [queryClient, threadId],
  );
  const socket = useInAppSocket({ threadKey: isInApp ? thread.thread_key : undefined, onMessage: onSocketMessage });

  const send = (text: string, attachment?: File): boolean => {
    if (!thread) return false;
    const remoteId = `console-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
    const media = attachment ? [{ name: attachment.name, mime: attachment.type || "application/octet-stream", size: attachment.size }] : undefined;
    const ok = socket.send({
      channel_id: thread.channel_id,
      remote_message_id: remoteId,
      thread_key: thread.thread_key,
      sender_address: SELF_ADDRESS,
      author_kind: "human",
      text,
      media,
      received_at: new Date().toISOString(),
    });
    if (!ok) return false;
    setBusy(true);
    setPending((current) => [...current, { id: remoteId, remote_id: remoteId, author_kind: "human", sender_address: SELF_ADDRESS, text, received_at: new Date().toISOString(), media }]);
    window.setTimeout(() => setBusy(false), 400);
    void queryClient.invalidateQueries({ queryKey: consoleKeys.thread(threadId) });
    return true;
  };

  const messages = useMemo(() => {
    const persisted = data?.messages ?? [];
    const seen = new Set(persisted.map((message) => message.remote_id));
    return [...persisted, ...pending.filter((message) => !seen.has(message.remote_id))];
  }, [data?.messages, pending]);
  const pendingIds = useMemo(() => new Set(pending.map((message) => String(message.id))), [pending]);

  const notFound = detail.error instanceof ConsoleApiError && detail.error.status === 404;
  const transcriptState = detail.isPending ? "loading" : notFound ? "empty" : detail.isError ? "error" : messages.length === 0 ? "empty" : "ready";
  const pendingGates = data?.gates.filter((gate) => gate.status === "pending") ?? [];
  const ceilingBlocksAll = thread?.ceiling_tier === "stranger";
  const { tone } = thread ? budgetPressure(thread.budget) : { tone: "neutral" as const };

  let composerReason: string | undefined;
  if (thread && !isInApp) composerReason = t(strings.console.conversations.repliesLiveOn, { channel: thread.channel_display_name ?? thread.channel_id });
  else if (isInApp && socket.state !== "open") composerReason = t(strings.console.conversations.connecting);

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden rounded-panel border border-app-border bg-app-surface">
      <header className="flex shrink-0 flex-col gap-2 border-b border-app-border px-3 py-2.5 md:px-4">
        <div className="flex items-center gap-3">
          <Link to="/conversations" aria-label={t(strings.console.common.back)} className="grid h-11 w-11 shrink-0 place-items-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted md:hidden">
            <ArrowLeft aria-hidden="true" className="h-4 w-4" />
          </Link>
          {thread ? (
            <>
              <AgentMark name={agentName} appearance={agent?.appearance} size="md" live={thread.channel_id === "in-app" ? socket.state === "open" : undefined} />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <h3 className="truncate text-sm font-semibold text-app-foreground">{agentName}</h3>
                  <ChannelChip id={thread.channel_id} name={thread.channel_display_name} accent={thread.channel_accent} />
                  {thread.is_group ? (
                    <span className="inline-flex items-center gap-1 text-xs text-app-muted-foreground">
                      <VolumeX aria-hidden="true" className="h-3.5 w-3.5" />
                      {t(strings.console.conversations.speaksWhenAddressed)}
                    </span>
                  ) : null}
                </div>
                <p className="truncate font-mono text-[11px] text-app-muted-foreground">{thread.thread_key}</p>
              </div>
              <div className="hidden shrink-0 items-center gap-2 sm:flex">
                <StatusBadge tone={ceilingBlocksAll ? "danger" : tone === "danger" ? "danger" : "success"} data-testid="conversations-permission">
                  {ceilingBlocksAll ? t(strings.console.conversations.agentRefusing) : tone === "danger" ? t(strings.console.conversations.agentSilenced) : t(strings.console.conversations.agentPermitted)}
                </StatusBadge>
              </div>
            </>
          ) : (
            <div className="h-9 flex-1 animate-pulse rounded-control bg-app-surface-muted" />
          )}
        </div>
        {thread && (data.participants.length > 0 || thread.is_group) ? (
          <div data-testid="conversations-roster" role="group" aria-label={t(strings.console.conversations.roster)} className="flex flex-wrap items-center gap-2 text-xs">
            <Users aria-hidden="true" className="h-3.5 w-3.5 text-app-muted-foreground" />
            {data.participants.map((participant) => (
              <Link key={participant.contact_id} to={`/contacts/${participant.contact_id}`} className="inline-flex items-center gap-1.5 rounded-pill border border-app-border py-0.5 pl-0.5 pr-2 hover:bg-app-surface-muted">
                <span aria-hidden="true" className="grid h-5 w-5 place-items-center rounded-full bg-app-surface-muted text-[9px] font-semibold text-app-muted-foreground">
                  {initials(participant.display_name || (participant.address === SELF_ADDRESS ? t(strings.console.conversations.you) : participant.address))}
                </span>
                <span className="max-w-[10rem] truncate font-medium text-app-foreground">{participant.display_name || (participant.address === SELF_ADDRESS ? t(strings.console.conversations.you) : participant.address)}</span>
                <TierBadge tier={participant.tier} />
              </Link>
            ))}
            <span className="ml-auto text-app-muted-foreground">
              {t(strings.console.conversations.roomCeiling)} <TierBadge tier={thread.ceiling_tier} testId="conversations-ceiling" />
            </span>
          </div>
        ) : null}
      </header>

      <Region
        surfaceId="transcript-region"
        testId="conversations-transcript-region"
        state={transcriptState}
        className="min-h-0 flex-1 overflow-y-auto px-2 md:px-3"
        errorDetail={detail.error instanceof Error ? detail.error.message : undefined}
        onRetry={() => void detail.refetch()}
        skeletonRows={5}
        empty={notFound ? <Quiet title={t(strings.console.conversations.threadNotFound)} description={t(strings.console.conversations.threadNotFoundDetail)} /> : <Quiet title={t(strings.console.conversations.noMessagesYet)} description={isInApp ? t(strings.console.conversations.sayHello) : undefined} />}
      >
        <Transcript messages={messages} agent={agent} agentName={agentName} pendingIds={pendingIds} selfAddresses={[SELF_ADDRESS, "browser"]} />
      </Region>

      <footer className="shrink-0 border-t border-app-border bg-app-background/60 p-2 pb-safe md:p-3">
        {pendingGates.length > 0 ? (
          <Region surfaceId="gate-region" testId="conversations-gate-region" state="ready" className="mb-2">
            {pendingGates.map((gate) => (
              <GateCard key={gate.id} gate={gate} compact testId="conversations-gate" />
            ))}
          </Region>
        ) : null}
        {thread && (tone === "warning" || tone === "danger") ? (
          <div className="mb-2 rounded-panel border border-app-border bg-app-surface p-2">
            <BudgetMeter budget={thread.budget} compact testId="conversations-budget" />
          </div>
        ) : null}
        {thread && ceilingBlocksAll ? (
          <p data-testid="conversations-silence" role="status" className="mb-2 rounded-panel border border-app-danger/30 bg-app-danger/5 px-3 py-2 text-xs text-app-foreground">
            {t(strings.console.conversations.strangerCeiling)}{" "}
            <Link to="/contacts" className="font-medium text-app-primary">
              {t(strings.console.conversations.reviewContacts)}
            </Link>
          </p>
        ) : null}
        {notFound ? null : <Composer disabledReason={composerReason} busy={busy} onSend={send} />}
      </footer>
    </div>
  );
}

export type { ThreadDetail };
