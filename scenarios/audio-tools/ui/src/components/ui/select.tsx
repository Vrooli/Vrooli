import * as React from "react";
import { ChevronDown } from "lucide-react";
import { cn } from "../../lib/utils";

export type SelectProps = React.SelectHTMLAttributes<HTMLSelectElement>;

/**
 * Styled native <select>. Wrapped so we can render a chevron without breaking
 * the native open-on-click affordance. The wrapper inherits `display: flex`
 * so consumers can size with `className`/`style` exactly like a bare select.
 */
export const Select = React.forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, children, ...props }, ref) => (
    <span className="relative inline-flex w-full">
      <select
        ref={ref}
        className={cn(
          "flex h-10 w-full appearance-none rounded-control border border-app-border bg-app-surface px-3 py-2 pr-9 text-base text-app-foreground transition-colors hover:border-app-border-strong focus:outline-none focus-visible:ring-2 focus-visible:ring-app-focus/50 disabled:cursor-not-allowed disabled:opacity-60 md:text-sm",
          className,
        )}
        {...props}
      >
        {children}
      </select>
      <ChevronDown
        aria-hidden="true"
        className="pointer-events-none absolute right-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-app-muted-foreground"
      />
    </span>
  ),
);
Select.displayName = "Select";
