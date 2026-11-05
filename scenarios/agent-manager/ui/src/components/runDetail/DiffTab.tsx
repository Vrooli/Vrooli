import { useCallback, useEffect, useState } from "react";
import { Check, ExternalLink, FileCode } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "../ui/input";
import { Label } from "../ui/label";
import type { Run, RunDiff } from "../../types";
import { DiffViewer } from "../DiffViewer";

interface DiffTabProps {
  run: Run;
  diff: RunDiff | null;
  diffLoading: boolean;
  sandboxUrl: string | null;
  openSandboxReview: () => void;
  // Partial-approve (select specific diff files) is the one agent-manager-native
  // review action the Web Console inbox does NOT offer (the inbox is binary
  // approve/reject), so it is preserved here. Plain full-run approve/reject is a
  // human action handled only in the inbox.
  onPartialApprove?: (fileIds: string[], actor?: string, commitMsg?: string) => Promise<unknown>;
}

export function DiffTab({ run, diff, diffLoading, sandboxUrl, openSandboxReview, onPartialApprove }: DiffTabProps) {
  const actions = run.actions;
  const [partialActive, setPartialActive] = useState(false);
  const [partialForm, setPartialForm] = useState({ actor: "", commitMsg: "" });
  const [partialSubmitting, setPartialSubmitting] = useState(false);
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());

  useEffect(() => {
    setPartialActive(false);
    setPartialForm({ actor: "", commitMsg: "" });
    setPartialSubmitting(false);
    setSelectedFiles(new Set());
  }, [run.id]);

  const handlePartialConfirm = useCallback(async () => {
    if (!onPartialApprove) return;
    setPartialSubmitting(true);
    try {
      const fileIds = (diff?.files ?? [])
        .filter((f) => selectedFiles.has(f.path) && f.id)
        .map((f) => f.id);
      if (fileIds.length > 0) {
        await onPartialApprove(fileIds, partialForm.actor.trim() || undefined, partialForm.commitMsg || undefined);
        setSelectedFiles(new Set());
        setPartialActive(false);
      }
    } catch (err) {
      console.error("Failed to partially approve run:", err);
    } finally {
      setPartialSubmitting(false);
    }
  }, [onPartialApprove, diff, selectedFiles, partialForm]);

  if (diffLoading) {
    return (
      <div className="py-8 text-center text-muted-foreground">
        <div className="h-8 w-8 mx-auto animate-spin rounded-full border-4 border-primary border-t-transparent" />
        <p className="mt-3 text-sm">Loading diff...</p>
      </div>
    );
  }
  return (
      <div className="space-y-4">
        {!diff ? (
          <div className="py-8 flex flex-col items-center gap-4 text-center">
            <FileCode className="h-10 w-10 text-muted-foreground/50" />
            <div>
              <p className="font-medium text-muted-foreground">No diff available</p>
              <p className="text-sm text-muted-foreground/70 mt-1">
                {run.sandboxId
                  ? "No complete diff evidence is available. The file count and changes are unverified."
                  : "This run didn't use a sandbox, so no diff was collected."}
              </p>
            </div>
            <div className="flex gap-2">
              {run.sandboxId && sandboxUrl && (
                <Button variant="secondary" size="sm" className="gap-2" onClick={openSandboxReview}>
                  <ExternalLink className="h-3.5 w-3.5" />
                  Open in Workspace Sandbox
                </Button>
              )}
            </div>
          </div>
        ) : diff.runId !== run.id || !diff.fileCountVerified ? (
          <div className="py-8 text-center text-muted-foreground" role="status">
            Diff evidence is incomplete; changes and file count are unverified.
          </div>
        ) : (
          <DiffViewer
            diff={diff}
            selectable={partialActive}
            selectedFiles={selectedFiles}
            onFileSelectionChange={(path, selected) => {
              setSelectedFiles((prev) => {
                const next = new Set(prev);
                if (selected) next.add(path);
                else next.delete(path);
                return next;
              });
            }}
          />
        )}

        {/* Review pointer — approving/rejecting a run is a human action
            handled only in the Web Console approvals inbox (the single
            operator surface), never from this read-only page. */}
        {(actions?.canApprove || actions?.canReject) && (
          <div className="border-t border-border pt-4 space-y-3">
            <div className="flex items-center gap-2 flex-wrap">
              {run.sandboxId && sandboxUrl && (
                <Button variant="secondary" size="sm" className="gap-1.5" onClick={openSandboxReview}>
                  <ExternalLink className="h-3.5 w-3.5" />
                  Open in Sandbox
                </Button>
              )}
              <div className="flex-1" />
              {/* Partial approve is the one review action the inbox cannot do
                  (file-selective). It is preserved here; full approve/reject
                  are handled only in the inbox. */}
              {actions?.canApprove && onPartialApprove && diff && (diff.files?.length ?? 0) > 1 && (
                <Button
                  variant={partialActive ? "secondary" : "secondary"}
                  size="sm"
                  className="gap-1.5"
                  onClick={() => {
                    if (partialActive) {
                      setPartialActive(false);
                      setSelectedFiles(new Set());
                    } else {
                      setPartialActive(true);
                    }
                  }}
                >
                  <Check className="h-3.5 w-3.5" />
                  Partial Approve
                </Button>
              )}
            </div>

            {partialActive && (
              <div className="space-y-3 rounded-lg border border-warning/30 bg-warning/5 p-3">
                <p className="text-sm text-muted-foreground">
                  Select files above, then approve the selected subset.
                </p>
                <div className="space-y-2">
                  <Label htmlFor="partial-approve-actor">Your Name (optional)</Label>
                  <Input
                    id="partial-approve-actor"
                    value={partialForm.actor}
                    onChange={(e) => setPartialForm({ ...partialForm, actor: e.target.value })}
                    placeholder="Leave blank to approve anonymously"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="partial-approve-msg">Commit Message (optional)</Label>
                  <Input
                    id="partial-approve-msg"
                    value={partialForm.commitMsg}
                    onChange={(e) => setPartialForm({ ...partialForm, commitMsg: e.target.value })}
                    placeholder="Custom commit message"
                  />
                </div>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={handlePartialConfirm}
                  disabled={partialSubmitting || selectedFiles.size === 0}
                  className="gap-2"
                >
                  <Check className="h-4 w-4" />
                  {partialSubmitting
                    ? "Approving..."
                    : `Approve Selected (${selectedFiles.size} file${selectedFiles.size !== 1 ? "s" : ""})`}
                </Button>
              </div>
            )}

            <div className="rounded-lg border border-border bg-muted/40 p-3 text-sm text-muted-foreground">
              Approve or reject the full run from the{" "}
              <span className="font-medium text-foreground">Web Console approvals inbox</span> — the single
              place operator decisions are answered.
            </div>
          </div>
        )}
      </div>
  );
}
