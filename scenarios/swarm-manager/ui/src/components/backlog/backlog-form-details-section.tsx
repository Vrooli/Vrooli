/**
 * BacklogFormDetailsSection
 *
 * Description / Status / Priority / Tags / Milestone / Dependencies /
 * Effort / Acceptance fields extracted from BacklogFormDialog.
 */

import { useEffect, useState } from "react";
import { Input } from "../ui/input";
import { Select } from "../ui/select";
import { selectors } from "../../consts/selectors";
import type { BacklogStatus } from "../../types";
import type { ExecutionLimits } from "../../types";
import { defaultApiClient } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";

const DEFAULT_LIMITS: ExecutionLimits = { maxSlices: 128, maxTokens: 2000000, maxWallSeconds: 604800, maxTurns: 2400, maxChargeMicroUsd: 250000000, maxChildren: 512, maxNodeAttempts: 512, maxRetries: 128 };
const LIMIT_FIELDS: Array<{ key: keyof ExecutionLimits; label: string }> = [
  { key: "maxSlices", label: "Maximum slices" }, { key: "maxTokens", label: "Maximum tokens" },
  { key: "maxWallSeconds", label: "Maximum wall seconds" }, { key: "maxTurns", label: "Maximum turns" },
  { key: "maxChargeMicroUsd", label: "Maximum charge (micro-USD)" }, { key: "maxChildren", label: "Maximum child runs" },
  { key: "maxNodeAttempts", label: "Maximum node attempts" }, { key: "maxRetries", label: "Maximum retries" },
];

const STATUS_OPTIONS: BacklogStatus[] = [
  "backlog",
  "researching",
  "ready",
  "queued",
  "in_progress",
  "completed",
  "failed",
];

export interface BacklogFormDetailsSectionProps {
  description: string;
  status: BacklogStatus;
  priority: number;
  tagsInput: string;
  milestone: string | undefined;
  dependsOn: string[] | undefined;
  effort: string | undefined;
  acceptanceAllow: string[] | undefined;
  acceptanceDeny: string[] | undefined;
  executionStrategy: string | undefined;
  executionLimits: ExecutionLimits | undefined;
  continuation: string | undefined;
  scopePolicy: string | undefined;
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
  milestone,
  dependsOn,
  effort,
  acceptanceAllow,
  acceptanceDeny,
  executionStrategy,
  executionLimits,
  continuation,
  scopePolicy,
  isEditMode,
  isSubmitting,
  onFieldChange,
  onTagsInputChange,
  onClearError,
}: BacklogFormDetailsSectionProps) {
  const [strategies, setStrategies] = useState<Array<{ id: string; display_name: string }>>([]);
  useEffect(() => {
    let active = true;
    void defaultApiClient.get<{ items?: Array<{ id: string; display_name: string }> }>(API_ENDPOINTS.executionStrategies)
      .then((response) => { if (active) setStrategies(response.items ?? []); })
      .catch(() => { if (active) setStrategies([]); });
    return () => { active = false; };
  }, []);

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
        <label htmlFor="backlog-form-milestone" className="text-sm font-medium text-slate-300">
          Milestone
        </label>
        <Input
          id="backlog-form-milestone"
          value={milestone ?? ""}
          onChange={(e) => {
            onFieldChange("milestone", e.target.value);
            onClearError();
          }}
          placeholder="e.g. core-billing"
          className="mt-2"
          disabled={isSubmitting}
        />
        <p className="mt-1 text-xs text-slate-500">Optional milestone grouping.</p>
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

      <section className="space-y-4 rounded-lg border border-cyan-300/15 bg-cyan-300/[0.03] p-4" aria-label="Execution">
        <div>
          <h3 className="text-sm font-semibold text-slate-100">Execution</h3>
          <p className="mt-1 text-xs leading-5 text-slate-400">These reviewed fields control the unattended route. Editing them invalidates plan acceptance and requires review again.</p>
        </div>
        <div>
          <label htmlFor="backlog-form-execution-strategy" className="text-sm font-medium text-slate-300">Execution strategy</label>
          <Select id="backlog-form-execution-strategy" value={executionStrategy ?? "phased-plan-drain"} onChange={(e) => { onFieldChange("executionStrategy", e.target.value); onClearError(); }} disabled={isSubmitting}>
            {strategies.length === 0 && <option value={executionStrategy ?? "phased-plan-drain"}>{executionStrategy ?? "phased-plan-drain"}</option>}
            {strategies.map((option) => <option key={option.id} value={option.id}>{option.display_name || option.id}</option>)}
          </Select>
        </div>
        <div>
          <p className="text-sm font-medium text-slate-300">Reviewed execution limits</p>
          <div className="mt-2 grid gap-3 sm:grid-cols-2">
            {LIMIT_FIELDS.map(({ key, label }) => <label key={key} className="text-xs text-slate-400">{label}
              <Input type="number" min={0} value={executionLimits?.[key] ?? ""} placeholder={String(DEFAULT_LIMITS[key])} onChange={(e) => { onFieldChange("executionLimits", { ...DEFAULT_LIMITS, ...(executionLimits ?? {}), [key]: Number(e.target.value) || 0 }); onClearError(); }} disabled={isSubmitting} />
            </label>)}
          </div>
        </div>
        <fieldset>
          <legend className="text-sm font-medium text-slate-300">Continuation</legend>
          <label className="mt-2 flex gap-2 text-xs text-slate-300"><input type="radio" checked={(continuation ?? "manual") === "manual"} onChange={() => onFieldChange("continuation", "manual")} disabled={isSubmitting} />Manual — stop after this run.</label>
          <label className="mt-2 flex gap-2 text-xs text-slate-300"><input type="radio" checked={continuation === "until-allowance"} onChange={() => onFieldChange("continuation", "until-allowance")} disabled={isSubmitting} />Until allowance is spent — continue automatically while safe.</label>
        </fieldset>
        <fieldset>
          <legend className="text-sm font-medium text-slate-300">Scope policy</legend>
          <label className="mt-2 flex gap-2 text-xs text-slate-300"><input type="radio" checked={(scopePolicy ?? "fixed") === "fixed"} onChange={() => onFieldChange("scopePolicy", "fixed")} disabled={isSubmitting} />Fixed — out-of-scope edits require an operator decision.</label>
          <label className="mt-2 flex gap-2 text-xs text-slate-300"><input type="radio" checked={scopePolicy === "extend-with-record"} onChange={() => onFieldChange("scopePolicy", "extend-with-record")} disabled={isSubmitting} />Extend with record — record a reason in Plan Manager before widening.</label>
        </fieldset>
      </section>
    </>
  );
}
