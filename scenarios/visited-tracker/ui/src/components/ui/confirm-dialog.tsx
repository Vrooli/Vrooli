import type { CSSProperties, ReactNode } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./dialog";
import { AlertCircle } from "lucide-react";
import { OverlayActionBar } from "@vrooli/react-component-library/OverlayActionBar/1";

interface ConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description: string | ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  onConfirm: () => void;
  destructive?: boolean;
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  onConfirm,
  destructive = false,
}: ConfirmDialogProps) {
  const handleConfirm = () => {
    onConfirm();
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          {destructive && (
            <div className="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-full bg-red-500/10">
              <AlertCircle className="h-6 w-6 text-red-400" aria-hidden="true" />
            </div>
          )}
          <DialogTitle className={destructive ? "text-center" : ""}>{title}</DialogTitle>
          <DialogDescription className={destructive ? "text-center" : ""}>
            {description}
          </DialogDescription>
        </DialogHeader>
        <DialogFooter className="sm:justify-center gap-2">
          <OverlayActionBar style={destructive ? { "--color-primary": "#dc2626", "--color-primary-foreground": "#ffffff" } as CSSProperties : undefined} actions={[
            { id: "cancel", label: cancelLabel, onSelect: () => onOpenChange(false) },
            { id: "confirm", label: confirmLabel, onSelect: handleConfirm, primary: true },
          ]} />
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
