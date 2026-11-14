import { useQueryClient } from "@tanstack/react-query";
import { Alert } from "@vrooli/react-component-library/Alert/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { ChangeSubject, CommissionState, lastResearchWrite, useCommissions, useOverview, useTopics, type GetOverviewResponse } from "../api/activity";
import { operatorApi, useNeedsYou } from "../api/operator";
import { ChangesLine } from "../components/Changes";
import { CommissionRow } from "../components/CommissionRow";
import { Confirm } from "../components/Confirm";
import { DeliveryBanner } from "../components/DeliveryBanner";
import { LaunchLink } from "../components/LaunchLink";
import { QueryView } from "../components/QueryView";
import { Section } from "../components/Section";
import { StateBadge } from "../components/StateBadge";
import { Unreadable } from "../components/Unreadable";
import { DerivedWhen, When } from "../components/When";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { toDate } from "../lib/format";
import { itemAnchor, itemKey, needsYouCount, waiting } from "../lib/needsYou";
import { overlapPairs, type OverlapPair } from "../lib/overlaps";
import { PERIODS, periodStart, type Period } from "../lib/period";
import { launchableAsIs, STATE_ORDER, WINDOW_OPEN } from "../lib/states";

/**
 * Now: the first screen says what is moving, in this order. One
 * status line (live, paused and limited, and the newest research write); what
 * needs the operator, at most three titles with the count; then the open
 * supervisors with their topics and what changed in each. Then what changed
 * since the last look or a chosen period, with one "Caught up". Below:
 * unreadable sources, what overlaps now, prepared packets and every
 * commission by state.
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const overview = useOverview();
  const generated = toDate(overview.data?.generatedAt);
  return (
    <div data-testid={selectors.pages.now} className="rd-page">
      <PageHeader
        headingId="now-heading"
        title={t(strings.now.title)}
        description={
          generated ? (
            <>
              {t(strings.now.generated)} <When date={generated} />
            </>
          ) : (
            t(strings.now.description)
          )
        }
        actions={<Button variant="primary" onClick={() => navigate("/launch")}>{t(strings.now.launch)}</Button>}
      />
      <QueryView query={overview}>{(o) => <StatusLine overview={o} />}</QueryView>
      <NeedsYouSummary />
      {overview.data ? <Moving overview={overview.data} /> : null}
    </div>
  );
}

/** The supervisors first, then what is below the first screen. */
function Moving({ overview: o }: { overview: GetOverviewResponse }) {
  const { t } = useTranslation();
  const [period, setPeriod] = useState<Period>("look");
  const since = periodStart(period);
  return (
    <>
      <Supervisors overview={o} since={since} />
      <Since period={period} onPeriod={setPeriod} since={since} />
      {o.lenses
        .filter((lens) => !lens.available)
        .map((lens) => (
          <Alert key={lens.name} tone="warning" title={t(strings.now.lensDown, { name: lens.name })} description={lens.reason || t(strings.now.lensDownHint)} />
        ))}
      <Unreadable evidence={o.unreadable} />
      <Overlaps pairs={overlapPairs(o.openOverlaps)} />
      <Prepared />
      <AllStates counts={o.stateCounts} />
    </>
  );
}

/** Live, paused and limited counts, and the newest research write anywhere. */
function StatusLine({ overview: o }: { overview: GetOverviewResponse }) {
  const { t } = useTranslation();
  const count = (state: CommissionState) => o.stateCounts.find((c) => c.state === state)?.count ?? 0;
  const newest = o.latestResearchWrite;
  return (
    <p className="rd-status" data-testid={selectors.now.status}>
      <span className="rd-status-counts">
        {t(strings.now.openSummary, {
          live: t(strings.now.liveCount, { count: count(CommissionState.LIVE) }),
          paused: t(strings.now.pausedCount, { count: count(CommissionState.PAUSED) }),
          limited: t(strings.now.limitedCount, { count: count(CommissionState.LIMITED) }),
        })}
      </span>
      {newest?.topic ? (
        <span className="rd-status-write">
          {t(strings.now.lastWrite)} <DerivedWhen time={newest.at} />
          {" · "}
          <Link to={`/topics/${encodeURIComponent(newest.topic)}`}>{newest.topic}</Link>
          {newest.note ? <span className="rd-write-note block">{newest.note}</span> : null}
        </span>
      ) : null}
    </p>
  );
}

/**
 * What changed: since the operator's last look (the last 24
 * hours when none is recorded) or a chosen period, for each topic with
 * research written in it, and one "Caught up" that records a look at
 * everything, after a confirmation.
 */
function Since({ period, onPeriod, since }: { period: Period; onPeriod: (p: Period) => void; since?: Date }) {
  const { t } = useTranslation();
  const client = useQueryClient();
  const topics = useTopics();
  const [confirming, setConfirming] = useState(false);
  const [done, setDone] = useState<{ at: Date | null; commissions: number; topics: number } | null>(null);
  // A topic can only have changed if research was written in it since the start; with "last look" the start is per topic, so the last week is read.
  const from = since ?? periodStart("week");
  const written = (topics.data?.topics ?? [])
    .filter((x) => {
      const at = toDate(lastResearchWrite(x)?.at?.value);
      return at !== null && from !== undefined && at > from;
    })
    .slice(0, SINCE_SHOWN);
  return (
    <Section
      id="since"
      title={t(strings.now.since)}
      note={t(period === "look" ? strings.now.sinceLookNote : strings.now.sinceNote)}
      actions={<Button variant="secondary" onClick={() => setConfirming(true)}>{t(strings.now.caughtUp)}</Button>}
    >
      <label className="flex max-w-xs flex-col gap-space-2xs text-body-sm">
        <span className="font-medium">{t(strings.now.period)}</span>
        <Select value={period} onValueChange={(value) => onPeriod(value as Period)} options={PERIODS.map((p) => ({ value: p, label: t(PERIOD_LABEL[p]) }))} />
      </label>
      {done ? (
        <p className="text-body-sm">
          {t(strings.now.caughtUpDone, { commissions: done.commissions, topics: done.topics })} {done.at ? <When date={done.at} /> : null}
        </p>
      ) : null}
      {topics.data && written.length === 0 ? <p className="text-body-sm">{t(strings.now.nothingWritten)}</p> : null}
      <ul className="rd-rows rd-rows-tight">
        {written.map((x) => (
          <li key={x.topic} className="rd-row">
            <Link to={`/topics/${encodeURIComponent(x.topic)}`} className="font-semibold">{x.title?.value || x.topic}</Link>
            <ChangesLine subject={ChangeSubject.TOPIC} id={x.topic} since={since} />
          </li>
        ))}
      </ul>
      <Confirm
        open={confirming}
        title={t(strings.now.caughtUpTitle)}
        description={t(strings.now.caughtUpDescription)}
        confirmLabel={t(strings.now.caughtUp)}
        onClose={() => setConfirming(false)}
        onConfirm={async () => {
          const res = await operatorApi.markAllSeen();
          setDone({ at: toDate(res.at), commissions: res.commissions, topics: res.topics });
          await client.invalidateQueries({ queryKey: ["activity", "changes"] });
        }}
      />
    </Section>
  );
}

/** At most this many topics in "What changed": the most recently written. */
const SINCE_SHOWN = 8;

const PERIOD_LABEL = {
  look: strings.now.periodLook,
  day: strings.now.periodDay,
  week: strings.now.periodWeek,
} as const satisfies Record<Period, string>;

const OPEN_TITLE = {
  [CommissionState.LIVE]: strings.now.liveTitle,
  [CommissionState.LIMITED]: strings.now.limitedTitle,
  [CommissionState.PAUSED]: strings.now.pausedTitle,
} as const;

const OPEN_NOTE = {
  [CommissionState.LIVE]: strings.now.liveNote,
  [CommissionState.LIMITED]: strings.now.limitedNote,
  [CommissionState.PAUSED]: strings.now.pausedNote,
} as const;

/**
 * Supervisors with a window open, split the way the operator needs to read
 * them: live is working; limited and paused are open but idle,
 * and each group says so, so "nothing live, two paused" never reads as an
 * empty page. Each row carries its topics; claim and state rule are on the
 * commission page.
 */
function Supervisors({ overview: o, since }: { overview: GetOverviewResponse; since?: Date }) {
  const { t } = useTranslation();
  return (
    <Section id="running" title={t(strings.now.supervisors)}>
      {o.running.length === 0 ? <p className="text-body-sm">{t(strings.now.nothingOpen)}</p> : null}
      {WINDOW_OPEN.map((state) => {
        const rows = o.running.filter((c) => c.state === state);
        if (rows.length === 0) return null;
        return (
          <div key={state} className="flex flex-col gap-space-xs">
            <h3 className="rd-h3">{t(OPEN_TITLE[state as keyof typeof OPEN_TITLE], { count: rows.length })}</h3>
            <p className="text-body-sm text-app-muted-foreground">{t(OPEN_NOTE[state as keyof typeof OPEN_NOTE])}</p>
            <ul className="rd-rows">
              {rows.map((c) => (
                <CommissionRow key={c.id} commission={c}>
                  <ChangesLine subject={ChangeSubject.COMMISSION} id={c.id} since={since} />
                </CommissionRow>
              ))}
            </ul>
          </div>
        );
      })}
    </Section>
  );
}

/**
 * Open commissions whose time on a topic overlaps, one row per pair with the
 * topics they share. Pairs with an ended or unknown commission are history,
 * shown on that commission's page.
 */
function Overlaps({ pairs }: { pairs: OverlapPair[] }) {
  const { t } = useTranslation();
  if (pairs.length === 0) return null;
  return (
    <Section id="overlaps" title={t(strings.now.overlaps, { count: pairs.length })} note={t(strings.now.overlapsNote)}>
      <ul className="rd-rows">
        {pairs.map((pair) => (
          <li key={pair.key} className="rd-row">
            <ul className="rd-chips" aria-label={t(strings.now.overlapPair)}>
              {pair.ids.map((id) => (
                <li key={id}>
                  <Link to={`/commissions/${encodeURIComponent(id)}`} className="rd-chip rd-id">{id}</Link>
                </li>
              ))}
            </ul>
            <ul className="rd-chips" aria-label={t(strings.now.overlapTopics)}>
              {pair.topics.map((topic) => (
                <li key={topic}>
                  <Link to={`/topics/${encodeURIComponent(topic)}`} className="rd-chip">{topic}</Link>
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
    </Section>
  );
}

/** Every commission by state, each count a link to that list. */
function AllStates({ counts }: { counts: GetOverviewResponse["stateCounts"] }) {
  const { t } = useTranslation();
  if (counts.length === 0) return null;
  return (
    <Section id="states" title={t(strings.now.allStates)}>
      <ul className="rd-stat-row" aria-label={t(strings.now.stateCounts)}>
        {STATE_ORDER.map((state) => {
          const count = counts.find((c) => c.state === state)?.count ?? 0;
          if (count === 0) return null;
          return (
            <li key={state}>
              <Link to={`/commissions?state=${state}`} className="rd-stat">
                <span className="rd-stat-value">{count}</span>
                <StateBadge state={state} />
              </Link>
            </li>
          );
        })}
      </ul>
    </Section>
  );
}

/** What needs the operator: the count and at most three titles, each one tap from its item. */
function NeedsYouSummary() {
  const { t } = useTranslation();
  const query = useNeedsYou();
  if (query.isError) return <DeliveryBanner error={query.error} />;
  if (!query.data) return null;
  const open = waiting(query.data.items);
  return (
    <Section
      id="needs-you"
      title={needsYouCount(t, query.data)}
      actions={open.length > 0 ? <Link to="/needs-you" className="rd-more">{t(strings.now.openFeed)}</Link> : undefined}
    >
      <DeliveryBanner courier={query.data.courier} open={query.data.itemCount} switchAt="/needs-you#delivery" />
      {open.length === 0 ? (
        <p className="text-body-sm">{t(strings.now.nothingNeedsYou)}</p>
      ) : (
        <ul className="rd-rows rd-rows-tight">
          {open.slice(0, 3).map((item) => (
            <li key={itemKey(item)} className="rd-row">
              <Link to={`/needs-you#${itemAnchor(item)}`} className="rd-row-link">
                <span className="block font-semibold">{item.title || item.id}</span>
                <span className="rd-id block">{item.commissionId || t(strings.needsYou.global)}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Section>
  );
}

/** Packets built but never launched: each launches as it is from its own page. */
function Prepared() {
  const { t } = useTranslation();
  const query = useCommissions();
  const prepared = query.data?.commissions.filter(launchableAsIs) ?? [];
  if (prepared.length === 0) return null;
  return (
    <Section id="prepared" title={t(strings.now.prepared, { count: prepared.length })} note={t(strings.now.preparedNote)}>
      <ul className="rd-rows">
        {prepared.map((c) => (
          <CommissionRow key={c.id} commission={c}>
            <LaunchLink id={c.id} />
          </CommissionRow>
        ))}
      </ul>
    </Section>
  );
}
