import { cva, type VariantProps } from "class-variance-authority";
import { forwardRef, type HTMLAttributes } from "react";
import { cn } from "../../lib/utils";

/**
 * Badge primitive — verdict + neutral variants.
 *
 * The verdict variants are the canonical visual surface for DTV's
 * pass/stale/unexpected/failure status. Surfaces never inline colors;
 * they reach for `<Badge variant="verdict-pass" />`.
 */
const badgeVariants = cva(
  "inline-flex items-center gap-1 rounded-pill px-2 py-0.5 text-xs font-medium",
  {
    variants: {
      variant: {
        neutral:
          "bg-status-neutral-bg text-status-neutral border border-app-border-subtle",
        info:
          "bg-app-surface-muted text-app-info border border-app-border-subtle",
        "verdict-pass":
          "bg-status-pass-bg text-status-pass border border-status-pass/30",
        "verdict-stale":
          "bg-status-stale-bg text-status-stale border border-status-stale/30",
        "verdict-unexpected":
          "bg-status-unexpected-bg text-status-unexpected border border-status-unexpected/30",
        "verdict-failure":
          "bg-status-failure-bg text-status-failure border border-status-failure/30",
      },
    },
    defaultVariants: {
      variant: "neutral",
    },
  },
);

export interface BadgeProps
  extends HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export const Badge = forwardRef<HTMLSpanElement, BadgeProps>(
  ({ className, variant, ...props }, ref) => (
    <span
      ref={ref}
      data-variant={variant ?? "neutral"}
      className={cn(badgeVariants({ variant }), className)}
      {...props}
    />
  ),
);
Badge.displayName = "Badge";
