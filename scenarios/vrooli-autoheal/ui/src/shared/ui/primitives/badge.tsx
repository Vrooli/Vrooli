import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-pill border px-3 py-1 text-xs font-medium tracking-wide",
  {
    variants: {
      tone: {
        neutral: "border-border-strong/70 bg-surface-overlay/60 text-text-primary",
        info: "border-accent-primary/40 bg-accent-primary/20 text-accent-primary",
        success: "border-accent-success/40 bg-accent-success/20 text-accent-success",
        warning: "border-accent-warning/40 bg-accent-warning/20 text-accent-warning",
        danger: "border-accent-danger/40 bg-accent-danger/20 text-accent-danger",
      },
      size: {
        default: "",
        sm: "px-2 py-0.5 text-[11px]",
      },
    },
    defaultVariants: {
      tone: "neutral",
      size: "default",
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, tone, size, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ tone, size, className }))} {...props} />;
}
