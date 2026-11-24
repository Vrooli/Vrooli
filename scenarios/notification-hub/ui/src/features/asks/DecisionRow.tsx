import { Check, Clock } from "lucide-react";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { ListItem, ListRow } from "../../components/ListRow";
import { Time } from "../../components/Time";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useNow } from "../../hooks/useNow";
import { useTranslation } from "../../i18n";
import { formatDueIn } from "../../lib/ago";
import { preserveRoutedTestMode } from "../routedTestMode";
import { askStateKey, STATE_LABEL, STATE_TONE, type BadgeTone } from "./askState";
import { askHeadline, askPreview, askTone, isAwaiting, optionLabel, senderLabel, urgencyLabel } from "./askView";
import { RequestPacketMeta } from "./RequestPacketMeta";

/**
 * One decision in a list. In the queue everything on the row is about what to
 * do next (due time, escalation, how much there is to read); in History the row
 * leads with how it ended.
 */
export function DecisionRow({ ask, queue, search, from, onOpen }: {
  ask: Ask;
  queue: boolean;
  /** The current location search, so a routed test mode survives the link. */
  search: string;
  /** Where the decision page should send the operator back to. */
  from: string;
  onOpen: () => void;
}) {
  const { t } = useTranslation();
  const now = useNow();
  const awaiting = isAwaiting(ask);
  const stateKey = askStateKey(ask);
  const tone = askTone(ask, now);
  const headline = askHeadline(ask);
  const preview = askPreview(ask);
  const due = awaiting ? formatDueIn(ask.deadline, now) : null;
  const outcomeAt = ask.answeredAt || ask.resolvedAt || ask.defaultAppliedAt || ask.createdAt;
  const answer = ask.state === "answered" ? ask.answerLabel || (ask.answer ? optionLabel(ask, ask.answer) : "") : "";
  const urgency = awaiting && ask.state !== "escalated" ? urgencyLabel(ask.urgency) : null;
  const urgencyTone: BadgeTone = ask.urgency === "critical" ? "danger" : ask.urgency === "high" ? "warning" : "neutral";
  const dueTone: BadgeTone = tone === "critical" ? "danger" : tone === "attention" ? "warning" : "neutral";
  return (
    <ListItem tone={tone} data-testid={queue ? selectors.asks.item : undefined}>
      <ListRow data-request-id={ask.id} to={preserveRoutedTestMode(`/asks/${encodeURIComponent(ask.id)}`, search)} state={{ from }} onClick={onOpen}>
        <p className="line-clamp-2 break-words text-base font-semibold leading-snug">{headline}</p>
        {headline !== ask.question && <span className="sr-only">{ask.question}</span>}
        {preview && <p className="mt-1 hidden line-clamp-2 text-sm text-app-muted-foreground md:block">{preview}</p>}
        <p className="mt-1 flex flex-wrap items-center gap-x-1.5 text-sm text-app-muted-foreground">
          <span className="[overflow-wrap:anywhere]">{senderLabel(ask) || t(strings.pages.history.unknownSource)}</span>
          <span aria-hidden="true">·</span>
          <Time value={outcomeAt} />
        </p>
        <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1.5 empty:hidden">
          {!queue && <StatusBadge tone={STATE_TONE[stateKey]}>{t(STATE_LABEL[stateKey])}</StatusBadge>}
          {queue && ask.state === "escalated" && <StatusBadge tone="danger">{t(strings.asks.row.escalated)}</StatusBadge>}
          {!queue && ask.state === "escalated" && <StatusBadge tone="danger">{t(strings.pages.history.escalated)}</StatusBadge>}
          {urgency && <StatusBadge tone={urgencyTone}>{t(urgency)}</StatusBadge>}
          {ask.decisionKind && <StatusBadge tone="neutral">{t(strings.asks.kind[ask.decisionKind as keyof typeof strings.asks.kind] ?? strings.asks.kind.review)}</StatusBadge>}
          {due && ask.closingSoon && (
            <StatusBadge tone={dueTone === "danger" ? "danger" : "warning"}>
              <span className="inline-flex items-center gap-1"><Clock size={14} aria-hidden="true" />{t(strings.asks.row.closesIn, { when: due })}</span>
            </StatusBadge>
          )}
          {due && !ask.closingSoon && <StatusBadge tone={dueTone}>{t(strings.asks.row.dueIn, { when: due })}</StatusBadge>}
          {answer && (
            <span className="inline-flex min-w-0 items-center gap-1 text-sm text-app-muted-foreground">
              <Check size={14} aria-hidden="true" className="shrink-0" />
              <span className="[overflow-wrap:anywhere]">{t(strings.pages.history.answerSummary, { answer })}</span>
            </span>
          )}
          <RequestPacketMeta ask={ask} />
        </div>
        {ask.state === "defaulted" && ask.reversible && ask.defaultAnswer && <p className="mt-2 text-sm text-app-muted-foreground">{t(strings.pages.history.lateAnswerAvailable)}</p>}
        {ask.state === "expired" && <span className="sr-only">{t(strings.pages.history.expiredNoReopen)}</span>}
      </ListRow>
    </ListItem>
  );
}
