/**
 * RetentionTab — diff-archive retention configuration.
 *
 * Three levers, each individually disable-able by setting the value
 * to 0:
 *   - maxArchiveAgeDays:      drop archives older than N days
 *   - maxArchiveSizeBytes:    keep total disk usage under N bytes
 *   - maxArchivesPerProject:  keep at most N archives per projectRoot
 *
 * The UI accepts human-readable byte sizes (e.g. "10 GiB", "500 MB")
 * and parses them to the integer byte count the API expects. We display
 * back the canonical IEC form so round-tripping a value doesn't
 * introduce silent drift.
 *
 * Submission semantics: `PUT /config/retention` accepts partial
 * updates; we only send fields the user actually changed since the
 * tab was opened. This avoids stomping out-of-band edits made via the
 * CLI or env vars between open and submit.
 */

import { useEffect, useMemo, useState } from "react";
import { Archive, Check, Info, Loader2, RotateCcw } from "lucide-react";

import { Button } from "../ui/button";
import { Input, Label } from "../ui/input";
import {
  useRetentionConfig,
  useUpdateRetentionConfig,
} from "../../lib/hooks";
import type { RetentionConfig, RetentionUpdate } from "../../lib/api";
import { bytesToHumanReadable, parseHumanBytes, parseNonNegativeInt } from "./byteFormat";

const DAY = 86_400;

interface FormState {
  maxArchiveAgeDays: string;
  maxArchiveSizeInput: string;
  maxArchivesPerProject: string;
}

function fromConfig(cfg: RetentionConfig): FormState {
  return {
    maxArchiveAgeDays: String(cfg.maxArchiveAgeDays),
    maxArchiveSizeInput: bytesToHumanReadable(cfg.maxArchiveSizeBytes),
    maxArchivesPerProject: String(cfg.maxArchivesPerProject),
  };
}

interface FieldHelpProps {
  children: React.ReactNode;
}
function FieldHelp({ children }: FieldHelpProps) {
  return (
    <p className="text-xs text-slate-500 mt-1 flex items-start gap-1.5">
      <Info className="h-3 w-3 mt-0.5 flex-shrink-0" />
      <span>{children}</span>
    </p>
  );
}

export function RetentionTab() {
  const query = useRetentionConfig();
  const mutation = useUpdateRetentionConfig();

  const [form, setForm] = useState<FormState | null>(null);
  // Snapshot of the form at last successful load — used to compute
  // "what changed" so we send a partial PUT.
  const [base, setBase] = useState<FormState | null>(null);
  const [savedFlash, setSavedFlash] = useState(false);

  useEffect(() => {
    if (query.data) {
      const next = fromConfig(query.data);
      setForm(next);
      setBase(next);
    }
  }, [query.data]);

  const dirty = useMemo(() => {
    if (!form || !base) return false;
    return (
      form.maxArchiveAgeDays !== base.maxArchiveAgeDays ||
      form.maxArchiveSizeInput !== base.maxArchiveSizeInput ||
      form.maxArchivesPerProject !== base.maxArchivesPerProject
    );
  }, [form, base]);

  const validation = useMemo(() => {
    if (!form) return { errors: {} as Record<string, string>, allValid: false };
    const errors: Record<string, string> = {};
    if (parseNonNegativeInt(form.maxArchiveAgeDays) === null) {
      errors.maxArchiveAgeDays = "Must be a non-negative integer (0 to disable).";
    }
    if (parseHumanBytes(form.maxArchiveSizeInput) === null) {
      errors.maxArchiveSizeInput = 'Use a number with optional unit, e.g. "10 GiB" or "0" to disable.';
    }
    if (parseNonNegativeInt(form.maxArchivesPerProject) === null) {
      errors.maxArchivesPerProject = "Must be a non-negative integer (0 = unlimited).";
    }
    return { errors, allValid: Object.keys(errors).length === 0 };
  }, [form]);

  if (query.isLoading || !form) {
    return (
      <div className="flex items-center justify-center py-12 text-slate-500">
        <Loader2 className="h-5 w-5 animate-spin mr-2" />
        <span className="text-sm">Loading retention configuration...</span>
      </div>
    );
  }

  if (query.isError) {
    return (
      <div className="p-6 rounded-lg bg-red-950/30 border border-red-800/50" data-testid="retention-error">
        <p className="text-sm text-red-300">Failed to load retention configuration.</p>
        <p className="text-xs text-slate-400 mt-1">{query.error?.message}</p>
        <Button variant="outline" size="sm" className="mt-3" onClick={() => query.refetch()}>
          <RotateCcw className="h-3.5 w-3.5 mr-1.5" /> Retry
        </Button>
      </div>
    );
  }

  const handleReset = () => {
    if (base) setForm(base);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!validation.allValid || !base) return;

    const update: RetentionUpdate = {};
    if (form.maxArchiveAgeDays !== base.maxArchiveAgeDays) {
      update.maxArchiveAgeDays = parseNonNegativeInt(form.maxArchiveAgeDays)!;
    }
    if (form.maxArchiveSizeInput !== base.maxArchiveSizeInput) {
      update.maxArchiveSizeBytes = parseHumanBytes(form.maxArchiveSizeInput)!;
    }
    if (form.maxArchivesPerProject !== base.maxArchivesPerProject) {
      update.maxArchivesPerProject = parseNonNegativeInt(form.maxArchivesPerProject)!;
    }
    if (Object.keys(update).length === 0) return;

    mutation.mutate(update, {
      onSuccess: (cfg) => {
        const next = fromConfig(cfg);
        setForm(next);
        setBase(next);
        setSavedFlash(true);
        window.setTimeout(() => setSavedFlash(false), 2000);
      },
    });
  };

  const ageDays = parseNonNegativeInt(form.maxArchiveAgeDays);
  const sizeBytes = parseHumanBytes(form.maxArchiveSizeInput);
  const perProject = parseNonNegativeInt(form.maxArchivesPerProject);

  return (
    <form onSubmit={handleSubmit} data-testid="retention-tab" className="space-y-6 py-2">
      <div className="flex items-start gap-3 p-4 rounded-lg bg-slate-900/50 border border-slate-800">
        <Archive className="h-5 w-5 text-slate-400 flex-shrink-0 mt-0.5" />
        <div>
          <h3 className="text-sm font-medium text-slate-200">Diff-archive retention</h3>
          <p className="text-xs text-slate-400 mt-1 leading-relaxed">
            When a sandbox transitions to a terminal state (Approved, Rejected, Deleted) it
            persists a snapshot of its diff. These archives are governed by the levers below
            and pruned on each pass of the <code className="text-slate-300">archive-retention</code>{" "}
            reconciler. Set any field to <code className="text-slate-300">0</code> to disable
            that lever individually.
          </p>
        </div>
      </div>

      <div>
        <Label htmlFor="retention-age-days">Maximum age (days)</Label>
        <Input
          id="retention-age-days"
          type="text"
          inputMode="numeric"
          value={form.maxArchiveAgeDays}
          onChange={(e) =>
            setForm({ ...form, maxArchiveAgeDays: e.target.value })
          }
          aria-invalid={!!validation.errors.maxArchiveAgeDays}
          data-testid="retention-age-days"
          className="mt-1"
        />
        {validation.errors.maxArchiveAgeDays ? (
          <p className="text-xs text-red-400 mt-1" data-testid="retention-age-days-error">
            {validation.errors.maxArchiveAgeDays}
          </p>
        ) : (
          <FieldHelp>
            Archives older than this are evicted. {ageDays === 0 ? "Currently disabled." : null}
            {ageDays !== null && ageDays > 0 ? ` ≈ ${(ageDays * DAY).toLocaleString()} seconds.` : null}
          </FieldHelp>
        )}
      </div>

      <div>
        <Label htmlFor="retention-size">Maximum total archive size</Label>
        <Input
          id="retention-size"
          type="text"
          value={form.maxArchiveSizeInput}
          onChange={(e) =>
            setForm({ ...form, maxArchiveSizeInput: e.target.value })
          }
          placeholder="e.g. 10 GiB"
          aria-invalid={!!validation.errors.maxArchiveSizeInput}
          data-testid="retention-size"
          className="mt-1"
        />
        {validation.errors.maxArchiveSizeInput ? (
          <p className="text-xs text-red-400 mt-1" data-testid="retention-size-error">
            {validation.errors.maxArchiveSizeInput}
          </p>
        ) : (
          <FieldHelp>
            Eviction starts at the oldest archive when total disk usage exceeds this budget.{" "}
            {sizeBytes === 0
              ? "Currently disabled."
              : sizeBytes !== null
                ? `${sizeBytes.toLocaleString()} bytes.`
                : null}
          </FieldHelp>
        )}
      </div>

      <div>
        <Label htmlFor="retention-per-project">Maximum archives per project</Label>
        <Input
          id="retention-per-project"
          type="text"
          inputMode="numeric"
          value={form.maxArchivesPerProject}
          onChange={(e) =>
            setForm({ ...form, maxArchivesPerProject: e.target.value })
          }
          aria-invalid={!!validation.errors.maxArchivesPerProject}
          data-testid="retention-per-project"
          className="mt-1"
        />
        {validation.errors.maxArchivesPerProject ? (
          <p className="text-xs text-red-400 mt-1" data-testid="retention-per-project-error">
            {validation.errors.maxArchivesPerProject}
          </p>
        ) : (
          <FieldHelp>
            Per-project cap on number of archive rows. {perProject === 0 ? "Unlimited." : null}
          </FieldHelp>
        )}
      </div>

      {mutation.isError && (
        <div className="p-3 rounded-lg bg-red-950/30 border border-red-800/50 text-xs text-red-300" data-testid="retention-save-error">
          Failed to save: {mutation.error?.message}
        </div>
      )}

      <div className="flex items-center justify-end gap-2 pt-4 border-t border-slate-800">
        {savedFlash && (
          <span
            className="flex items-center gap-1.5 text-xs text-emerald-400"
            data-testid="retention-saved"
          >
            <Check className="h-3.5 w-3.5" />
            Saved
          </span>
        )}
        <Button
          type="button"
          variant="outline"
          onClick={handleReset}
          disabled={!dirty || mutation.isPending}
          data-testid="retention-reset"
        >
          Reset
        </Button>
        <Button
          type="submit"
          disabled={!dirty || !validation.allValid || mutation.isPending}
          data-testid="retention-save"
        >
          {mutation.isPending ? (
            <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />
          ) : null}
          Save
        </Button>
      </div>
    </form>
  );
}
