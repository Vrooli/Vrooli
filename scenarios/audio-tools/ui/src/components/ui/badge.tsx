import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-pill px-2 py-0.5 text-xs font-medium",
  {
    variants: {
      variant: {
        neutral: "bg-app-surface-muted text-app-muted-foreground border border-app-border",
        info: "bg-app-info-soft text-app-info border border-app-info/30",
        success: "bg-app-success-soft text-app-success border border-app-success/30",
        warning: "bg-app-warning-soft text-app-warning border border-app-warning/30",
        danger: "bg-app-danger-soft text-app-danger border border-app-danger/30",
        primary: "bg-app-primary text-app-primary-foreground",
      },
    },
    defaultVariants: { variant: "neutral" },
  },
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}
