import { Alert } from "@vrooli/react-component-library/Alert/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { ArrowUpRight, FlaskConical } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import { ChangeSubject, lastResearchWrite, useCommissions, useTopics } from "../api/activity";
import { homeFileUrl } from "../api/client";
import { ChangesPanel } from "../components/Changes";
import { CommissionRow } from "../components/CommissionRow";
import { Fact, Unknown } from "../components/Fact";
import { PathText } from "../components/PathText";
import { ResearchWriteWhen } from "../components/ResearchWrite";
import { QueryView } from "../components/QueryView";
import { Facts, Section } from "../components/Section";
import { LastObserved, When } from "../components/When";
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
      <LastObserved date={toDate(query.data?.observedAt)} />
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
          const workspace = topic.workspaceFiles;
          const currentUrl = workspace?.currentView?.file ? homeFileUrl(workspace.currentView.file.path, query.data?.researchHome) : null;
          const toolsUrl = workspace?.tools?.file ? homeFileUrl(workspace.tools.file.path, query.data?.researchHome) : null;
          return (
            <>
              <PageHeader headingId="topic-heading" title={topic.title?.value || topic.topic} eyebrow={<span className="rd-id">{topic.topic}</span>} />
              <div className="rd-topic-actions">
                <div>
                  <p className="rd-topic-actions-title">{t(strings.topics.researchActions)}</p>
                  <p>{t(strings.topics.researchActionsNote)}</p>
                </div>
                {!check?.mode ? null : (
                  <Link className="rd-topic-commission" to={`/launch?topic=${encodeURIComponent(topic.topic)}`}>
                    <FlaskConical aria-hidden="true" size={18} />
                    {t(strings.topics.commissionTopic)}
                  </Link>
                )}
                <a className="rd-more" href="#topic-commissions">
                  {t(strings.topics.viewCommissions)} <ArrowUpRight aria-hidden="true" size={14} />
                </a>
              </div>
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
              <Section id="topic-workspace-files" title={t(strings.topics.workspaceFiles)} note={workspace?.feedbackTruncated ? t(strings.topics.feedbackPartial) : undefined}>
                <ul className="rd-rows">
                  <li className="rd-row">
                    <span className="font-semibold">{t(strings.topics.currentView)}</span>
                    {workspace?.currentView?.file ? (
                      currentUrl ? <a href={currentUrl}>{workspace.currentView.file.name}</a> : <PathText path={workspace.currentView.file.path} />
                    ) : (
                      <span className="text-body-sm text-app-muted-foreground">
                        {workspace?.currentView?.evidence?.known ? t(strings.topics.noOptionalFile) : workspace?.currentView?.evidence?.reason || t(strings.common.unknown)}
                      </span>
                    )}
                  </li>
                  <li className="rd-row">
                    <span className="font-semibold">{t(strings.topics.tools)}</span>
                    {workspace?.tools?.file ? (
                      toolsUrl ? <a href={toolsUrl}>{workspace.tools.file.name}</a> : <PathText path={workspace.tools.file.path} />
                    ) : (
                      <span className="text-body-sm text-app-muted-foreground">
                        {workspace?.tools?.evidence?.known ? t(strings.topics.noOptionalFile) : workspace?.tools?.evidence?.reason || t(strings.common.unknown)}
                      </span>
                    )}
                  </li>
                  <li className="rd-row">
                    <span className="font-semibold">{t(strings.topics.feedback)}</span>
                    {workspace?.feedbackFiles.length ? (
                      <ul className="rd-list">
                        {workspace.feedbackFiles.map((file) => {
                          const url = homeFileUrl(file.path, query.data?.researchHome);
                          return <li key={file.path}>{url ? <a href={url}>{file.name}</a> : <PathText path={file.path} />}</li>;
                        })}
                      </ul>
                    ) : (
                      <span className="text-body-sm text-app-muted-foreground">
                        {workspace?.feedbackEvidence?.known ? t(strings.topics.noFeedback) : workspace?.feedbackEvidence?.reason || t(strings.common.unknown)}
                      </span>
                    )}
                  </li>
                </ul>
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
