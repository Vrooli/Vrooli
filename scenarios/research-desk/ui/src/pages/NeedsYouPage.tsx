import { useQueryClient } from "@tanstack/react-query";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Switch } from "@vrooli/react-component-library/Switch/1";
import type { TFunction } from "i18next";
import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";

import { Delivery, DeliveryReason, operatorApi, useDryRun, useNeedsYou, type CourierStatus, type NeedsYouItem } from "../api/operator";
import { Confirm } from "../components/Confirm";
import { CourierHealth } from "../components/CourierHealth";
import { DeliveryBanner } from "../components/DeliveryBanner";
import { EntryField } from "../components/EntryField";
import { Markdown } from "../components/Markdown";
import { PathText } from "../components/PathText";
import { QueryView } from "../components/QueryView";
import { Section } from "../components/Section";
import { Unreadable } from "../components/Unreadable";
import { When } from "../components/When";
import { WouldWrite } from "../components/WouldWrite";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { entryTooLong } from "../lib/entry";
import { toDate } from "../lib/format";
import { isOlderList, itemKey, keyFromHash, needsYouCount, waiting } from "../lib/needsYou";
import { useDebounced } from "../lib/useDebounced";

/**
 * Needs you: one feed of what supervisors are waiting on. Each item is
 * one collapsed row (title, where it came from, delivery state, the
 * recommended answer); its body opens on tap, and a link from Now opens it
 *. Answer with an offered option or your own words; the answer
 * goes to the commission's OPERATOR.md. A secret is never typed here.
 */
export function NeedsYouPage() {
  const { t } = useTranslation();
  const query = useNeedsYou();
  const target = keyFromHash(useLocation().hash);
  const loaded = query.data !== undefined;
  useEffect(() => {
    if (!loaded || !target) return;
    document.getElementById(target)?.scrollIntoView({ block: "start" });
  }, [loaded, target]);
  return (
    <div data-testid={selectors.pages.needsYou} className="rd-page">
      <PageHeader headingId="needs-you-heading" title={t(strings.needsYou.title)} description={t(strings.needsYou.description)} />
      {query.isError ? <DeliveryBanner error={query.error} /> : null}
      <QueryView query={query}>
        {(feed) => {
          const { items, courier, unreadable } = feed;
          const open = waiting(items);
          const answered = items.filter((item) => item.answer);
          return (
            <>
              <DeliveryBanner courier={courier} open={feed.itemCount} switchAt="#delivery" />
              <Unreadable evidence={unreadable} />
              {open.length === 0 && unreadable.length === 0 ? (
                <EmptyState title={t(strings.needsYou.emptyTitle)} description={t(strings.needsYou.empty)} />
              ) : (
                <>
                  <p className="rd-lede">{needsYouCount(t, feed)}</p>
                  <ul className="rd-rows">
                    {open.map((item) => <Item key={itemKey(item)} item={item} focus={target === itemKey(item)} />)}
                  </ul>
                </>
              )}
              {answered.length > 0 ? (
                <Section id="answered" title={t(strings.needsYou.answered, { count: answered.length })}>
                  <ul className="rd-rows">{answered.map((item) => <Item key={itemKey(item)} item={item} focus={target === itemKey(item)} />)}</ul>
                </Section>
              ) : null}
              {courier && courier.orphans.length > 0 ? (
                <Section id="orphans" title={t(strings.needsYou.orphans, { count: courier.orphans.length })} note={t(strings.needsYou.orphansNote)}>
                  <ul className="rd-list">
                    {courier.orphans.map((o) => (
                      <li key={o.askId || o.key}>
                        <span className="rd-id">{o.key || o.askId}</span> {o.askState ? `· ${o.askState}` : null} {o.why ? `· ${o.why}` : null}
                      </li>
                    ))}
                  </ul>
                </Section>
              ) : null}
              <Section id="delivery" title={t(strings.courier.title)}>
                <CourierHealth />
                {courier ? <GlobalCourier courier={courier} /> : null}
              </Section>
            </>
          );
        }}
      </QueryView>
    </div>
  );
}

/**
 * One row that names it, opening to its body and its answer. An older list
 * is the same row: its text read in place and where it is answered.
 */
function Item({ item, focus }: { item: NeedsYouItem; focus: boolean }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(focus);
  useEffect(() => {
    if (focus) setOpen(true);
  }, [focus]);
  const answeredAt = toDate(item.answer?.at);
  return (
    <li className="rd-row rd-item" id={itemKey(item)}>
      <details open={open} onToggle={(e) => setOpen(e.currentTarget.open)}>
        <summary className="rd-item-summary">
          <span className="font-semibold">{item.title || item.id}</span>
          <span className="rd-item-meta">
            <DeliveryBadge item={item} />
            <span className="rd-id text-app-muted-foreground">{item.commissionId || t(strings.needsYou.global)}</span>
          </span>
          {item.recommended ? (
            <span className="block text-body-sm">
              <span className="font-semibold">{t(strings.needsYou.recommended)}</span> {item.recommended.label || item.recommended.key}
            </span>
          ) : null}
        </summary>
        <div className="rd-item-body">
          <p className="text-body-sm text-app-muted-foreground">
            {item.id ? <span className="rd-id">{item.id}</span> : null}
            {item.id && item.commissionId ? " · " : null}
            {item.commissionId ? (
              <>
                <Link to={`/commissions/${encodeURIComponent(item.commissionId)}?tab=steer`} className="rd-id">{item.commissionId}</Link>
              </>
            ) : null}
          </p>
          <Markdown content={item.body} />
          {!item.recommended && item.recommendedProblem ? <p className="text-body-sm text-app-muted-foreground">{item.recommendedProblem}</p> : null}
          {item.packet ? (
            <p className="text-body-sm">
              {t(strings.needsYou.packet)} <PathText path={item.packet} />
            </p>
          ) : null}
          {item.answer ? (
            <div className="rd-answer">
              <span className="text-body-sm font-semibold">
                {t(strings.needsYou.yourAnswer)} {answeredAt ? <When date={answeredAt} /> : null}
              </span>
              <Markdown content={item.answer.body} />
              <span className="text-body-sm text-app-muted-foreground">
                {item.read?.value ? t(strings.needsYou.read) : item.read?.evidence?.known === false ? item.read.evidence.reason : t(strings.needsYou.notRead)}
              </span>
            </div>
          ) : isReused(item) ? (
            <Alert
              tone="warning"
              title={t(strings.needsYou.reusedTitle)}
              description={t(strings.needsYou.reused, { id: item.id })}
              actions={item.commissionId ? <Link to={`/commissions/${encodeURIComponent(item.commissionId)}?tab=steer`} className="rd-more">{t(strings.needsYou.reusedAction)}</Link> : undefined}
            />
          ) : item.secret ? (
            <Alert tone="info" title={t(strings.needsYou.secretTitle)} description={t(strings.needsYou.secret, { where: item.secret })} />
          ) : item.answerable ? (
            <AnswerForm item={item} />
          ) : (
            <AnsweredElsewhere item={item} />
          )}
          <p className="text-caption text-app-muted-foreground">{deliveryWords(item, t)}</p>
          {item.sources.length > 0 ? (
            <details className="rd-fold">
              <summary>{t(strings.needsYou.sources, { count: item.sources.length })}</summary>
              <ul className="rd-list">{item.sources.map((s) => <li key={s}><PathText path={s} /></li>)}</ul>
            </details>
          ) : null}
        </div>
      </details>
    </li>
  );
}

function deliveryWords(item: NeedsYouItem, t: TFunction): string {
  const parts = [item.deliveryReason, item.courier?.why].filter((part, n, all) => part && all.indexOf(part) === n);
  // The item's own Delivered: line is the supervisor's only when the API says the scheduler sent it;
  // a line naming the desk's own notice is already worded by the reason ("delivered by the desk at …").
  if (item.deliveredByScheduler && item.deliveryReasonCode === DeliveryReason.SCHEDULER_DELIVERED) parts.unshift(t(strings.needsYou.deliveredByScheduler, { line: item.deliveredByScheduler }));
  return parts.join(" · ");
}

/** The courier found this ID already answered or delivered before the item appeared. */
function isReused(item: NeedsYouItem): boolean {
  return item.deliveryReasonCode === DeliveryReason.ID_REUSED;
}

/**
 * An item the desk cannot answer: what the item itself says to do and its
 * links, so the operator knows where it is answered.
 */
function AnsweredElsewhere({ item }: { item: NeedsYouItem }) {
  const { t } = useTranslation();
  return (
    <div className="rd-answer">
      <span className="text-body-sm font-semibold">{t(strings.needsYou.answeredWhere)}</span>
      {item.instruction ? <Markdown content={item.instruction} /> : <p className="text-body-sm">{item.notAnswerableReason || t(strings.common.unknown)}</p>}
      {item.links.length > 0 ? (
        <ul className="rd-list text-body-sm">
          {item.links.map((link) => (
            <li key={`${link.label}|${link.target}`}>
              {/^https?:\/\//.test(link.target) ? (
                <a href={link.target} target="_blank" rel="noreferrer">{link.label || link.target}</a>
              ) : (
                <PathText path={link.target} label={link.label || undefined} />
              )}
            </li>
          ))}
        </ul>
      ) : null}
      {item.instruction && item.notAnswerableReason ? <p className="text-caption text-app-muted-foreground">{item.notAnswerableReason}</p> : null}
    </div>
  );
}

function DeliveryBadge({ item }: { item: NeedsYouItem }) {
  const { t } = useTranslation();
  if (isReused(item)) return <StatusBadge tone="warning">{t(strings.needsYou.badgeReused)}</StatusBadge>;
  if (isOlderList(item)) return <StatusBadge tone="neutral">{t(strings.needsYou.olderBadge)}</StatusBadge>;
  if (item.delivered) return <StatusBadge tone="success">{t(strings.needsYou.badgeDelivered)}</StatusBadge>;
  if (item.courier?.on) return <StatusBadge tone="info">{t(strings.needsYou.badgeQueued)}</StatusBadge>;
  if (item.delivery === Delivery.NONE) return <StatusBadge tone="neutral">{t(strings.needsYou.badgeHere)}</StatusBadge>;
  return <StatusBadge tone="neutral">{t(strings.needsYou.badgeNotSent)}</StatusBadge>;
}

/** Answer with an offered option (plus an optional note) or in your own words. */
function AnswerForm({ item }: { item: NeedsYouItem }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const [choice, setChoice] = useState<string | null>(null);
  const [note, setNote] = useState("");
  const [warning, setWarning] = useState("");
  // The page's own needs-you query: the limit arrives with the list.
  const limit = useNeedsYou().data?.entryCharsMax ?? 0;
  const words = useDebounced(note.trim(), 400);
  const preview = useDryRun(["answer", item.commissionId, item.id, choice, words], () => operatorApi.answer(item.commissionId, item.id, choice ?? "", words, true), choice !== null && !(choice === "other" && !words) && !entryTooLong(words, limit));
  return (
    <div className="flex flex-col gap-space-sm">
      <div className="rd-actions" role="group" aria-label={t(strings.needsYou.options)}>
        {item.options.map((option) => (
          <Button key={option.key} variant={item.recommended?.key === option.key ? "primary" : "secondary"} onClick={() => setChoice(option.key)}>
            {option.label || option.key}
          </Button>
        ))}
        <Button variant="ghost" onClick={() => setChoice("other")}>{t(strings.needsYou.ownWords)}</Button>
      </div>
      {warning ? <Alert tone="warning" title={t(strings.say.writtenWithWarning)} description={warning} /> : null}
      <Confirm
        open={choice !== null}
        title={t(strings.needsYou.confirmTitle, { id: item.id })}
        description={t(strings.needsYou.confirmDescription)}
        confirmLabel={t(strings.needsYou.confirm)}
        onClose={() => {
          setChoice(null);
          setNote("");
        }}
        onConfirm={async () => {
          if (choice === "other" && !note.trim()) throw new Error(t(strings.needsYou.ownWordsRequired));
          if (entryTooLong(note, limit)) throw new Error(t(strings.entry.refused));
          const res = await operatorApi.answer(item.commissionId, item.id, choice ?? "", note.trim());
          setWarning(res.warning);
          await client.invalidateQueries({ queryKey: ["operator", "needs-you"] });
        }}
      >
        <EntryField label={t(choice === "other" ? strings.needsYou.ownWordsLabel : strings.needsYou.noteLabel)} rows={3} value={note} limit={limit} onChange={setNote} />
        {choice === "other" && !words ? null : <WouldWrite label={t(strings.needsYou.appendsTo, { id: item.commissionId })} query={preview} />}
        {item.courier?.askId ? <p className="text-body-sm">{t(strings.needsYou.askStaysOpen)}</p> : null}
      </Confirm>
    </div>
  );
}

/** Global ATTENTION.md items: off until the operator turns them on. */
function GlobalCourier({ courier }: { courier: CourierStatus }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const [next, setNext] = useState<boolean | null>(null);
  return (
    <div id="global-courier" className="flex flex-col gap-space-xs">
      <h3 className="rd-h3">{t(strings.courier.globalTitle)}</h3>
      <p className="text-body-sm text-app-muted-foreground">{t(strings.courier.globalNote)}</p>
      <Switch label={t(strings.courier.globalLabel)} checked={courier.globalOn} disabled={!courier.writer} onCheckedChange={(value) => setNext(value)} />
      <p className="text-caption text-app-muted-foreground">
        {t(strings.courier.perCommission)} <PathText path={courier.switchesFile} />
      </p>
      <Confirm
        open={next !== null}
        title={t(next ? strings.courier.turnOnTitle : strings.courier.turnOffTitle)}
        description={t(next ? strings.courier.globalOn : strings.courier.globalOff)}
        confirmLabel={t(next ? strings.courier.turnOnLabel : strings.courier.turnOffLabel)}
        onClose={() => setNext(null)}
        onConfirm={async () => {
          await operatorApi.setCourier({ global: true }, Boolean(next));
          await client.invalidateQueries({ queryKey: ["operator", "needs-you"] });
        }}
      />
    </div>
  );
}
