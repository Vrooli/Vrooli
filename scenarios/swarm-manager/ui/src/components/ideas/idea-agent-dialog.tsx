import { useEffect, useMemo, useState } from "react";
import { X, Sparkles } from "lucide-react";
import { Button } from "../ui/button";
import { selectors } from "../../consts/selectors";
import { IDEA_AGENT_FILE_PATHS } from "../../lib";
import type { IdeaAgentMode } from "../../types";

interface IdeaAgentDialogProps {
  isOpen: boolean;
  isSubmitting?: boolean;
  ideaTitle: string;
  errorMessage?: string | null;
  onClose: () => void;
  onSubmit: (payload: { mode: IdeaAgentMode; prompt: string }) => void;
}

const MODE_OPTIONS: Array<{
  value: IdeaAgentMode;
  title: string;
  description: string;
  output: string;
}> = [
  {
    value: "clarify",
    title: "Clarify",
    description:
      "Gather the most relevant questions needed to clarify scope, constraints, and implementation details.",
    output: `Writes questions to ${IDEA_AGENT_FILE_PATHS.clarify}`,
  },
  {
    value: "suggest",
    title: "Suggest",
    description:
      "Generate improvements and alternative approaches for the idea, ready for review and selection.",
    output: `Writes suggestions to ${IDEA_AGENT_FILE_PATHS.suggest}`,
  },
  {
    value: "enhance",
    title: "Enhance",
    description:
      "Produce a refined plan using answered clarifications and accepted suggestions.",
    output: `Writes enhancements to ${IDEA_AGENT_FILE_PATHS.enhance}`,
  },
];

export function IdeaAgentDialog({
  isOpen,
  isSubmitting = false,
  ideaTitle,
  errorMessage = null,
  onClose,
  onSubmit,
}: IdeaAgentDialogProps) {
  const [prompt, setPrompt] = useState("");
  const [mode, setMode] = useState<IdeaAgentMode>("clarify");

  useEffect(() => {
    if (isOpen) {
      setPrompt("");
      setMode("clarify");
    }
  }, [isOpen]);

  const activeMode = useMemo(() => MODE_OPTIONS.find((option) => option.value === mode), [mode]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" data-testid={selectors.ideaForm.agentDialog}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} aria-hidden="true" />
      <div className="relative z-10 w-full max-w-2xl rounded-xl border border-white/10 bg-slate-900 p-6 shadow-2xl">
        <button
          type="button"
          onClick={onClose}
          className="absolute right-4 top-4 rounded-full p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        <div className="flex items-center gap-3">
          <div className="rounded-full bg-cyan-500/20 p-2">
            <Sparkles className="h-5 w-5 text-cyan-400" />
          </div>
          <div>
            <h2 className="text-xl font-semibold text-slate-100">Run Idea Agent</h2>
            <p className="text-sm text-slate-400">{ideaTitle}</p>
          </div>
        </div>

        <div className="mt-4 rounded-lg border border-white/10 bg-slate-800/40 p-3 text-sm text-slate-300">
          This agent will update files inside the idea folder so you can review, edit, and build on the output.
        </div>

        <fieldset className="mt-4 space-y-3" data-testid={selectors.ideaForm.agentMode}>
          <legend className="text-sm font-medium text-slate-300">Agent type</legend>
          <div className="grid gap-3 md:grid-cols-3">
            {MODE_OPTIONS.map((option) => {
              const isSelected = option.value === mode;
              return (
                <label
                  key={option.value}
                  className={`flex h-full cursor-pointer flex-col gap-2 rounded-lg border p-3 text-left transition ${
                    isSelected
                      ? "border-cyan-500/60 bg-cyan-500/10"
                      : "border-white/10 bg-slate-800/40 hover:border-white/20"
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-semibold text-slate-100">{option.title}</span>
                    <input
                      type="radio"
                      name="idea-agent-mode"
                      value={option.value}
                      checked={isSelected}
                      onChange={() => setMode(option.value)}
                      className="h-4 w-4 accent-cyan-500"
                    />
                  </div>
                  <p className="text-xs text-slate-400">{option.description}</p>
                  <p className="text-xs text-slate-500">{option.output}</p>
                </label>
              );
            })}
          </div>
        </fieldset>

        <div className="mt-4 space-y-3">
          <label htmlFor="idea-agent-context" className="text-sm font-medium text-slate-300">
            Additional context (optional)
          </label>
          <textarea
            id="idea-agent-context"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="Add any constraints, focus areas, or implementation notes."
            className="w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={4}
            data-testid={selectors.ideaForm.agentContext}
            disabled={isSubmitting}
          />
          {activeMode && (
            <div className="text-xs text-slate-400">
              Next output: <span className="text-slate-200">{activeMode.output}</span>
            </div>
          )}
          {errorMessage && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {errorMessage}
            </div>
          )}
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button
            onClick={() => onSubmit({ mode, prompt })}
            disabled={isSubmitting}
            data-testid={selectors.ideaForm.agentSubmit}
          >
            {isSubmitting ? "Spawning..." : `Run ${activeMode?.title ?? "Agent"}`}
          </Button>
        </div>
      </div>
    </div>
  );
}
