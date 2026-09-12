import { useState, useEffect } from "react";
import { Button } from "../ui/button";
import { Drawer } from "../ui/drawer";
import { Input } from "../ui/input";
import { Select } from "../ui/select";
import type { ArchiveTargetFormValues } from "../../types";

export type TargetFormMode = "create" | "edit";

interface TargetFormDialogProps {
  isOpen: boolean;
  mode: TargetFormMode;
  initialValues?: Partial<ArchiveTargetFormValues>;
  isSubmitting?: boolean;
  submitError?: string | null;
  onClose: () => void;
  onSubmit: (values: ArchiveTargetFormValues) => void;
}

const CRITICALITY_OPTIONS = ["P0", "P1", "P2"] as const;
const STATUS_OPTIONS = ["pending", "complete"] as const;

export function TargetFormDialog({
  isOpen,
  mode,
  initialValues,
  isSubmitting = false,
  submitError = null,
  onClose,
  onSubmit,
}: TargetFormDialogProps) {
  const [id, setId] = useState("");
  const [title, setTitle] = useState("");
  const [criticality, setCriticality] = useState("P0");
  const [status, setStatus] = useState("pending");
  const [notes, setNotes] = useState("");
  const [linkedReqs, setLinkedReqs] = useState("");
  const [error, setError] = useState<string | null>(null);

  const isEditMode = mode === "edit";

  useEffect(() => {
    if (isOpen) {
      setId(initialValues?.id ?? "");
      setTitle(initialValues?.title ?? "");
      setCriticality(initialValues?.criticality ?? "P0");
      setStatus(initialValues?.status ?? "pending");
      setNotes(initialValues?.notes ?? "");
      setLinkedReqs(initialValues?.linked_requirement_ids?.join(", ") ?? "");
      setError(null);
    }
  }, [isOpen, initialValues]);

  const handleSubmit = () => {
    if (!title.trim()) {
      setError("Title is required.");
      return;
    }
    const parsedReqs = linkedReqs
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);

    onSubmit({
      id: id.trim(),
      title: title.trim(),
      criticality,
      status,
      notes: notes.trim(),
      linked_requirement_ids: parsedReqs,
    });
  };

  const displayError = error ?? submitError;

  return (
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      title={isEditMode ? "Edit Target" : "Create Target"}
      description={isEditMode ? "Update operational target details." : "Add a new operational target."}
    >

      <div className="space-y-4 p-4">
        <div>
          <label htmlFor="target-form-id" className="text-sm font-medium text-slate-300">ID</label>
          <Input
            id="target-form-id"
            value={id}
            onChange={(e) => { setId(e.target.value); setError(null); }}
            placeholder="OT-P0-001 (auto-generated if empty)"
            className="mt-2"
            disabled={isSubmitting || isEditMode}
          />
        </div>

        <div>
          <label htmlFor="target-form-title" className="text-sm font-medium text-slate-300">Title</label>
          <Input
            id="target-form-title"
            value={title}
            onChange={(e) => { setTitle(e.target.value); setError(null); }}
            placeholder="Target title"
            className="mt-2"
            disabled={isSubmitting}
          />
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="target-form-criticality" className="text-sm font-medium text-slate-300">Criticality</label>
            <div className="mt-2">
              <Select
                id="target-form-criticality"
                value={criticality}
                onChange={(e) => setCriticality(e.target.value)}
                disabled={isSubmitting}
              >
                {CRITICALITY_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>{opt}</option>
                ))}
              </Select>
            </div>
          </div>

          <div>
            <label htmlFor="target-form-status" className="text-sm font-medium text-slate-300">Status</label>
            <div className="mt-2">
              <Select
                id="target-form-status"
                value={status}
                onChange={(e) => setStatus(e.target.value)}
                disabled={isSubmitting}
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>{opt}</option>
                ))}
              </Select>
            </div>
          </div>
        </div>

        <div>
          <label htmlFor="target-form-notes" className="text-sm font-medium text-slate-300">Notes</label>
          <textarea
            id="target-form-notes"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Additional notes..."
            className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={2}
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="target-form-reqs" className="text-sm font-medium text-slate-300">Linked Requirements</label>
          <Input
            id="target-form-reqs"
            value={linkedReqs}
            onChange={(e) => setLinkedReqs(e.target.value)}
            placeholder="REQ-001, REQ-002 (comma-separated)"
            className="mt-2"
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
          {isSubmitting ? "Saving..." : isEditMode ? "Save Changes" : "Create Target"}
        </Button>
      </div>
    </Drawer>
  );
}
