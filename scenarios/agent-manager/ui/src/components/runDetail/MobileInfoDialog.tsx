import { Check, Copy } from "lucide-react";
import { Dialog, DialogContent, DialogHeader } from "../ui/dialog";
import type { Run, RunDiff } from "../../types";
import { RunDetailsContent, type CostTotals } from "../RunDetailParts";

interface MobileInfoDialogProps {
  run: Run;
  diff: RunDiff | null;
  taskTitle: string;
  profileName: string;
  durationMs: number | null;
  costTotals: CostTotals;
  runIdCopyStatus: string;
  onCopyRunId: () => void;
  onClose: () => void;
}

/** Run details behind the info icon on phones. */
export function MobileInfoDialog({ run, diff, taskTitle, profileName, durationMs, costTotals, runIdCopyStatus, onCopyRunId, onClose }: MobileInfoDialogProps) {
  return (
    <Dialog open onOpenChange={onClose}>
      <DialogContent className="max-h-[80vh] overflow-y-auto">
        <DialogHeader onClose={onClose}><h3 className="font-semibold">Run Details</h3></DialogHeader>
        <div className="space-y-4 pt-4">
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">Run ID:</span>
            <button
              type="button"
              className="flex min-h-11 items-center gap-1 px-1 text-xs text-muted-foreground transition-colors hover:text-foreground sm:min-h-8"
              onClick={onCopyRunId}
              title={`Copy run ID: ${run.id}`}
            >
              <code className="font-mono">{run.id.slice(0, 8)}</code>
              {runIdCopyStatus === "copied" ? <Check className="h-3 w-3 text-emerald-500" /> : <Copy className="h-3 w-3" />}
            </button>
          </div>
          <RunDetailsContent
            run={run}
            diff={diff}
            taskTitle={taskTitle}
            profileName={profileName}
            durationMs={durationMs}
            costTotals={costTotals}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
