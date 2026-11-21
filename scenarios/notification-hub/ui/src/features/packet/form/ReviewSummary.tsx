import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { optionLabel } from "../../asks/askView";
import { buildAnswer, missingRequired, OTHER_ANSWER } from "../model/answer";
import type { PacketDraft } from "../model/draft";
import { describeResponse } from "../model/describeResponse";

/**
 * The last look before sending: the main answer and what was said to each
 * question, as the requester will receive it. A required question that is
 * still open is named, with a way to jump back to it.
 */
export function ReviewSummary({ ask, draft, onEdit }: { ask: Ask; draft: PacketDraft; onEdit: (target: "answer" | { question: string }) => void }) {
  const { t } = useTranslation();
  const payload = buildAnswer(ask, draft);
  const missing = new Set(missingRequired(ask, draft).map((question) => question.id));
  const answerLabel = draft.answer === OTHER_ANSWER ? t(strings.packet.form.otherOption) : optionLabel(ask, draft.answer);

  return (
    <section aria-labelledby="review-heading" className="flex flex-col gap-3">
      <h3 id="review-heading" className="text-base font-semibold">{t(strings.packet.form.reviewTitle)}</h3>
      <dl className="flex flex-col divide-y divide-app-border rounded-panel border border-app-border bg-app-surface">
        <div className="flex items-start justify-between gap-3 p-3">
          <div className="min-w-0">
            <dt className="text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{t(strings.packet.form.reviewDecision)}</dt>
            <dd className="mt-1 break-words font-medium">{draft.answer ? answerLabel : <span className="text-app-danger">{t(strings.packet.form.reviewNothing)}</span>}</dd>
          </div>
          <button type="button" onClick={() => onEdit("answer")} className="min-h-11 shrink-0 text-sm font-medium underline underline-offset-4">{t(strings.packet.form.reviewEdit)}</button>
        </div>
        {ask.questions.map((question) => {
          const lines = describeResponse(question, payload.responses.find((response) => response.questionId === question.id), t);
          return (
            <div key={question.id} className="flex items-start justify-between gap-3 p-3">
              <div className="min-w-0">
                <dt className="break-words text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">{question.prompt}</dt>
                <dd className="mt-1 flex flex-col gap-0.5 break-words">
                  {lines.length > 0 ? lines.map((line) => <span key={line}>{line}</span>) : <span className={missing.has(question.id) ? "text-app-danger" : "text-app-muted-foreground"}>{missing.has(question.id) ? t(strings.packet.form.reviewRequired) : t(strings.packet.form.reviewSkipped)}</span>}
                </dd>
              </div>
              <button type="button" onClick={() => onEdit({ question: question.id })} className="min-h-11 shrink-0 text-sm font-medium underline underline-offset-4">{t(strings.packet.form.reviewEdit)}</button>
            </div>
          );
        })}
      </dl>
    </section>
  );
}
