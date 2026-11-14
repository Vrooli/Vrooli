import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { orderedOptions } from "../../asks/askView";
import { OTHER_ANSWER } from "../model/answer";
import type { PacketDraft } from "../model/draft";
import { ChoiceCard } from "./ChoiceCard";
import { NoteDisclosure } from "./NoteDisclosure";
import { TextArea } from "./TextArea";

interface MainOptionsProps {
  ask: Ask;
  draft: PacketDraft;
  disabled: boolean;
  onChange: (next: PacketDraft) => void;
}

/**
 * The decision's own answer key: the requester's options, "Other" always
 * available, and the note. The note is the explanation of "Other" (required),
 * or an optional comment on any other option.
 */
export function MainOptions({ ask, draft, disabled, onChange }: MainOptionsProps) {
  const { t } = useTranslation();
  const choices = orderedOptions(ask).map((option) => ({ key: option.key, label: option.label }));
  // "Other" is always available, even when the requester's list lacks it.
  if (!choices.some((option) => option.key === OTHER_ANSWER)) choices.push({ key: OTHER_ANSWER, label: "" });
  return (
    <div className="flex flex-col gap-3">
      <fieldset className="flex min-w-0 flex-col gap-2.5">
        <legend className="mb-2 text-sm font-semibold">{t(strings.packet.form.choose)}</legend>
        {choices.map((option) => {
          const isOther = option.key === OTHER_ANSWER;
          return (
            <ChoiceCard
              key={option.key}
              type="radio"
              name={`answer-${ask.id}`}
              value={option.key}
              data-testid={selectors.asks.option}
              data-option-key={option.key}
              checked={draft.answer === option.key}
              disabled={disabled}
              onChange={() => onChange({ ...draft, answer: option.key })}
              label={isOther ? t(strings.packet.form.otherOption) : option.label}
              recommendedLabel={option.key === ask.recommended ? t(strings.packet.form.recommended) : undefined}
            />
          );
        })}
      </fieldset>
      {draft.answer === OTHER_ANSWER ? (
        <TextArea
          label={t(strings.packet.form.otherPrompt)}
          required
          testId={selectors.asks.note}
          value={draft.note}
          disabled={disabled}
          onChange={(note) => onChange({ ...draft, note })}
        />
      ) : (
        <NoteDisclosure label={t(strings.packet.form.noteLabel)} open={Boolean(draft.note)}>
          <TextArea
            label={t(strings.packet.form.noteLabel)}
            hideLabel
            testId={selectors.asks.note}
            value={draft.note}
            disabled={disabled}
            onChange={(note) => onChange({ ...draft, note })}
          />
        </NoteDisclosure>
      )}
    </div>
  );
}
