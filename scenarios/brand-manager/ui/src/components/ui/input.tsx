import { forwardRef } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const inputVariants = cva(
  "w-full rounded-lg border bg-white/5 text-sm text-slate-50 placeholder:text-slate-500 focus:outline-none transition-colors",
  {
    variants: {
      variant: {
        default: "border-white/10 focus:border-white/30 px-3 py-2",
        search: "border-white/10 focus:border-white/30 pl-10 pr-4 py-2",
      },
      inputSize: {
        default: "",
        sm: "text-xs px-2 py-1",
      },
    },
    defaultVariants: {
      variant: "default",
      inputSize: "default",
    },
  },
);

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement>,
    VariantProps<typeof inputVariants> {}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, variant, inputSize, ...props }, ref) => {
    return (
      <input
        ref={ref}
        className={cn(inputVariants({ variant, inputSize, className }))}
        {...props}
      />
    );
  },
);

Input.displayName = "Input";

export interface TextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  className?: string;
}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => {
    return (
      <textarea
        ref={ref}
        className={cn(
          "w-full rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-slate-50 placeholder:text-slate-500 focus:border-white/30 focus:outline-none resize-none transition-colors",
          className,
        )}
        {...props}
      />
    );
  },
);

Textarea.displayName = "Textarea";
