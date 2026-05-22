import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-pill px-2 py-0.5 text-xs font-medium",
  {
    variants: {
      variant: {
        default: "bg-app-surface-muted text-app-foreground border border-app-border",
        info: "bg-app-info/15 text-app-info border border-app-info/40",
        success: "bg-app-success/15 text-app-success border border-app-success/40",
        warning: "bg-app-warning/15 text-app-warning border border-app-warning/40",
        danger: "bg-app-danger/15 text-app-danger border border-app-danger/40",
        outline: "border border-app-border text-app-foreground bg-transparent",
      },
    },
    defaultVariants: { variant: "default" },
  },
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
