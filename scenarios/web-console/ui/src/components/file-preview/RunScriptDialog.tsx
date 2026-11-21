import { useEffect, useState } from "react";
import { Loader2, Play, X } from "lucide-react";
import { scriptRunPlan } from "../../lib/fileExecution";

interface RunScriptDialogProps {
  path: string;
  text: string;
  onClose: () => void;
  onRun: (command: string, workingDir: string) => Promise<void>;
}

export default function RunScriptDialog({ path, text, onClose, onRun }: RunScriptDialogProps) {
  const plan = scriptRunPlan(path, text);
  const [command, setCommand] = useState(plan?.command ?? "");
  const [workingDir, setWorkingDir] = useState(plan?.workingDir ?? "");
  const [confirmed, setConfirmed] = useState(false);
  const [running, setRunning] = useState(false);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => { if (event.key === "Escape" && !running) { onClose(); } };
    window.addEventListener("keydown", onKeyDown);
    return () => { window.removeEventListener("keydown", onKeyDown); };
  }, [onClose, running]);

  if (!plan) return null;
  const submit = async () => {
    setRunning(true);
    try { await onRun(command, workingDir); } finally { setRunning(false); }
  };

  return (
    <div className="fixed inset-0 z-wc-modal flex items-center justify-center bg-black/60 p-4" role="dialog" aria-modal="true" aria-labelledby="run-script-title" data-testid="run-script-dialog">
      <div className="w-full max-w-xl rounded-2xl border border-wc-default bg-wc-surface-raised p-5 shadow-2xl">
        <div className="flex items-start justify-between gap-4"><div><h2 id="run-script-title" className="text-base font-semibold">Run script</h2><p className="mt-1 break-all text-xs text-wc-text-muted">{path}</p></div><button type="button" onClick={onClose} disabled={running} aria-label="Close" className="rounded p-1 text-wc-text-muted hover:bg-wc-surface-input"><X className="h-4 w-4" /></button></div>
        <div className="mt-4 space-y-3">
          <label className="block text-xs font-medium">Command<input value={command} onChange={(e) => { setCommand(e.target.value); }} className="mt-1 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 font-mono text-sm" data-testid="run-script-command" /></label>
          <label className="block text-xs font-medium">Working directory<input value={workingDir} onChange={(e) => { setWorkingDir(e.target.value); }} className="mt-1 w-full rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 font-mono text-sm" data-testid="run-script-working-dir" /></label>
          <p className="text-xs text-wc-text-muted">Runs in an interactive scratch terminal. It stays out of the sidebar until you hand it off.</p>
          {plan.risky && <label className="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-200"><input type="checkbox" checked={confirmed} onChange={(e) => { setConfirmed(e.target.checked); }} className="mt-0.5" data-testid="run-script-risk-confirm" />I understand this command may modify the system or install software.</label>}
        </div>
        <div className="mt-5 flex justify-end gap-2"><button type="button" onClick={onClose} disabled={running} className="rounded-lg border border-wc-default px-3 py-2 text-sm">Cancel</button><button type="button" onClick={() => void submit()} disabled={running || !command.trim() || !workingDir.trim() || (plan.risky && !confirmed)} className="inline-flex items-center gap-2 rounded-lg bg-wc-accent px-3 py-2 text-sm font-medium text-white disabled:opacity-50" data-testid="run-script-submit">{running ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}Run</button></div>
      </div>
    </div>
  );
}
