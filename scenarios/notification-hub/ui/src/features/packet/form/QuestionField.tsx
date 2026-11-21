import { useId } from "react";
import type { Ask, AskQuestion } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { Chip } from "../Chip";
import { ANSWER_LIMITS, isChoiceKind, questionKind } from "../model/questions";
import type { ResponseDraft } from "../model/draft";
import type { AttachmentTarget } from "../viewer/attachments";
import { ChoiceQuestion } from "./ChoiceQuestion";
import { SectionReviewQuestion } from "./SectionReviewQuestion";
import { NoteDisclosure } from "./NoteDisclosure";
import { TextArea } from "./TextArea";
import { VariantQuestion } from "./VariantQuestion";

export interface QuestionProps {
  ask: Ask;
  question: AskQuestion;
  value: ResponseDraft;
  disabled: boolean;
  onChange: (next: ResponseDraft) => void;
  onOpenAttachment: (target: AttachmentTarget) => void;
}

/**
 * One typed question of the answer form. A question type this reader does not
 * know is drawn as its prompt with a text box and answered with text, so a
 * newer sender can never leave the operator unable to answer.
 */
export function QuestionField(props: QuestionProps) {
  const { t } = useTranslation();
  const { question, value, disabled, onChange } = props;
  const promptId = useId();
  const helpId = useId();
  const kind = questionKind(question);
  const reason = question.recommendationReason;
  return (
    <section
      data-testid={selectors.packet.question}
      data-question-id={question.id}
      data-question-kind={kind}
      aria-labelledby={promptId}
      aria-describedby={question.help ? helpId : undefined}
      className="flex flex-col gap-3"
    >
      <div className="flex flex-col gap-1">
        <h3 id={promptId} className="break-words text-base font-semibold leading-snug">
          {question.prompt}
          {question.required && (
            <Chip className="ms-2 align-middle" tone="info">
              {t(strings.packet.form.required)}
            </Chip>
          )}
        </h3>
        {question.help && (
          <p id={helpId} className="break-words text-sm text-app-muted-foreground">
            {question.help}
          </p>
        )}
      </div>
      {kind === "single_choice" || kind === "multi_choice" ? <ChoiceQuestion {...props} labelledBy={promptId} multiple={kind === "multi_choice"} /> : null}
      {kind === "variant" ? <VariantQuestion {...props} labelledBy={promptId} /> : null}
      {kind === "section_review" ? <SectionReviewQuestion {...props} /> : null}
      {kind === "text" ? (
        <TextArea
          label={question.prompt}
          hideLabel
          required={question.required}
          disabled={disabled}
          maxLength={ANSWER_LIMITS.responseTextChars}
          value={value.text}
          onChange={(text) => onChange({ ...value, text })}
        />
      ) : null}
      {reason && question.recommended && (
        <p className="break-words rounded-control border-s-4 border-app-warning bg-app-surface-muted px-3 py-2 text-sm text-app-muted-foreground">
          <span className="font-semibold text-app-foreground">{t(strings.packet.form.recommendedBecause)} </span>
          {reason}
        </p>
      )}
      {isChoiceKind(kind) && (
        <NoteDisclosure label={t(strings.packet.form.questionNote)} open={Boolean(value.text)}>
          <TextArea
            label={t(strings.packet.form.questionNoteFor, { prompt: question.prompt })}
            hideLabel
            rows={2}
            disabled={disabled}
            maxLength={ANSWER_LIMITS.responseTextChars}
            value={value.text}
            onChange={(text) => onChange({ ...value, text })}
          />
        </NoteDisclosure>
      )}
    </section>
  );
}
