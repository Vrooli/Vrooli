import { Clock, RefreshCcw } from "lucide-react";
import type { Ask, AskContextRequest } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { selectors } from "../../../consts/selectors";
import { Time } from "../../../components/Time";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { DecisionText } from "../DecisionText";
import type { TextLinks } from "../decisionLinks";

/** A request the requester has not yet answered. */
const isWaiting = (request: AskContextRequest) => !request.addressedAt;

/**
 * The questions one reply covered. A reply answers every question that was open
 * when it was sent, so those questions share one `response` (possibly empty,
 * when the requester revised the decision without writing back) and one time.
 */
interface Round {
  requests: AskContextRequest[];
  response: string;
}

function roundsOf(requests: readonly AskContextRequest[]): Round[] {
  const rounds = new Map<string, Round>();
  for (const request of requests) {
    const key = `${request.addressedAt}\u0000${request.response}`;
    const round = rounds.get(key);
    if (round) round.requests.push(request);
    else rounds.set(key, { requests: [request], response: request.response });
  }
  return [...rounds.values()];
}

function RequestNote({ request, label }: { request: AskContextRequest; label?: string }) {
  const { t } = useTranslation();
  return (
    <figure className="mt-1 text-sm text-app-muted-foreground">
      {label && <figcaption className="text-xs font-semibold">{label}</figcaption>}
      <blockquote dir="auto" className="line-clamp-4 break-words border-s-2 border-app-border ps-2 italic">{request.question}</blockquote>
      <figcaption className="mt-1 text-xs">
        {t(strings.packet.context.askedAt)} <Time value={request.requestedAt} />
      </figcaption>
    </figure>
  );
}

/** One reply: the questions it covered, then the requester's answer in full, line breaks kept. */
function RoundNote({ round, links }: { round: Round; links: TextLinks }) {
  const { t } = useTranslation();
  const label = round.response ? t(strings.packet.context.yourQuestion) : undefined;
  return (
    <div className="mt-2 flex flex-col gap-2">
      {round.requests.map((request) => <RequestNote key={request.id} request={request} label={label} />)}
      {round.response && (
        <div className="flex flex-col gap-1">
          <p className="text-xs font-semibold text-app-muted-foreground">{t(strings.packet.context.answer)}</p>
          <DecisionText value={round.response} links={links} className="block text-base text-app-foreground" />
        </div>
      )}
    </div>
  );
}

/**
 * The operator's questions to the requester. While one is unanswered the page
 * says so; once the requester has replied it shows the reply, and when the
 * decision itself changed it points at what changed.
 *
 * The latest reply is the one answered at the newest time. Everything answered
 * before it is shown folded. (A revision count would not do: an answer that
 * changes no term leaves it unchanged, and a replaced decision carries earlier
 * rounds forward with none.)
 */
export function ContextRequests({ ask, links, onSeeChanges }: { ask: Ask; links: TextLinks; onSeeChanges: () => void }) {
  const { t } = useTranslation();
  if (ask.contextRequests.length === 0) return null;
  const waiting = ask.contextRequests.filter(isWaiting);
  const answered = ask.contextRequests.filter((request) => !isWaiting(request));
  const newest = Math.max(...answered.map((request) => Date.parse(request.addressedAt)));
  const recent = answered.filter((request) => Date.parse(request.addressedAt) === newest);
  const earlier = answered.filter((request) => Date.parse(request.addressedAt) !== newest);
  // An answer alone makes no revision: offer the changes only when the decision was revised after a question was asked.
  const revised = recent.some((request) => ask.revisions.some((revision) => Date.parse(revision.revisedAt) >= Date.parse(request.requestedAt)));
  return (
    <div data-testid={selectors.packet.contextRequests} className="flex flex-col gap-2">
      {waiting.length > 0 && (
        <section role="status" className="rounded-panel border border-app-warning/60 bg-app-warning/15 p-3">
          <h2 className="inline-flex items-center gap-2 text-sm font-semibold">
            <Clock size={16} aria-hidden="true" />
            {t(strings.packet.context.waiting)}
          </h2>
          {waiting.map((request) => <RequestNote key={request.id} request={request} />)}
        </section>
      )}
      {recent.length > 0 && (
        <section role="status" className="rounded-panel border border-app-primary/60 bg-app-primary/10 p-3">
          <h2 className="inline-flex items-center gap-2 text-sm font-semibold">
            <RefreshCcw size={16} aria-hidden="true" />
            {t(strings.packet.context.updated)}
          </h2>
          {roundsOf(recent).map((round) => <RoundNote key={round.requests[0]?.id} round={round} links={links} />)}
          {revised && (
            <button type="button" onClick={onSeeChanges} className="mt-1 inline-flex min-h-11 items-center text-sm font-medium underline underline-offset-4">
              {t(strings.packet.context.seeChanges)}
            </button>
          )}
        </section>
      )}
      {earlier.length > 0 && (
        <details className="rounded-panel border border-app-border bg-app-surface p-3">
          <summary className="min-h-11 cursor-pointer py-2 text-sm font-medium">{t(strings.packet.context.earlier, { count: earlier.length })}</summary>
          {roundsOf(earlier).map((round) => <RoundNote key={round.requests[0]?.id} round={round} links={links} />)}
        </details>
      )}
    </div>
  );
}
