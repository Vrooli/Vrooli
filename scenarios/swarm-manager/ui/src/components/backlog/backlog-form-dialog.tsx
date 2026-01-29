import { useEffect, useMemo, useState } from "react";
import { X } from "lucide-react";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { selectors } from "../../consts/selectors";
import { parseTagsInput, sanitizeBacklogName, tagsToInput } from "../../lib";
import type { BacklogKind, BacklogResearchTarget, BacklogStatus } from "../../types";

export type BacklogFormMode = "create" | "edit";

export interface BacklogFormValues {
  name: string;
  title: string;
  description: string;
  status: BacklogStatus;
  priority: number;
  tags: string[];
  kind: BacklogKind;
  researchTarget?: BacklogResearchTarget;
}

interface BacklogFormDialogProps {
  isOpen: boolean;
  mode: BacklogFormMode;
  initialValues?: BacklogFormValues;
  defaultKind?: BacklogKind;
  isSubmitting?: boolean;
  submitError?: string | null;
  onClose: () => void;
  onSubmit: (values: BacklogFormValues) => void;
}

const STATUS_OPTIONS: BacklogStatus[] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "completed",
  "archived",
];

const KIND_OPTIONS: Array<{ value: BacklogKind; label: string; helper: string }> = [
  { value: "idea", label: "Idea", helper: "New scenario concepts to evolve into full builds." },
  { value: "research", label: "Research", helper: "Investigations that feed into ideas, fixes, or execution." },
  { value: "fix", label: "Fix", helper: "Targeted fixes to existing scenarios or tooling." },
  { value: "execute", label: "Execute", helper: "Focused tasks to carry out inside the swarm." },
];

const RESEARCH_TARGET_OPTIONS: Array<{ value: BacklogResearchTarget; label: string; helper: string }> = [
  { value: "idea", label: "Idea", helper: "Research feeds scenario ideation." },
  { value: "fix", label: "Fix", helper: "Research supports a fix backlog item." },
  { value: "execute", label: "Execute", helper: "Research supports a task to execute." },
  { value: "unspecified", label: "Unspecified", helper: "Open-ended research with no target yet." },
];

const kindLabelFor = (kind: BacklogKind): string =>
  KIND_OPTIONS.find((option) => option.value === kind)?.label ?? "Backlog";

export function BacklogFormDialog({
  isOpen,
  mode,
  initialValues,
  defaultKind = "idea",
  isSubmitting = false,
  submitError = null,
  onClose,
  onSubmit,
}: BacklogFormDialogProps) {
  const [name, setName] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<BacklogStatus>("backlog");
  const [priority, setPriority] = useState(5);
  const [tagsInput, setTagsInput] = useState("");
  const [nameDirty, setNameDirty] = useState(false);
  const [kind, setKind] = useState<BacklogKind>(defaultKind);
  const [researchTarget, setResearchTarget] = useState<BacklogResearchTarget>("idea");
  const [error, setError] = useState<string | null>(null);

  const isEditMode = mode === "edit";

  useEffect(() => {
    if (isOpen) {
      const nextKind = initialValues?.kind ?? defaultKind;
      setName(initialValues?.name ?? "");
      setTitle(initialValues?.title ?? "");
      setDescription(initialValues?.description ?? "");
      setStatus(initialValues?.status ?? "backlog");
      setPriority(initialValues?.priority ?? 5);
      setTagsInput(tagsToInput(initialValues?.tags ?? []));
      setKind(nextKind);
      setResearchTarget(initialValues?.researchTarget ?? "idea");
      setNameDirty(isEditMode);
      setError(null);
    }
  }, [isOpen, initialValues, isEditMode, defaultKind]);

  const handleTitleChange = (value: string) => {
    setTitle(value);
    if (error) setError(null);
    if (!nameDirty && !isEditMode) {
      setName(sanitizeBacklogName(value));
    }
  };

  const derivedTags = useMemo(() => parseTagsInput(tagsInput), [tagsInput]);

  const handleSubmit = () => {
    if (title.trim() === "") {
      setError("Title is required.");
      return;
    }
    const finalName = isEditMode ? name : sanitizeBacklogName(name || title);
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
      kind,
      researchTarget: kind === "research" ? researchTarget : undefined,
    });
  };

  const displayError = error ?? submitError;
  const kindLabel = kindLabelFor(kind);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" data-testid={selectors.backlogForm.dialog}>
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
          {isEditMode ? `Edit ${kindLabel}` : `Create ${kindLabel}`}
        </h2>
        <p className="mt-1 text-sm text-slate-400">
          {isEditMode
            ? "Update backlog details and lifecycle status."
            : "Capture a new backlog item and add it to the swarm."}
        </p>

        <div className="mt-6 space-y-4">
          <div>
            <label htmlFor="backlog-form-kind" className="text-sm font-medium text-slate-300">
              Backlog type
            </label>
            <div className="mt-2">
              {isEditMode ? (
                <div className="rounded-lg border border-white/10 bg-slate-800/50 px-4 py-2 text-sm text-slate-200">
                  {kindLabel}
                </div>
              ) : (
                <select
                  id="backlog-form-kind"
                  value={kind}
                  onChange={(e) => {
                    setKind(e.target.value as BacklogKind);
                    if (error) setError(null);
                  }}
                  className="w-full appearance-none rounded-lg border border-white/10 bg-slate-800/50 px-4 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                  data-testid={selectors.backlogForm.kindSelect}
                  disabled={isSubmitting}
                >
                  {KIND_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              )}
              {!isEditMode && (
                <p className="mt-1 text-xs text-slate-500">
                  {KIND_OPTIONS.find((option) => option.value === kind)?.helper}
                </p>
              )}
            </div>
          </div>

          {kind === "research" && (
            <div>
              <label htmlFor="backlog-form-research-target" className="text-sm font-medium text-slate-300">
                Research target
              </label>
              <div className="mt-2">
                <select
                  id="backlog-form-research-target"
                  value={researchTarget}
                  onChange={(e) => {
                    setResearchTarget(e.target.value as BacklogResearchTarget);
                    if (error) setError(null);
                  }}
                  className="w-full appearance-none rounded-lg border border-white/10 bg-slate-800/50 px-4 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                  data-testid={selectors.backlogForm.researchTargetSelect}
                  disabled={isSubmitting}
                >
                  {RESEARCH_TARGET_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
              <p className="mt-1 text-xs text-slate-500">
                {RESEARCH_TARGET_OPTIONS.find((option) => option.value === researchTarget)?.helper}
              </p>
            </div>
          )}

          <div>
            <label htmlFor="backlog-form-title" className="text-sm font-medium text-slate-300">
              Title
            </label>
            <Input
              id="backlog-form-title"
              value={title}
              onChange={(e) => handleTitleChange(e.target.value)}
              placeholder="Backlog item title"
              className="mt-2"
              data-testid={selectors.backlogForm.titleInput}
              disabled={isSubmitting}
            />
          </div>

          <div>
            <label htmlFor="backlog-form-name" className="text-sm font-medium text-slate-300">
              Name
            </label>
            <Input
              id="backlog-form-name"
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                setNameDirty(true);
                if (error) setError(null);
              }}
              placeholder="folder-safe-name"
              className="mt-2"
              data-testid={selectors.backlogForm.nameInput}
              disabled={isSubmitting || isEditMode}
            />
            {!isEditMode && (
              <p className="mt-1 text-xs text-slate-500">Auto-generated from title if left empty.</p>
            )}
          </div>

          <div>
            <label htmlFor="backlog-form-description" className="text-sm font-medium text-slate-300">
              Description
            </label>
            <textarea
              id="backlog-form-description"
              value={description}
              onChange={(e) => {
                setDescription(e.target.value);
                if (error) setError(null);
              }}
              placeholder="Describe the task, goals, and constraints..."
              className="mt-2 w-full rounded-lg border border-white/10 bg-slate-800/50 px-4 py-3 text-sm text-slate-100 placeholder:text-slate-500 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
              rows={4}
              data-testid={selectors.backlogForm.descriptionInput}
              disabled={isSubmitting}
            />
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <label htmlFor="backlog-form-priority" className="text-sm font-medium text-slate-300">
                Priority (1-10)
              </label>
              <Input
                id="backlog-form-priority"
                type="number"
                min={1}
                max={10}
                value={priority}
                onChange={(e) => {
                  setPriority(Number(e.target.value) || 1);
                  if (error) setError(null);
                }}
                className="mt-2"
                data-testid={selectors.backlogForm.priorityInput}
                disabled={isSubmitting}
              />
            </div>
            {isEditMode ? (
              <div>
                <label htmlFor="backlog-form-status" className="text-sm font-medium text-slate-300">
                  Status
                </label>
                <div className="relative mt-2">
                  <select
                    id="backlog-form-status"
                    value={status}
                    onChange={(e) => {
                      setStatus(e.target.value as BacklogStatus);
                      if (error) setError(null);
                    }}
                    className="w-full appearance-none rounded-lg border border-white/10 bg-slate-800/50 px-4 py-2 text-sm text-slate-100 focus:border-cyan-500 focus:outline-none focus:ring-1 focus:ring-cyan-500"
                    data-testid={selectors.backlogForm.statusSelect}
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
            <label htmlFor="backlog-form-tags" className="text-sm font-medium text-slate-300">
              Tags
            </label>
            <Input
              id="backlog-form-tags"
              value={tagsInput}
              onChange={(e) => {
                setTagsInput(e.target.value);
                if (error) setError(null);
              }}
              placeholder="ai, automation, ops"
              className="mt-2"
              data-testid={selectors.backlogForm.tagsInput}
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
            data-testid={selectors.backlogForm.cancelButton}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={isSubmitting}
            data-testid={selectors.backlogForm.submitButton}
          >
            {isSubmitting ? "Saving..." : isEditMode ? "Save Changes" : `Create ${kindLabel}`}
          </Button>
        </div>
      </div>
    </div>
  );
}
