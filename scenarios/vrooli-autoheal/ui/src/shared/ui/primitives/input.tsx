import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../../lib/utils";

const inputVariants = cva(
  "w-full rounded-md border bg-surface-overlay/40 px-3 py-2 text-sm text-text-primary placeholder:text-text-muted/70 transition-colors duration-fast focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-primary/70 disabled:cursor-not-allowed disabled:opacity-60",
  {
    variants: {
      intent: {
        default: "border-border-default/70 focus-visible:border-accent-primary/60",
        danger: "border-accent-danger/50 focus-visible:border-accent-danger/70 focus-visible:ring-accent-danger/60",
      },
      size: {
        sm: "h-8 px-2.5 text-xs",
        default: "h-10",
        lg: "h-11 px-4",
      },
    },
    defaultVariants: {
      intent: "default",
      size: "default",
    },
  }
);

export interface InputProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, "size">,
    VariantProps<typeof inputVariants> {}

export function Input({ className, intent, size, ...props }: InputProps) {
  return <input className={cn(inputVariants({ intent, size, className }))} {...props} />;
}
