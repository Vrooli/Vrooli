import * as React from "react";
import { cn } from "../../lib/utils";

export type TextareaProps = React.TextareaHTMLAttributes<HTMLTextAreaElement>;

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        // text-base on mobile (16px) prevents iOS Safari auto-zoom on focus; md:text-sm restores desktop density.
        className={cn(
          "flex min-h-[80px] w-full rounded-md border border-white/20 bg-white/5 px-3 py-2 text-base md:text-sm text-white placeholder:text-white/40 focus:outline-none focus:ring-2 focus:ring-white/40 disabled:cursor-not-allowed disabled:opacity-60",
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
