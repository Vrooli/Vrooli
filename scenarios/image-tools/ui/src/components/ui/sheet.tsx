import { useEffect, useRef, type ReactNode } from "react";
import { X } from "lucide-react";

export interface SheetProps {
  /** Whether the sheet is mounted/open. */
  open: boolean;
  /** Close request (backdrop click, Escape, or the close button). */
  onClose: () => void;
  /** Accessible title rendered in the header and wired to aria-labelledby. */
  title: string;
  /** Optional sub-line under the title (e.g. a hardware summary). */
  subtitle?: ReactNode;
  /** Localized accessible label for the close button. */
  closeLabel: string;
  /** Stable test id for the dialog container. */
  testId?: string;
  children: ReactNode;
}

/**
 * Sheet is the app's overlay primitive. It renders as a centered modal dialog on
 * desktop (`sm:` and up) and as a full-screen drawer on mobile — the pattern the
 * model picker uses so a one-handed phone tap gets the full menu, while a desktop
 * click gets a focused dialog. It owns the modal essentials: a backdrop, body
 * scroll-lock, Escape-to-close, an initial-focus + focus-trap loop, and
 * role="dialog"/aria-modal wired to the title. Purely presentational — callers
 * own `open` and the content.
 */
export function Sheet({ open, onClose, title, subtitle, closeLabel, testId, children }: SheetProps) {
  const panelRef = useRef<HTMLDivElement | null>(null);
  const titleId = useRef(`sheet-title-${Math.random().toString(36).slice(2)}`).current;

  // Lock body scroll while open so the page behind the drawer can't scroll.
  useEffect(() => {
    if (!open) {
      return;
    }
    const previous = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = previous;
    };
  }, [open]);

  // Move focus into the panel on open (the close button's focus is fine).
  useEffect(() => {
    if (open) {
      panelRef.current?.focus();
    }
  }, [open]);

  // Escape-to-close + a minimal Tab focus-trap, attached at the document level
  // (more robust than an element handler — it works no matter where focus sits).
  useEffect(() => {
    if (!open) {
      return;
    }
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        onClose();
        return;
      }
      if (e.key !== "Tab") {
        return;
      }
      const focusable = panelRef.current?.querySelectorAll<HTMLElement>(
        'a[href],button:not([disabled]),textarea,input,select,[tabindex]:not([tabindex="-1"])',
      );
      if (!focusable || focusable.length === 0) {
        return;
      }
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      const active = document.activeElement;
      if (e.shiftKey && active === first) {
        e.preventDefault();
        last?.focus();
      } else if (!e.shiftKey && active === last) {
        e.preventDefault();
        first?.focus();
      }
    };
    document.addEventListener("keydown", handler, true);
    return () => document.removeEventListener("keydown", handler, true);
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-stretch justify-center sm:items-center sm:p-4"
      role="presentation"
    >
      <button
        type="button"
        tabIndex={-1}
        aria-label={closeLabel}
        className="absolute inset-0 bg-app-overlay backdrop-blur-sm"
        onClick={onClose}
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        data-testid={testId}
        tabIndex={-1}
        className="relative flex h-full w-full flex-col overflow-hidden bg-app-surface shadow-xl outline-none sm:h-auto sm:max-h-[85vh] sm:max-w-2xl sm:rounded-panel sm:border sm:border-app-border"
      >
        <header className="flex items-start justify-between gap-3 border-b border-app-border px-4 py-3">
          <div className="min-w-0">
            <h2 id={titleId} className="text-base font-semibold text-app-foreground">
              {title}
            </h2>
            {subtitle ? (
              <div className="mt-0.5 text-xs text-app-muted-foreground">{subtitle}</div>
            ) : null}
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label={closeLabel}
            data-testid={testId ? `${testId}-close` : undefined}
            className="shrink-0 rounded-control p-1.5 text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
          >
            <X aria-hidden="true" className="h-5 w-5" />
          </button>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto">{children}</div>
      </div>
    </div>
  );
}
