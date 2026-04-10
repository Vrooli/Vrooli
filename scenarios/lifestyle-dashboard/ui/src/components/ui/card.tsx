/**
 * Card primitive component
 *
 * Provides a consistent card appearance for dashboard panels and content blocks.
 * Uses CVA for variant support.
 */
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const cardVariants = cva(
  "rounded-xl border border-white/10 bg-white/5",
  {
    variants: {
      interactive: {
        true: "hover:bg-white/[0.07] transition-colors cursor-pointer",
        false: "",
      },
      padding: {
        default: "p-4",
        lg: "p-6",
        none: "",
      },
    },
    defaultVariants: {
      interactive: false,
      padding: "default",
    },
  }
);

export interface CardProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof cardVariants> {}

export function Card({
  className,
  interactive,
  padding,
  ...props
}: CardProps) {
  return (
    <div
      className={cn(cardVariants({ interactive, padding, className }))}
      {...props}
    />
  );
}

/**
 * CardHeader provides consistent spacing for card headers with title and optional action.
 */
export function CardHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex items-center justify-between mb-4", className)}
      {...props}
    />
  );
}

/**
 * CardTitle provides consistent heading styling for cards.
 */
export function CardTitle({
  className,
  ...props
}: React.HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h2
      className={cn("text-lg font-medium", className)}
      {...props}
    />
  );
}

/**
 * StatBox provides a compact stat display inside cards.
 * Used for grid layouts with multiple stats (e.g., storage overview).
 */
export function StatBox({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("rounded-lg bg-slate-800/50 p-4", className)}
      {...props}
    />
  );
}
