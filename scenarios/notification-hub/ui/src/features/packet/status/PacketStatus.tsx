import type { ReactNode } from "react";
import { Link, useLocation } from "react-router-dom";
import { Ban, CheckCircle2, Clock, XCircle } from "lucide-react";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { Facts } from "../../../components/Facts";
import { selectors } from "../../../consts/selectors";
import { Time } from "../../../components/Time";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { optionLabel, remainingUntil } from "../../asks/askView";
import { preserveRoutedTestMode } from "../../routedTestMode";
import { DecisionText } from "../DecisionText";
import { AnswerRecord } from "./AnswerRecord";
import type { TextLinks } from "../decisionLinks";

/**
 * How a closed decision ended: an icon and one line, then the facts. It is a
 * status line, not a box, so it can't be mistaken for a field to fill in.
 */
function StatusLine({ icon, title, children }: { icon: ReactNode; title: ReactNode; children?: ReactNode }) {
  return (
    <section data-testid={selectors.asks.status} role="status" className="flex flex-col gap-2">
      <p className="flex items-start gap-2 font-medium">
        <span aria-hidden="true" className="mt-0.5 shrink-0">{icon}</span>
        <span dir="auto" className="min-w-0 [overflow-wrap:anywhere]">{title}</span>
      </p>
      {children && <div className="flex flex-col gap-2 ps-7 text-sm">{children}</div>}
    </section>
  );
}

const MUTED_ICON = "text-app-muted-foreground";

/** Where a decision stands: answered, defaulted, expired, withdrawn, dismissed, or counting down to its default. */
export function PacketStatus({ ask, now, links }: { ask: Ask; now: number; links: TextLinks }) {
  const { t } = useTranslation();
  if (ask.state === "answered") {
    // Two options may share a label; only then is the requester's key the way to tell them apart.
    const answered = ask.answerLabel || ask.answer;
    const ambiguous = ask.options.filter((option) => option.label === answered).length > 1;
    const facts = [
      ...(ask.answeredBy ? [{ label: t(strings.packet.status.answeredBy), value: ask.answeredBy }] : []),
      ...(ask.answeredAt ? [{ label: t(strings.packet.status.answeredAt), value: <Time value={ask.answeredAt} /> }] : []),
      ...(ask.note ? [{ label: t(strings.packet.status.answerNote), value: <DecisionText value={ask.note} links={links} inheritText /> }] : []),
    ];
    return (
      <StatusLine icon={<CheckCircle2 size={20} className="text-app-success" />} title={t(strings.packet.status.answered, { option: ambiguous ? `${answered} (${ask.answer})` : answered })}>
        {facts.length > 0 && <Facts items={facts} />}
        {ask.late && ask.defaultAnswer && (
          <div className="flex flex-col gap-2">
            <p className="font-medium">{t(strings.packet.status.lateRecorded)}</p>
            <Facts
              className="text-app-muted-foreground"
              items={[
                { label: t(strings.packet.status.recordedDefault), value: optionLabel(ask, ask.defaultAnswer) },
                { label: t(strings.packet.status.defaultApplied), value: <Time value={ask.defaultAppliedAt} /> },
              ]}
            />
            <p className="text-app-muted-foreground">{t(strings.packet.status.lateDoesNotUndo)}</p>
          </div>
        )}
        <AnswerRecord ask={ask} links={links} />
      </StatusLine>
    );
  }
  if (ask.state === "defaulted") {
    return (
      <StatusLine icon={<Clock size={20} className={MUTED_ICON} />} title={t(strings.packet.status.defaulted, { option: optionLabel(ask, ask.defaultAnswer) })}>
        {ask.defaultAppliedAt && (
          <p className="text-app-muted-foreground">
            <Time value={ask.defaultAppliedAt} />
          </p>
        )}
        {ask.reversible && <p className="text-app-muted-foreground">{t(strings.packet.status.lateAvailable)}</p>}
      </StatusLine>
    );
  }
  if (ask.state === "expired") return <StatusLine icon={<Clock size={20} className={MUTED_ICON} />} title={t(strings.packet.status.expired)} />;
  if (ask.state === "withdrawn") return <WithdrawnStatus ask={ask} links={links} />;
  if (ask.state === "dismissed") {
    return <StatusLine icon={<XCircle size={20} className={MUTED_ICON} />} title={t(strings.packet.status.dismissed, { reason: ask.reason })} />;
  }
  if (!ask.defaultAnswer) return null;
  const remaining = remainingUntil(ask.defaultEligibleAt, now);
  const option = optionLabel(ask, ask.defaultAnswer);
  return (
    <p data-testid={selectors.asks.defaultNotice} className="rounded-panel bg-app-surface-muted p-3 text-sm">
      {remaining ? t(strings.packet.status.defaultIn, { option, time: remaining }) : t(strings.packet.status.defaultWaiting, { option })}
    </p>
  );
}

/**
 * A decision its requester closed. The reason is the requester's own words (or
 * the Hub's, for a replaced or lapsed decision) and, when a newer decision took
 * its place, a link to it.
 */
function WithdrawnStatus({ ask, links }: { ask: Ask; links: TextLinks }) {
  const { t } = useTranslation();
  const location = useLocation();
  return (
    <StatusLine icon={<Ban size={20} className={MUTED_ICON} />} title={t(strings.packet.status.withdrawn)}>
      {ask.reason && (
        <p>
          <span className="text-app-muted-foreground">{t(strings.packet.status.closedReason)} </span>
          <DecisionText value={ask.reason} links={links} inheritText />
        </p>
      )}
      {ask.supersededByAskId && (
        <p>
          <Link to={preserveRoutedTestMode(`/asks/${ask.supersededByAskId}`, location.search)} className="inline-flex min-h-11 items-center font-medium underline">
            {t(strings.packet.status.replacedBy)}
          </Link>
        </p>
      )}
    </StatusLine>
  );
}
