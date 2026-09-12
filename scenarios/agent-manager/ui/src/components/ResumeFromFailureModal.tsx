import { useEffect, useRef } from "react";
import { AlertCircle, Paperclip, PlayCircle } from "lucide-react";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import { AttachmentPreview } from "./AttachmentPreview";
import { useAttachments } from "../hooks/useAttachments";
import type { Run } from "../types";
import { useState } from "react";

interface ResumeFromFailureModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The failed/cancelled run that will be resumed. */
  failedRun: Run | null;
  onSubmit: (customContext: string, attachmentIds?: string[]) => Promise<void>;
  loading?: boolean;
  error?: string | null;
}

/**
 * ResumeFromFailureModal launches a brand-new run that inherits the
 * failed/cancelled run's task and profile, seeded with that attempt's
 * transcript and diff so the agent can finish what's left without redoing
 * completed work. Distinct from Retry (no context) and Continue (Codex
 * session resume — fragile and only when SessionID is present).
 */
export function ResumeFromFailureModal({
  open,
  onOpenChange,
  failedRun,
  onSubmit,
  loading = false,
  error = null,
}: ResumeFromFailureModalProps) {
  const [customContext, setCustomContext] = useState("");
  const {
    attachments,
    addAttachment,
    removeAttachment,
    clearAttachments,
    getUploadedIds,
    isUploading,
  } = useAttachments();
  const imageInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) {
      setCustomContext("");
      clearAttachments();
    }
  }, [open, clearAttachments]);

  const handleImageSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      addAttachment(file);
      e.target.value = "";
    }
  };

  const handleSubmit = async () => {
    const uploadedIds = getUploadedIds();
    const attachmentIds = uploadedIds.length > 0 ? uploadedIds : undefined;
    await onSubmit(customContext.trim(), attachmentIds);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg flex flex-col">
        <DialogHeader onClose={() => onOpenChange(false)}>
          <DialogTitle className="flex items-center gap-2">
            <PlayCircle className="h-5 w-5" />
            Resume from Failure
          </DialogTitle>
          <DialogDescription>
            Start a new run that inherits this run's task and profile. The new agent will be given the prior attempt's transcript and diff so it can pick up where the previous run left off without redoing completed work.
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="flex-1 overflow-y-auto space-y-4">
          <div className="space-y-2">
            <Label htmlFor="resumeContext">
              Additional Guidance (optional)
            </Label>
            <Textarea
              id="resumeContext"
              value={customContext}
              onChange={(e) => setCustomContext(e.target.value)}
              placeholder="e.g. 'the schema migration was applied; just retry the API wiring' or 'ignore the partial test edits, redo them from scratch'"
              rows={5}
            />
            <p className="text-xs text-muted-foreground">
              Tell the resumed agent what to skip, retry, or focus on. Leave blank to let it decide from the prior transcript.
            </p>
          </div>

          <div className="space-y-2 pt-2 border-t">
            <Label>Image Attachments (optional)</Label>
            {attachments.length > 0 && (
              <AttachmentPreview
                attachments={attachments}
                onRemove={removeAttachment}
                isUploading={isUploading}
              />
            )}
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => imageInputRef.current?.click()}
            >
              <Paperclip className="h-4 w-4 mr-2" />
              Attach Image
            </Button>
            <input
              ref={imageInputRef}
              type="file"
              accept="image/jpeg,image/png,image/gif,image/webp"
              onChange={handleImageSelect}
              className="hidden"
            />
            <p className="text-xs text-muted-foreground">
              Screenshots or diagrams that clarify the remaining work.
            </p>
          </div>

          {error && (
            <div className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{error}</span>
            </div>
          )}

          {failedRun?.actions?.canResumeFromFailureReason && !failedRun.actions.canResumeFromFailure && (
            <div className="flex items-start gap-2 rounded-md border border-amber-500/50 bg-amber-500/10 px-3 py-2 text-sm">
              <AlertCircle className="mt-0.5 h-4 w-4 text-amber-500 shrink-0" />
              <span>{failedRun.actions.canResumeFromFailureReason}</span>
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={loading || isUploading || !failedRun}
            className="gap-2"
          >
            {loading ? (
              "Resuming..."
            ) : isUploading ? (
              "Uploading..."
            ) : (
              <>
                <PlayCircle className="h-4 w-4" />
                Resume Run
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
