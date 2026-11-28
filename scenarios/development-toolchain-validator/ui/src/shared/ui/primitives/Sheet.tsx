import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { cva } from "class-variance-authority";
import { cn } from "../../lib/utils";
import { useGlobalKeydown } from "../../../hooks/useGlobalKeydown";

/**
 * Sheet primitive — side-mounted modal panel.
 *
 * Sides:
 *   - right (desktop drawer)
 *   - bottom (mobile bottom sheet)
 *   - left (alternate drawer; used for hamburger menus)
 *
 * Hand-rolled; mirrors Radix Dialog with `side` variants. Includes backdrop
 * dismiss + Escape dismiss + focus trap basics. Animation duration honors
 * `prefers-reduced-motion` via the design-tokens.css global rule.
 */
const sheetVariants = cva(
  "fixed z-50 flex flex-col gap-3 overflow-y-auto border-app-border bg-app-surface-overlay p-6 shadow-2xl transition-transform duration-default ease-out",
  {
    variants: {
      side: {
        right: "inset-y-0 right-0 w-full max-w-md border-l",
        left: "inset-y-0 left-0 w-full max-w-md border-r",
        bottom: "inset-x-0 bottom-0 max-h-[90vh] rounded-t-sheet border-t pb-[max(env(safe-area-inset-bottom),1.5rem)]",
      },
    },
    defaultVariants: { side: "right" },
  },
);

export interface SheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  side?: "right" | "left" | "bottom";
  children: ReactNode;
  ariaLabel?: string;
}

export function Sheet({ open, onOpenChange, side = "right", children, ariaLabel }: SheetProps) {
  const ref = useRef<HTMLDivElement | null>(null);

  useGlobalKeydown((_seq, event) => {
    if (!open) return false;
    if (event.key === "Escape") {
      onOpenChange(false);
      return true;
    }
    return false;
  });

  useEffect(() => {
    if (!open) return undefined;
    const previouslyFocused = document.activeElement as HTMLElement | null;
    const t = setTimeout(() => {
      const focusables = ref.current?.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])',
      );
      focusables?.[0]?.focus();
    }, 0);
    return () => {
      clearTimeout(t);
      if (previouslyFocused && typeof previouslyFocused.focus === "function") {
        previouslyFocused.focus();
      }
    };
  }, [open]);

  if (!open || typeof document === "undefined") return null;

  return createPortal(
    <div className="fixed inset-0 z-50" role="presentation">
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
        data-side={side}
        className={cn(sheetVariants({ side }))}
      >
        {children}
      </div>
    </div>,
    document.body,
  );
}

export function SheetHeader({ children, className }: { children: ReactNode; className?: string }) {
  return <div className={cn("flex items-start justify-between gap-3", className)}>{children}</div>;
}

export function SheetTitle({ children, className }: { children: ReactNode; className?: string }) {
  return <h2 className={cn("text-lg font-semibold text-app-foreground", className)}>{children}</h2>;
}

export function SheetDescription({ children, className }: { children: ReactNode; className?: string }) {
  return <p className={cn("text-sm text-app-muted-foreground", className)}>{children}</p>;
}
