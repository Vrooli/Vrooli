import { useEffect } from "react";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { selectors } from "../../consts/selectors";
import { sanitizeBacklogName } from "../../lib";
import type { BacklogFormValues, BacklogKind } from "../../types";
import { useBacklogFormStore } from "../../stores";
import { BacklogFormIdentitySection, kindLabelFor } from "./backlog-form-identity-section";
import { BacklogFormDetailsSection } from "./backlog-form-details-section";

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

  const handleClearError = () => {
    if (error) setError(null);
  };

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
        <BacklogFormIdentitySection
          kind={kind}
          title={title}
          name={name}
          isEditMode={isEditMode}
          isSubmitting={isSubmitting}
          nameDirty={nameDirty}
          error={error}
          onKindChange={(k) => setField("kind", k)}
          onTitleChange={handleTitleChange}
          onNameChange={(v) => setField("name", v)}
          onNameDirty={() => setNameDirty(true)}
          onClearError={handleClearError}
        />

        <BacklogFormDetailsSection
          description={description}
          status={status}
          priority={priority}
          tagsInput={tagsInput}
          initiative={initiative}
          dependsOn={dependsOn}
          effort={effort}
          acceptanceAllow={acceptanceAllow}
          acceptanceDeny={acceptanceDeny}
          isEditMode={isEditMode}
          isSubmitting={isSubmitting}
          onFieldChange={(field, value) => setField(field as keyof BacklogFormValues, value as never)}
          onTagsInputChange={(v) => {
            setTagsInput(v);
            handleClearError();
          }}
          onClearError={handleClearError}
        />

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
