import { Alert } from "@vrooli/react-component-library/Alert/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Link, useParams } from "react-router-dom";

import { ChangeSubject, lastResearchWrite, useCommissions, useTopics } from "../api/activity";
import { ChangesPanel } from "../components/Changes";
import { CommissionRow } from "../components/CommissionRow";
import { Fact, Unknown } from "../components/Fact";
import { PathText } from "../components/PathText";
import { ResearchWriteWhen } from "../components/ResearchWrite";
import { QueryView } from "../components/QueryView";
import { Facts, Section } from "../components/Section";
import { When } from "../components/When";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { kb, toDate, tokens } from "../lib/format";

/** One topic: its mission's health, its workstreams, who has held it, and what changed. */
export function TopicPage() {
  const { t } = useTranslation();
  const { topic: name = "" } = useParams();
  const query = useTopics();
  const commissions = useCommissions();
  return (
    <div data-testid={selectors.pages.topic} className="rd-page">
      <Link to="/topics" className="rd-back">{t(strings.topics.back)}</Link>
      <QueryView query={query}>
        {({ topics }) => {
          const topic = topics.find((x) => x.topic === name);
          if (!topic) return <Alert tone="danger" title={t(strings.topics.missing, { topic: name })} />;
          const check = topic.missionCheck;
          const write = lastResearchWrite(topic);
          const missionAt = toDate(topic.mission?.modified);
          const review = topic.lastReview;
          const reviewAt = toDate(review?.file?.modified);
          const related = (commissions.data?.commissions ?? []).filter((c) => topic.commissionIds.includes(c.id));
          return (
            <>
              <PageHeader headingId="topic-heading" title={topic.title?.value || topic.topic} eyebrow={<span className="rd-id">{topic.topic}</span>} />
              <Facts>
                <Fact label={t(strings.topics.lastWrite)} evidence={write?.at?.evidence}><ResearchWriteWhen write={write} /></Fact>
                <Fact label={t(strings.topics.mode)} evidence={check?.evidence}>{check?.mode || <Unknown />}</Fact>
                <Fact label={t(strings.topics.mission)}>
                  {topic.mission ? (
                    <span className="flex flex-col">
                      <PathText path={topic.mission.path} />
                      <span className="text-caption text-app-muted-foreground">
                        {t(strings.topics.missionSize, { kb: kb(topic.mission.bytes), tokens: tokens(topic.mission.bytes) })}
                        {missionAt ? (
                          <>
                            {" · "}
                            {t(strings.topics.edited)} <When date={missionAt} />
                          </>
                        ) : null}
                      </span>
                    </span>
                  ) : (
                    <Unknown />
                  )}
                </Fact>
                <Fact label={t(strings.topics.lastReview)}>
                  {review ? (
                    <span className="flex flex-col">
                      <Link to={`/reviews/${encodeURIComponent(review.reviewId)}`} className="rd-id">{review.reviewId}</Link>
                      {reviewAt ? <When date={reviewAt} /> : null}
                    </span>
                  ) : (
                    t(strings.topics.noReview)
                  )}
                </Fact>
              </Facts>
              {check?.errors.map((e) => <Alert key={e} tone="danger" title={e} />)}
              {check?.warnings.map((w) => <Alert key={w} tone="warning" title={w} />)}
              <Section id="workstreams" title={t(strings.topics.workstreams)} note={check ? t(strings.topics.work, { open: check.open, active: check.active, waiting: check.waitingOperator, landed: check.landed }) : undefined}>
                {!check || check.workstreams.length === 0 ? (
                  <p className="text-body-sm">{t(strings.topics.noWorkstreams)}</p>
                ) : (
                  <ul className="rd-rows">
                    {check.workstreams.map((w) => (
                      <li key={w.id} className="rd-row">
                        <span className="flex flex-wrap items-center gap-space-xs">
                          <span className="font-semibold">{w.id}</span>
                          <StatusBadge tone={w.status === "active" ? "success" : w.status === "landed" ? "neutral" : "info"}>{w.status}</StatusBadge>
                        </span>
                        {w.showcase ? <span className="text-body-sm">{w.showcase}</span> : null}
                      </li>
                    ))}
                  </ul>
                )}
              </Section>
              <Section id="topic-commissions" title={t(strings.topics.commissions, { count: topic.commissionIds.length })}>
                {related.length === 0 ? (
                  <p className="text-body-sm">{t(strings.topics.noCommissions)}</p>
                ) : (
                  <ul className="rd-rows">{related.map((c) => <CommissionRow key={c.id} commission={c} />)}</ul>
                )}
              </Section>
              <Section id="topic-changes" title={t(strings.changes.title)}>
                <ChangesPanel subject={ChangeSubject.TOPIC} id={topic.topic} />
              </Section>
            </>
          );
        }}
      </QueryView>
    </div>
  );
}
