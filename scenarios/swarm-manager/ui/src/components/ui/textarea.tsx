import { forwardRef } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

// text-base on mobile (16px) prevents iOS Safari auto-zoom on focus; md:text-sm restores desktop density.
const textareaVariants = cva(
  "w-full rounded-md border bg-slate-900/60 px-3 py-2 text-base md:text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-1 transition-colors disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        default: "border-white/10 focus:border-cyan-500 focus:ring-cyan-500",
        error: "border-red-500/50 focus:border-red-500 focus:ring-red-500",
      },
    },
    defaultVariants: { variant: "default" },
  },
);

export interface TextareaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement>,
    VariantProps<typeof textareaVariants> {}

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, variant, ...props }, ref) => (
    <textarea ref={ref} className={cn(textareaVariants({ variant, className }))} {...props} />
  ),
);
Textarea.displayName = "Textarea";
