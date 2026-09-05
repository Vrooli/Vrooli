import { forwardRef } from "react";
import { cn } from "../../lib/utils";

/**
 * Select primitive — native `<select>` wrapped with token-driven styling.
 *
 * Native select is the right call here:
 *   - mobile gets the platform picker (better UX than a custom dropdown on
 *     a 360w phone)
 *   - it's inherently keyboard accessible
 *   - it doesn't need a focus trap
 *
 * For a richer combobox we'd reach for Radix Select; DTV doesn't need it.
 */
export type SelectProps = React.SelectHTMLAttributes<HTMLSelectElement>;

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, children, ...props }, ref) => (
    <select
      ref={ref}
      className={cn(
        "flex h-10 w-full rounded-control border border-app-border bg-app-surface-input px-3 py-2 text-base text-app-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-app-accent/60 disabled:cursor-not-allowed disabled:opacity-60 md:text-sm",
        className,
      )}
      {...props}
    >
      {children}
    </select>
  ),
);
Select.displayName = "Select";
