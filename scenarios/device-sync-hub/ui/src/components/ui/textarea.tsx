import * as React from "react";
import { cn } from "../../lib/utils";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        // text-base on mobile (16px) prevents iOS Safari auto-zoom on focus; md:text-sm restores desktop density.
        // Token-driven so the field themes correctly in both light and dark.
        className={cn(
          "flex min-h-[80px] w-full rounded-control border border-app-border bg-app-surface px-3 py-2 text-base md:text-sm text-app-foreground placeholder:text-app-muted-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:cursor-not-allowed disabled:opacity-60",
          className
        )}
        ref={ref}
        {...props}
      />
    );
  }
);
Textarea.displayName = "Textarea";

export { Textarea };
