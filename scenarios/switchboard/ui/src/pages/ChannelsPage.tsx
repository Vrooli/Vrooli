import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Link2, Radio } from "lucide-react";
import { useState, type FormEvent } from "react";
import { Link, useParams } from "react-router-dom";

import { Button } from "@vrooli/react-component-library/Button/2";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";

import { consoleApi, consoleKeys, type ChannelListing } from "../api/console";
import { AgentMark } from "../components/console/AgentMark";
import { Page } from "../components/console/Page";
import { Quiet, Region } from "../components/console/Region";
import { strings } from "../consts/strings";
import { useSession } from "../features/session/SessionProvider";
import { useTranslation } from "../i18n";

const FRICTION_KEY = {
  none: strings.console.channels.friction.none,
  low: strings.console.channels.friction.low,
  medium: strings.console.channels.friction.medium,
  high: strings.console.channels.friction.high,
} as const;
const COST_KEY = { byok: strings.console.channels.cost.byok, metered: strings.console.channels.cost.metered } as const;

/**
 * The attach catalogue: every channel this install could use, ordered by how
 * much work each takes. Unavailable ones show what would satisfy their
 * requirement instead of failing three screens later.
 */
export function ChannelsPage() {
  const { t } = useTranslation();
  const { channelId } = useParams<{ channelId?: string }>();
  const [attaching, setAttaching] = useState<ChannelListing>();
  const [attached, setAttached] = useState<string>();
  const [showAll, setShowAll] = useState(false);

  const channels = useQuery({ queryKey: consoleKeys.channels, queryFn: ({ signal }) => consoleApi.channels(signal), refetchInterval: 30_000 });
  const overview = useQuery({ queryKey: consoleKeys.overview, queryFn: ({ signal }) => consoleApi.overview(signal), staleTime: 15_000 });
  const health = Object.fromEntries((overview.data?.channels ?? []).map((channel) => [channel.id, channel]));

  const list = [...(channels.data ?? [])].sort((a, b) => a.descriptor.setup.friction - b.descriptor.setup.friction);
  const available = list.filter((channel) => channel.availability === "available" && channel.implemented);
  const visible = showAll || available.length === 0 || channelId ? list : list.filter((channel) => channel.availability === "available" || (health[channel.descriptor.id]?.bindings ?? 0) > 0);
  const hidden = list.length - visible.length;
  const state = channels.isPending ? "loading" : channels.isError ? "error" : list.length === 0 ? "empty" : available.length < list.length ? "partial" : "ready";

  return (
    <Page headingId="channels-heading" testId="page-channels" title={t(strings.console.channels.title)} description={t(strings.console.channels.description)}>
      {attached ? (
        <div role="status" data-testid="channels-attached" className="flex items-center gap-2 rounded-panel border border-app-success/40 bg-app-success/5 px-3 py-2 text-sm">
          <CheckCircle2 aria-hidden="true" className="h-4 w-4 text-app-success" />
          {t(strings.console.channels.attached, { channel: attached })}
          <Link to="/conversations" className="ml-auto text-xs font-medium text-app-primary">
            {t(strings.console.channels.goToConversations)}
          </Link>
        </div>
      ) : null}
      <Region
        surfaceId="catalog-region"
        testId="channels-catalog-region"
        state={state}
        errorDetail={channels.error instanceof Error ? channels.error.message : undefined}
        onRetry={() => void channels.refetch()}
        skeletonRows={4}
        empty={<Quiet icon={<Radio className="h-6 w-6" />} title={t(strings.console.channels.emptyTitle)} description={t(strings.console.channels.emptyDetail)} />}
      >
        <ul data-testid="channels-catalog" className="flex flex-col gap-3">
          {visible.map((channel) => (
            <ChannelRow
              key={channel.descriptor.id}
              channel={channel}
              bindings={health[channel.descriptor.id]?.bindings ?? 0}
              threads={health[channel.descriptor.id]?.threads ?? 0}
              highlighted={channel.descriptor.id === channelId}
              onAttach={() => setAttaching(channel)}
            />
          ))}
        </ul>
        {hidden > 0 ? (
          <div className="flex justify-center">
            <Button type="button" variant="ghost" size="sm" data-testid="channels-show-all" onClick={() => setShowAll(true)}>
              {t(strings.console.channels.showAll, { count: hidden })}
            </Button>
          </div>
        ) : null}
      </Region>
      {attaching ? (
        <AttachDialog
          channel={attaching}
          onClose={() => setAttaching(undefined)}
          onAttached={(name) => {
            setAttached(name);
            setAttaching(undefined);
          }}
        />
      ) : null}
    </Page>
  );
}

function FrictionDots({ friction }: { friction: number }) {
  const { t } = useTranslation();
  const level = friction <= 0 ? "none" : friction === 1 ? "low" : friction === 2 ? "medium" : "high";
  return (
    <span data-testid="channels-friction" role="img" aria-label={t(strings.console.channels.setupEffort, { level: t(FRICTION_KEY[level]) })} className="inline-flex items-center gap-1.5 text-xs text-app-muted-foreground">
      <span aria-hidden="true" className="flex items-center gap-0.5">
        {[1, 2, 3, 4].map((step) => (
          <span key={step} className={["h-1.5 w-1.5 rounded-full", step <= friction ? "bg-app-foreground/70" : "bg-app-border"].join(" ")} />
        ))}
      </span>
      {t(FRICTION_KEY[level])}
    </span>
  );
}

function ChannelRow({ channel, bindings, threads, highlighted, onAttach }: { channel: ChannelListing; bindings: number; threads: number; highlighted: boolean; onAttach: () => void }) {
  const { t } = useTranslation();
  const d = channel.descriptor;
  const live = channel.availability === "available" && channel.implemented;
  const capabilities = [
    d.supports?.images ? t(strings.console.channels.supports.images) : null,
    d.supports?.files ? t(strings.console.channels.supports.files) : null,
    d.supports?.groups ? t(strings.console.channels.supports.groups) : null,
    d.supports?.threads ? t(strings.console.channels.supports.threads) : null,
  ].filter((item): item is string => item !== null);
  return (
    <li
      data-testid="channels-row"
      data-channel-id={d.id}
      className={["relative flex flex-col gap-3 overflow-hidden rounded-panel border bg-app-surface p-4 pl-5 sm:flex-row sm:items-center", highlighted ? "border-app-primary" : "border-app-border"].join(" ")}
    >
      <span aria-hidden="true" className="absolute inset-y-0 left-0 w-1.5" style={{ background: d.accent ?? "var(--color-accent)" }} />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="text-sm font-semibold text-app-foreground">{d.displayName}</h3>
          <StatusBadge tone={live ? "success" : channel.implemented ? "warning" : "neutral"} data-testid="channels-availability">
            {live ? t(strings.console.channels.availability.available) : channel.implemented ? t(strings.console.channels.availability.unavailable) : t(strings.console.channels.availability.unimplemented)}
          </StatusBadge>
          {d.cost && d.cost !== "free" ? (
            <span data-testid="channels-gated" role="note" className="rounded-pill bg-app-surface-muted px-2 py-0.5 text-xs text-app-muted-foreground">
              {t(COST_KEY[d.cost === "byok" ? "byok" : "metered"])}
            </span>
          ) : null}
        </div>
        <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-app-muted-foreground">
          <FrictionDots friction={d.setup.friction} />
          {capabilities.length > 0 ? <span>{capabilities.join(" · ")}</span> : null}
          {d.limits?.maxTextBytes ? <span className="font-mono">{t(strings.console.channels.textLimit, { kb: Math.round(d.limits.maxTextBytes / 1024) })}</span> : null}
        </div>
        {!live && channel.reason ? (
          <p className="mt-2 text-sm text-app-foreground">
            <span className="text-app-muted-foreground">{t(strings.console.channels.toEnable)}</span> {channel.reason}
          </p>
        ) : null}
        {live ? (
          <p data-testid="channels-health" className="mt-2 text-xs text-app-muted-foreground">
            {t(strings.console.overview.channelCounts, { bindings, threads })}
          </p>
        ) : null}
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {live ? (
          <Button type="button" size="sm" data-testid="channels-attach" onClick={onAttach}>
            <Link2 aria-hidden="true" className="h-4 w-4" />
            {t(strings.console.channels.attachAgent)}
          </Button>
        ) : (
          <Link to="/settings" data-testid="channels-requirement" className="inline-flex min-h-11 items-center rounded-control border border-app-border px-3 text-sm font-medium text-app-foreground hover:bg-app-surface-muted">
            {t(strings.console.channels.satisfyRequirement)}
          </Link>
        )}
      </div>
    </li>
  );
}

function AttachDialog({ channel, onClose, onAttached }: { channel: ChannelListing; onClose: () => void; onAttached: (channelName: string) => void }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { withSession } = useSession();
  const [agentId, setAgentId] = useState("");
  const [address, setAddress] = useState("");
  const [threadKey, setThreadKey] = useState("");
  const roster = useQuery({ queryKey: consoleKeys.agents, queryFn: ({ signal }) => consoleApi.agents(signal) });
  const attach = useMutation({
    mutationFn: () => withSession(() => consoleApi.createBinding({ agentId, channelId: channel.descriptor.id, address, threadKey })),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["console"] });
      onAttached(channel.descriptor.displayName);
      setAgentId("");
      setAddress("");
      setThreadKey("");
    },
  });
  const agents = (roster.data?.agents ?? []).filter((agent) => !agent.broken);
  const submit = (event: FormEvent) => {
    event.preventDefault();
    if (agentId && address.trim()) attach.mutate();
  };

  return (
    <ResponsiveDialog
      open
      onClose={onClose}
      title={t(strings.console.channels.attachTo, { channel: channel.descriptor.displayName })}
      closeLabel={t(strings.console.common.close)}
      size="sm"
      testId="channels-attach-dialog"
      footer={
        <div className="flex w-full justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>
            {t(strings.console.common.cancel)}
          </Button>
          <Button type="submit" form="attach-form" data-testid="channels-attach-confirm" disabled={!agentId || !address.trim()} pending={attach.isPending}>
            {t(strings.console.channels.confirmAttachment)}
          </Button>
        </div>
      }
    >
      <form id="attach-form" onSubmit={submit} aria-label={t(strings.console.channels.attachFormLabel)} className="flex flex-col gap-4">
        <p className="text-sm text-app-muted-foreground">{t(strings.console.channels.bindingDescription)}</p>
        <fieldset className="flex flex-col gap-1.5">
          <legend className="mb-1 text-sm font-medium">{t(strings.console.channels.agent)}</legend>
          {roster.isPending ? <p className="text-xs text-app-muted-foreground">{t(strings.console.region.loading)}</p> : null}
          <div role="radiogroup" aria-label={t(strings.console.channels.agent)} className="flex max-h-48 flex-col gap-1 overflow-y-auto">
            {agents.map((agent) => (
              <button
                  key={agent.id}
                  type="button"
                  role="radio"
                  aria-checked={agentId === agent.id}
                  onClick={() => setAgentId(agent.id)}
                  className={["flex w-full items-center gap-2 rounded-control border px-2.5 py-1.5 text-left text-sm", agentId === agent.id ? "border-app-primary bg-app-primary/5" : "border-app-border hover:bg-app-surface-muted"].join(" ")}
                >
                  <AgentMark name={agent.display_name} appearance={agent.appearance} size="xs" />
                  <span className="truncate">{agent.display_name}</span>
                </button>
            ))}
          </div>
        </fieldset>
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="binding-address">
          {t(strings.console.channels.address)}
          <input id="binding-address" data-testid="channels-binding-address" value={address} onChange={(event) => setAddress(event.target.value)} required placeholder={t(strings.console.channels.addressPlaceholder)} className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 text-base font-normal md:text-sm" />
          <span className="text-xs font-normal text-app-muted-foreground">{t(strings.console.channels.addressHint)}</span>
        </label>
        <label className="flex flex-col gap-1 text-sm font-medium" htmlFor="binding-thread">
          {t(strings.console.channels.threadKey)}
          <input id="binding-thread" value={threadKey} onChange={(event) => setThreadKey(event.target.value)} className="min-h-10 rounded-control border border-app-border bg-app-surface px-3 font-mono text-sm font-normal" />
          <span className="text-xs font-normal text-app-muted-foreground">{t(strings.console.channels.threadKeyHint)}</span>
        </label>
        {attach.isError ? (
          <p role="alert" className="text-sm text-app-danger">
            {attach.error instanceof Error ? attach.error.message : t(strings.errors.unknown)}
          </p>
        ) : null}
      </form>
    </ResponsiveDialog>
  );
}
