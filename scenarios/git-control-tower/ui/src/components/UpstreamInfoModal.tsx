import { GitBranch, Info, X } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";

interface UpstreamInfoModalProps {
  isOpen: boolean;
  localBranch?: string;
  upstreamRef?: string;
  ahead?: number;
  behind?: number;
  onClose: () => void;
}

function getRemoteName(upstreamRef?: string) {
  if (!upstreamRef) return "origin";
  const parts = upstreamRef.split("/");
  return parts[0] || "origin";
}

export function UpstreamInfoModal({
  isOpen,
  localBranch,
  upstreamRef,
  ahead = 0,
  behind = 0,
  onClose
}: UpstreamInfoModalProps) {
  const isMobile = useIsMobile();
  if (!isOpen) return null;

  const remote = getRemoteName(upstreamRef);
  const hasLocal = Boolean(localBranch);
  const desiredUpstream = localBranch ? `${remote}/${localBranch}` : "";
  const canSuggestSetUpstream = Boolean(localBranch);
  const copyCommand = (command: string) => {
    if (!navigator?.clipboard) return;
    void navigator.clipboard.writeText(command);
  };

  const summary = localBranch && upstreamRef
    ? `Local branch ${localBranch} tracks ${upstreamRef}.`
    : upstreamRef
      ? `Upstream set to ${upstreamRef}.`
      : "No upstream is configured yet.";

  const implication = upstreamRef
    ? "Push and pull will target that upstream ref. Ahead/behind counts are measured against it."
    : "Without an upstream, pushes need an explicit remote and branch.";

  const mismatchNote =
    localBranch && upstreamRef && !upstreamRef.endsWith(`/${localBranch}`)
      ? `This means pushes from ${localBranch} will update ${upstreamRef}, not ${remote}/${localBranch}.`
      : "";

  const fetchCommand = `git fetch ${remote}`;
  const pushCommand = hasLocal ? `git push -u ${remote} ${localBranch}` : `git push -u ${remote} <branch>`;
  const setUpstreamCommand = hasLocal
    ? `git branch --set-upstream-to=${desiredUpstream} ${localBranch}`
    : "git branch --set-upstream-to=<remote>/<branch> <branch>";

  const syncSummary = upstreamRef
    ? `Status vs ${upstreamRef}: ${ahead} ahead, ${behind} behind.`
    : "";

  const actions = [
    {
      id: "fetch",
      title: "Fetch remote updates",
      description: `Refresh ${remote} refs so ahead/behind is accurate.`,
      safety: "Safe: does not change your working tree or staged files.",
      command: fetchCommand
    },
    {
      id: "push",
      title: "Push and set upstream",
      description: hasLocal
        ? `Push ${localBranch} and track ${remote}/${localBranch} by default.`
        : `Push your branch and set its upstream on ${remote}.`,
      safety: "Safe: pushes commits only; does not touch local files.",
      command: pushCommand
    },
    {
      id: "set-upstream",
      title: "Change tracking target",
      description: hasLocal
        ? `Set ${localBranch} to track ${desiredUpstream}.`
        : "Set your branch to track a different upstream ref.",
      safety: "Safe: changes tracking metadata only; no file changes.",
      command: setUpstreamCommand,
      disabled: !canSuggestSetUpstream
    }
  ];

  const content = (
    <>
      <div className="space-y-2">
        <div className="flex items-center gap-2 text-slate-200">
          <GitBranch className="h-4 w-4 text-slate-400" />
          <h2 className="text-sm font-semibold">Branch tracking</h2>
        </div>
        <p className="text-xs text-slate-500">{summary}</p>
        <p className="text-xs text-slate-500">{implication}</p>
        {mismatchNote && <p className="text-xs text-amber-300">{mismatchNote}</p>}
        {syncSummary && <p className="text-xs text-slate-500">{syncSummary}</p>}
      </div>

      <div className="space-y-3">
        <div className="flex items-center gap-2 text-xs text-slate-300">
          <Info className="h-3.5 w-3.5 text-slate-400" />
          Recommended actions
        </div>
        <div className="grid gap-3">
          {actions.map((action) => (
            <button
              key={action.id}
              type="button"
              onClick={() => copyCommand(action.command)}
              disabled={action.disabled}
              className={`w-full rounded-xl border px-4 py-4 text-left transition ${
                action.disabled
                  ? "border-slate-800/50 bg-slate-900/20 text-slate-600"
                  : "border-slate-800 bg-slate-900/40 text-slate-200 hover:bg-slate-800/60"
              }`}
            >
              <div className="text-sm font-semibold">{action.title}</div>
              <div className="mt-1 text-xs text-slate-400">{action.description}</div>
              <div className="mt-2 text-[11px] text-emerald-300">{action.safety}</div>
              <div className="mt-2 text-[11px] font-mono text-slate-500">
                {action.command}
              </div>
              <div className="mt-3 text-xs text-slate-300">Click to copy</div>
            </button>
          ))}
        </div>
      </div>
    </>
  );

  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Branch tracking"
      >
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <div className="flex items-center gap-2">
            <GitBranch className="h-5 w-5 text-slate-400" />
            <h2 className="text-base font-semibold text-slate-100">Branch tracking</h2>
          </div>
          <button
            type="button"
            className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-4 py-6 space-y-6">
          {content}
        </div>

        <div className="border-t border-slate-800 px-4 py-4 pb-safe">
          <Button
            variant="default"
            size="sm"
            onClick={onClose}
            className="w-full h-12 text-sm touch-target"
          >
            Done
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="Branch tracking"
    >
      <div className="w-full max-w-lg rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <div className="flex items-center gap-2">
            <GitBranch className="h-4 w-4 text-slate-400" />
            <div>
              <h2 className="text-sm font-semibold text-slate-100">Branch tracking</h2>
              <p className="text-xs text-slate-500">How upstream affects push and pull</p>
            </div>
          </div>
          <button
            type="button"
            className="rounded-md p-2 text-slate-400 hover:bg-slate-900 hover:text-slate-200"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="px-4 py-4 space-y-4">
          {content}
        </div>

        <div className="flex items-center justify-end gap-2 border-t border-slate-800 px-4 py-3">
          <Button variant="outline" size="sm" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>
    </div>
  );
}
