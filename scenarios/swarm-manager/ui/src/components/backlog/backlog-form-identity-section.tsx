/**
 * BacklogFormIdentitySection
 *
 * Name / Title / Kind fields extracted from BacklogFormDialog.
 */

import { Input } from "../ui/input";
import { Select } from "../ui/select";
import { selectors } from "../../consts/selectors";
import type { BacklogKind } from "../../types";

// eslint-disable-next-line react-refresh/only-export-components -- helper constant shared with backlog-form-dialog
export const KIND_OPTIONS: Array<{ value: BacklogKind; label: string; helper: string }> = [
  { value: "idea", label: "Idea", helper: "New scenario concepts to evolve into full builds." },
  { value: "research", label: "Research", helper: "Investigations that feed into ideas, fixes, or execution." },
  { value: "fix", label: "Fix", helper: "Targeted fixes to existing scenarios or tooling." },
  { value: "execute", label: "Execute", helper: "Focused tasks to carry out inside the swarm." },
  { value: "chore", label: "Chore", helper: "Maintenance, cleanup, dependency updates, or infrastructure work." },
];

// eslint-disable-next-line react-refresh/only-export-components -- helper function shared with backlog-form-dialog
export const kindLabelFor = (kind: BacklogKind): string =>
  KIND_OPTIONS.find((option) => option.value === kind)?.label ?? "Backlog";

export interface BacklogFormIdentitySectionProps {
  kind: BacklogKind;
  title: string;
  name: string;
  isEditMode: boolean;
  isSubmitting: boolean;
  nameDirty: boolean;
  error: string | null;
  onKindChange: (kind: BacklogKind) => void;
  onTitleChange: (value: string) => void;
  onNameChange: (value: string) => void;
  onNameDirty: () => void;
  onClearError: () => void;
}

export function BacklogFormIdentitySection({
  kind,
  title,
  name,
  isEditMode,
  isSubmitting,
  nameDirty: _nameDirty,
  error: _error,
  onKindChange,
  onTitleChange,
  onNameChange,
  onNameDirty,
  onClearError,
}: BacklogFormIdentitySectionProps) {
  const kindLabel = kindLabelFor(kind);

  return (
    <>
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
                onKindChange(e.target.value as BacklogKind);
                onClearError();
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
          onChange={(e) => onTitleChange(e.target.value)}
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
            onNameChange(e.target.value);
            onNameDirty();
            onClearError();
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
    </>
  );
}
