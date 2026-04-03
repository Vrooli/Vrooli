/**
 * BacklogFormDetailsSection
 *
 * Description / Status / Priority / Tags / Initiative / Dependencies /
 * Effort / Acceptance fields extracted from BacklogFormDialog.
 */

import { Input } from "../ui/input";
import { Select } from "../ui/select";
import { selectors } from "../../consts/selectors";
import type { BacklogStatus } from "../../types";

const STATUS_OPTIONS: BacklogStatus[] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "completed",
  "archived",
];

export interface BacklogFormDetailsSectionProps {
  description: string;
  status: BacklogStatus;
  priority: number;
  tagsInput: string;
  initiative: string | undefined;
  dependsOn: string[] | undefined;
  effort: string | undefined;
  acceptanceAllow: string[] | undefined;
  acceptanceDeny: string[] | undefined;
  isEditMode: boolean;
  isSubmitting: boolean;
  onFieldChange: (field: string, value: unknown) => void;
  onTagsInputChange: (value: string) => void;
  onClearError: () => void;
}

export function BacklogFormDetailsSection({
  description,
  status,
  priority,
  tagsInput,
  initiative,
  dependsOn,
  effort,
  acceptanceAllow,
  acceptanceDeny,
  isEditMode,
  isSubmitting,
  onFieldChange,
  onTagsInputChange,
  onClearError,
}: BacklogFormDetailsSectionProps) {
  return (
    <>
      <div>
        <label htmlFor="backlog-form-description" className="text-sm font-medium text-slate-300">
          Description
        </label>
        <textarea
          id="backlog-form-description"
          value={description}
          onChange={(e) => {
            onFieldChange("description", e.target.value);
            onClearError();
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
              onFieldChange("priority", Number(e.target.value) || 1);
              onClearError();
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
                  onFieldChange("status", e.target.value as BacklogStatus);
                  onClearError();
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
            onTagsInputChange(e.target.value);
            onClearError();
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
            onFieldChange("initiative", e.target.value);
            onClearError();
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
            onFieldChange("dependsOn", deps);
            onClearError();
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
              onFieldChange("effort", e.target.value);
              onClearError();
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
            onFieldChange("acceptanceAllow", patterns);
            onClearError();
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
            onFieldChange("acceptanceDeny", patterns);
            onClearError();
          }}
          placeholder="*.lock, node_modules/**"
          className="mt-2"
          disabled={isSubmitting}
        />
        <p className="mt-1 text-xs text-slate-500">Glob patterns for file paths that must NOT be modified.</p>
      </div>
    </>
  );
}
