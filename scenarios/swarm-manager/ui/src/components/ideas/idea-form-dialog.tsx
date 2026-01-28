import { useEffect, useMemo, useState } from "react";
import { X } from "lucide-react";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { selectors } from "../../consts/selectors";
import { parseTagsInput, sanitizeIdeaName, tagsToInput } from "../../lib";
import type { IdeaStatus } from "../../types";

export type IdeaFormMode = "create" | "edit";

export interface IdeaFormValues {
  name: string;
  title: string;
  description: string;
  status: IdeaStatus;
  priority: number;
  tags: string[];
}

interface IdeaFormDialogProps {
  isOpen: boolean;
  mode: IdeaFormMode;
  initialValues?: IdeaFormValues;
  isSubmitting?: boolean;
  submitError?: string | null;
  onClose: () => void;
  onSubmit: (values: IdeaFormValues) => void;
}

const STATUS_OPTIONS: IdeaStatus[] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "completed",
  "archived",
];

export function IdeaFormDialog({
  isOpen,
  mode,
  initialValues,
  isSubmitting = false,
  submitError = null,
  onClose,
  onSubmit,
}: IdeaFormDialogProps) {
  const [name, setName] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<IdeaStatus>("backlog");
  const [priority, setPriority] = useState(5);
  const [tagsInput, setTagsInput] = useState("");
  const [nameDirty, setNameDirty] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isEditMode = mode === "edit";

  useEffect(() => {
    if (isOpen) {
      setName(initialValues?.name ?? "");
      setTitle(initialValues?.title ?? "");
      setDescription(initialValues?.description ?? "");
      setStatus(initialValues?.status ?? "backlog");
      setPriority(initialValues?.priority ?? 5);
      setTagsInput(tagsToInput(initialValues?.tags ?? []));
      setNameDirty(isEditMode);
      setError(null);
    }
  }, [isOpen, initialValues, isEditMode]);

  const handleTitleChange = (value: string) => {
    setTitle(value);
    if (error) setError(null);
    if (!nameDirty && !isEditMode) {
      setName(sanitizeIdeaName(value));
    }
  };

  const derivedTags = useMemo(() => parseTagsInput(tagsInput), [tagsInput]);

  const handleSubmit = () => {
    if (title.trim() === "") {
      setError("Title is required.");
      return;
    }
    const finalName = isEditMode ? name : sanitizeIdeaName(name || title);
    if (finalName.trim() === "") {
      setError("Name is required.");
      return;
    }

    onSubmit({
      name: finalName,
      title: title.trim(),
      description: description.trim(),
      status: isEditMode ? status : "backlog",
      priority,
      tags: derivedTags,
    });
  };

  const displayError = error ?? submitError;

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" data-testid={selectors.ideaForm.dialog}>
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} aria-hidden="true" />
      <div className="relative z-10 w-full max-w-xl rounded-xl border border-white/10 bg-slate-900 p-6 shadow-2xl">
        <button
          type="button"
          onClick={onClose}
          className="absolute right-4 top-4 rounded-full p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
          aria-label="Close dialog"
        >
          <X className="h-5 w-5" />
        </button>

        <h2 className="text-xl font-semibold text-slate-100">
          {isEditMode ? "Edit Idea" : "Create Idea"}
        </h2>
        <p className="mt-1 text-sm text-slate-400">
          {isEditMode
            ? "Update idea details and lifecycle status."
            : "Capture a new scenario idea and add it to your backlog."}
        </p>

        <div className="mt-6 space-y-4">
          <div>
            <label className="text-sm font-medium text-slate-300">Title</label>
            <Input
              value={title}
              onChange={(e) => handleTitleChange(e.target.value)}
              placeholder="Idea title"
              className="mt-2"
              data-testid={selectors.ideaForm.titleInput}
              disabled={isSubmitting}
            />
          </div>

          <div>
            <label className="text-sm font-medium text-slate-300">Name</label>
            <Input
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                setNameDirty(true);
                if (error) setError(null);
              }}
              placeholder="folder-safe-name"
              className="mt-2"
              data-testid={selectors.ideaForm.nameInput}
              disabled={isSubmitting || isEditMode}
            />
            {!isEditMode && (
              <p className="mt-1 text-xs text-slate-500">Auto-generated from title if left empty.</p>
            )}
          </div>

          <div>
            <label className="text-sm font-medium text-slate-300">Description</label>
            <textarea
              value={description}
              onChange={(e) => {
                setDescription(e.target.value);
                if (error) setError(null);
              }}
              placeholder="Describe the idea, goals, and constraints..."
              className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
              rows={4}
              data-testid={selectors.ideaForm.descriptionInput}
              disabled={isSubmitting}
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label className="text-sm font-medium text-slate-300">Priority (1-10)</label>
              <Input
                type="number"
                min={1}
                max={10}
                value={priority}
                onChange={(e) => {
                  setPriority(Number(e.target.value) || 1);
                  if (error) setError(null);
                }}
                className="mt-2"
                data-testid={selectors.ideaForm.priorityInput}
                disabled={isSubmitting}
              />
            </div>
            {isEditMode ? (
              <div>
                <label className="text-sm font-medium text-slate-300">Status</label>
                <div className="relative mt-2">
                  <select
                    value={status}
                    onChange={(e) => {
                      setStatus(e.target.value as IdeaStatus);
                      if (error) setError(null);
                    }}
                    className="w-full appearance-none rounded-lg border border-white/10 bg-slate-800/50 px-4 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                    data-testid={selectors.ideaForm.statusSelect}
                    disabled={isSubmitting}
                  >
                    {STATUS_OPTIONS.map((option) => (
                      <option key={option} value={option}>
                        {option.replace(/_/g, " ")}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            ) : (
              <div className="flex flex-col justify-end text-sm text-slate-400">
                <span className="font-medium text-slate-300">Status</span>
                <span className="mt-2 rounded-lg border border-white/10 bg-slate-800/50 px-4 py-2">Backlog</span>
              </div>
            )}
          </div>

          <div>
            <label className="text-sm font-medium text-slate-300">Tags</label>
            <Input
              value={tagsInput}
              onChange={(e) => {
                setTagsInput(e.target.value);
                if (error) setError(null);
              }}
              placeholder="ai, automation, ops"
              className="mt-2"
              data-testid={selectors.ideaForm.tagsInput}
              disabled={isSubmitting}
            />
            <p className="mt-1 text-xs text-slate-500">Separate tags with commas.</p>
          </div>

          {displayError && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {displayError}
            </div>
          )}
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <Button
            variant="outline"
            onClick={onClose}
            disabled={isSubmitting}
            data-testid={selectors.ideaForm.cancelButton}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isSubmitting}
            data-testid={selectors.ideaForm.submitButton}
          >
            {isSubmitting ? "Saving..." : isEditMode ? "Save Changes" : "Create Idea"}
          </Button>
        </div>
      </div>
    </div>
  );
}
