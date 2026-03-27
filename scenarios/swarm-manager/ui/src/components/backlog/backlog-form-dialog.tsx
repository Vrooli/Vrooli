import { useEffect } from "react";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { Input } from "../ui/input";
import { Select } from "../ui/select";
import { selectors } from "../../consts/selectors";
import { sanitizeBacklogName } from "../../lib";
import type { BacklogFormValues, BacklogKind, BacklogStatus } from "../../types";
import { useBacklogFormStore } from "../../stores";

export type BacklogFormMode = "create" | "edit";

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
  { value: "chore", label: "Chore", helper: "Maintenance, cleanup, dependency updates, or infrastructure work." },
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
  const values = useBacklogFormStore((state) => state.values);
  const tagsInput = useBacklogFormStore((state) => state.tagsInput);
  const nameDirty = useBacklogFormStore((state) => state.nameDirty);
  const error = useBacklogFormStore((state) => state.error);
  const setField = useBacklogFormStore((state) => state.setField);
  const setTagsInput = useBacklogFormStore((state) => state.setTagsInput);
  const setNameDirty = useBacklogFormStore((state) => state.setNameDirty);
  const setError = useBacklogFormStore((state) => state.setError);
  const initialize = useBacklogFormStore((state) => state.initialize);
  const { name, title, description, status, priority, kind, tags, initiative, dependsOn, effort, acceptanceAllow, acceptanceDeny } = values;

  const isEditMode = mode === "edit";

  useEffect(() => {
    if (isOpen) {
      initialize({ isEditMode, defaultKind, initialValues });
    }
  }, [isOpen, initialValues, isEditMode, defaultKind, initialize]);

  const handleTitleChange = (value: string) => {
    setField("title", value);
    if (error) setError(null);
    if (!nameDirty && !isEditMode) {
      setField("name", sanitizeBacklogName(value));
    }
  };

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
      tags,
      kind,
      dependsOn: dependsOn && dependsOn.length > 0 ? dependsOn : undefined,
      initiative: initiative?.trim() || undefined,
      effort: effort?.trim() || undefined,
      acceptanceAllow: acceptanceAllow && acceptanceAllow.length > 0 ? acceptanceAllow : undefined,
      acceptanceDeny: acceptanceDeny && acceptanceDeny.length > 0 ? acceptanceDeny : undefined,
    });
  };

  const displayError = error ?? submitError;
  const kindLabel = kindLabelFor(kind);

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      maxWidth="max-w-xl"
      isLoading={isSubmitting}
      testId={selectors.backlogForm.dialog}
    >
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
              <Select
                id="backlog-form-kind"
                value={kind}
                onChange={(e) => {
                  setField("kind", e.target.value as BacklogKind);
                  if (error) setError(null);
                }}
                data-testid={selectors.backlogForm.kindSelect}
                disabled={isSubmitting}
              >
                {KIND_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
            )}
            {!isEditMode && (
              <p className="mt-1 text-xs text-slate-500">
                {KIND_OPTIONS.find((option) => option.value === kind)?.helper}
              </p>
            )}
          </div>
        </div>

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
              setField("name", e.target.value);
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
              setField("description", e.target.value);
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
                setField("priority", Number(e.target.value) || 1);
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
              <div className="mt-2">
                <Select
                  id="backlog-form-status"
                  value={status}
                  onChange={(e) => {
                    setField("status", e.target.value as BacklogStatus);
                    if (error) setError(null);
                  }}
                  data-testid={selectors.backlogForm.statusSelect}
                  disabled={isSubmitting}
                >
                  {STATUS_OPTIONS.map((option) => (
                    <option key={option} value={option}>
                      {option.replace(/_/g, " ")}
                    </option>
                  ))}
                </Select>
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

        <div>
          <label htmlFor="backlog-form-initiative" className="text-sm font-medium text-slate-300">
            Initiative
          </label>
          <Input
            id="backlog-form-initiative"
            value={initiative ?? ""}
            onChange={(e) => {
              setField("initiative", e.target.value);
              if (error) setError(null);
            }}
            placeholder="e.g. core-billing"
            className="mt-2"
            disabled={isSubmitting}
          />
          <p className="mt-1 text-xs text-slate-500">Optional initiative grouping.</p>
        </div>

        <div>
          <label htmlFor="backlog-form-depends-on" className="text-sm font-medium text-slate-300">
            Dependencies
          </label>
          <Input
            id="backlog-form-depends-on"
            value={(dependsOn ?? []).join(", ")}
            onChange={(e) => {
              const deps = e.target.value
                .split(",")
                .map((s) => s.trim())
                .filter(Boolean);
              setField("dependsOn", deps);
              if (error) setError(null);
            }}
            placeholder="e.g. fix/auth-bug, idea/dashboard"
            className="mt-2"
            disabled={isSubmitting}
          />
          <p className="mt-1 text-xs text-slate-500">Comma-separated kind/name references.</p>
        </div>

        <div>
          <label htmlFor="backlog-form-effort" className="text-sm font-medium text-slate-300">
            Effort
          </label>
          <div className="mt-2">
            <Select
              id="backlog-form-effort"
              value={effort ?? ""}
              onChange={(e) => {
                setField("effort", e.target.value);
                if (error) setError(null);
              }}
              disabled={isSubmitting}
            >
              <option value="">-- None --</option>
              <option value="XS">XS</option>
              <option value="S">S</option>
              <option value="M">M</option>
              <option value="L">L</option>
              <option value="XL">XL</option>
            </Select>
          </div>
        </div>

        <div>
          <label htmlFor="backlog-form-acceptance-allow" className="text-sm font-medium text-slate-300">
            Acceptance Allow
          </label>
          <Input
            id="backlog-form-acceptance-allow"
            value={(acceptanceAllow ?? []).join(", ")}
            onChange={(e) => {
              const patterns = e.target.value
                .split(",")
                .map((s) => s.trim())
                .filter(Boolean);
              setField("acceptanceAllow", patterns);
              if (error) setError(null);
            }}
            placeholder="src/**, docs/**"
            className="mt-2"
            disabled={isSubmitting}
          />
          <p className="mt-1 text-xs text-slate-500">Glob patterns for file paths expected to be modified.</p>
        </div>

        <div>
          <label htmlFor="backlog-form-acceptance-deny" className="text-sm font-medium text-slate-300">
            Acceptance Deny
          </label>
          <Input
            id="backlog-form-acceptance-deny"
            value={(acceptanceDeny ?? []).join(", ")}
            onChange={(e) => {
              const patterns = e.target.value
                .split(",")
                .map((s) => s.trim())
                .filter(Boolean);
              setField("acceptanceDeny", patterns);
              if (error) setError(null);
            }}
            placeholder="*.lock, node_modules/**"
            className="mt-2"
            disabled={isSubmitting}
          />
          <p className="mt-1 text-xs text-slate-500">Glob patterns for file paths that must NOT be modified.</p>
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
    </Dialog>
  );
}
