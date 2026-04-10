import { useEffect, useRef, useId } from "react";
import { AlertTriangle } from "lucide-react";
import { Button } from "./button";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  pendingLabel?: string;
  variant?: "danger" | "default";
  isPending?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  pendingLabel,
  variant = "default",
  isPending = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const descriptionId = useId();

  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onCancel();
        return;
      }
      if (e.key === "Tab" && dialogRef.current) {
        const focusable = dialogRef.current.querySelectorAll<HTMLElement>(
          'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
        );
        if (focusable.length === 0) return;
        const first = focusable[0];
        const last = focusable[focusable.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
      onClick={onCancel}
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
      aria-describedby={descriptionId}
      data-testid="confirm-dialog"
    >
      <div
        ref={dialogRef}
        className="w-full max-w-sm rounded-xl border border-white/10 bg-slate-900 p-5 shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start gap-3">
          {variant === "danger" && (
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-red-500/10">
              <AlertTriangle className="h-5 w-5 text-red-400" aria-hidden="true" />
            </div>
          )}
          <div className="min-w-0">
            <h3 id="confirm-dialog-title" className="text-base font-semibold text-slate-100">
              {title}
            </h3>
            <p id={descriptionId} className="mt-1 text-sm text-slate-300">{description}</p>
          </div>
        </div>

        <div className="mt-5 flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <Button
            variant="outline"
            size="sm"
            onClick={onCancel}
            disabled={isPending}
            autoFocus
            data-testid="confirm-dialog-cancel"
          >
            {cancelLabel}
          </Button>
          <Button
            size="sm"
            onClick={onConfirm}
            disabled={isPending}
            className={variant === "danger" ? "bg-red-600 hover:bg-red-700 text-white border-red-600" : ""}
            data-testid="confirm-dialog-confirm"
          >
            {isPending ? (pendingLabel ?? `${confirmLabel}...`) : confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
