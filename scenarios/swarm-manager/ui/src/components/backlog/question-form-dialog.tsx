import { useState, useEffect } from "react";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Input } from "../ui/input";
import type { ClarifyQuestionFormValues } from "../../types";

export type QuestionFormMode = "create" | "edit";

interface QuestionFormDialogProps {
  isOpen: boolean;
  mode: QuestionFormMode;
  initialValues?: Partial<ClarifyQuestionFormValues>;
  isSubmitting?: boolean;
  submitError?: string | null;
  onClose: () => void;
  onSubmit: (values: ClarifyQuestionFormValues) => void;
}

export function QuestionFormDialog({
  isOpen,
  mode,
  initialValues,
  isSubmitting = false,
  submitError = null,
  onClose,
  onSubmit,
}: QuestionFormDialogProps) {
  const [question, setQuestion] = useState("");
  const [options, setOptions] = useState("");
  const [answer, setAnswer] = useState("");
  const [error, setError] = useState<string | null>(null);

  const isEditMode = mode === "edit";

  useEffect(() => {
    if (isOpen) {
      setQuestion(initialValues?.question ?? "");
      setOptions(initialValues?.options?.join(", ") ?? "");
      setAnswer(initialValues?.answer ?? "");
      setError(null);
    }
  }, [isOpen, initialValues]);

  const handleSubmit = () => {
    if (!question.trim()) {
      setError("Question text is required.");
      return;
    }
    const parsedOptions = options
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);

    onSubmit({
      question: question.trim(),
      options: parsedOptions.length > 0 ? parsedOptions : undefined,
      answer: answer.trim() || undefined,
    });
  };

  const displayError = error ?? submitError;

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="max-w-xl" isLoading={isSubmitting}>
      <h2 className="text-xl font-semibold text-slate-100">
        {isEditMode ? "Edit Question" : "Add Question"}
      </h2>
      <p className="mt-1 text-sm text-slate-400">
        {isEditMode ? "Update the clarifying question." : "Add a new clarifying question."}
      </p>

      <div className="mt-6 space-y-4">
        <div>
          <label htmlFor="question-form-text" className="text-sm font-medium text-slate-300">Question</label>
          <textarea
            id="question-form-text"
            value={question}
            onChange={(e) => { setQuestion(e.target.value); setError(null); }}
            placeholder="What needs to be clarified?"
            className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={3}
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="question-form-options" className="text-sm font-medium text-slate-300">Options (optional)</label>
          <Input
            id="question-form-options"
            value={options}
            onChange={(e) => setOptions(e.target.value)}
            placeholder="Option A, Option B, Option C (comma-separated)"
            className="mt-2"
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="question-form-answer" className="text-sm font-medium text-slate-300">Answer (optional)</label>
          <textarea
            id="question-form-answer"
            value={answer}
            onChange={(e) => setAnswer(e.target.value)}
            placeholder="Pre-fill an answer..."
            className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={2}
            disabled={isSubmitting}
          />
        </div>

        {displayError && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {displayError}
          </div>
        )}
      </div>

      <div className="mt-6 flex justify-end gap-3">
        <Button variant="outline" onClick={onClose} disabled={isSubmitting}>Cancel</Button>
        <Button onClick={handleSubmit} disabled={isSubmitting}>
          {isSubmitting ? "Saving..." : isEditMode ? "Save Changes" : "Add Question"}
        </Button>
      </div>
    </Dialog>
  );
}
