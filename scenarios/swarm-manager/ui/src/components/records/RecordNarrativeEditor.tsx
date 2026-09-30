/**
 * RecordNarrativeEditor — one-shot form to fill a stub record's narrative.
 *
 * Refuses to render on non-stub records; once a record has narrative content,
 * amendments require writing a superseding record.
 */

import { useState } from "react";
import type { RecordItem, RecordNarrativeInput, RecordOutcome } from "../../types";
import { ALL_RECORD_OUTCOMES } from "../../types";

interface RecordNarrativeEditorProps {
  record: RecordItem;
  onSubmit: (input: RecordNarrativeInput) => Promise<void>;
  onCancel?: () => void;
}

export function RecordNarrativeEditor({ record, onSubmit, onCancel }: RecordNarrativeEditorProps) {
  const [trigger, setTrigger] = useState(record.trigger);
  const [approach, setApproach] = useState(record.approach);
  const [ruledOutText, setRuledOutText] = useState(record.ruledOut.join("\n"));
  const [commit, setCommit] = useState(record.commit ?? "");
  const [filesText, setFilesText] = useState(record.filesChanged.join("\n"));
  const [outcome, setOutcome] = useState<RecordOutcome>(record.outcome);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!record.stub) {
    return (
      <div className="rounded border border-amber-700 bg-amber-950/40 p-3 text-sm text-amber-200">
        This record's narrative is already filled. Amendments require creating a superseding record.
      </div>
    );
  }

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!trigger.trim() || !approach.trim()) {
      setError("Trigger and approach are required.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await onSubmit({
        trigger: trigger.trim(),
        approach: approach.trim(),
        ruledOut: ruledOutText.split("\n").map((s) => s.trim()).filter(Boolean),
        commit: commit.trim() || undefined,
        filesChanged: filesText.split("\n").map((s) => s.trim()).filter(Boolean),
        outcome,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="flex flex-col gap-3" data-testid="record-narrative-editor">
      <label className="flex flex-col gap-1 text-sm text-slate-300">
        Trigger (one-line symptom or goal)
        <input
          type="text"
          className="rounded border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100"
          value={trigger}
          onChange={(e) => setTrigger(e.target.value)}
          data-testid="record-trigger-input"
        />
      </label>
      <label className="flex flex-col gap-1 text-sm text-slate-300">
        Approach (root cause + what the fix does)
        <textarea
          className="min-h-24 rounded border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100"
          value={approach}
          onChange={(e) => setApproach(e.target.value)}
          data-testid="record-approach-input"
        />
      </label>
      <label className="flex flex-col gap-1 text-sm text-slate-300">
        Ruled out (one per line)
        <textarea
          className="min-h-16 rounded border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100"
          value={ruledOutText}
          onChange={(e) => setRuledOutText(e.target.value)}
          data-testid="record-ruled-out-input"
        />
      </label>
      <div className="grid grid-cols-2 gap-3">
        <label className="flex flex-col gap-1 text-sm text-slate-300">
          Commit
          <input
            type="text"
            className="rounded border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100"
            value={commit}
            onChange={(e) => setCommit(e.target.value)}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm text-slate-300">
          Outcome
          <select
            className="rounded border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100"
            value={outcome}
            onChange={(e) => setOutcome(e.target.value as RecordOutcome)}
          >
            {ALL_RECORD_OUTCOMES.map((o) => (
              <option key={o} value={o}>
                {o}
              </option>
            ))}
          </select>
        </label>
      </div>
      <label className="flex flex-col gap-1 text-sm text-slate-300">
        Files changed (one path per line)
        <textarea
          className="min-h-16 rounded border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100"
          value={filesText}
          onChange={(e) => setFilesText(e.target.value)}
        />
      </label>

      {error ? (
        <div className="rounded border border-red-700 bg-red-950/40 p-2 text-sm text-red-200">{error}</div>
      ) : null}

      <div className="flex items-center gap-2">
        <button
          type="submit"
          disabled={busy}
          className="rounded bg-emerald-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-50"
          data-testid="record-narrative-submit"
        >
          {busy ? "Saving…" : "Fill narrative"}
        </button>
        {onCancel ? (
          <button
            type="button"
            onClick={onCancel}
            className="rounded border border-slate-700 px-3 py-2 text-sm text-slate-200"
          >
            Cancel
          </button>
        ) : null}
      </div>
    </form>
  );
}
