import { useEffect, useId, useRef, type ReactNode } from "react";

import { useEscapeKey } from "@vrooli/react-component-library/useEscapeKey/1.0.0";
import { useFocusTrap } from "@vrooli/react-component-library/useFocusTrap/1.0.0";
import { cn } from "../lib/classnames";

interface ConfirmDialogProps {
  /** Whether the dialog is mounted/visible. Returns null when false. */
  open: boolean;
  /** Heading of the dialog (pre-translated). */
  title: ReactNode;
  /** Question / consequence copy (pre-translated; interpolation done by caller). */
  body: ReactNode;
  /** Label of the safe button (auto-focused on open). */
  cancelLabel: string;
  /** Label of the action button. */
  confirmLabel: string;
  /** Style the confirm button as destructive (red). */
  destructive?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
  /**
   * Prefix for the data-testids: `<prefix>-dialog`, `<prefix>-cancel`,
   * `<prefix>-confirm`.
   */
  testIdPrefix: string;
}

/**
 * ConfirmDialog is the single confirm primitive for destructive yes/no
 * decisions (close session, discard attachments). It renders a centered card
 * on the confirm z tier — above the drawer tier — so it works standalone and
 * layered over an open DrawerShell. Owns role=alertdialog semantics, Escape
 * (= cancel), focus trapping, and auto-focusing the safe Cancel button.
 */
export function ConfirmDialog({
  open,
  title,
  body,
  cancelLabel,
  confirmLabel,
  destructive = false,
  onCancel,
  onConfirm,
  testIdPrefix,
}: ConfirmDialogProps) {
  useEscapeKey(open, onCancel);

  const panelRef = useRef<HTMLDivElement>(null);
  useFocusTrap(open, panelRef);

  // Auto-focus Cancel for safety: Enter never confirms destruction by default.
  const cancelRef = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    if (open) cancelRef.current?.focus();
  }, [open]);

  const titleId = useId();
  const bodyId = useId();

  if (!open) return null;

  return (
    <div
      data-testid={`${testIdPrefix}-dialog`}
      className="fixed inset-0 z-wc-confirm flex items-center justify-center bg-wc-backdrop p-4"
      onClick={onCancel}
    >
      <div
        ref={panelRef}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={bodyId}
        className="wc-stable-theme w-full max-w-sm rounded-lg border border-wc-default bg-wc-surface-raised p-5 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 id={titleId} className="mb-2 text-sm font-semibold text-wc-text-primary">
          {title}
        </h2>
        <p id={bodyId} className="mb-4 text-xs text-wc-text-secondary">
          {body}
        </p>
        <div className="flex justify-end gap-2">
          <button
            ref={cancelRef}
            type="button"
            data-testid={`${testIdPrefix}-cancel`}
            className="rounded-full px-4 py-1.5 text-sm font-medium text-wc-text-primary transition hover:bg-white/10"
            onClick={onCancel}
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            data-testid={`${testIdPrefix}-confirm`}
            className={cn(
              "rounded-full px-4 py-1.5 text-sm font-medium transition",
              destructive
                ? "bg-red-600 text-white hover:bg-red-700"
                : "bg-wc-accent text-wc-accent-fg hover:opacity-90",
            )}
            onClick={onConfirm}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
