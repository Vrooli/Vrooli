import * as React from "react";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";

export interface DialogProps {
  open: boolean;
  onClose: () => void;
  /** Accessible label for the dialog. Required for SR users. */
  ariaLabel: string;
  /** Optional title element id if the dialog has a visible heading. */
  ariaLabelledBy?: string;
  children: React.ReactNode;
  className?: string;
}

export function Dialog({
  open,
  onClose,
  ariaLabel,
  ariaLabelledBy,
  children,
  className,
}: DialogProps) {
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      data-testid={selectors.ui.dialog.root}
      role="dialog"
      aria-modal="true"
      aria-label={ariaLabelledBy ? undefined : ariaLabel}
      aria-labelledby={ariaLabelledBy}
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
    >
      <button
        type="button"
        data-testid={selectors.ui.dialog.backdrop}
        aria-label={ariaLabel}
        onClick={onClose}
        className="absolute inset-0 bg-black/50 backdrop-blur-sm cursor-default"
        tabIndex={-1}
      />
      <div
        data-testid={selectors.ui.dialog.panel}
        className={cn(
          "relative z-10 w-full max-w-md rounded-panel border border-app-border bg-app-surface p-6 text-app-foreground shadow-xl",
          className,
        )}
      >
        {children}
      </div>
    </div>
  );
}
