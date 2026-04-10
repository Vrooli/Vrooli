import { useState } from "react";
import { Button } from "./ui/button";
import { ConfirmDialog } from "./ConfirmDialog";

export function FixActions({
  ruleId,
  scenarioNames,
  fixable,
  dryRun,
  onToggleDryRun,
  onFix,
  isPending
}: {
  ruleId: string;
  scenarioNames: string[];
  fixable: boolean;
  dryRun: boolean;
  onToggleDryRun: () => void;
  onFix: (ruleId: string, scenarioNames: string[], dryRun: boolean) => void;
  isPending: boolean;
}) {
  const [confirmTarget, setConfirmTarget] = useState<string[] | null>(null);

  if (!fixable || scenarioNames.length === 0) return null;

  const handleConfirm = () => {
    if (confirmTarget) {
      onFix(ruleId, confirmTarget, dryRun);
      setConfirmTarget(null);
    }
  };

  const handleClick = () => {
    if (dryRun) {
      // Dry run: confirm before previewing
      setConfirmTarget(scenarioNames);
    } else {
      // Real fix: skip ConfirmDialog — diff review modal in App.tsx handles confirmation
      onFix(ruleId, scenarioNames, false);
    }
  };

  return (
    <>
      <div className="mt-3 flex flex-wrap items-center gap-2">
        <label className="flex items-center gap-1.5 text-xs text-slate-400">
          <input
            type="checkbox"
            checked={dryRun}
            onChange={onToggleDryRun}
            className="h-3 w-3 accent-slate-100"
          />
          Dry run
        </label>
        <Button
          size="sm"
          variant="outline"
          disabled={isPending}
          onClick={handleClick}
        >
          {isPending ? "Fixing..." : `Fix All (${scenarioNames.length})`}
        </Button>
      </div>

      <ConfirmDialog
        open={confirmTarget !== null}
        title="Dry Run Fix"
        message={`Preview fixes for ${ruleId} across ${confirmTarget?.length ?? 0} scenario(s). No files will be changed.`}
        confirmLabel="Preview"
        onConfirm={handleConfirm}
        onCancel={() => setConfirmTarget(null)}
      />
    </>
  );
}

export function FixScenarioButton({
  ruleId,
  scenarioName,
  dryRun,
  onFix,
  isPending
}: {
  ruleId: string;
  scenarioName: string;
  dryRun: boolean;
  onFix: (ruleId: string, scenarioNames: string[], dryRun: boolean) => void;
  isPending: boolean;
}) {
  const [showConfirm, setShowConfirm] = useState(false);

  const handleClick = () => {
    if (dryRun) {
      setShowConfirm(true);
    } else {
      // Real fix: skip ConfirmDialog — diff review modal in App.tsx handles confirmation
      onFix(ruleId, [scenarioName], false);
    }
  };

  return (
    <>
      <button
        className="ml-2 rounded border border-white/10 px-2 py-0.5 text-xs text-slate-400 hover:bg-white/5 hover:text-slate-200 disabled:opacity-50"
        disabled={isPending}
        onClick={handleClick}
      >
        Fix
      </button>
      <ConfirmDialog
        open={showConfirm}
        title="Dry Run Fix"
        message={`Preview fix for ${ruleId} on ${scenarioName}. No files will be changed.`}
        confirmLabel="Preview"
        onConfirm={() => {
          onFix(ruleId, [scenarioName], dryRun);
          setShowConfirm(false);
        }}
        onCancel={() => setShowConfirm(false)}
      />
    </>
  );
}
