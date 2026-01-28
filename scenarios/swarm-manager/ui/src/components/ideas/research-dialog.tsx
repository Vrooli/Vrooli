import { useEffect, useState } from "react";
import { X, Sparkles } from "lucide-react";
import { Button } from "../ui/button";
import { selectors } from "../../consts/selectors";

interface ResearchDialogProps {
  isOpen: boolean;
  isSubmitting?: boolean;
  ideaTitle: string;
  errorMessage?: string | null;
  onClose: () => void;
  onSubmit: (prompt: string) => void;
}

export function ResearchDialog({
  isOpen,
  isSubmitting = false,
  ideaTitle,
  errorMessage = null,
  onClose,
  onSubmit,
}: ResearchDialogProps) {
  const [prompt, setPrompt] = useState("");

  useEffect(() => {
    if (isOpen) {
      setPrompt("");
    }
  }, [isOpen]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" data-testid={selectors.ideaForm.researchDialog}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} aria-hidden="true" />
      <div className="relative z-10 w-full max-w-lg rounded-xl border border-white/10 bg-slate-900 p-6 shadow-2xl">
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
            <h2 className="text-xl font-semibold text-slate-100">Research Idea</h2>
            <p className="text-sm text-slate-400">{ideaTitle}</p>
          </div>
        </div>

        <div className="mt-4 space-y-3">
          <p className="text-sm text-slate-400">
            Provide any focus areas or constraints. The agent will analyze feasibility, risks, and
            recommended next steps.
          </p>
          <textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="e.g., Focus on integrations with agent-manager and expected user flows"
            className="w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={5}
            data-testid={selectors.ideaForm.researchPrompt}
            disabled={isSubmitting}
          />
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
            onClick={() => onSubmit(prompt)}
            disabled={isSubmitting}
            data-testid={selectors.ideaForm.researchSubmit}
          >
            {isSubmitting ? "Spawning..." : "Start Research"}
          </Button>
        </div>
      </div>
    </div>
  );
}
