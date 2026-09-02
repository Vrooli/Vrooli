import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangle, ArrowLeft, Users } from "lucide-react";
import { useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { Button } from "@vrooli/react-component-library/Button/2";

import { ConsoleApiError, TRUST_TIERS, consoleApi, consoleKeys, type Contact, type TierChange, type TrustTier } from "../api/console";
import { ChannelChip } from "../components/console/ChannelChip";
import { Page } from "../components/console/Page";
import { Quiet, Region } from "../components/console/Region";
import { TIER_LABEL_KEY, TierBadge, tierRank } from "../components/console/TierBadge";
import { strings } from "../consts/strings";
import { useSession } from "../features/session/SessionProvider";
import { useTranslation } from "../i18n";
import { initials } from "../lib/identity";
import { relativeTime } from "../lib/time";

const TIER_EFFECT_KEY: Record<TrustTier, (typeof strings.console.tiers.effect)[TrustTier]> = {
  stranger: strings.console.tiers.effect.stranger,
  known: strings.console.tiers.effect.known,
  trusted: strings.console.tiers.effect.trusted,
  owner: strings.console.tiers.effect.owner,
};

/**
 * Everyone who has ever reached an agent here, and the tier each one holds.
 * The tier change is the only edit on this page and the one with the largest
 * blast radius, so what it permits and which rooms it narrows are stated
 * before it is confirmed.
 */
export function ContactsPage() {
  const { t } = useTranslation();
  const { contactId } = useParams<{ contactId?: string }>();
  const contacts = useQuery({ queryKey: consoleKeys.contacts, queryFn: ({ signal }) => consoleApi.contacts(signal), refetchInterval: 30_000 });
  const [query, setQuery] = useState("");
  const list = useMemo(() => contacts.data ?? [], [contacts.data]);
  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return list.filter((contact) => !needle || `${contact.display_name ?? ""} ${contact.address} ${contact.channel_id}`.toLowerCase().includes(needle));
  }, [list, query]);
  const state = contacts.isPending ? "loading" : contacts.isError ? "error" : list.length === 0 ? "empty" : "ready";
  const showListOnMobile = !contactId;

  return (
    <Page headingId="contacts-heading" testId="page-contacts" title={t(strings.console.contacts.title)} description={t(strings.console.contacts.description)} layout="fill">
      <div className="grid min-h-0 flex-1 gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(320px,2fr)]">
        <Region
          surfaceId="contact-region"
          testId="contacts-contact-region"
          state={state}
          className="min-h-0"
          errorDetail={contacts.error instanceof Error ? contacts.error.message : undefined}
          onRetry={() => void contacts.refetch()}
          skeletonRows={5}
          empty={<Quiet icon={<Users className="h-6 w-6" />} title={t(strings.console.contacts.emptyTitle)} description={t(strings.console.contacts.empty)} />}
        >
          {showListOnMobile ? null : (
            <div role="list" aria-label={t(strings.console.contacts.title)} data-testid="contacts-strip" className="flex gap-2 overflow-x-auto pb-1 lg:hidden">
              {visible.map((contact) => {
                const name = contact.display_name?.trim() || contact.address;
                const selected = contact.id === contactId;
                return (
                  <Link key={contact.id} role="listitem" to={`/contacts/${encodeURIComponent(contact.id)}`} aria-current={selected ? "page" : undefined} className={["inline-flex min-h-11 shrink-0 items-center gap-2 rounded-pill border px-3 text-xs font-medium", selected ? "border-app-primary bg-app-primary/10 text-app-primary" : "border-app-border text-app-foreground"].join(" ")}>
                    <span aria-hidden="true" className="h-2 w-2 rounded-full" style={{ background: contact.channel_accent ?? "var(--color-accent)" }} />
                    <span className="max-w-[9rem] truncate">{name}</span>
                  </Link>
                );
              })}
            </div>
          )}
          <div className={showListOnMobile ? "flex min-h-0 flex-1 flex-col gap-3" : "hidden min-h-0 flex-1 flex-col gap-3 lg:flex"}>
          <label className="block">
            <span className="sr-only">{t(strings.console.contacts.search)}</span>
            <input
              type="search"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder={t(strings.console.contacts.search)}
              data-testid="contacts-search"
              className="min-h-11 w-full rounded-control border border-app-border bg-app-surface px-3 text-base text-app-foreground placeholder:text-app-muted-foreground md:min-h-10 md:text-sm"
            />
          </label>
          <ul data-testid="contacts-list" className="min-h-0 flex-1 divide-y divide-app-border overflow-y-auto rounded-panel border border-app-border bg-app-surface">
            {visible.map((contact) => (
              <ContactRow key={contact.id} contact={contact} selected={contact.id === contactId} />
            ))}
            {visible.length === 0 ? <li className="px-3 py-6 text-center text-sm text-app-muted-foreground">{t(strings.console.contacts.noMatches)}</li> : null}
          </ul>
          </div>
        </Region>
        <div className={["min-h-0", showListOnMobile ? "hidden lg:block" : "block"].join(" ")}>
          {contactId ? (
            <ContactPanel key={contactId} contactId={contactId} />
          ) : (
            <div className="flex h-full min-h-[12rem] items-center justify-center rounded-panel border border-dashed border-app-border p-6 text-center text-sm text-app-muted-foreground">
              {t(strings.console.contacts.pickContact)}
            </div>
          )}
        </div>
      </div>
    </Page>
  );
}

function ContactRow({ contact, selected }: { contact: Contact; selected: boolean }) {
  const { t } = useTranslation();
  const name = contact.display_name?.trim() || contact.address;
  return (
    <li data-testid="contacts-row" data-contact-id={contact.id}>
      <Link
        to={`/contacts/${encodeURIComponent(contact.id)}`}
        aria-current={selected ? "page" : undefined}
        className={["flex items-center gap-3 px-3 py-2.5 transition-colors", selected ? "bg-app-primary/5" : "hover:bg-app-surface-muted"].join(" ")}
      >
        <span aria-hidden="true" className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-app-surface-muted text-xs font-semibold text-app-muted-foreground">
          {initials(name)}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="truncate text-sm font-semibold text-app-foreground">{name}</span>
            {contact.display_name ? <code className="hidden truncate font-mono text-[11px] text-app-muted-foreground sm:inline">{contact.address}</code> : null}
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-app-muted-foreground">
            <ChannelChip id={contact.channel_id} name={contact.channel_display_name} accent={contact.channel_accent} />
            <span>{t(strings.console.contacts.lastSeen, { when: relativeTime(contact.last_seen) })}</span>
            {contact.room_count > 1 ? <span>{t(strings.console.contacts.roomCount, { count: contact.room_count })}</span> : null}
          </div>
        </div>
        <TierBadge tier={contact.tier} testId="contacts-tier-badge" />
      </Link>
    </li>
  );
}

function ContactPanel({ contactId }: { contactId: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { withSession } = useSession();
  const detail = useQuery({ queryKey: consoleKeys.contact(contactId), queryFn: ({ signal }) => consoleApi.contact(contactId, signal) });
  const [draftTier, setDraftTier] = useState<TrustTier>();
  const [result, setResult] = useState<TierChange>();
  const contact = detail.data?.contact;
  const rooms = detail.data?.rooms ?? [];
  const currentTier = contact?.tier;
  const nextTier = draftTier ?? currentTier;
  const changed = nextTier !== undefined && nextTier !== currentTier;
  const narrowing = contact && nextTier && tierRank(nextTier) < tierRank(contact.tier);
  const affected = contact && narrowing ? rooms.filter((room) => room.participant_count > 1 && tierRank(room.ceiling_tier) >= tierRank(contact.tier)) : [];

  const update = useMutation({
    mutationFn: (tier: TrustTier) => withSession(() => consoleApi.updateContact(contactId, { tier })),
    onSuccess: async (change) => {
      setResult(change);
      setDraftTier(undefined);
      await queryClient.invalidateQueries({ queryKey: ["console"] });
    },
  });

  const notFound = detail.error instanceof ConsoleApiError && detail.error.status === 404;
  const roomsState = detail.isPending ? "loading" : notFound ? "empty" : detail.isError ? "error" : rooms.length === 0 ? "empty" : "ready";
  const name = contact?.display_name?.trim() || contact?.address || contactId;

  return (
    <div className="flex h-full min-h-0 flex-col gap-4 overflow-y-auto rounded-panel border border-app-border bg-app-surface p-4">
      <div className="flex items-center gap-3">
        <button type="button" onClick={() => navigate("/contacts")} aria-label={t(strings.console.common.back)} className="grid h-11 w-11 place-items-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted lg:hidden">
          <ArrowLeft aria-hidden="true" className="h-4 w-4" />
        </button>
        <span aria-hidden="true" className="grid h-12 w-12 shrink-0 place-items-center rounded-full bg-app-surface-muted text-sm font-semibold text-app-muted-foreground">
          {initials(name)}
        </span>
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-base font-semibold text-app-foreground">{name}</h3>
          {contact ? (
            <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-app-muted-foreground">
              <ChannelChip id={contact.channel_id} name={contact.channel_display_name} accent={contact.channel_accent} />
              <code className="font-mono">{contact.address}</code>
            </div>
          ) : null}
        </div>
        {contact ? <TierBadge tier={contact.tier} size="md" /> : null}
      </div>

      {contact ? (
        <dl className="grid grid-cols-3 gap-px overflow-hidden rounded-panel border border-app-border bg-app-border text-xs">
          {[
            [t(strings.console.contacts.firstSeen), relativeTime(contact.first_seen)],
            [t(strings.console.contacts.messages), String(contact.message_count)],
            [t(strings.console.contacts.rooms), String(contact.room_count)],
          ].map(([label, value]) => (
            <div key={label} className="bg-app-surface px-3 py-2">
              <dt className="text-app-muted-foreground">{label}</dt>
              <dd className="font-mono text-sm font-semibold text-app-foreground">{value}</dd>
            </div>
          ))}
        </dl>
      ) : null}

      <fieldset data-testid="contacts-tier-control" className="flex flex-col gap-2" disabled={!contact || update.isPending}>
        <legend className="mb-1 text-sm font-semibold">{t(strings.console.contacts.trustTier)}</legend>
        <div role="radiogroup" aria-label={t(strings.console.contacts.trustTier)} className="flex flex-col gap-1.5">
          {TRUST_TIERS.map((tier) => {
            const selected = nextTier === tier;
            return (
              <button
                key={tier}
                type="button"
                role="radio"
                aria-checked={selected}
                data-testid={`contacts-tier-${tier}`}
                onClick={() => setDraftTier(tier)}
                className={["flex items-start gap-3 rounded-panel border px-3 py-2.5 text-left", selected ? "border-app-primary bg-app-primary/5" : "border-app-border hover:bg-app-surface-muted"].join(" ")}
              >
                <TierBadge tier={tier} className="mt-0.5 shrink-0" />
                <span className="min-w-0 text-sm">
                  <span className="block font-medium text-app-foreground">{t(TIER_LABEL_KEY[tier])}</span>
                  <span data-testid={selected ? "contacts-tier-effect" : undefined} role={selected ? "note" : undefined} className="block text-xs text-app-muted-foreground">
                    {t(TIER_EFFECT_KEY[tier])}
                  </span>
                </span>
              </button>
            );
          })}
        </div>
        {changed && affected.length > 0 ? (
          <div data-testid="contacts-ceiling-warning" role="alert" className="flex items-start gap-2 rounded-panel border border-app-warning/50 bg-app-warning/10 px-3 py-2 text-xs text-app-foreground">
            <AlertTriangle aria-hidden="true" className="mt-0.5 h-3.5 w-3.5 shrink-0 text-app-warning" />
            <p>{t(strings.console.contacts.ceilingWarning, { count: affected.length, tier: t(TIER_LABEL_KEY[nextTier]) })}</p>
          </div>
        ) : null}
        {changed && nextTier === "owner" ? (
          <div role="alert" className="flex items-start gap-2 rounded-panel border border-app-danger/40 bg-app-danger/5 px-3 py-2 text-xs text-app-foreground">
            <AlertTriangle aria-hidden="true" className="mt-0.5 h-3.5 w-3.5 shrink-0 text-app-danger" />
            <p>{t(strings.console.contacts.ownerWarning)}</p>
          </div>
        ) : null}
        <div className="flex items-center gap-2 pt-1">
          <Button type="button" size="sm" data-testid="contacts-confirm" disabled={!changed} pending={update.isPending} onClick={() => nextTier && update.mutate(nextTier)}>
            {t(strings.console.contacts.changeTier)}
          </Button>
          {changed ? (
            <Button type="button" size="sm" variant="ghost" onClick={() => setDraftTier(undefined)}>
              {t(strings.console.common.cancel)}
            </Button>
          ) : null}
        </div>
        {update.isError ? (
          <p role="alert" className="text-xs text-app-danger">
            {update.error instanceof Error ? update.error.message : t(strings.errors.unknown)}
          </p>
        ) : null}
        {result && !changed ? (
          <p role="status" className="text-xs text-app-success">
            {t(strings.console.contacts.tierChanged, { tier: t(TIER_LABEL_KEY[result.contact.tier]), rooms: result.affected_rooms.length })}
          </p>
        ) : null}
      </fieldset>

      <Region surfaceId="rooms-region" testId="contacts-rooms-region" state={roomsState} title={t(strings.console.contacts.roomsTitle)} empty={<Quiet title={t(strings.console.contacts.noRooms)} />}>
        <ul data-testid="contacts-rooms" className="divide-y divide-app-border rounded-panel border border-app-border">
          {rooms.map((room) => (
            <li key={room.thread_id} className="flex items-center gap-2 px-3 py-2 text-xs">
              <ChannelChip id={room.channel_id} name={room.channel_display_name} accent={room.channel_accent} />
              <Link to={`/conversations/${room.thread_id}`} className="min-w-0 flex-1 truncate font-mono text-app-foreground hover:text-app-primary">
                {room.thread_key}
              </Link>
              {room.is_group ? <span className="text-app-muted-foreground">{t(strings.console.contacts.people, { count: room.participant_count })}</span> : null}
              <span className="text-app-muted-foreground">{t(strings.console.conversations.roomCeiling)}</span>
              <TierBadge tier={room.ceiling_tier} />
            </li>
          ))}
        </ul>
      </Region>
    </div>
  );
}
