/**
 * FileActionDialogs
 *
 * Rename / move / copy / delete dialogs extracted from BacklogFileBrowser.
 */

import { Loader2 } from "lucide-react";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Input } from "../ui/input";
import type { FileActionType } from "./backlog-file-browser";
import type { BacklogFile } from "../../types";

export interface FileActionDialogsProps {
  activeAction: { action: FileActionType; target: BacklogFile } | null;
  fileActionInput: string;
  fileActionError: string | null;
  fileActionPending: boolean;
  onInputChange: (value: string) => void;
  onConfirm: () => void;
  onClose: () => void;
}

export function FileActionDialogs({
  activeAction,
  fileActionInput,
  fileActionError,
  fileActionPending,
  onInputChange,
  onConfirm,
  onClose,
}: FileActionDialogsProps) {
  return (
    <>
      {/* File action dialogs (rename/move/copy) */}
      <Dialog
        isOpen={Boolean(activeAction && activeAction.action !== "delete")}
        onClose={onClose}
        title={
          activeAction?.action === "rename"
            ? `Rename ${activeAction.target.type}`
            : activeAction?.action === "move"
              ? `Move ${activeAction.target.type}`
              : activeAction?.action === "copy"
                ? `Copy ${activeAction.target.type}`
                : "File Action"
        }
        maxWidth="max-w-md"
      >
        {activeAction && activeAction.action !== "delete" && (
          <div className="space-y-4">
            <div className="text-sm text-slate-300">
              <p className="text-xs uppercase tracking-wide text-slate-500">Source</p>
              <p className="mt-1 break-all rounded-lg bg-slate-800/60 px-3 py-2">{activeAction.target.path}</p>
            </div>
            <div className="space-y-2">
              <label className="text-xs uppercase tracking-wide text-slate-500">
                {activeAction.action === "rename" ? "New name" : "Destination path"}
              </label>
              <Input
                value={fileActionInput}
                onChange={(event) => onInputChange(event.target.value)}
                placeholder={activeAction.action === "rename" ? "new-name.ext" : "path/to/target"}
              />
            </div>
            {fileActionError && (
              <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-200">
                {fileActionError}
              </div>
            )}
            <div className="flex justify-end gap-2">
              <Button
                variant="outline"
                onClick={onClose}
                disabled={fileActionPending}
              >
                Cancel
              </Button>
              <Button
                variant="default"
                onClick={onConfirm}
                disabled={fileActionPending}
                data-testid="confirm-file-action"
              >
                {fileActionPending ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : null}
                Apply
              </Button>
            </div>
          </div>
        )}
      </Dialog>

      {/* File delete confirmation dialog */}
      <ConfirmDialog
        isOpen={Boolean(activeAction && activeAction.action === "delete")}
        onClose={onClose}
        onConfirm={onConfirm}
        title={`Delete ${activeAction?.target.type ?? "file"}`}
        description={`Delete "${activeAction?.target.path ?? ""}" from this backlog item? This cannot be undone.`}
        confirmLabel="Delete"
        isLoading={fileActionPending}
      />
    </>
  );
}
