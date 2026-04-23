import * as React from "react";
import { cn } from "../lib/utils";

export type InputSize = "sm" | "md" | "lg";

const inputSizeClassName: Record<InputSize, string> = {
  sm: "",
  md: "px-4 py-3 text-base",
  lg: "px-5 py-4 text-lg",
};

export type InputProps = Omit<React.InputHTMLAttributes<HTMLInputElement>, "size"> & {
  size?: InputSize;
};

// text-base on mobile (16px) prevents iOS Safari auto-zoom on focus; md:text-sm restores desktop density. Shared with textareaBaseClassName below.
export const inputBaseClassName =
  "w-full rounded-lg border border-white/10 bg-surface-primary/70 px-3 py-2 text-base md:text-sm text-white placeholder:text-slate-500 focus:border-white/20 focus:outline-none focus:ring-1 focus:ring-white/20 disabled:cursor-not-allowed disabled:opacity-50";

export const textareaBaseClassName = `${inputBaseClassName} resize-none`;

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, type, size = "sm", ...props }, ref) => {
    return (
      <input
        type={type}
        className={cn(inputBaseClassName, inputSizeClassName[size], className)}
        ref={ref}
        {...props}
      />
    );
  }
);
Input.displayName = "Input";

export { Input };

export interface TextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  size?: InputSize;
}

const Textarea = React.forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, size = "sm", ...props }, ref) => {
    return (
      <textarea
        className={cn(textareaBaseClassName, inputSizeClassName[size], className)}
        ref={ref}
        {...props}
      />
    );
  }
);

Textarea.displayName = "Textarea";

export { Textarea };
