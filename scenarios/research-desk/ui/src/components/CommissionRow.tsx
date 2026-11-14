import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import type { CommissionSummary } from "../api/activity";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { toDate } from "../lib/format";
import { pastAllowance } from "../lib/states";
import { StateBadge } from "./StateBadge";
import { DerivedWhen, When } from "./When";

/**
 * One commission in a list: what it is, its state, what it holds, when it
 * last wrote ("at least" when the walk was cut short), and any edit after
 * its end, which is never counted as activity.
 */
export function CommissionRow({ commission, children }: { commission: CommissionSummary; children?: ReactNode }) {
  const { t } = useTranslation();
  const last = toDate(commission.lastWrite?.value);
  const end = toDate(commission.allowanceEnd?.value);
  const touched = toDate(commission.touchedAfterEnd?.value);
  const title = commission.title?.value || commission.id;
  return (
    <li className="rd-row">
      <div className="rd-row-head">
        <Link to={`/commissions/${encodeURIComponent(commission.id)}`} className="rd-row-link min-w-0 flex-1">
          <span className="block font-semibold">{title}</span>
          {title !== commission.id ? <span className="rd-id block">{commission.id}</span> : null}
        </Link>
        <StateBadge state={commission.state} stalled={commission.stalled} pastAllowance={pastAllowance(commission)} />
      </div>
      {commission.topics.length > 0 ? (
        <ul className="rd-chips" aria-label={t(strings.commission.topics)}>
          {commission.topics.map((topic) => (
            <li key={topic.topic}>
              <Link to={`/topics/${encodeURIComponent(topic.topic)}`} className="rd-chip">{topic.topic}</Link>
            </li>
          ))}
        </ul>
      ) : null}
      <p className="text-body-sm text-app-muted-foreground">
        {last ? (
          <>
            {t(strings.commission.lastWrite)} <DerivedWhen time={commission.lastWrite} />
          </>
        ) : (
          t(strings.commission.noWrite)
        )}
        {end ? (
          <>
            {" · "}
            {t(end.getTime() > Date.now() ? strings.commission.allowanceEnds : strings.commission.allowanceEnded)} <When date={end} />
          </>
        ) : null}
        {touched ? (
          <>
            {" · "}
            {t(strings.commission.touchedAfterEnd)} <When date={touched} />
          </>
        ) : null}
      </p>
      {children}
    </li>
  );
}
