// Purge confirmation dialog with DELETE confirmation

import { useState } from "react";
import { Button } from "../ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { Input } from "../ui/input";
import { Label } from "../ui/label";

export interface PurgePreview {
  profiles: number;
  tasks: number;
  runs: number;
}

interface PurgeConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  actionLabel: string;
  pattern: string;
  preview: PurgePreview | null;
  loading: boolean;
  error: string | null;
  onConfirm: () => Promise<void>;
}

export function PurgeConfirmDialog({
  open,
  onOpenChange,
  actionLabel,
  pattern,
  preview,
  loading,
  error,
  onConfirm,
}: PurgeConfirmDialogProps) {
  const [confirmText, setConfirmText] = useState("");

  const handleConfirm = async () => {
    if (confirmText.trim() !== "DELETE") return;
    await onConfirm();
    setConfirmText("");
  };

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      setConfirmText("");
    }
    onOpenChange(isOpen);
  };

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{actionLabel}</DialogTitle>
          <DialogDescription>
            This action deletes data that matches the regex pattern. This cannot be undone.
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="space-y-4">
          <div className="space-y-2 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Pattern</span>
              <span className="font-mono text-xs">{pattern}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Profiles</span>
              <span>{preview?.profiles ?? 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Tasks</span>
              <span>{preview?.tasks ?? 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span>Runs</span>
              <span>{preview?.runs ?? 0}</span>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="purgeConfirm">Type DELETE to confirm</Label>
            <Input
              id="purgeConfirm"
              value={confirmText}
              onChange={(event) => setConfirmText(event.target.value)}
              placeholder="DELETE"
            />
          </div>
          {error && (
            <div className="rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          )}
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={loading || confirmText.trim() !== "DELETE"}
          >
            {loading ? "Deleting..." : "Confirm Delete"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
