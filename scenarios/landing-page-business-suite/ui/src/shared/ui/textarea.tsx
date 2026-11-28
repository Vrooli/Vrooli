import * as React from "react";
import { cn } from "../lib/utils";

export type TextareaSize = "sm" | "md" | "lg";

const textareaSizeClassName: Record<TextareaSize, string> = {
  sm: "",
  md: "px-4 py-3 text-base",
  lg: "px-5 py-4 text-lg",
};

export interface TextareaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  size?: TextareaSize;
}

// text-base on mobile (16px) prevents iOS Safari auto-zoom on focus; md:text-sm restores desktop density.
export const textareaBaseClassName =
  "min-h-0 w-full resize-none rounded-lg border border-white/10 bg-surface-primary/70 px-3 py-2 text-base text-white placeholder:text-slate-500 focus:border-white/20 focus:outline-none focus:ring-1 focus:ring-white/20 disabled:cursor-not-allowed disabled:opacity-50 md:text-sm";

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, size = "sm", ...props }, ref) => (
    <textarea
      className={cn(textareaBaseClassName, textareaSizeClassName[size], className)}
      ref={ref}
      {...props}
    />
  )
);

Textarea.displayName = "Textarea";

export { Textarea };
