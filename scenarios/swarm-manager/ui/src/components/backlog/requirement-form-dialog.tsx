import { useState, useEffect } from "react";
import { Button } from "../ui/button";
import { Drawer } from "../ui/drawer";
import { Input } from "../ui/input";
import { Select } from "../ui/select";
import type { ArchiveRequirementRecord } from "../../types";

export type RequirementFormMode = "create" | "edit";

interface RequirementFormDialogProps {
  isOpen: boolean;
  mode: RequirementFormMode;
  initialValues?: Partial<ArchiveRequirementRecord>;
  isSubmitting?: boolean;
  submitError?: string | null;
  onClose: () => void;
  onSubmit: (values: ArchiveRequirementRecord) => void;
}

const STATUS_OPTIONS = ["pending", "in_progress", "complete"] as const;

export function RequirementFormDialog({
  isOpen,
  mode,
  initialValues,
  isSubmitting = false,
  submitError = null,
  onClose,
  onSubmit,
}: RequirementFormDialogProps) {
  const [id, setId] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState("pending");
  const [category, setCategory] = useState("");
  const [prdRef, setPrdRef] = useState("");
  const [notes, setNotes] = useState("");
  const [error, setError] = useState<string | null>(null);

  const isEditMode = mode === "edit";

  useEffect(() => {
    if (isOpen) {
      setId(initialValues?.id ?? "");
      setTitle(initialValues?.title ?? "");
      setDescription(initialValues?.description ?? "");
      setStatus(initialValues?.status ?? "pending");
      setCategory(initialValues?.category ?? "");
      setPrdRef(initialValues?.prd_ref ?? "");
      setNotes(initialValues?.notes ?? "");
      setError(null);
    }
  }, [isOpen, initialValues]);

  const handleSubmit = () => {
    if (!id.trim()) {
      setError("ID is required.");
      return;
    }
    if (!title.trim()) {
      setError("Title is required.");
      return;
    }
    onSubmit({
      id: id.trim(),
      title: title.trim(),
      description: description.trim(),
      status,
      category: category.trim(),
      prd_ref: prdRef.trim(),
      notes: notes.trim() || undefined,
    });
  };

  const displayError = error ?? submitError;

  return (
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      title={isEditMode ? "Edit Requirement" : "Create Requirement"}
      description={isEditMode ? "Update requirement details." : "Add a new requirement to this module."}
    >

      <div className="space-y-4 p-4">
        <div>
          <label htmlFor="req-form-id" className="text-sm font-medium text-slate-300">ID</label>
          <Input
            id="req-form-id"
            value={id}
            onChange={(e) => { setId(e.target.value); setError(null); }}
            placeholder="REQ-P0-001"
            className="mt-2"
            disabled={isSubmitting || isEditMode}
          />
        </div>

        <div>
          <label htmlFor="req-form-title" className="text-sm font-medium text-slate-300">Title</label>
          <Input
            id="req-form-title"
            value={title}
            onChange={(e) => { setTitle(e.target.value); setError(null); }}
            placeholder="Requirement title"
            className="mt-2"
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="req-form-description" className="text-sm font-medium text-slate-300">Description</label>
          <textarea
            id="req-form-description"
            value={description}
            onChange={(e) => { setDescription(e.target.value); setError(null); }}
            placeholder="Describe the requirement..."
            className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={3}
            disabled={isSubmitting}
          />
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label htmlFor="req-form-status" className="text-sm font-medium text-slate-300">Status</label>
            <div className="mt-2">
              <Select
                id="req-form-status"
                value={status}
                onChange={(e) => setStatus(e.target.value)}
                disabled={isSubmitting}
              >
                {STATUS_OPTIONS.map((opt) => (
                  <option key={opt} value={opt}>{opt.replace(/_/g, " ")}</option>
                ))}
              </Select>
            </div>
          </div>

          <div>
            <label htmlFor="req-form-category" className="text-sm font-medium text-slate-300">Category</label>
            <Input
              id="req-form-category"
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              placeholder="functional"
              className="mt-2"
              disabled={isSubmitting}
            />
          </div>
        </div>

        <div>
          <label htmlFor="req-form-prd-ref" className="text-sm font-medium text-slate-300">PRD Reference</label>
          <Input
            id="req-form-prd-ref"
            value={prdRef}
            onChange={(e) => setPrdRef(e.target.value)}
            placeholder="PRD-001"
            className="mt-2"
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="req-form-notes" className="text-sm font-medium text-slate-300">Notes</label>
          <textarea
            id="req-form-notes"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Additional notes..."
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
          {isSubmitting ? "Saving..." : isEditMode ? "Save Changes" : "Create Requirement"}
        </Button>
      </div>
    </Drawer>
  );
}
