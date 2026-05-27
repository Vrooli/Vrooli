// BaselineEditModal (Plan B §4.2) — power-user surface pointer swap.
//
// EditBaseline re-points one surface at a different (already-pinned) test-genie
// run, so only the test-genie-backed surfaces (workflows, tests) are editable.
// A run picker that lists historical runs arrives with the Workflows tab's
// WorkflowReplayService; until then this takes an explicit run ID.

import { useEffect, useState } from "react";
import { Loader2, X } from "lucide-react";
import { Button } from "../../components/ui/button";
import { MutationErrorBanner } from "../../components/ScenarioReviewPanelShared";
import { useIsMobile } from "../../hooks";
import { useEditBaseline } from "../../lib/hooks-baselines";
import { SURFACE_META } from "./model";
import type { BaselineManifest } from "@vrooli/proto-types/git-control-tower/v1/baselines/baselines_pb";

// Only surfaces backed by test-genie runs can be re-pointed by run ID.
const EDITABLE_SURFACES = ["workflows", "tests"] as const;
type EditableSurface = (typeof EDITABLE_SURFACES)[number];

interface BaselineEditModalProps {
  isOpen: boolean;
  scenario: string;
  baseline: BaselineManifest;
  repoId?: string | null;
  onClose: () => void;
}

export function BaselineEditModal({ isOpen, scenario, baseline, repoId, onClose }: BaselineEditModalProps) {
  const isMobile = useIsMobile();
  const edit = useEditBaseline(repoId);
  const [surface, setSurface] = useState<EditableSurface>("workflows");
  const [runId, setRunId] = useState("");

  useEffect(() => {
    if (isOpen) {
      setSurface("workflows");
      setRunId("");
      edit.reset();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  if (!isOpen) return null;

  const currentRef = baseline.surfaces[surface]?.ref ?? "(not captured)";
  const canSubmit = runId.trim().length > 0 && !edit.isPending;

  const handleSave = () => {
    if (!canSubmit) return;
    edit.mutate(
      { scenario, name: baseline.name, branch: baseline.branch, surface, pinRunId: runId.trim(), reason: "ui-edit" },
      { onSuccess: () => onClose() },
    );
  };

  const body = (
    <div className="space-y-4">
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-400">Surface</label>
        <div className="flex gap-2">
          {EDITABLE_SURFACES.map((s) => (
            <button
              key={s}
              type="button"
              onClick={() => setSurface(s)}
              className={`px-3 py-1.5 rounded-lg border text-xs transition-colors ${
                surface === s
                  ? "border-blue-500 bg-blue-950/40 text-blue-300"
                  : "border-slate-700 text-slate-400 hover:text-slate-200"
              }`}
            >
              {SURFACE_META[s].label}
            </button>
          ))}
        </div>
      </div>

      <div className="text-xs text-slate-500">
        Current pointer: <span className="font-mono text-slate-300 break-all">{currentRef}</span>
      </div>

      <div className="space-y-1">
        <label htmlFor="edit-run-id" className="text-xs font-medium text-slate-400">
          New test-genie run ID
        </label>
        <input
          id="edit-run-id"
          type="text"
          value={runId}
          onChange={(e) => setRunId(e.target.value)}
          placeholder="run-id to pin"
          className="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 font-mono focus:border-blue-500 focus:outline-none"
        />
      </div>

      <MutationErrorBanner error={edit.error} onDismiss={() => edit.reset()} />
    </div>
  );

  const footer = (
    <>
      <Button variant="outline" size="sm" onClick={onClose} disabled={edit.isPending} className="h-8 px-3">
        Cancel
      </Button>
      <Button size="sm" onClick={handleSave} disabled={!canSubmit} className="h-8 px-3 gap-1.5">
        {edit.isPending && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
        Save
      </Button>
    </>
  );

  if (isMobile) {
    return (
      <div className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200" role="dialog" aria-modal="true" aria-label="Edit baseline">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <h2 className="text-base font-semibold text-slate-100">Edit baseline</h2>
          <button type="button" className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 touch-target" onClick={onClose} aria-label="Close" disabled={edit.isPending}>
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-6">{body}</div>
        <div className="border-t border-slate-800 px-4 py-4 pb-safe flex gap-3 [&>button]:flex-1 [&>button]:h-12">{footer}</div>
      </div>
    );
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="Edit baseline"
      onClick={(e) => {
        if (e.target === e.currentTarget && !edit.isPending) onClose();
      }}
    >
      <div className="w-full max-w-md rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-100">Edit {baseline.name}</h2>
          <button type="button" className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60" onClick={onClose} disabled={edit.isPending} aria-label="Close">
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="px-4 py-4">{body}</div>
        <div className="flex items-center justify-end gap-2 border-t border-slate-800 px-4 py-3">{footer}</div>
      </div>
    </div>
  );
}
