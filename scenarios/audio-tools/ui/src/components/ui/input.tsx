import * as React from "react";
import { cn } from "../../lib/utils";

export type InputProps = React.InputHTMLAttributes<HTMLInputElement>;

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, ...props }, ref) => {
    return (
      <input
        type={type}
        // text-base on mobile (16px) prevents iOS Safari auto-zoom on focus; md:text-sm restores desktop density.
        className={cn(
          "flex h-10 w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-base md:text-sm text-app-foreground placeholder:text-app-muted-foreground transition-colors hover:border-app-border-strong focus:outline-none focus-visible:ring-2 focus-visible:ring-app-focus/50 disabled:cursor-not-allowed disabled:opacity-60",
          className
        )}
        ref={ref}
        {...props}
      />
    );
  }
);
Input.displayName = "Input";

export { Input };
