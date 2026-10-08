import { cva, type VariantProps } from "class-variance-authority";
import { forwardRef, type HTMLAttributes } from "react";
import { cn } from "../../lib/utils";

/**
 * Card primitive set.
 *
 * Variants:
 *   - surface:
 *     - base (default) — `bg-app-surface`
 *     - raised — `bg-app-surface-raised` with subtle elevation shadow
 *     - muted — `bg-app-surface-muted` for nested sections
 *   - padding:
 *     - default — p-4
 *     - compact — p-3
 *     - none — no padding (consumer-managed)
 */
const cardVariants = cva(
  "rounded-panel border border-app-border text-app-foreground",
  {
    variants: {
      surface: {
        base: "bg-app-surface",
        raised: "bg-app-surface-raised shadow-sm",
        muted: "bg-app-surface-muted",
      },
      padding: {
        default: "p-4",
        compact: "p-3",
        none: "",
      },
    },
    defaultVariants: {
      surface: "base",
      padding: "default",
    },
  },
);

export interface CardProps
  extends HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof cardVariants> {}

export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ className, surface, padding, ...props }, ref) => (
    <div
      ref={ref}
      data-variant={surface ?? "base"}
      className={cn(cardVariants({ surface, padding }), className)}
      {...props}
    />
  ),
);
Card.displayName = "Card";

export const CardHeader = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("flex flex-col gap-1", className)} {...props} />
  ),
);
CardHeader.displayName = "CardHeader";

export const CardTitle = forwardRef<HTMLHeadingElement, HTMLAttributes<HTMLHeadingElement>>(
  ({ className, children, ...props }, ref) => (
    <h3
      ref={ref}
      className={cn("text-sm font-semibold text-app-foreground", className)}
      {...props}
    >
      {children}
    </h3>
  ),
);
CardTitle.displayName = "CardTitle";

export const CardDescription = forwardRef<HTMLParagraphElement, HTMLAttributes<HTMLParagraphElement>>(
  ({ className, ...props }, ref) => (
    <p
      ref={ref}
      className={cn("text-xs text-app-muted-foreground", className)}
      {...props}
    />
  ),
);
CardDescription.displayName = "CardDescription";

export const CardContent = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div ref={ref} className={cn("mt-3", className)} {...props} />
  ),
);
CardContent.displayName = "CardContent";

export const CardFooter = forwardRef<HTMLDivElement, HTMLAttributes<HTMLDivElement>>(
  ({ className, ...props }, ref) => (
    <div
      ref={ref}
      className={cn("mt-3 flex items-center gap-2", className)}
      {...props}
    />
  ),
);
CardFooter.displayName = "CardFooter";
