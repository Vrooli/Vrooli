import { Link } from "react-router-dom";

import { ChangeSubject, MatchKind, type GetCommissionResponse } from "../../api/activity";
import { useLaunchRecord } from "../../api/launch";
import { ChangesPanel } from "../../components/Changes";
import { Fact, Unknown } from "../../components/Fact";
import { PathText } from "../../components/PathText";
import { Facts, Section } from "../../components/Section";
import { DerivedWhen, When } from "../../components/When";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { toDate } from "../../lib/format";

export function StatusPanel({ data }: { data: GetCommissionResponse }) {
  const { t } = useTranslation();
  const c = data.commission;
  if (!c) return null;
  const time = (value: Parameters<typeof toDate>[0]) => {
    const d = toDate(value);
    return d ? <When date={d} /> : <Unknown />;
  };
  return (
    <div className="flex flex-col gap-space-lg">
      <Facts>
        <Fact label={t(strings.commission.state)} evidence={c.stateEvidence}>
          {c.stateRule || <Unknown />}
        </Fact>
        <Fact label={t(strings.commission.topics)} evidence={c.topicsEvidence}>
          {c.topics.length === 0 ? (
            <Unknown />
          ) : (
            <span className="rd-chips">
              {c.topics.map((topic) => (
                <Link key={topic.topic} to={`/topics/${encodeURIComponent(topic.topic)}`} className="rd-chip">{topic.topic}</Link>
              ))}
            </span>
          )}
        </Fact>
        <Fact label={t(strings.commission.started)} evidence={c.started?.evidence}>{time(c.started?.value)}</Fact>
        <Fact label={t(strings.commission.allowanceEnd)} evidence={c.allowanceEnd?.evidence}>{time(c.allowanceEnd?.value)}</Fact>
        <Fact label={t(strings.commission.lastWriteLabel)} evidence={c.lastWrite?.evidence}>
          {c.lastWrite?.value ? <DerivedWhen time={c.lastWrite} /> : <Unknown />}
        </Fact>
        {c.ended?.value ? (
          <Fact label={t(strings.commission.ended)} evidence={c.ended.evidence}>{time(c.ended.value)}</Fact>
        ) : null}
        {c.touchedAfterEnd?.value ? (
          <Fact label={t(strings.commission.touchedAfterEndLabel)} evidence={c.touchedAfterEnd.evidence}>
            {time(c.touchedAfterEnd.value)} {t(strings.commission.touchedNote)}
          </Fact>
        ) : null}
        <Fact label={t(strings.commission.claim)} evidence={c.claim?.evidence}>
          {c.claim?.value ? <q className="rd-claim">{c.claim.value}</q> : <Unknown />}
        </Fact>
        <Fact label={t(strings.commission.launchedBy)}>{c.launchedBy || t(strings.commission.byHand)}</Fact>
        <Fact label={t(strings.commission.folder)}>
          <PathText path={c.folder} />
        </Fact>
      </Facts>
      <Section id="changes" title={t(strings.changes.title)}>
        <ChangesPanel subject={ChangeSubject.COMMISSION} id={c.id} />
      </Section>
      <Sessions data={data} />
      <LaunchHistory id={c.id} />
    </div>
  );
}

/** Which supervisor session this is, and how sure the desk is (SYSTEM.md §3). */
function Sessions({ data }: { data: GetCommissionResponse }) {
  const { t } = useTranslation();
  return (
    <Section id="sessions" title={t(strings.commission.sessions)} note={t(strings.commission.sessionsNote)}>
      {data.matches.length === 0 ? (
        <p className="text-body-sm"><Unknown reason={t(strings.commission.noSession)} /></p>
      ) : (
        <ul className="rd-rows">
          {data.matches.map((m) => {
            const at = toDate(m.at);
            return (
              <li key={`${m.webConsoleSessionId}/${m.basis}/${m.threadId}`} className="rd-row">
                <span className="font-semibold">
                  {t(m.kind === MatchKind.EXACT ? strings.commission.exactMatch : strings.commission.relatedMatch)}
                  {" · "}
                  {m.windowOpen ? t(strings.commission.windowOpen) : t(strings.commission.windowClosed)}
                </span>
                <span className="rd-id">{m.webConsoleSessionId}</span>
                <span className="text-body-sm text-app-muted-foreground">
                  {t(strings.commission.matchDetail, { basis: m.basis, goal: m.goalStatus || t(strings.common.unknown) })}
                  {at ? (
                    <>
                      {" · "}
                      <When date={at} />
                    </>
                  ) : null}
                </span>
                {m.source ? <PathText path={m.source} /> : null}
              </li>
            );
          })}
        </ul>
      )}
    </Section>
  );
}

/** LAUNCH.json as a timeline: steps, controls, a failure, a discard. */
function LaunchHistory({ id }: { id: string }) {
  const { t } = useTranslation();
  const record = useLaunchRecord(id).data?.launch;
  if (!record || (record.steps.length === 0 && record.controls.length === 0 && !record.failedStep)) return null;
  const discarded = toDate(record.discardedAt);
  return (
    <Section id="launch" title={t(strings.commission.launchRecord)} actions={<Link to={`/launch/${encodeURIComponent(id)}`} className="rd-more">{t(strings.commission.openLaunch)}</Link>}>
      <ol className="rd-timeline">
        {record.steps.map((step, n) => {
          const at = toDate(step.at);
          return (
            <li key={`s${n}`}>
              <span>{step.step}</span> {at ? <When date={at} /> : null}
            </li>
          );
        })}
        {record.controls.map((control, n) => {
          const at = toDate(control.at);
          return (
            <li key={`c${n}`}>
              <span>{t(strings.commission.controlEntry, { action: control.action })}</span> {at ? <When date={at} /> : null}
            </li>
          );
        })}
      </ol>
      {record.failedStep ? <p className="text-body-sm text-app-danger">{t(strings.launch.failedAt, { step: record.failedStep, reason: record.failedReason })}</p> : null}
      {discarded ? (
        <p className="text-body-sm">
          {t(strings.commission.discarded, { reason: record.discardedReason })} <When date={discarded} />
        </p>
      ) : null}
      <PathText path={record.path} />
    </Section>
  );
}
