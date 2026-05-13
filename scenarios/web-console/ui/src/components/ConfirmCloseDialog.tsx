import { useEffect, useRef } from "react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";

interface ConfirmCloseDialogProps {
  open: boolean;
  sessionName: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export default function ConfirmCloseDialog({
  open,
  sessionName,
  onConfirm,
  onCancel,
}: ConfirmCloseDialogProps) {
  const { t } = useTranslation();
  const cancelRef = useRef<HTMLButtonElement>(null);

  // Auto-focus Cancel for safety, handle Escape
  useEffect(() => {
    if (!open) return;
    cancelRef.current?.focus();

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onCancel();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, onCancel]);

  if (!open) return null;

  return (
    <div
      data-testid="confirm-close-dialog"
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={onCancel}
    >
      <div
        className="mx-4 w-full max-w-sm rounded-lg border border-wc-default bg-wc-surface-raised p-5 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-2 text-sm font-semibold text-wc-text-primary">
          {t(strings.confirmClose.title)}
        </h2>
        <p className="mb-4 text-xs text-wc-text-secondary">
          {t(strings.confirmClose.body, { name: sessionName })}
        </p>
        <div className="flex justify-end gap-2">
          <button
            ref={cancelRef}
            className="rounded-full px-4 py-1.5 text-sm font-medium text-white hover:bg-white/10 transition-colors"
            onClick={onCancel}
            data-testid="confirm-close-cancel"
          >
            {t(strings.confirmClose.cancel)}
          </button>
          <button
            className="rounded-full px-4 py-1.5 text-sm font-medium bg-red-600 text-white hover:bg-red-700 transition-colors"
            onClick={onConfirm}
            data-testid="confirm-close-confirm"
          >
            {t(strings.confirmClose.confirm)}
          </button>
        </div>
      </div>
    </div>
  );
}
