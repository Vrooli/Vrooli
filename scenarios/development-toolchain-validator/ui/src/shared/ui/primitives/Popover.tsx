import { useEffect, useRef, type ReactNode } from "react";
import { cn } from "../../lib/utils";

/**
 * Popover primitive — anchored content panel.
 *
 * Consumer-controlled (`open` / `onOpenChange`). Renders the popover as a
 * sibling to the trigger via absolute positioning. Outside-click + Escape
 * dismiss. Does NOT trap focus (popovers are not modal).
 */
export interface PopoverProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  trigger: ReactNode;
  children: ReactNode;
  align?: "left" | "right";
  className?: string;
}

export function Popover({
  open,
  onOpenChange,
  trigger,
  children,
  align = "left",
  className,
}: PopoverProps) {
  const wrapperRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!open) return undefined;
    const onClickOutside = (e: MouseEvent) => {
      if (wrapperRef.current && !wrapperRef.current.contains(e.target as Node)) {
        onOpenChange(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onOpenChange(false);
    };
    document.addEventListener("mousedown", onClickOutside);
    window.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClickOutside);
      window.removeEventListener("keydown", onKey);
    };
  }, [open, onOpenChange]);

  return (
    <div ref={wrapperRef} className="relative inline-block">
      {trigger}
      {open ? (
        <div
          role="dialog"
          className={cn(
            "absolute z-30 mt-1 min-w-[12rem] rounded-panel border border-app-border bg-app-surface-overlay p-2 shadow-lg",
            align === "right" ? "right-0" : "left-0",
            className,
          )}
        >
          {children}
        </div>
      ) : null}
    </div>
  );
}
