import { useEffect, useState } from "react";
import { Check, ExternalLink, FileCode, X } from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
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
import { buildSandboxReviewUrl } from "../lib/utils";
import type { ApproveFormData, RejectFormData, Run, RunDiff } from "../types";
import { ApprovalState } from "../types";

interface ReviewModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  run: Run;
  diff: RunDiff | null;
  diffLoading: boolean;
  onApprove: (req: ApproveFormData) => Promise<void>;
  onReject: (req: RejectFormData) => Promise<void>;
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

export function ReviewModal({
  open,
  onOpenChange,
  run,
  diff,
  diffLoading,
  onApprove,
  onReject,
}: ReviewModalProps) {
  const [action, setAction] = useState<"none" | "approve" | "reject">("none");
  const [approvalForm, setApprovalForm] = useState({ actor: "", commitMsg: "" });
  const [rejectForm, setRejectForm] = useState({ actor: "", reason: "" });
  const [submitting, setSubmitting] = useState(false);

  // Reset state when modal closes
  useEffect(() => {
    if (!open) {
      setAction("none");
      setApprovalForm({ actor: "", commitMsg: "" });
      setRejectForm({ actor: "", reason: "" });
      setSubmitting(false);
    }
  }, [open]);

  const actions = run.actions;
  const canApprove = actions?.canApprove ?? false;
  const canReject = actions?.canReject ?? false;

  const handleConfirm = async () => {
    setSubmitting(true);
    try {
      if (action === "approve") {
        await onApprove({
          actor: approvalForm.actor.trim() || undefined,
          commitMsg: approvalForm.commitMsg || undefined,
        });
      } else if (action === "reject") {
        await onReject({
          actor: rejectForm.actor.trim() || undefined,
          reason: rejectForm.reason || undefined,
        });
      }
      onOpenChange(false);
    } catch (err) {
      console.error(`Failed to ${action} run:`, err);
    } finally {
      setSubmitting(false);
    }
  };

  const approvalLabel = approvalStateLabel(run.approvalState);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg lg:max-w-[90vw] xl:max-w-6xl flex flex-col overflow-hidden">
        <DialogHeader onClose={() => onOpenChange(false)}>
          <DialogTitle className="flex items-center gap-2">
            <FileCode className="h-5 w-5" />
            Review Changes
          </DialogTitle>
          <DialogDescription>
            {run.changedFiles > 0
              ? `${run.changedFiles} changed file${run.changedFiles !== 1 ? "s" : ""} awaiting review`
              : "Review run changes before approving or rejecting"}
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="flex-1 min-h-0 overflow-y-auto">
          <div className="grid gap-6 lg:grid-cols-[320px_1fr]">
            {/* Left column: Summary + Actions */}
            <div className="space-y-5">
              {/* Summary */}
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
                    {run.changedFiles > 0 ? run.changedFiles : "None"}
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
                {run.sandboxId && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="w-full gap-2"
                    onClick={() => {
                      const url = buildSandboxReviewUrl(run.sandboxId ?? "");
                      window.open(url, "_blank", "noopener,noreferrer");
                    }}
                  >
                    <ExternalLink className="h-3.5 w-3.5" />
                    Open in Sandbox
                  </Button>
                )}
              </div>

              {/* Action selector */}
              {(canApprove || canReject) && (
                <div className="space-y-3">
                  <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">Action</h4>
                  <div className="flex gap-2">
                    {canApprove && (
                      <Button
                        variant={action === "approve" ? "success" : "outline"}
                        size="sm"
                        className="flex-1 gap-1.5"
                        onClick={() => setAction(action === "approve" ? "none" : "approve")}
                      >
                        <Check className="h-3.5 w-3.5" />
                        Approve
                      </Button>
                    )}
                    {canReject && (
                      <Button
                        variant={action === "reject" ? "destructive" : "outline"}
                        size="sm"
                        className="flex-1 gap-1.5"
                        onClick={() => setAction(action === "reject" ? "none" : "reject")}
                      >
                        <X className="h-3.5 w-3.5" />
                        Reject
                      </Button>
                    )}
                  </div>

                  {/* Approve form */}
                  {action === "approve" && (
                    <div className="space-y-3 rounded-lg border border-success/30 bg-success/5 p-3">
                      <div className="space-y-2">
                        <Label htmlFor="review-actor">Your Name (optional)</Label>
                        <Input
                          id="review-actor"
                          value={approvalForm.actor}
                          onChange={(e) =>
                            setApprovalForm({ ...approvalForm, actor: e.target.value })
                          }
                          placeholder="Leave blank to approve anonymously"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="review-commit-msg">Commit Message (optional)</Label>
                        <Input
                          id="review-commit-msg"
                          value={approvalForm.commitMsg}
                          onChange={(e) =>
                            setApprovalForm({ ...approvalForm, commitMsg: e.target.value })
                          }
                          placeholder="Custom commit message"
                        />
                      </div>
                    </div>
                  )}

                  {/* Reject form */}
                  {action === "reject" && (
                    <div className="space-y-3 rounded-lg border border-destructive/30 bg-destructive/5 p-3">
                      <div className="space-y-2">
                        <Label htmlFor="review-reject-actor">Your Name (optional)</Label>
                        <Input
                          id="review-reject-actor"
                          value={rejectForm.actor}
                          onChange={(e) =>
                            setRejectForm({ ...rejectForm, actor: e.target.value })
                          }
                          placeholder="Leave blank for anonymous"
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="review-reject-reason">Rejection Reason</Label>
                        <Textarea
                          id="review-reject-reason"
                          value={rejectForm.reason}
                          onChange={(e) =>
                            setRejectForm({ ...rejectForm, reason: e.target.value })
                          }
                          placeholder="Why are you rejecting these changes?"
                          rows={3}
                        />
                      </div>
                    </div>
                  )}
                </div>
              )}
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
              ) : diff ? (
                <DiffViewer diff={diff} />
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
            Cancel
          </Button>
          {action === "approve" ? (
            <Button
              variant="success"
              onClick={handleConfirm}
              disabled={submitting || !canApprove}
              className="gap-2"
            >
              <Check className="h-4 w-4" />
              {submitting ? "Approving..." : "Confirm Approval"}
            </Button>
          ) : action === "reject" ? (
            <Button
              variant="destructive"
              onClick={handleConfirm}
              disabled={submitting || !canReject}
              className="gap-2"
            >
              <X className="h-4 w-4" />
              {submitting ? "Rejecting..." : "Confirm Rejection"}
            </Button>
          ) : null}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
