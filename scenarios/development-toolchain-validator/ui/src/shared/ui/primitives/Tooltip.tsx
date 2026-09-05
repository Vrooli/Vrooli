import { cloneElement, useId, useState, type ReactElement, type ReactNode } from "react";
import { cn } from "../../lib/utils";

/**
 * Tooltip primitive — hover/focus reveal.
 *
 * Lightweight, no portal — the tooltip renders as a sibling positioned via
 * absolute + transform. Wraps a single trigger element via cloneElement so
 * the trigger keeps its native handlers; consumers pass children. Mobile
 * users get the tooltip on focus (long-press / tap-and-hold via the
 * keyboard substrate); explicit `defaultOpen` is supported for tests.
 *
 * For complex hover patterns (delay, follow-cursor, etc.) we'd reach for
 * Radix Tooltip; this DTV-scoped version covers the "explain a chip" use
 * case the surfaces need.
 */
export interface TooltipProps {
  /** Trigger element. Must accept ref + native handlers. */
  children: ReactElement<{
    onMouseEnter?: React.MouseEventHandler;
    onMouseLeave?: React.MouseEventHandler;
    onFocus?: React.FocusEventHandler;
    onBlur?: React.FocusEventHandler;
    "aria-describedby"?: string;
  }>;
  /** Tooltip content (string or simple node). */
  content: ReactNode;
  /** Render-side override for tests/debug. */
  defaultOpen?: boolean;
  className?: string;
}

export function Tooltip({ children, content, defaultOpen = false, className }: TooltipProps) {
  const id = useId();
  const [open, setOpen] = useState(defaultOpen);

  const trigger = cloneElement(children, {
    onMouseEnter: (e: React.MouseEvent) => {
      children.props.onMouseEnter?.(e);
      setOpen(true);
    },
    onMouseLeave: (e: React.MouseEvent) => {
      children.props.onMouseLeave?.(e);
      setOpen(false);
    },
    onFocus: (e: React.FocusEvent) => {
      children.props.onFocus?.(e);
      setOpen(true);
    },
    onBlur: (e: React.FocusEvent) => {
      children.props.onBlur?.(e);
      setOpen(false);
    },
    "aria-describedby": id,
  });

  return (
    <span className="relative inline-flex">
      {trigger}
      {open ? (
        <span
          role="tooltip"
          id={id}
          className={cn(
            "pointer-events-none absolute bottom-full left-1/2 z-30 mb-1 -translate-x-1/2 whitespace-nowrap rounded-control border border-app-border bg-app-surface-overlay px-2 py-1 text-xs text-app-foreground shadow-lg",
            className,
          )}
        >
          {content}
        </span>
      ) : null}
    </span>
  );
}
