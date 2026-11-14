import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { useState } from "react";
import { Link } from "react-router-dom";

import { CommissionState, lastResearchWrite, useTopics, type TopicSummary } from "../api/activity";
import { Fact } from "../components/Fact";
import { ResearchWriteWhen } from "../components/ResearchWrite";
import { QueryView } from "../components/QueryView";
import { Facts } from "../components/Section";
import { StateBadge } from "../components/StateBadge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { kb, tokens } from "../lib/format";
import { matching, sortByResearchWrite } from "../lib/topics";

/**
 * Every research topic, newest research write first, with a filter: when research last wrote (its source one tap away), its mode, its open
 * work, who is on it, and how much an owner reads to start.
 */
export function TopicsPage() {
  const { t } = useTranslation();
  const query = useTopics();
  const [filter, setFilter] = useState("");
  return (
    <div data-testid={selectors.pages.topics} className="rd-page">
      <PageHeader headingId="topics-heading" title={t(strings.topics.title)} description={t(strings.topics.description)} />
      <QueryView query={query}>
        {({ topics }) => {
          if (topics.length === 0) return <EmptyState title={t(strings.topics.emptyTitle)} description={t(strings.topics.empty)} />;
          const shown = sortByResearchWrite(matching(topics, filter));
          return (
            <>
              <Input type="search" value={filter} onChange={(e) => setFilter(e.target.value)} aria-label={t(strings.topics.filter)} placeholder={t(strings.topics.filter)} />
              {shown.length === 0 ? (
                <p className="text-body-sm">{t(strings.topics.noMatch, { filter })}</p>
              ) : (
                <ul className="rd-rows">
                  {shown.map((topic) => (
                    <TopicRow key={topic.topic} topic={topic} />
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

function TopicRow({ topic }: { topic: TopicSummary }) {
  const { t } = useTranslation();
  const write = lastResearchWrite(topic);
  const check = topic.missionCheck;
  const problems = (check?.errors.length ?? 0) + (check?.warnings.length ?? 0);
  return (
    <li className="rd-row">
      <div className="rd-row-head">
        <Link to={`/topics/${encodeURIComponent(topic.topic)}`} className="rd-row-link min-w-0 flex-1">
          <span className="block font-semibold">{topic.title?.value || topic.topic}</span>
          <span className="rd-id block">{topic.topic}</span>
        </Link>
        <span className="inline-flex flex-wrap gap-space-3xs">
          {topic.liveCommissionIds.length > 0 ? <StateBadge state={CommissionState.LIVE} /> : null}
          {topic.pausedCommissionIds.length > 0 ? <StateBadge state={CommissionState.PAUSED} /> : null}
          {topic.limitedCommissionIds.length > 0 ? <StateBadge state={CommissionState.LIMITED} /> : null}
          {check?.mode ? <StatusBadge tone="neutral">{check.mode}</StatusBadge> : null}
          {problems > 0 ? <StatusBadge tone="warning">{t(strings.topics.problems, { count: problems })}</StatusBadge> : null}
        </span>
      </div>
      <Facts>
        <Fact label={t(strings.topics.lastWrite)} evidence={write?.at?.evidence}>
          <ResearchWriteWhen write={write} />
        </Fact>
      </Facts>
      <p className="text-body-sm">
        {check
          ? t(strings.topics.work, { open: check.open, active: check.active, waiting: check.waitingOperator, landed: check.landed })
          : t(strings.topics.noCheck)}
        {" · "}
        {t(strings.topics.commissions, { count: topic.commissionIds.length })}
        {topic.mission ? (
          <>
            {" · "}
            {t(strings.topics.missionSize, { kb: kb(topic.mission.bytes), tokens: tokens(topic.mission.bytes) })}
          </>
        ) : null}
      </p>
    </li>
  );
}
