import { cva, type VariantProps } from "class-variance-authority";
import type { HTMLAttributes } from "react";
import { cn } from "../../lib/utils";

const skeletonVariants = cva(
  "animate-pulse rounded-control bg-app-surface-muted",
  {
    variants: {
      variant: {
        row: "h-4 w-full",
        card: "h-24 w-full",
        grid: "h-20 w-full",
        chip: "h-5 w-16",
      },
    },
    defaultVariants: { variant: "row" },
  },
);

export interface LoadingSkeletonProps
  extends HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof skeletonVariants> {
  /** Number of skeleton rows/cards to render. Default 1. */
  count?: number;
}

/**
 * Loading skeleton. Renders `count` placeholder blocks of the chosen variant.
 * Variants: `row` (default), `card`, `grid`, `chip`.
 */
export function LoadingSkeleton({
  variant,
  count = 1,
  className,
  ...props
}: LoadingSkeletonProps) {
  return (
    <div className={cn("flex flex-col gap-2", className)} role="status" aria-live="polite" {...props}>
      {Array.from({ length: count }).map((_, i) => (
        <div key={i} data-variant={variant ?? "row"} className={cn(skeletonVariants({ variant }))} />
      ))}
    </div>
  );
}
