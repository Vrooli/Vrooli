import * as React from "react";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";

/**
 * Lightweight tooltip. Shows on focus / mouseenter; hides on blur / mouseleave.
 * Uses an inline visually-rendered span rather than a portal, so it works in
 * the test environment without DOM portals. Accessible: tooltip is owned by
 * the trigger via aria-describedby; the trigger is the actual focusable
 * element.
 */
export interface TooltipProps {
  /** Tooltip body text. */
  label: string;
  /** Element to attach the tooltip to. Must be a single focusable child. */
  children: React.ReactElement<{
    "aria-describedby"?: string;
    onFocus?: (e: React.FocusEvent) => void;
    onBlur?: (e: React.FocusEvent) => void;
    onMouseEnter?: (e: React.MouseEvent) => void;
    onMouseLeave?: (e: React.MouseEvent) => void;
  }>;
  className?: string;
}

let TOOLTIP_ID_COUNTER = 0;

export function Tooltip({ label, children, className }: TooltipProps) {
  const [open, setOpen] = React.useState(false);
  const id = React.useMemo(() => `tooltip-${++TOOLTIP_ID_COUNTER}`, []);

  const trigger = React.cloneElement(children, {
    "aria-describedby": open ? id : undefined,
    onFocus: (e: React.FocusEvent) => {
      setOpen(true);
      children.props.onFocus?.(e);
    },
    onBlur: (e: React.FocusEvent) => {
      setOpen(false);
      children.props.onBlur?.(e);
    },
    onMouseEnter: (e: React.MouseEvent) => {
      setOpen(true);
      children.props.onMouseEnter?.(e);
    },
    onMouseLeave: (e: React.MouseEvent) => {
      setOpen(false);
      children.props.onMouseLeave?.(e);
    },
  });

  return (
    <span className="relative inline-flex">
      {trigger}
      {open ? (
        <span
          id={id}
          role="tooltip"
          data-testid={selectors.ui.tooltip.root}
          className={cn(
            "pointer-events-none absolute left-1/2 top-full z-50 mt-1 -translate-x-1/2 whitespace-nowrap rounded-control border border-app-border bg-app-surface px-2 py-1 text-xs text-app-foreground shadow-md",
            className,
          )}
        >
          {label}
        </span>
      ) : null}
    </span>
  );
}
