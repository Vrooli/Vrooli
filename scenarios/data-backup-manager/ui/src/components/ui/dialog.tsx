import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";

import { cn } from "../../lib/utils";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Lightweight controlled dialog. On desktop it's a centered compact modal; on
 * mobile it becomes a full-screen panel (DESIGN.md: complex dialogs are
 * full-screen on mobile, especially forms and multi-step decisions). Closes on
 * Escape and backdrop click; moves focus to the panel on open.
 */
export function Dialog({
  open,
  onClose,
  title,
  children,
  footer,
  "data-testid": testId,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  footer?: ReactNode;
  "data-testid"?: string;
}) {
  const { t } = useTranslation();
  const panelRef = useRef<HTMLDivElement>(null);

  // Focus the panel only on the open transition — not on every render — so
  // typing into a field inside the dialog doesn't get its focus stolen back.
  useEffect(() => {
    if (open) panelRef.current?.focus();
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-end justify-center sm:items-center sm:p-4">
      {/* Real button backdrop: click-to-close that is keyboard-reachable (and
          Escape closes too), satisfying the interactive-element a11y rule. */}
      <button
        type="button"
        aria-label={t(strings.common.close)}
        onClick={onClose}
        className="absolute inset-0 cursor-default bg-black/40"
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-label={title}
        tabIndex={-1}
        data-testid={testId}
        className={cn(
          "relative z-10 flex max-h-[90vh] w-full flex-col overflow-hidden bg-app-surface shadow-xl outline-none",
          "rounded-t-sheet sm:max-w-lg sm:rounded-panel",
        )}
      >
        <div className="flex items-center justify-between border-b border-app-border px-4 py-3">
          <h2 className="text-sm font-semibold text-app-foreground">{title}</h2>
          <button
            type="button"
            aria-label={t(strings.common.close)}
            onClick={onClose}
            className="rounded-control p-1 text-app-muted-foreground hover:bg-app-surface-muted"
          >
            <X aria-hidden="true" className="h-4 w-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto px-4 py-4">{children}</div>
        {footer && (
          <div className="flex items-center justify-end gap-2 border-t border-app-border px-4 py-3">
            {footer}
          </div>
        )}
      </div>
    </div>,
    document.body,
  );
}
