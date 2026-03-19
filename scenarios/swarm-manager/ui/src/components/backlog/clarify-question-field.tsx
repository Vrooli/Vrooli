import { useEffect, useMemo, useRef, useState } from "react";
import { Textarea } from "../ui/textarea";
import { RadioGroup } from "../ui/radio-group";
import type { IdeaClarificationQuestion } from "../../types";

interface ClarifyQuestionFieldProps {
  question: IdeaClarificationQuestion;
  index: number;
  onChange: (updated: IdeaClarificationQuestion) => void;
  testIdPrefix?: string;
  disabled?: boolean;
}

const OTHER_VALUE = "__other__";

/** Check whether an answer matches one of the predefined options (trimmed comparison). */
const answerMatchesOption = (answer: string, options: string[]): boolean =>
  options.some((opt) => opt.trim() === answer.trim());

export function ClarifyQuestionField({ question, index, onChange, testIdPrefix, disabled }: ClarifyQuestionFieldProps) {
  const options = useMemo(() => question.options ?? [], [question.options]);
  const hasOptions = options.length > 0;

  // Track whether the user explicitly clicked "Other" during this interaction.
  // This prevents the empty-answer sync from flipping back to "no selection"
  // when the user clicks Other and hasn't typed yet.
  const userClickedOther = useRef(false);

  // Determine initial otherSelected state from the data
  const [otherSelected, setOtherSelected] = useState(
    () => hasOptions && !!question.answer && !answerMatchesOption(question.answer, options),
  );

  // Sync otherSelected when the question data changes (e.g., after save + reload
  // or when different question data arrives). Reset the userClickedOther ref since
  // the data source changed.
  useEffect(() => {
    if (!hasOptions) return;
    if (question.answer) {
      const isOther = !answerMatchesOption(question.answer, options);
      setOtherSelected(isOther);
      userClickedOther.current = false;
    } else if (!userClickedOther.current) {
      // Answer is empty and user didn't explicitly click Other — reset to no selection
      setOtherSelected(false);
    }
  }, [hasOptions, question.id, question.answer, options]);

  // When not "Other", find the exact option value that matches (handles trimmed comparison)
  const selectedRadio = otherSelected
    ? OTHER_VALUE
    : (question.answer
      ? (options.find((opt) => opt.trim() === question.answer?.trim()) ?? question.answer)
      : "");

  const handleRadioChange = (value: string) => {
    if (value === OTHER_VALUE) {
      userClickedOther.current = true;
      setOtherSelected(true);
      onChange({ ...question, answer: "" });
    } else {
      userClickedOther.current = false;
      setOtherSelected(false);
      onChange({ ...question, answer: value });
    }
  };

  const testId = testIdPrefix ? `${testIdPrefix}-q-${question.id}` : undefined;

  return (
    <div className="rounded-lg border border-white/10 bg-slate-900/40 p-3" data-testid={testId}>
      <p className="text-sm font-medium text-slate-100">
        {index + 1}. {question.question}
      </p>

      {hasOptions ? (
        <div className="mt-3 space-y-3">
          <RadioGroup
            name={`clarify-${question.id}`}
            value={selectedRadio}
            onChange={handleRadioChange}
            disabled={disabled}
            options={[
              ...options.map((opt) => ({ value: opt, label: opt })),
              { value: OTHER_VALUE, label: "Other" },
            ]}
            testIdPrefix={testId ? `${testId}-opt` : undefined}
          />
          {otherSelected && (
            <Textarea
              value={question.answer ?? ""}
              onChange={(e) => onChange({ ...question, answer: e.target.value })}
              placeholder="Specify your answer..."
              rows={2}
              disabled={disabled}
              data-testid={testId ? `${testId}-other-input` : undefined}
            />
          )}
        </div>
      ) : (
        <Textarea
          value={question.answer ?? ""}
          onChange={(e) => onChange({ ...question, answer: e.target.value })}
          placeholder="Your answer..."
          rows={2}
          className="mt-2"
          disabled={disabled}
          data-testid={testId ? `${testId}-input` : undefined}
        />
      )}
    </div>
  );
}
