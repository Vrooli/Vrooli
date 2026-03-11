import { useCallback, useEffect, useRef } from "react";
import { AlertTriangle } from "lucide-react";
import { Button } from "./ui/button";

/**
 * Confirmation dialog for destructive actions
 *
 * Implements replay safety by requiring explicit confirmation before
 * side-effect operations like delete.
 *
 * [REQ:P1-017a] Route-Level Error Boundaries (provides controlled teardown)
 */
export interface ConfirmDialogProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
  variant?: "danger" | "warning";
}

export function ConfirmDialog({
  isOpen,
  title,
  message,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  onConfirm,
  onCancel,
  variant = "danger"
}: ConfirmDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const confirmButtonRef = useRef<HTMLButtonElement>(null);

  // Focus trap and keyboard handling
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!isOpen) return;

      if (e.key === "Escape") {
        e.preventDefault();
        onCancel();
      }
    },
    [isOpen, onCancel]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  // Focus confirm button when dialog opens
  useEffect(() => {
    if (isOpen && confirmButtonRef.current) {
      confirmButtonRef.current.focus();
    }
  }, [isOpen]);

  if (!isOpen) return null;

  const variantStyles = {
    danger: {
      icon: "text-red-400",
      button: "bg-red-600 hover:bg-red-700 text-white"
    },
    warning: {
      icon: "text-yellow-400",
      button: "bg-yellow-600 hover:bg-yellow-700 text-white"
    }
  };

  const styles = variantStyles[variant];

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
      data-testid="confirm-dialog"
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onCancel}
        data-testid="confirm-dialog-backdrop"
      />

      {/* Dialog */}
      <div
        ref={dialogRef}
        className="relative z-10 mx-4 w-full max-w-md rounded-xl border border-white/10 bg-slate-900 p-6 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex gap-4">
          <div className={`flex-shrink-0 rounded-full bg-white/10 p-2 ${styles.icon}`}>
            <AlertTriangle className="h-6 w-6" />
          </div>
          <div className="flex-1">
            <h3
              id="confirm-dialog-title"
              className="text-lg font-semibold text-white"
            >
              {title}
            </h3>
            <p className="mt-2 text-sm text-slate-400">{message}</p>
          </div>
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <Button
            variant="outline"
            onClick={onCancel}
            data-testid="confirm-dialog-cancel"
          >
            {cancelLabel}
          </Button>
          <button
            ref={confirmButtonRef}
            onClick={onConfirm}
            className={`rounded-lg px-4 py-2 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-white/20 ${styles.button}`}
            data-testid="confirm-dialog-confirm"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
