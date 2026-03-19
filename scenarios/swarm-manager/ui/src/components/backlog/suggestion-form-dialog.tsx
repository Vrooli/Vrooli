import { useState, useEffect } from "react";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Select } from "../ui/select";
import type { SuggestionFormValues, IdeaSuggestionDecision } from "../../types";

export type SuggestionFormMode = "create" | "edit";

interface SuggestionFormDialogProps {
  isOpen: boolean;
  mode: SuggestionFormMode;
  initialValues?: Partial<SuggestionFormValues>;
  isSubmitting?: boolean;
  submitError?: string | null;
  onClose: () => void;
  onSubmit: (values: SuggestionFormValues) => void;
}

const STATUS_OPTIONS: Array<{ value: IdeaSuggestionDecision; label: string }> = [
  { value: "pending", label: "Pending" },
  { value: "accepted", label: "Accepted" },
  { value: "rejected", label: "Rejected" },
];

export function SuggestionFormDialog({
  isOpen,
  mode,
  initialValues,
  isSubmitting = false,
  submitError = null,
  onClose,
  onSubmit,
}: SuggestionFormDialogProps) {
  const [suggestion, setSuggestion] = useState("");
  const [details, setDetails] = useState("");
  const [status, setStatus] = useState<IdeaSuggestionDecision>("pending");
  const [notes, setNotes] = useState("");
  const [error, setError] = useState<string | null>(null);

  const isEditMode = mode === "edit";

  useEffect(() => {
    if (isOpen) {
      setSuggestion(initialValues?.suggestion ?? "");
      setDetails(initialValues?.details ?? "");
      setStatus(initialValues?.status ?? "pending");
      setNotes(initialValues?.notes ?? "");
      setError(null);
    }
  }, [isOpen, initialValues]);

  const handleSubmit = () => {
    if (!suggestion.trim()) {
      setError("Suggestion text is required.");
      return;
    }
    onSubmit({
      suggestion: suggestion.trim(),
      details: details.trim() || undefined,
      status,
      notes: notes.trim() || undefined,
    });
  };

  const displayError = error ?? submitError;

  return (
    <Dialog isOpen={isOpen} onClose={onClose} maxWidth="max-w-xl" isLoading={isSubmitting}>
      <h2 className="text-xl font-semibold text-slate-100">
        {isEditMode ? "Edit Suggestion" : "Add Suggestion"}
      </h2>
      <p className="mt-1 text-sm text-slate-400">
        {isEditMode ? "Update the suggestion details." : "Add a new suggestion."}
      </p>

      <div className="mt-6 space-y-4">
        <div>
          <label htmlFor="suggestion-form-text" className="text-sm font-medium text-slate-300">Suggestion</label>
          <textarea
            id="suggestion-form-text"
            value={suggestion}
            onChange={(e) => { setSuggestion(e.target.value); setError(null); }}
            placeholder="What do you suggest?"
            className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={3}
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="suggestion-form-details" className="text-sm font-medium text-slate-300">Details (optional)</label>
          <textarea
            id="suggestion-form-details"
            value={details}
            onChange={(e) => setDetails(e.target.value)}
            placeholder="Additional context or rationale..."
            className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={2}
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="suggestion-form-status" className="text-sm font-medium text-slate-300">Status</label>
          <div className="mt-2">
            <Select
              id="suggestion-form-status"
              value={status}
              onChange={(e) => setStatus(e.target.value as IdeaSuggestionDecision)}
              disabled={isSubmitting}
            >
              {STATUS_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </Select>
          </div>
        </div>

        {(status === "accepted" || status === "rejected") && (
          <div>
            <label htmlFor="suggestion-form-notes" className="text-sm font-medium text-slate-300">Notes (optional)</label>
            <textarea
              id="suggestion-form-notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="Add a note about this decision..."
              className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
              rows={2}
              disabled={isSubmitting}
            />
          </div>
        )}

        {displayError && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {displayError}
          </div>
        )}
      </div>

      <div className="mt-6 flex justify-end gap-3">
        <Button variant="outline" onClick={onClose} disabled={isSubmitting}>Cancel</Button>
        <Button onClick={handleSubmit} disabled={isSubmitting}>
          {isSubmitting ? "Saving..." : isEditMode ? "Save Changes" : "Add Suggestion"}
        </Button>
      </div>
    </Dialog>
  );
}
