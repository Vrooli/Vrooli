import { ClarifyQuestionField } from "./clarify-question-field";
import type { IdeaClarificationQuestion } from "../../types";

interface ClarifyQuestionListProps {
  questions: IdeaClarificationQuestion[];
  onChange: (questions: IdeaClarificationQuestion[]) => void;
  testIdPrefix?: string;
}

export function ClarifyQuestionList({ questions, onChange, testIdPrefix }: ClarifyQuestionListProps) {
  const handleUpdate = (updated: IdeaClarificationQuestion) => {
    onChange(questions.map((q) => (q.id === updated.id ? updated : q)));
  };

  return (
    <div className="space-y-3">
      {questions.map((q, idx) => (
        <ClarifyQuestionField
          key={q.id}
          question={q}
          index={idx}
          onChange={handleUpdate}
          testIdPrefix={testIdPrefix}
        />
      ))}
    </div>
  );
}
