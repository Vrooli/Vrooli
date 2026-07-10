// SetBaselineModal (Plan B §4.2) — the capture flow.
//
// Every capture is one comprehensive baseline-profile Test Genie run. The
// current branch is auto-detected and a dirty tree remains explicit.

import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, Anchor, Loader2, X } from "lucide-react";
import { Button } from "../../components/ui/button";
import { MutationErrorBanner } from "../../components/ScenarioReviewPanelShared";
import { useIsMobile } from "../../hooks";
import { useRepoStatus } from "../../lib/hooks-core";
import { useSnapshotBaseline } from "../../lib/hooks-baselines";
import { dirtyStateFromStatus } from "./model";

interface SetBaselineModalProps {
  isOpen: boolean;
  scenario: string;
  repoId?: string | null;
  onClose: () => void;
  onCreated: (name: string) => void;
}

export function SetBaselineModal({
  isOpen,
  scenario,
  repoId,
  onClose,
  onCreated,
}: SetBaselineModalProps) {
  const isMobile = useIsMobile();
  const repoStatus = useRepoStatus(repoId);
  const create = useSnapshotBaseline(repoId);

  const branch = repoStatus.data?.branch?.head ?? "";
  const dirty = useMemo(() => dirtyStateFromStatus(repoStatus.data), [repoStatus.data]);

  const [name, setName] = useState("");

  // Reset form each time the modal opens.
  useEffect(() => {
    if (isOpen) {
      setName("");
      create.reset();
    }
    // create is stable from useMutation.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen]);

  if (!isOpen) return null;

  const canSubmit = name.trim().length > 0 && !create.isPending;

  const handleCapture = () => {
    if (!canSubmit) return;
    create.mutate(
      {
        scenario,
        name: name.trim(),
        branch,
      },
      {
        onSuccess: () => {
          onCreated(name.trim());
          onClose();
        },
      },
    );
  };

  const body = (
    <div className="space-y-4">
      <div className="space-y-1">
        <label htmlFor="baseline-name" className="text-xs font-medium text-slate-400">
          Name
        </label>
        <input
          id="baseline-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="pre-launch-1.3"
          autoFocus
          className="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-600 focus:border-blue-500 focus:outline-none"
        />
      </div>

      <div className="text-xs text-slate-400">
        Branch:{" "}
        <span className="font-mono text-slate-200">
          {branch || (repoStatus.isLoading ? "detecting…" : "unknown")}
        </span>{" "}
        (auto-detected)
      </div>

      {dirty.dirty && (
        <div className="flex items-start gap-2 rounded-lg border border-amber-800 bg-amber-950/40 p-3">
          <AlertTriangle className="h-4 w-4 text-amber-400 mt-0.5 shrink-0" />
          <p className="text-xs text-amber-300">
            Working tree is dirty ({dirty.modified} changed). This run will reflect
            uncommitted changes, not the committed state.
          </p>
        </div>
      )}

      <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-3 text-xs text-slate-400">
        Captures one comprehensive run with full descriptor and typed evidence catalogs.
      </div>

      {create.isPending && (
        <div className="flex items-center gap-2 text-xs text-blue-300">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          Capturing comprehensive run — this can take a few minutes.
        </div>
      )}
      <MutationErrorBanner error={create.error} onDismiss={() => create.reset()} />
    </div>
  );

  const footer = (
    <>
      <Button variant="outline" size="sm" onClick={onClose} disabled={create.isPending} className="h-8 px-3">
        Cancel
      </Button>
      <Button size="sm" onClick={handleCapture} disabled={!canSubmit} className="h-8 px-3 gap-1.5">
        {create.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Anchor className="h-3.5 w-3.5" />}
        Capture
      </Button>
    </>
  );

  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Set baseline"
      >
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <h2 className="text-base font-semibold text-slate-100">Set baseline</h2>
          <button
            type="button"
            className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 touch-target"
            onClick={onClose}
            aria-label="Close"
            disabled={create.isPending}
          >
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
      aria-label="Set baseline"
      onClick={(e) => {
        if (e.target === e.currentTarget && !create.isPending) onClose();
      }}
    >
      <div className="w-full max-w-md rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-100">Set baseline for {scenario}</h2>
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60"
            onClick={onClose}
            disabled={create.isPending}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="px-4 py-4 max-h-[70vh] overflow-y-auto">{body}</div>
        <div className="flex items-center justify-end gap-2 border-t border-slate-800 px-4 py-3">{footer}</div>
      </div>
    </div>
  );
}
