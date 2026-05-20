import { useEffect, useState } from "react";
import { Loader2, Play, Save, XCircle } from "lucide-react";
import { Button } from "./ui/button";
import { usePrecommitConfig, useSavePrecommitConfig, useStreamPrecommit } from "../lib/hooks-core";
import type { PrecommitConfig } from "../lib/api";

function formatElapsed(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  const seconds = ms / 1000;
  if (seconds < 60) return `${seconds.toFixed(1)}s`;
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}m ${s.toString().padStart(2, "0")}s`;
}

interface SettingsTabPrecommitProps {
  repoId?: string | null;
  isMobile?: boolean;
}

export function SettingsTabPrecommit({ repoId, isMobile = false }: SettingsTabPrecommitProps) {
  const configQuery = usePrecommitConfig(repoId);
  const saveMutation = useSavePrecommitConfig(repoId);
  const precommitStream = useStreamPrecommit(repoId);
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
  const streamingResult = precommitStream.state.finished?.result;
  const lastResult = streamingResult ?? draft.last_result;
  const streamRunning = precommitStream.state.running;

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
          onClick={() => {
            void precommitStream.run({}).catch(() => {});
          }}
          disabled={streamRunning || !draft.command.trim()}
          className="h-8 gap-2"
        >
          <Play className="h-3.5 w-3.5" />
          Run
        </Button>
        {streamRunning && (
          <Button
            size="sm"
            variant="outline"
            onClick={() => precommitStream.cancel()}
            className="h-8 gap-2 text-red-300 hover:text-red-200"
          >
            <XCircle className="h-3.5 w-3.5" />
            Cancel
          </Button>
        )}
      </div>

      {streamRunning && (
        <div
          className="rounded-md border border-sky-800/60 bg-sky-950/30 p-3 text-xs text-sky-100"
          role="status"
          aria-live="polite"
          data-testid="precommit-running"
        >
          <div className="flex items-center justify-between gap-3">
            <span className="flex items-center gap-2 font-medium">
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
              Running pre-commit: <code className="text-sky-200">{precommitStream.state.command ?? draft.command}</code>
            </span>
            <span className="tabular-nums">{formatElapsed(precommitStream.state.elapsedMs)}</span>
          </div>
          <p className="mt-2 text-[11px] text-sky-200/70">
            Drift check may take 30s–2min if many shared packages were touched.
          </p>
          {precommitStream.state.tail.length > 0 && (
            <pre className="mt-3 max-h-40 overflow-auto whitespace-pre-wrap rounded bg-slate-950 p-2 text-[11px] text-slate-200">
              {precommitStream.state.tail.join("\n")}
            </pre>
          )}
        </div>
      )}

      {!streamRunning && precommitStream.state.error && (
        <div className="rounded-md border border-red-800/60 bg-red-950/30 p-3 text-xs text-red-200">
          {precommitStream.state.error}
        </div>
      )}

      <HookStatusBadge hook={draft.hook} enabled={draft.enabled} />

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

interface HookStatusBadgeProps {
  hook?: PrecommitConfig["hook"];
  enabled: boolean;
}

function HookStatusBadge({ hook, enabled }: HookStatusBadgeProps) {
  if (!enabled) {
    return (
      <div className="rounded-md border border-slate-800 bg-slate-900/40 px-3 py-2 text-xs text-slate-400">
        Precommit disabled — no hook installed; commits via raw <code>git</code> will skip checks.
      </div>
    );
  }
  const status = hook?.status ?? "fallback";
  const installed = status === "installed";
  const toneClass = installed
    ? "border-emerald-700 bg-emerald-950/40 text-emerald-200"
    : "border-amber-700 bg-amber-950/40 text-amber-200";
  const headline = installed
    ? "Hook installed — protected from both GCT and raw `git commit`."
    : "In-process only — `git commit` from a shell will skip the precommit check.";
  return (
    <div className={`rounded-md border px-3 py-2 text-xs ${toneClass}`}>
      <div className="font-medium">{headline}</div>
      {hook?.reason && <div className="mt-1 opacity-90">Reason: {hook.reason}</div>}
      {hook?.path && <div className="mt-1 opacity-70 font-mono">{hook.path}</div>}
      {hook?.existing_hook_preview && (
        <details className="mt-2">
          <summary className="cursor-pointer opacity-90">
            Existing hook ({hook.existing_kind ?? "user"})
          </summary>
          <pre className="mt-2 max-h-32 overflow-auto whitespace-pre-wrap rounded bg-slate-950/70 p-2 text-[11px]">
            {hook.existing_hook_preview}
          </pre>
        </details>
      )}
    </div>
  );
}
