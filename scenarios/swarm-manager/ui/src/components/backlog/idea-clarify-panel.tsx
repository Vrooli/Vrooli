// DOC: docs/guides/idea-agent-workflow.md#phase-1-clarify
import { useEffect, useMemo, useState } from "react";
import { MessageSquareText } from "lucide-react";
import { Button } from "../ui/button";
import { Card } from "../ui/card";
import { ClarifyQuestionList } from "./clarify-question-list";
import { selectors } from "../../consts/selectors";
import type { IdeaAgentMode, IdeaClarificationQuestion } from "../../types";

type NextAction = IdeaAgentMode | "none";

interface IdeaClarifyPanelProps {
  questions: IdeaClarificationQuestion[];
  filePath: string;
  parseError?: string;
  isSubmitting: boolean;
  submitError?: string | null;
  onSubmit: (payload: { questions: IdeaClarificationQuestion[]; nextMode: NextAction }) => void;
}

const NEXT_ACTION_OPTIONS: Array<{ value: NextAction; label: string; helper: string }> = [
  {
    value: "none",
    label: "No action",
    helper: "Save your answers without triggering an agent.",
  },
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
  const [nextMode, setNextMode] = useState<NextAction>("none");

  useEffect(() => {
    setLocalQuestions(questions);
  }, [questions]);

  const hasQuestions = localQuestions.length > 0;
  const hasUnanswered = localQuestions.some((item) => !item.answer || !item.answer.trim());
  const selectionHelper = useMemo(
    () => NEXT_ACTION_OPTIONS.find((option) => option.value === nextMode)?.helper,
    [nextMode],
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

  const submitLabel = isSubmitting
    ? "Submitting..."
    : nextMode === "none"
      ? "Save Answers"
      : `Save Answers & Run ${nextMode === "suggest" ? "Suggest" : "Enhance"}`;

  return (
    <Card padding="lg" data-testid={selectors.backlogDetails.clarifyPanel}>
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-sm font-medium text-slate-200">
            <MessageSquareText className="h-4 w-4 text-cyan-400" />
            Clarify Questions
          </div>
          <p className="mt-2 text-sm text-slate-400">
            Answer the agent's questions below. Your responses will be saved back into {filePath}.
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
        <div className="mt-4">
          <ClarifyQuestionList
            questions={localQuestions}
            onChange={setLocalQuestions}
            testIdPrefix="clarify"
          />
        </div>
      )}

      <div className="mt-4 rounded-lg border border-white/10 bg-slate-800/50 p-4">
        <label htmlFor="clarify-next-mode" className="text-sm font-medium text-slate-200">
          What happens next?
        </label>
        <div className="mt-2 flex flex-wrap gap-2" data-testid={selectors.backlogDetails.clarifyNextMode}>
          {NEXT_ACTION_OPTIONS.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => setNextMode(option.value)}
              data-testid={option.value === "none" ? selectors.backlogDetails.clarifyNextModeNone : undefined}
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

      {hasUnanswered && hasQuestions && nextMode !== "none" && (
        <div className="mt-3 text-xs text-slate-400">
          Answer each question before running an agent.
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
          disabled={isSubmitting || !hasQuestions || (hasUnanswered && nextMode !== "none")}
          data-testid={selectors.backlogDetails.clarifySubmit}
        >
          {submitLabel}
        </Button>
      </div>
    </Card>
  );
}
