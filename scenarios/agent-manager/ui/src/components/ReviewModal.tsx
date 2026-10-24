import { ExternalLink, FileCode } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogBody,
  DialogFooter,
} from "./ui/dialog";
import { DiffViewer } from "./DiffViewer";
import type { Run, RunDiff } from "../types";
import { ApprovalState } from "../types";
import { reviewFileCount } from "../lib/reviewFileCount";

interface ReviewModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  run: Run;
  diff: RunDiff | null;
  diffLoading: boolean;
  onOpenSandbox?: () => void;
}

function approvalStateLabel(state: ApprovalState): string {
  switch (state) {
    case ApprovalState.APPROVED:
      return "Approved";
    case ApprovalState.REJECTED:
      return "Rejected";
    case ApprovalState.PARTIALLY_APPROVED:
      return "Partial Approval";
    case ApprovalState.PENDING:
      return "Pending Review";
    default:
      return "";
  }
}

// ReviewModal is a READ-ONLY view of a run's changes. Approving or rejecting a
// run is handled by a human only in the Web Console approvals inbox (the single
// operator surface), not here — so this modal shows the diff and points the
// operator at the inbox rather than offering a duplicate approve/reject action.
export function ReviewModal({
  open,
  onOpenChange,
  run,
  diff,
  diffLoading,
  onOpenSandbox,
}: ReviewModalProps) {
  const approvalLabel = approvalStateLabel(run.approvalState);
  const awaitingReview = run.approvalState === ApprovalState.PENDING;
  const currentDiff = diffLoading ? null : diff;
  const fileCount = reviewFileCount(run, currentDiff);
  const hasVerifiedDiff = currentDiff?.runId === run.id && currentDiff.fileCountVerified;
  let countDescription = "Review run changes";
  if (!fileCount.available) {
    countDescription = "File count unavailable";
  } else if (fileCount.count === 0) {
    countDescription = "0 changed files in the verified diff";
  } else if (fileCount.count > 0) {
    countDescription = `${fileCount.count} changed file${fileCount.count !== 1 ? "s" : ""} awaiting review`;
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg lg:max-w-[90vw] xl:max-w-6xl flex flex-col overflow-hidden">
        <DialogHeader onClose={() => onOpenChange(false)}>
          <DialogTitle className="flex items-center gap-2">
            <FileCode className="h-5 w-5" />
            Review Changes
          </DialogTitle>
          <DialogDescription>
            {countDescription}
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="flex-1 min-h-0 overflow-y-auto">
          <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
            {/* Left column: Summary */}
            <div className="space-y-5">
              <div className="rounded-lg border border-border bg-card/50 p-4 space-y-3">
                <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Summary</h4>
                <div className="space-y-2 text-sm">
                  {approvalLabel && (
                    <div className="flex items-center gap-2">
                      <span className="text-muted-foreground">Status:</span>
                      <Badge variant="outline">{approvalLabel}</Badge>
                    </div>
                  )}
                  <div>
                    <span className="text-muted-foreground">Changed files: </span>
                    {fileCount.available ? fileCount.count : "Unavailable"}
                  </div>
                  {run.sandboxId && (
                    <div>
                      <span className="text-muted-foreground">Sandbox: </span>
                      <code className="text-xs bg-muted px-1 py-0.5 rounded">
                        {run.sandboxId.slice(0, 12)}...
                      </code>
                    </div>
                  )}
                </div>
                {run.sandboxId && onOpenSandbox && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full gap-2"
                    onClick={onOpenSandbox}
                  >
                    <ExternalLink className="h-3.5 w-3.5" />
                    Open in Sandbox
                  </Button>
                )}
              </div>

              {/* Approvals are handled in the Web Console approvals inbox, not here. */}
              {awaitingReview && (run.actions?.reviewUnavailableReason ? (
                <div role="status" className="rounded-lg border border-amber-500/40 bg-amber-500/5 p-3 text-sm text-amber-700 dark:text-amber-400">
                  {run.actions.reviewUnavailableReason}
                </div>
              ) : (
                <div className="rounded-lg border border-border bg-muted/40 p-3 text-sm text-muted-foreground">
                  Approve or reject this run from the <span className="font-medium text-foreground">Web Console approvals inbox</span>, the single place operator decisions are answered.
                </div>
              ))}
            </div>

            {/* Right column: Diff */}
            <div className="min-h-0">
              <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground mb-3">
                Changes
              </h4>
              {diffLoading ? (
                <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                  <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
                  <p className="mt-3 text-sm">Loading diff...</p>
                </div>
              ) : hasVerifiedDiff && currentDiff ? (
                <DiffViewer diff={currentDiff} />
              ) : currentDiff ? (
                <div className="py-12 text-center text-muted-foreground text-sm">
                  Diff evidence is incomplete; changes and file count are unverified.
                </div>
              ) : (
                <div className="py-12 text-center text-muted-foreground text-sm">
                  No diff available
                </div>
              )}
            </div>
          </div>
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
