import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { cn } from "../../lib/utils";

/**
 * Minimal modal Dialog primitive.
 *
 * Renders a centered modal with backdrop dismiss + Escape dismiss + focus
 * trap. Hand-rolled (no Radix dep) — kept intentionally simple. The shape
 * mirrors the Radix Dialog surface so a future swap is mechanical.
 *
 * For mobile bottom-sheet behavior use `<Sheet side="bottom" />` instead.
 */
export interface DialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
  /**
   * Optional aria-label for the dialog. Required when no DialogTitle is
   * rendered (e.g. confirmation prompts that use a description only).
   */
  ariaLabel?: string;
}

export function Dialog({ open, onOpenChange, children, ariaLabel }: DialogProps) {
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) return undefined;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onOpenChange(false);
    };
    window.addEventListener("keydown", onKey);
    const previouslyFocused = document.activeElement as HTMLElement | null;
    // Focus the first focusable element inside the dialog.
    const focusFirst = () => {
      const focusables = ref.current?.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
      );
      focusables?.[0]?.focus();
    };
    const t = setTimeout(focusFirst, 0);
    return () => {
      window.removeEventListener("keydown", onKey);
      clearTimeout(t);
      if (previouslyFocused && typeof previouslyFocused.focus === "function") {
        previouslyFocused.focus();
      }
    };
  }, [open, onOpenChange]);

  if (!open || typeof document === "undefined") return null;

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      role="presentation"
    >
      <button
        type="button"
        aria-hidden="true"
        tabIndex={-1}
        className="absolute inset-0 cursor-default bg-black/60 backdrop-blur-sm"
        onClick={() => onOpenChange(false)}
      />
      <div
        ref={ref}
        role="dialog"
        aria-modal="true"
        aria-label={ariaLabel}
        className="relative z-10 w-full max-w-md rounded-sheet border border-app-border bg-app-surface-overlay p-6 shadow-2xl"
      >
        {children}
      </div>
    </div>,
    document.body,
  );
}

export function DialogTitle({ children, className }: { children: ReactNode; className?: string }) {
  return <h2 className={cn("text-lg font-semibold text-app-foreground", className)}>{children}</h2>;
}

export function DialogDescription({ children, className }: { children: ReactNode; className?: string }) {
  return <p className={cn("mt-2 text-sm text-app-muted-foreground", className)}>{children}</p>;
}

export function DialogFooter({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("mt-6 flex justify-end gap-2", className)}>{children}</div>;
}
