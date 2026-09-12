import { useState, useEffect } from "react";
import { Button } from "../ui/button";
import { Drawer } from "../ui/drawer";
import { Input } from "../ui/input";
import type { ModuleFormValues } from "../../types";

export type ModuleFormMode = "create" | "edit";

interface ModuleFormDialogProps {
  isOpen: boolean;
  mode: ModuleFormMode;
  initialValues?: Partial<ModuleFormValues>;
  isSubmitting?: boolean;
  submitError?: string | null;
  onClose: () => void;
  onSubmit: (values: ModuleFormValues) => void;
}

export function ModuleFormDialog({
  isOpen,
  mode,
  initialValues,
  isSubmitting = false,
  submitError = null,
  onClose,
  onSubmit,
}: ModuleFormDialogProps) {
  const [id, setId] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState<string | null>(null);

  const isEditMode = mode === "edit";

  useEffect(() => {
    if (isOpen) {
      setId(initialValues?.id ?? "");
      setTitle(initialValues?.title ?? "");
      setDescription(initialValues?.description ?? "");
      setError(null);
    }
  }, [isOpen, initialValues]);

  const handleSubmit = () => {
    if (!isEditMode && !id.trim()) {
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
    });
  };

  const displayError = error ?? submitError;

  return (
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      title={isEditMode ? "Edit Module" : "Create Module"}
      description={isEditMode ? "Update module metadata." : "Add a new requirements module."}
    >

      <div className="space-y-4 p-4">
        <div>
          <label htmlFor="module-form-id" className="text-sm font-medium text-slate-300">ID</label>
          <Input
            id="module-form-id"
            value={id}
            onChange={(e) => { setId(e.target.value); setError(null); }}
            placeholder="module-id"
            className="mt-2"
            disabled={isSubmitting || isEditMode}
          />
        </div>

        <div>
          <label htmlFor="module-form-title" className="text-sm font-medium text-slate-300">Title</label>
          <Input
            id="module-form-title"
            value={title}
            onChange={(e) => { setTitle(e.target.value); setError(null); }}
            placeholder="Module title"
            className="mt-2"
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="module-form-description" className="text-sm font-medium text-slate-300">Description</label>
          <textarea
            id="module-form-description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Describe the module..."
            className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
            rows={3}
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
          {isSubmitting ? "Saving..." : isEditMode ? "Save Changes" : "Create Module"}
        </Button>
      </div>
    </Drawer>
  );
}
