import { useEffect, useState } from "react";
import { Play, Save } from "lucide-react";
import { Button } from "./ui/button";
import { usePrecommitConfig, useRunPrecommit, useSavePrecommitConfig } from "../lib/hooks-core";
import type { PrecommitConfig } from "../lib/api";

interface SettingsTabPrecommitProps {
  repoId?: string | null;
  isMobile?: boolean;
}

export function SettingsTabPrecommit({ repoId, isMobile = false }: SettingsTabPrecommitProps) {
  const configQuery = usePrecommitConfig(repoId);
  const saveMutation = useSavePrecommitConfig(repoId);
  const runMutation = useRunPrecommit(repoId);
  const [draft, setDraft] = useState<PrecommitConfig | null>(null);

  useEffect(() => {
    if (configQuery.data) {
      setDraft(configQuery.data);
    }
  }, [configQuery.data]);

  if (configQuery.isLoading || !draft) {
    return <p className="text-sm text-slate-400">Loading precommit settings...</p>;
  }

  const controlClass = "w-full rounded-md border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-100";
  const labelClass = "text-xs font-medium text-slate-300";
  const lastResult = runMutation.data?.result ?? draft.last_result;

  return (
    <div className={`space-y-4 ${isMobile ? "pb-6" : ""}`}>
      <label className="flex items-center justify-between gap-3 rounded-md border border-slate-800 px-3 py-2">
        <span className={labelClass}>Enabled</span>
        <input
          type="checkbox"
          checked={draft.enabled}
          onChange={(event) => setDraft({ ...draft, enabled: event.target.checked })}
        />
      </label>

      <div className="space-y-1">
        <label className={labelClass} htmlFor="precommit-command">Command</label>
        <input
          id="precommit-command"
          className={controlClass}
          value={draft.command}
          onChange={(event) => setDraft({ ...draft, command: event.target.value })}
          placeholder="make precommit"
        />
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1">
          <label className={labelClass} htmlFor="precommit-timeout">Timeout seconds</label>
          <input
            id="precommit-timeout"
            type="number"
            min={1}
            max={1800}
            className={controlClass}
            value={draft.timeout_seconds}
            onChange={(event) => setDraft({ ...draft, timeout_seconds: Number(event.target.value) || 300 })}
          />
        </div>
        <div className="space-y-1">
          <label className={labelClass} htmlFor="precommit-working-dir">Working directory</label>
          <input
            id="precommit-working-dir"
            className={controlClass}
            value={draft.working_directory}
            onChange={(event) => setDraft({ ...draft, working_directory: event.target.value })}
          />
        </div>
      </div>

      <label className="flex items-center justify-between gap-3 rounded-md border border-slate-800 px-3 py-2">
        <span className={labelClass}>Run before commit</span>
        <input
          type="checkbox"
          checked={draft.run_before_commit}
          onChange={(event) => setDraft({ ...draft, run_before_commit: event.target.checked })}
        />
      </label>

      <label className="flex items-center justify-between gap-3 rounded-md border border-slate-800 px-3 py-2">
        <span className={labelClass}>Allow one-time override</span>
        <input
          type="checkbox"
          checked={draft.allow_override}
          onChange={(event) => setDraft({ ...draft, allow_override: event.target.checked })}
        />
      </label>

      <div className="flex flex-wrap gap-2">
        <Button
          size="sm"
          onClick={() => saveMutation.mutate(draft)}
          disabled={saveMutation.isPending}
          className="h-8 gap-2"
        >
          <Save className="h-3.5 w-3.5" />
          Save
        </Button>
        <Button
          size="sm"
          variant="outline"
          onClick={() => runMutation.mutate({})}
          disabled={runMutation.isPending || !draft.command.trim()}
          className="h-8 gap-2"
        >
          <Play className="h-3.5 w-3.5" />
          Run
        </Button>
      </div>

      {lastResult && (
        <div className="rounded-md border border-slate-800 bg-slate-900/70 p-3 text-xs text-slate-300">
          <div className="flex items-center justify-between gap-3">
            <span className="font-medium text-slate-100">{lastResult.summary}</span>
            <span>{lastResult.status}</span>
          </div>
          {(lastResult.stdout || lastResult.stderr) && (
            <pre className="mt-3 max-h-48 overflow-auto whitespace-pre-wrap rounded bg-slate-950 p-2 text-[11px] text-slate-300">
              {[lastResult.stdout, lastResult.stderr].filter(Boolean).join("\n")}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}
