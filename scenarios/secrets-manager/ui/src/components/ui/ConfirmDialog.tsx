import { AlertTriangle, X } from "lucide-react";
import { Button } from "./button";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  variant?: "warning" | "danger" | "info";
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "Confirm",
  cancelLabel = "Cancel",
  variant = "warning",
  onConfirm,
  onCancel
}: ConfirmDialogProps) {
  if (!open) return null;

  const variantStyles = {
    warning: {
      border: "border-amber-400/30",
      bg: "bg-amber-500/10",
      icon: "text-amber-400",
      button: "bg-amber-600 hover:bg-amber-500"
    },
    danger: {
      border: "border-red-400/30",
      bg: "bg-red-500/10",
      icon: "text-red-400",
      button: "bg-red-600 hover:bg-red-500"
    },
    info: {
      border: "border-cyan-400/30",
      bg: "bg-cyan-500/10",
      icon: "text-cyan-400",
      button: "bg-cyan-600 hover:bg-cyan-500"
    }
  };

  const styles = variantStyles[variant];

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 p-4"
      onClick={onCancel}
    >
      <div
        className={`w-full max-w-md rounded-2xl border ${styles.border} bg-slate-950 p-5 shadow-2xl`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start gap-3">
          <div className={`rounded-full ${styles.bg} p-2`}>
            <AlertTriangle className={`h-5 w-5 ${styles.icon}`} />
          </div>
          <div className="flex-1">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold text-white">{title}</h3>
              <button
                onClick={onCancel}
                className="rounded-full p-1 text-white/40 hover:bg-white/10 hover:text-white"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <p className="mt-2 text-sm text-white/70">{message}</p>
          </div>
        </div>

        <div className="mt-5 flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onCancel}>
            {cancelLabel}
          </Button>
          <Button
            size="sm"
            onClick={onConfirm}
            className={`${styles.button} text-white`}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
