import * as React from "react";
import { cn } from "../../lib/utils";

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

/**
 * Input primitive — token-driven. `text-base` on mobile (16px) prevents iOS Safari
 * auto-zoom on focus; `md:text-sm` restores desktop density.
 */
export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(
          "flex h-10 w-full rounded-control border border-app-border bg-app-surface-input px-3 py-2 text-base text-app-foreground placeholder:text-app-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-app-accent/60 disabled:cursor-not-allowed disabled:opacity-60 md:text-sm",
          className,
        )}
        ref={ref}
        {...props}
      />
    );
  },
);
Input.displayName = "Input";
