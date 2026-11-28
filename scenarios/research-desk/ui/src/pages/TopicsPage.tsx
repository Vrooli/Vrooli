import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { BookOpen, Compass, FlaskConical, Globe, Search } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";

import { CommissionState, lastResearchWrite, useTopics, type TopicSummary } from "../api/activity";
import { Fact } from "../components/Fact";
import { IconTile, MiniBars, StatusPill } from "../components/daylight";
import { ResearchWriteWhen } from "../components/ResearchWrite";
import { LastObserved } from "../components/When";
import { QueryView } from "../components/QueryView";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { toDate } from "../lib/format";
import { matching, sortByResearchWrite } from "../lib/topics";
import { miniWeek } from "../lib/week";
import { STATE_META } from "../lib/states";

type View = "all" | "attention" | "quiet";

const ICONS = [<Compass />, <FlaskConical />, <BookOpen />, <Globe />];
const TONES = ["lavender", "blue", "mint", "peach"] as const;

/**
 * Every research topic, newest research write first, with a search and three
 * filters. The grid is the fashion reference applied to research: pastel icon
 * tiles, a title, the work it holds, a seven-day pulse, and the write's source
 * one tap away.
 */
export function TopicsPage() {
  const { t } = useTranslation();
  const query = useTopics();
  const [filter, setFilter] = useState("");
  const [view, setView] = useState<View>("all");
  return (
    <div data-testid={selectors.pages.topics} className="rd-page">
      <PageHeader headingId="topics-heading" title={t(strings.topics.title)} description={t(strings.topics.description)} />
      <LastObserved date={toDate(query.data?.observedAt)} />
      <QueryView query={query}>
        {({ topics }) => {
          if (topics.length === 0) return <EmptyState title={t(strings.topics.emptyTitle)} description={t(strings.topics.empty)} />;
          const searched = sortByResearchWrite(matching(topics, filter));
          const shown = searched.filter((topic) => {
            if (view === "attention") return needsAttention(topic);
            if (view === "quiet") return !needsAttention(topic);
            return true;
          });
          const attention = topics.filter(needsAttention).length;
          return (
            <>
              <div className="rd-tools">
                <label className="rd-search">
                  <Search aria-hidden="true" />
                  <input
                    type="search"
                    value={filter}
                    onChange={(event) => setFilter(event.target.value)}
                    aria-label={t(strings.topics.filter)}
                    placeholder={t(strings.topics.search)}
                  />
                </label>
                <div className="rd-toolbar" role="group" aria-label={t(strings.topics.filter)}>
                  <FilterPill active={view === "all"} onClick={() => setView("all")}>{t(strings.topics.all)}</FilterPill>
                  <FilterPill active={view === "attention"} onClick={() => setView("attention")}>{t(strings.topics.needsAttention)}</FilterPill>
                  <FilterPill active={view === "quiet"} onClick={() => setView("quiet")}>{t(strings.topics.quiet)}</FilterPill>
                </div>
              </div>
              {attention > 0 && view !== "quiet" ? (
                <div className="rd-feature">
                  <small>{t(strings.topics.needsAttention)}</small>
                  <b>{t(strings.topics.attentionCount, { count: attention })}</b>
                  <span>{t(strings.topics.attentionNote)}</span>
                  <button type="button" className="rd-pill rd-pill--ink rd-feature-action" onClick={() => setView("attention")}>
                    {t(strings.topics.reviewAttention)}
                  </button>
                </div>
              ) : null}
              {shown.length === 0 ? (
                <p className="text-body-sm">{t(filter ? strings.topics.noMatch : strings.topics.noAttention, { filter })}</p>
              ) : (
                <ul className="rd-topic-grid">
                  {shown.map((topic, index) => (
                    <TopicCard key={topic.topic} topic={topic} icon={ICONS[index % ICONS.length]} tone={TONES[index % TONES.length] ?? "blue"} />
                  ))}
                </ul>
              )}
            </>
          );
        }}
      </QueryView>
    </div>
  );
}

function FilterPill({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button type="button" className={`rd-filter${active ? " rd-filter--on" : ""}`} aria-pressed={active} onClick={onClick}>
      {children}
    </button>
  );
}

function dominantState(topic: TopicSummary): CommissionState | null {
  if (topic.liveCommissionIds.length > 0) return CommissionState.LIVE;
  if (topic.haltedCommissionIds.length > 0) return CommissionState.HALTED;
  if (topic.limitedCommissionIds.length > 0) return CommissionState.LIMITED;
  if (topic.pausedCommissionIds.length > 0) return CommissionState.PAUSED;
  return null;
}

function needsAttention(topic: TopicSummary): boolean {
  return dominantState(topic) !== null || (topic.missionCheck?.errors.length ?? 0) > 0 || (topic.missionCheck?.warnings.length ?? 0) > 0 || (topic.missionCheck?.waitingOperator ?? 0) > 0;
}

function TopicCard({ topic, icon, tone }: { topic: TopicSummary; icon: React.ReactNode; tone: (typeof TONES)[number] }) {
  const { t } = useTranslation();
  const write = lastResearchWrite(topic);
  const check = topic.missionCheck;
  const problems = (check?.errors.length ?? 0) + (check?.warnings.length ?? 0);
  const state = dominantState(topic);
  const written = toDate(write?.at?.value);
  return (
    <li className="rd-topic-card">
      <Link to={`/topics/${encodeURIComponent(topic.topic)}`} className="rd-topic-card-main">
        <span className="rd-topic-top">
          <IconTile tone={tone} state={state === CommissionState.LIVE ? "live" : state === CommissionState.HALTED ? "stop" : state ? "wait" : "quiet"}>{icon}</IconTile>
          {state ? (
            <StatusPill
              tone={state === CommissionState.LIVE ? "mint" : "peach"}
              dot={state === CommissionState.LIVE ? "live" : "wait"}
            >
              {t(STATE_META[state].label)}
            </StatusPill>
          ) : (
            <StatusPill tone="quiet">{t(strings.topics.quiet)}</StatusPill>
          )}
        </span>
        <span className="rd-topic-title">{topic.title?.value || topic.topic}</span>
        <span className="rd-topic-meta">
          {check ? t(strings.topics.openRuns, { open: check.open, count: topic.commissionIds.length }) : t(strings.topics.noCheck)}
        </span>
        {check?.mode || problems > 0 ? (
          <span className="rd-topic-pills">
            {check?.mode ? <StatusPill tone="quiet">{check.mode}</StatusPill> : null}
            {problems > 0 ? <StatusPill tone="peach">{t(strings.topics.problems, { count: problems })}</StatusPill> : null}
          </span>
        ) : null}
        <span className="rd-topic-meta">{topic.topic}</span>
        <MiniBars values={miniWeek(written ? [written] : [])} />
      </Link>
      <Fact label={t(strings.topics.lastWrite)} evidence={write?.at?.evidence}>
        <ResearchWriteWhen write={write} />
      </Fact>
    </li>
  );
}
