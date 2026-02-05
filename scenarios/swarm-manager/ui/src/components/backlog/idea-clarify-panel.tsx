import { useEffect, useMemo, useState } from "react";
import { MessageSquareText } from "lucide-react";
import { Button } from "../ui/button";
import { Card } from "../ui/card";
import { selectors } from "../../consts/selectors";
import type { IdeaAgentMode, IdeaClarificationQuestion } from "../../types";

interface IdeaClarifyPanelProps {
  questions: IdeaClarificationQuestion[];
  filePath: string;
  parseError?: string;
  isSubmitting: boolean;
  submitError?: string | null;
  onSubmit: (payload: { questions: IdeaClarificationQuestion[]; nextMode: IdeaAgentMode }) => void;
}

const NEXT_MODE_OPTIONS: Array<{ value: IdeaAgentMode; label: string; helper: string }> = [
  {
    value: "suggest",
    label: "Suggest",
    helper: "Generate improvements based on your answers.",
  },
  {
    value: "enhance",
    label: "Enhance",
    helper: "Move directly to a refined plan based on your answers.",
  },
];

export function IdeaClarifyPanel({
  questions = [],
  filePath,
  parseError,
  isSubmitting,
  submitError,
  onSubmit,
}: IdeaClarifyPanelProps) {
  const [localQuestions, setLocalQuestions] = useState<IdeaClarificationQuestion[]>(questions);
  const [nextMode, setNextMode] = useState<IdeaAgentMode>("suggest");

  useEffect(() => {
    setLocalQuestions(questions);
  }, [questions]);

  const hasQuestions = localQuestions.length > 0;
  const hasUnanswered = localQuestions.some((item) => !item.answer || !item.answer.trim());
  const selectionHelper = useMemo(
    () => NEXT_MODE_OPTIONS.find((option) => option.value === nextMode)?.helper,
    [nextMode]
  );

  if (parseError) {
    return (
      <Card padding="lg" data-testid={selectors.backlogDetails.clarifyPanel}>
        <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
          <MessageSquareText className="h-4 w-4 text-cyan-400" />
          Clarify Questions
        </div>
        <p className="mt-3 text-sm text-red-300">Unable to parse {filePath}: {parseError}</p>
        <p className="mt-2 text-xs text-slate-400">Open the file in the preview to review or fix formatting.</p>
      </Card>
    );
  }

  return (
    <Card padding="lg" data-testid={selectors.backlogDetails.clarifyPanel}>
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
            <MessageSquareText className="h-4 w-4 text-cyan-400" />
            Clarify Questions
          </div>
          <p className="mt-2 text-sm text-slate-400">
            Answer the agent’s questions below. Your responses will be saved back into {filePath}.
          </p>
        </div>
        <span className="rounded-full bg-slate-800 px-3 py-1 text-xs text-slate-400">
          {localQuestions.length} question{localQuestions.length === 1 ? "" : "s"}
        </span>
      </div>

      {!hasQuestions && (
        <div className="mt-4 rounded-lg border border-white/10 bg-slate-800/40 p-4 text-sm text-slate-400">
          No questions found yet. If the agent is still running, refresh the file list once it finishes.
        </div>
      )}

      {hasQuestions && (
        <div className="mt-4 space-y-4">
          {localQuestions.map((question, index) => (
            <div key={question.id} className="rounded-lg border border-white/10 bg-slate-800/40 p-4">
              <p className="text-sm font-medium text-slate-100">{index + 1}. {question.question}</p>
              <textarea
                value={question.answer ?? ""}
                onChange={(event) => {
                  const value = event.target.value;
                  setLocalQuestions((current) =>
                    current.map((item) => (item.id === question.id ? { ...item, answer: value } : item))
                  );
                }}
                placeholder="Add your answer..."
                className="mt-3 w-full rounded-md border border-white/10 bg-slate-900/60 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                rows={3}
              />
            </div>
          ))}
        </div>
      )}

      <div className="mt-4 rounded-lg border border-white/10 bg-slate-800/50 p-4">
        <label htmlFor="clarify-next-mode" className="text-sm font-medium text-slate-200">
          Next agent
        </label>
        <div className="mt-2 flex flex-wrap gap-2" data-testid={selectors.backlogDetails.clarifyNextMode}>
          {NEXT_MODE_OPTIONS.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => setNextMode(option.value)}
              className={`rounded-full px-3 py-1 text-xs font-medium transition ${
                nextMode === option.value
                  ? "bg-cyan-500/20 text-cyan-200 ring-1 ring-cyan-500/50"
                  : "bg-slate-800 text-slate-300 hover:bg-slate-700"
              }`}
            >
              {option.label}
            </button>
          ))}
        </div>
        {selectionHelper && <p className="mt-2 text-xs text-slate-400">{selectionHelper}</p>}
      </div>

      {hasUnanswered && hasQuestions && (
        <div className="mt-3 text-xs text-slate-400">
          Answer each question before continuing.
        </div>
      )}

      {submitError && (
        <div className="mt-3 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
          {submitError}
        </div>
      )}

      <div className="mt-4 flex justify-end">
        <Button
          onClick={() => onSubmit({ questions: localQuestions, nextMode })}
          disabled={isSubmitting || !hasQuestions || hasUnanswered}
          data-testid={selectors.backlogDetails.clarifySubmit}
        >
          {isSubmitting ? "Submitting..." : `Save Answers & Run ${nextMode === "suggest" ? "Suggest" : "Enhance"}`}
        </Button>
      </div>
    </Card>
  );
}
