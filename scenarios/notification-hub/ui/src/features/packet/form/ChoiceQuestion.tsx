import { strings } from "../../../consts/strings";
import { useTranslation } from "../../../i18n";
import { ChoiceCard } from "./ChoiceCard";
import type { QuestionProps } from "./QuestionField";

/** single_choice (radio cards) and multi_choice (checkboxes). */
export function ChoiceQuestion({ question, value, disabled, onChange, multiple, labelledBy }: QuestionProps & { multiple: boolean; labelledBy: string }) {
  const { t } = useTranslation();
  const recommended = question.recommended?.selected ?? [];
  return (
    <div role={multiple ? "group" : "radiogroup"} aria-labelledby={labelledBy} className="flex flex-col gap-2.5">
      {question.choices.map((choice) => (
        <ChoiceCard
          key={choice.key}
          type={multiple ? "checkbox" : "radio"}
          name={`question-${question.id}`}
          value={choice.key}
          checked={value.selected.includes(choice.key)}
          disabled={disabled}
          onChange={(event) => {
            const checked = event.target.checked;
            const selected = multiple
              ? checked
                ? [...value.selected, choice.key]
                : value.selected.filter((key) => key !== choice.key)
              : [choice.key];
            onChange({ ...value, selected });
          }}
          label={choice.label}
          detail={choice.detail}
          recommendedLabel={recommended.includes(choice.key) ? t(strings.packet.form.recommended) : undefined}
        />
      ))}
    </div>
  );
}
