import { useEffect, useState } from "react";
import { Textarea } from "../ui/textarea";
import { RadioGroup } from "../ui/radio-group";
import type { IdeaClarificationQuestion } from "../../types";

interface ClarifyQuestionFieldProps {
  question: IdeaClarificationQuestion;
  index: number;
  onChange: (updated: IdeaClarificationQuestion) => void;
  testIdPrefix?: string;
}

const OTHER_VALUE = "__other__";

export function ClarifyQuestionField({ question, index, onChange, testIdPrefix }: ClarifyQuestionFieldProps) {
  const hasOptions = question.options && question.options.length > 0;

  // "Other" is selected when the answer doesn't match any predefined option
  const [otherSelected, setOtherSelected] = useState(
    () => hasOptions && !!question.answer && !question.options!.includes(question.answer),
  );

  // Sync otherSelected when the question prop changes (e.g., after save + reload).
  // Only sync when answer is non-empty so that clicking "Other" (which clears the
  // answer to "") still keeps the textarea visible.
  useEffect(() => {
    if (hasOptions && question.answer) {
      setOtherSelected(!question.options!.includes(question.answer));
    }
  }, [hasOptions, question.answer, question.options]);

  const selectedRadio = otherSelected ? OTHER_VALUE : (question.answer ?? "");

  const handleRadioChange = (value: string) => {
    if (value === OTHER_VALUE) {
      setOtherSelected(true);
      onChange({ ...question, answer: "" });
    } else {
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
            options={[
              ...question.options!.map((opt) => ({ value: opt, label: opt })),
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
          data-testid={testId ? `${testId}-input` : undefined}
        />
      )}
    </div>
  );
}
