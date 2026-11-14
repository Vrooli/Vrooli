import { Link, useLocation } from "react-router-dom";
import { ArrowRight, CheckCircle2 } from "lucide-react";
import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { askHeadline, optionLabel } from "../../asks/askView";
import { preserveRoutedTestMode } from "../../routedTestMode";
import type { Outcome } from "../useAnswerFlow";
import { useNextDecision } from "../useNextDecision";

interface OutcomeNoticesProps {
  ask: Ask;
  outcome: Outcome | null;
  recordedLabel: string;
  attemptedAnswer: string;
  sending: boolean;
  checking: boolean;
  readError: boolean;
  /** Where the list the operator came from lives, for "all decisions". */
  returnTo: string;
}

/**
 * What happened to the answer just sent, in the words the operator needs when
 * the connection is uncertain: recorded, still open, or "do not send it again".
 * After an answer is recorded it offers the next waiting decision.
 */
export function OutcomeNotices({ ask, outcome, recordedLabel, attemptedAnswer, sending, checking, readError, returnTo }: OutcomeNoticesProps) {
  const { t } = useTranslation();
  const location = useLocation();
  const finished = outcome === "recorded" || outcome === "dismissed";
  const next = useNextDecision(ask.id, finished);
  return (
    <>
      {sending && <p role="status">{t(strings.packet.outcome.sending)}</p>}
      {checking && <p role="status">{t(strings.packet.outcome.checking)}</p>}
      {outcome === "recorded" && (
        <p role="status" className="flex items-center gap-2 rounded-panel border border-app-success/60 bg-app-success/15 p-3 font-medium">
          <CheckCircle2 size={18} aria-hidden="true" className="shrink-0 text-app-success" />
          {t(strings.packet.outcome.recorded, { option: recordedLabel })}
        </p>
      )}
      {outcome === "uncertain" && <p role="alert">{t(strings.packet.outcome.uncertain)}</p>}
      {outcome === "changed" && <p role="alert" className="rounded-panel border border-app-warning/60 bg-app-warning/15 p-3">{t(strings.packet.outcome.changed)}</p>}
      {outcome === "open" && <p role="status">{t(strings.packet.outcome.stillOpen)}</p>}
      {outcome === "defaulted" && <p role="status">{t(strings.packet.outcome.defaultPending)}</p>}
      {outcome === "conflict" && <p role="alert">{t(strings.packet.outcome.conflict, { option: recordedLabel, attempted: attemptedAnswer ? optionLabel(ask, attemptedAnswer) : "" })}</p>}
      {readError && <p role="alert">{t(strings.packet.outcome.readbackFailed)}</p>}
      {finished && (
        <nav aria-label={t(strings.packet.outcome.nextLabel)} className="flex flex-col gap-2">
          {next.data && (
            <Link
              data-testid={selectors.packet.nextDecision}
              to={preserveRoutedTestMode(`/asks/${encodeURIComponent(next.data.id)}`, location.search)}
              state={{ from: returnTo }}
              className="inline-flex min-h-12 items-center justify-between gap-3 rounded-panel bg-app-primary px-4 py-2 font-medium text-app-primary-foreground"
            >
              <span className="flex min-w-0 flex-col">
                <span>{t(strings.packet.outcome.next)}</span>
                <span className="line-clamp-1 break-words text-sm font-normal opacity-90">{askHeadline(next.data)}</span>
              </span>
              <ArrowRight size={18} aria-hidden="true" className="shrink-0 rtl:rotate-180" />
            </Link>
          )}
          {next.isSuccess && !next.data && <p role="status" className="text-sm text-app-muted-foreground">{t(strings.packet.outcome.allDone)}</p>}
          <Link to={returnTo} className="inline-flex min-h-11 items-center self-start text-sm font-medium underline underline-offset-4">
            {t(strings.packet.outcome.allDecisions)}
          </Link>
        </nav>
      )}
    </>
  );
}
