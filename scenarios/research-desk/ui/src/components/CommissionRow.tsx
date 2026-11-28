import { Compass, Gauge, Pause, Rocket, TriangleAlert } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import { CommissionState, type CommissionSummary } from "../api/activity";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { toDate } from "../lib/format";
import { pastAllowance } from "../lib/states";
import { displayTitle } from "../lib/titles";
import { IconTile, type Tone } from "./daylight";
import { StateBadge } from "./StateBadge";
import { DerivedWhen, When } from "./When";

const ICON: Record<CommissionState, ReactNode> = {
  [CommissionState.UNSPECIFIED]: <Compass />,
  [CommissionState.LIVE]: <Compass />,
  [CommissionState.PAUSED]: <Pause />,
  [CommissionState.LIMITED]: <Gauge />,
  [CommissionState.HALTED]: <TriangleAlert />,
  [CommissionState.STOPPED]: <Compass />,
  [CommissionState.PREPARED]: <Rocket />,
  [CommissionState.ENDED]: <Compass />,
  [CommissionState.UNKNOWN]: <Compass />,
};

const TONE: Record<CommissionState, Tone> = {
  [CommissionState.UNSPECIFIED]: "neutral",
  [CommissionState.LIVE]: "blue",
  [CommissionState.PAUSED]: "peach",
  [CommissionState.LIMITED]: "peach",
  [CommissionState.HALTED]: "peach",
  [CommissionState.STOPPED]: "neutral",
  [CommissionState.PREPARED]: "lavender",
  [CommissionState.ENDED]: "neutral",
  [CommissionState.UNKNOWN]: "neutral",
};

const DOT: Record<CommissionState, "live" | "wait" | "stop" | "quiet"> = {
  [CommissionState.UNSPECIFIED]: "quiet",
  [CommissionState.LIVE]: "live",
  [CommissionState.PAUSED]: "wait",
  [CommissionState.LIMITED]: "wait",
  [CommissionState.HALTED]: "stop",
  [CommissionState.STOPPED]: "quiet",
  [CommissionState.PREPARED]: "quiet",
  [CommissionState.ENDED]: "quiet",
  [CommissionState.UNKNOWN]: "quiet",
};

/**
 * One commission in a list: a soft icon tile, what it is, its state as a word
 * plus a dot, the newest write and the allowance, then whatever the caller
 * adds below (its changes, or a launch action).
 */
export function CommissionRow({ commission, children }: { commission: CommissionSummary; children?: ReactNode }) {
  const { t } = useTranslation();
  const last = toDate(commission.lastWrite?.value);
  const end = toDate(commission.allowanceEnd?.value);
  const touched = toDate(commission.touchedAfterEnd?.value);
  const fullTitle = commission.title?.value || commission.id;
  const title = displayTitle(fullTitle, commission.id);
  const state = commission.state;
  return (
    <li className="rd-row" data-state={state}>
      <div className="rd-row-line">
        <IconTile size="sm" tone={TONE[state]} state={DOT[state]}>
          {ICON[state]}
        </IconTile>
        <span className="rd-row-copy">
          <Link to={`/commissions/${encodeURIComponent(commission.id)}`} className="rd-row-link rd-row-title" title={fullTitle}>{title}</Link>
          <span className="rd-row-meta">
            {title !== commission.id ? `${commission.id} · ` : ""}
            {commission.topics.map((topic) => topic.topic).join(" · ") || t(strings.commission.topics)}
          </span>
        </span>
        <span className="rd-row-end">
          <StateBadge state={commission.state} stalled={commission.stalled} pastAllowance={pastAllowance(commission)} />
        </span>
      </div>
      <p className="rd-row-meta">
        {last ? <>{t(strings.commission.lastWrite)} <DerivedWhen time={commission.lastWrite} /></> : t(strings.commission.noWrite)}
        {end ? <> · {t(end.getTime() > Date.now() ? strings.commission.allowanceEnds : strings.commission.allowanceEnded)} <When date={end} /></> : null}
        {touched ? <> · {t(strings.commission.touchedAfterEnd)} <When date={touched} /></> : null}
      </p>
      {children ? <div className="rd-row-extra">{children}</div> : null}
    </li>
  );
}
