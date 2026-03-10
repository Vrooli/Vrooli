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
          onClick={() => setConfirmTarget(scenarioNames)}
        >
          {isPending ? "Fixing..." : `Fix All (${scenarioNames.length})`}
        </Button>
      </div>

      <ConfirmDialog
        open={confirmTarget !== null}
        title={dryRun ? "Dry Run Fix" : "Apply Fix"}
        message={
          dryRun
            ? `Preview fixes for ${ruleId} across ${confirmTarget?.length ?? 0} scenario(s). No files will be changed.`
            : `Apply fixes for ${ruleId} across ${confirmTarget?.length ?? 0} scenario(s). This will modify files on disk.`
        }
        confirmLabel={dryRun ? "Preview" : "Fix"}
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

  return (
    <>
      <button
        className="ml-2 rounded border border-white/10 px-2 py-0.5 text-xs text-slate-400 hover:bg-white/5 hover:text-slate-200 disabled:opacity-50"
        disabled={isPending}
        onClick={() => setShowConfirm(true)}
      >
        Fix
      </button>
      <ConfirmDialog
        open={showConfirm}
        title={dryRun ? "Dry Run Fix" : "Apply Fix"}
        message={
          dryRun
            ? `Preview fix for ${ruleId} on ${scenarioName}. No files will be changed.`
            : `Apply fix for ${ruleId} on ${scenarioName}. This will modify files on disk.`
        }
        confirmLabel={dryRun ? "Preview" : "Fix"}
        onConfirm={() => {
          onFix(ruleId, [scenarioName], dryRun);
          setShowConfirm(false);
        }}
        onCancel={() => setShowConfirm(false)}
      />
    </>
  );
}
