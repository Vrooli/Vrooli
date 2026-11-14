import { cva, type VariantProps } from "class-variance-authority";
import * as React from "react";

import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-pill px-2 py-0.5 text-xs font-medium",
  {
    variants: {
      tone: {
        neutral: "bg-app-surface-muted text-app-foreground",
        info: "bg-app-info/15 text-app-info",
        success: "bg-app-success/15 text-app-success",
        warn: "bg-app-warning/15 text-app-warning",
        error: "bg-app-danger/15 text-app-danger",
        ok: "bg-app-success/15 text-app-success",
      },
    },
    defaultVariants: { tone: "neutral" },
  },
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export const Badge = React.forwardRef<HTMLSpanElement, BadgeProps>(function Badge(
  { className, tone, ...props },
  ref,
) {
  return <span ref={ref} className={cn(badgeVariants({ tone, className }))} {...props} />;
});
