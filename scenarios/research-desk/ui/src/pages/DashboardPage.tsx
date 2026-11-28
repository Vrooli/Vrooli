import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Select } from "@vrooli/react-component-library/Select/1";
import { ArrowUpRight, BookOpenCheck, Compass, Layers, Rocket } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";

import { ChangeSubject, CommissionState, lastResearchWrite, useCommissions, useOverview, useTopics, type CommissionSummary, type GetOverviewResponse, type TopicSummary } from "../api/activity";
import { operatorApi, useNeedsYou } from "../api/operator";
import { ChangesLine } from "../components/Changes";
import { CommissionRow } from "../components/CommissionRow";
import { Confirm } from "../components/Confirm";
import { DeliveryBanner } from "../components/DeliveryBanner";
import { IconTile, MiniBars, Panel, PanelHead, WeekStrip } from "../components/daylight";
import { LaunchLink } from "../components/LaunchLink";
import { QueryView } from "../components/QueryView";
import { Section } from "../components/Section";
import { StateBadge } from "../components/StateBadge";
import { Unreadable } from "../components/Unreadable";
import { DerivedWhen, LastObserved, When } from "../components/When";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { formatDate } from "../i18n/format";
import { toDate } from "../lib/format";
import { deliveryOn, itemAnchor, itemKey, needsYouCount, waiting } from "../lib/needsYou";
import { miniWeek, sameDay, weekDays } from "../lib/week";
import { overlapPairs, type OverlapPair } from "../lib/overlaps";
import { PERIODS, periodStart, type Period } from "../lib/period";
import { launchableAsIs, STATE_ORDER, WINDOW_OPEN } from "../lib/states";

/**
 * Home: a bright day-at-a-glance. The first thing is one blue block holding
 * the single most important question; beside it the open supervisors and the
 * live counts. Then the week, what changed, the topics to watch, and the four
 * actions that start most days. Everything below is the same evidence the
 * desk has always shown, just no longer competing with the question.
 */
export function DashboardPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const overview = useOverview();
  const feed = useNeedsYou();
  const generated = toDate(overview.data?.generatedAt);
  return (
    <div data-testid={selectors.pages.now} className="rd-page">
      <PageHeader
        headingId="now-heading"
        eyebrow={formatDate(new Date(), { weekday: "long", month: "long", day: "numeric" })}
        title={t(strings.now.title)}
        description={overview.data ? <StatusLine overview={overview.data} /> : generated ? <>{t(strings.now.generated)} <When date={generated} /></> : t(strings.now.description)}
        actions={<Button variant="primary" onClick={() => navigate("/launch")}>{t(strings.now.launch)}</Button>}
      />
      {feed.data ? <DeliveryBanner courier={feed.data.courier} open={feed.data.itemCount} switchAt="/needs-you#delivery" /> : null}
      {feed.isError ? <DeliveryBanner error={feed.error} /> : null}
      <LastObserved date={toDate(feed.data?.observedAt)} />
      <div className="rd-bento">
        <NeedsYouHero />
        {overview.data ? <Moving overview={overview.data} /> : <LiveSkeleton />}
        {overview.data ? <OpenStates counts={overview.data.stateCounts} /> : <StatsSkeleton />}
        <WeekCard />
        <Since />
        <TopicsToWatch />
        <QuickActions />
      </div>
      {overview.data ? <Below overview={overview.data} /> : null}
      {!overview.data ? <QueryView query={overview}>{() => null}</QueryView> : null}
    </div>
  );
}

/** Status in one line: what is open, and when research last wrote. */
function StatusLine({ overview: o }: { overview: GetOverviewResponse }) {
  const { t } = useTranslation();
  const count = (state: CommissionState) => o.stateCounts.find((c) => c.state === state)?.count ?? 0;
  const newest = o.latestResearchWrite;
  return (
    <span className="rd-status" data-testid={selectors.now.status}>
      <span className="rd-status-counts">
        {t(strings.now.openSummary, {
          live: t(strings.now.liveCount, { count: count(CommissionState.LIVE) }),
          halted: t(strings.now.haltedCount, { count: count(CommissionState.HALTED) }),
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
    </span>
  );
}

function isToday(date: Date | null): boolean {
  if (!date) return false;
  const now = new Date();
  return date.getFullYear() === now.getFullYear() && date.getMonth() === now.getMonth() && date.getDate() === now.getDate();
}

/** The one blue block: how many decisions wait, the top one, and one way in. */
function NeedsYouHero() {
  const { t } = useTranslation();
  const query = useNeedsYou();
  const feed = query.data;
  if (!feed) {
    return (
      <div className="rd-hero" data-testid="home-hero" aria-busy="true">
        <p className="rd-hero-label">{t(strings.now.needsYouLabel)}</p>
        <div className="rd-skeleton"><i /><i /><i /></div>
      </div>
    );
  }
  const open = waiting(feed.items);
  const top = open[0];
  const older = feed.olderListCount;
  const deliveredToday = open.filter((item) => isToday(toDate(item.delivered?.at))).length;
  const answeredToday = feed.items.filter((item) => isToday(toDate(item.answer?.at))).length;
  return (
    <section className="rd-hero" data-testid="home-hero" aria-labelledby="needs-you-title">
      <h2 className="rd-visually-hidden" id="needs-you-title">{needsYouCount(t, feed)}</h2>
      <p className="rd-hero-label">{t(strings.now.needsYouLabel)}</p>
      <div className="rd-hero-metric">
        <b>{feed.itemCount}</b>
        <span>{t(strings.now.waiting)}</span>
      </div>
      {deliveredToday > 0 ? <span className="rd-hero-calendar">+{deliveredToday} {t(strings.now.deliveredToday, { count: deliveredToday })}</span> : null}
      {top ? (
        <Link className="rd-hero-question" to={`/needs-you#${itemAnchor(top)}`}>
          <b>{top.title || top.id}</b>
          <span>
            {top.recommended
              ? `${t(strings.needsYou.recommended)} ${top.recommended.label || top.recommended.key}`
              : top.recommendedProblem || top.commissionId || t(strings.needsYou.global)}
          </span>
        </Link>
      ) : (
        <div className="rd-hero-question">
          <b>{t(strings.now.nothingWaiting)}</b>
          <span>{t(strings.now.nothingNeedsYou)}</span>
        </div>
      )}
      {open.length > 1 ? (
        <ul className="rd-hero-upnext" aria-label={t(strings.needsYou.upNext)}>
          {open.slice(1, 3).map((item) => (
            <li key={itemKey(item)}>
              <Link to={`/needs-you#${itemAnchor(item)}`}>{item.title || item.id}</Link>
            </li>
          ))}
        </ul>
      ) : null}
      <div className="rd-hero-actions">
        {top ? (
          <Link className="rd-pill-white" to={`/needs-you#${itemAnchor(top)}`}>
            {t(strings.now.answer)} <ArrowUpRight aria-hidden="true" size={15} />
          </Link>
        ) : (
          <Link className="rd-pill-white" to="/needs-you">{t(strings.needsYou.openFullList)}</Link>
        )}
        {older > 0 ? <Link className="rd-quiet" to="/needs-you">{t(strings.now.olderLists, { count: older })}</Link> : null}
      </div>
      <div className="rd-hero-foot">
        <span>{feed.courier && deliveryOn(feed.courier) ? t(strings.delivery.on) : t(strings.delivery.off)}</span>
        <span>{answeredToday > 0 ? `${answeredToday} ${t(strings.needsYou.answeredToday).toLowerCase()}` : t(strings.now.noQuestions)}</span>
      </div>
    </section>
  );
}

const OPEN_TITLE = {
  [CommissionState.LIVE]: strings.now.liveTitle,
  [CommissionState.HALTED]: strings.now.haltedTitle,
  [CommissionState.LIMITED]: strings.now.limitedTitle,
  [CommissionState.PAUSED]: strings.now.pausedTitle,
} as const;

const OPEN_NOTE = {
  [CommissionState.LIVE]: strings.now.liveNote,
  [CommissionState.HALTED]: strings.now.haltedNote,
  [CommissionState.LIMITED]: strings.now.limitedNote,
  [CommissionState.PAUSED]: strings.now.pausedNote,
} as const;

/**
 * Open supervisors, grouped by state, in the middle column. Three rows are the
 * day's glance; the rest stay one tap away on Runs.
 */
function Moving({ overview: o }: { overview: GetOverviewResponse }) {
  const { t } = useTranslation();
  const rows = WINDOW_OPEN.flatMap((state) => o.running.filter((c) => c.state === state));
  const shown = rows.slice(0, 3);
  return (
    <section className="rd-bento-live" aria-labelledby="live-now-title">
      <Panel>
        <PanelHead
          id="live-now-title"
          title={t(strings.now.liveNow)}
          action={<Link className="rd-card-link" to="/commissions">{t(strings.now.seeAll)}</Link>}
        />
        {o.running.length === 0 ? <p className="rd-card-note">{t(strings.now.nothingOpen)}</p> : null}
        {WINDOW_OPEN.map((state) => {
          const list = shown.filter((c) => c.state === state);
          if (list.length === 0) return null;
          return (
            <div key={state} className="flex flex-col gap-space-3xs">
              <h3 className="rd-h3">{t(OPEN_TITLE[state as keyof typeof OPEN_TITLE], { count: o.running.filter((c) => c.state === state).length })}</h3>
              <p className="rd-visually-hidden">{t(OPEN_NOTE[state as keyof typeof OPEN_NOTE])}</p>
              <ul className="rd-rows rd-rows-tight">
                {list.map((c) => <CommissionRow key={c.id} commission={c} />)}
              </ul>
            </div>
          );
        })}
        {rows[0] ? (
          <ChangesLine subject={ChangeSubject.COMMISSION} id={rows[0].id} />
        ) : null}
        {rows.length > shown.length ? (
          <Link className="rd-more" to="/commissions">{t(strings.now.seeAll)}</Link>
        ) : null}
      </Panel>
    </section>
  );
}

/** The bento keeps its shape while the first read lands. */
function LiveSkeleton() {
  const { t } = useTranslation();
  return (
    <section className="rd-bento-live">
      <Panel>
        <PanelHead title={t(strings.now.liveNow)} />
        <div className="rd-skeleton"><i /><i /><i /></div>
      </Panel>
    </section>
  );
}

function StatsSkeleton() {
  return (
    <div className="rd-bento-stats" aria-hidden="true">
      {[0, 1, 2, 3].map((n) => (
        <span className="rd-stat rd-stat--zero" key={n}>
          <span className="rd-stat-value">—</span>
          <span className="rd-stat-label">—</span>
        </span>
      ))}
    </div>
  );
}

/** The states with an open window, as quiet white pills: the at-a-glance count. */
function OpenStates({ counts }: { counts: GetOverviewResponse["stateCounts"] }) {
  const { t } = useTranslation();
  return (
    <ul className="rd-stat-row rd-bento-stats" aria-label={t(strings.now.stateCounts)}>
      {WINDOW_OPEN.map((state) => {
        const count = counts.find((c) => c.state === state)?.count ?? 0;
        const body = (
          <>
            <span className="rd-stat-value">{count}</span>
            <StateBadge state={state} />
          </>
        );
        return (
          <li key={state}>
            {count > 0 ? (
              <Link to={`/commissions?state=${state}`} className="rd-stat">{body}</Link>
            ) : (
              <span className="rd-stat rd-stat--zero">{body}</span>
            )}
          </li>
        );
      })}
    </ul>
  );
}

/** Every state present, each count a link to that list. */
function AllStates({ counts }: { counts: GetOverviewResponse["stateCounts"] }) {
  const { t } = useTranslation();
  if (counts.length === 0) return null;
  return (
    <Section id="states" title={t(strings.now.allStates)}>
      <ul className="rd-stat-row" aria-label={t(strings.now.allStates)}>
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

/** This week as seven real dates and a rounded bar per day. */
function WeekCard() {
  const { t } = useTranslation();
  const topics = useTopics();
  const writes = (topics.data?.topics ?? [])
    .map((topic) => toDate(lastResearchWrite(topic)?.at?.value))
    .filter((date): date is Date => date !== null);
  const days = weekDays(writes);
  const total = writes.filter((date) => days.some((day) => sameDay(date, day.at))).length;
  return (
    <section className="rd-bento-week">
      <Panel>
        <PanelHead
          title={t(strings.now.thisWeek)}
          note={t(strings.now.thisWeekNote)}
          action={<span className="rd-pill rd-pill--quiet">{t(strings.now.thisWeekTopics, { count: total })}</span>}
        />
        <WeekStrip days={days.map((day) => ({ key: day.key, letter: day.letter, date: day.date, count: day.count, today: day.today, off: day.off }))} total={total} />
      </Panel>
    </section>
  );
}

/**
 * What changed since the operator's last look (the last 24 hours when none is
 * recorded) or a chosen period, for each topic with research written in it,
 * and one "Caught up" that records a look at everything, after a confirmation.
 */
function Since() {
  const { t } = useTranslation();
  const client = useQueryClient();
  const topics = useTopics();
  const [period, setPeriod] = useState<Period>("look");
  const [confirming, setConfirming] = useState(false);
  const [done, setDone] = useState<{ at: Date | null; commissions: number; topics: number } | null>(null);
  const since = periodStart(period);
  const from = since ?? periodStart("week");
  const written = (topics.data?.topics ?? [])
    .filter((x) => {
      const at = toDate(lastResearchWrite(x)?.at?.value);
      return at !== null && from !== undefined && at > from;
    })
    .slice(0, SINCE_SHOWN);
  return (
    <div className="rd-bento-since">
      <Section
      id="since"
      title={t(strings.now.since)}
      note={t(period === "look" ? strings.now.sinceLookNote : strings.now.sinceNote)}
      actions={<Button variant="primary" onClick={() => setConfirming(true)}>{t(strings.now.caughtUp)}</Button>}
    >
      <label className="flex max-w-xs flex-col gap-space-2xs text-body-sm">
        <span className="font-medium">{t(strings.now.period)}</span>
        <Select value={period} onValueChange={(value) => setPeriod(value as Period)} options={PERIODS.map((p) => ({ value: p, label: t(PERIOD_LABEL[p]) }))} />
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
    </div>
  );
}

const SINCE_SHOWN = 6;

const PERIOD_LABEL = {
  look: strings.now.periodLook,
  day: strings.now.periodDay,
  week: strings.now.periodWeek,
} as const satisfies Record<Period, string>;

/** Three topics the operator is most likely to look at, newest write first. */
function TopicsToWatch() {
  const { t } = useTranslation();
  const topics = useTopics();
  const items = [...(topics.data?.topics ?? [])]
    .sort((a, b) => Number(lastResearchWrite(b)?.at?.value?.seconds ?? 0n) - Number(lastResearchWrite(a)?.at?.value?.seconds ?? 0n))
    .slice(0, 3);
  if (items.length === 0) {
    return (
      <section className="rd-bento-topics">
        <Panel>
          <PanelHead title={t(strings.now.topicsToWatch)} />
          <p className="rd-card-note">{t(strings.topics.empty)}</p>
        </Panel>
      </section>
    );
  }
  return (
    <section className="rd-bento-topics">
      <Panel>
        <PanelHead title={t(strings.now.topicsToWatch)} action={<Link className="rd-card-link" to="/topics">{t(strings.now.seeAll)}</Link>} />
        <ul className="rd-rows rd-rows-tight">
          {items.map((topic, index) => (
            <li key={topic.topic} className="rd-row">
              <Link to={`/topics/${encodeURIComponent(topic.topic)}`} className="rd-row-line">
                <IconTile size="sm" tone={TOPIC_TONES[index % TOPIC_TONES.length]}>
                  <Compass />
                </IconTile>
                <span className="rd-row-copy">
                  <span className="rd-row-title">{topic.title?.value || topic.topic}</span>
                  <span className="rd-row-meta">
                    {topic.missionCheck ? t(strings.topics.openRuns, { open: topic.missionCheck.open, count: topic.commissionIds.length }) : t(strings.topics.commissions, { count: topic.commissionIds.length })}
                  </span>
                </span>
                <MiniBars values={miniWeek(writtenDates(topic))} />
              </Link>
            </li>
          ))}
        </ul>
      </Panel>
    </section>
  );
}

const TOPIC_TONES = ["lavender", "blue", "mint"] as const;

function writtenDates(topic: TopicSummary): Date[] {
  const at = toDate(lastResearchWrite(topic)?.at?.value);
  return at ? [at] : [];
}

/** The four things a morning usually needs, one tap away. */
function QuickActions() {
  const { t } = useTranslation();
  return (
    <div className="rd-bento-quick">
      <nav className="rd-quick" aria-label={t(strings.now.quickActions)}>
        <Link to="/launch"><Rocket aria-hidden="true" />{t(strings.layout.nav.launch)}</Link>
        <Link to="/topics"><Layers aria-hidden="true" />{t(strings.layout.nav.topics)}</Link>
        <Link to="/commissions"><Compass aria-hidden="true" />{t(strings.layout.nav.commissions)}</Link>
        <Link to="/reviews"><BookOpenCheck aria-hidden="true" />{t(strings.layout.nav.reviews)}</Link>
      </nav>
    </div>
  );
}

/** Everything below the first screen, in the same white-card grammar. */
function Below({ overview: o }: { overview: GetOverviewResponse }) {
  const { t } = useTranslation();
  return (
    <>
      {o.lenses.filter((lens) => !lens.available).map((lens) => (
        <p key={lens.name} className="rd-warn-line" role="status">
          <span>{t(strings.now.lensDown, { name: lens.name })}</span>
          <span className="text-app-muted-foreground">{lens.reason || t(strings.now.lensDownHint)}</span>
        </p>
      ))}
      <Unreadable evidence={o.unreadable} />
      <Overlaps pairs={overlapPairs(o.openOverlaps)} />
      <Prepared />
      <AllStates counts={o.stateCounts} />
    </>
  );
}

/** Open commissions whose time on a topic overlaps, one row per pair. */
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
                <li key={id}><Link to={`/commissions/${encodeURIComponent(id)}`} className="rd-chip rd-id">{id}</Link></li>
              ))}
            </ul>
            <ul className="rd-chips" aria-label={t(strings.now.overlapTopics)}>
              {pair.topics.map((topic) => (
                <li key={topic}><Link to={`/topics/${encodeURIComponent(topic)}`} className="rd-chip">{topic}</Link></li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
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
        {prepared.map((c: CommissionSummary) => (
          <CommissionRow key={c.id} commission={c}>
            <LaunchLink id={c.id} />
          </CommissionRow>
        ))}
      </ul>
    </Section>
  );
}
