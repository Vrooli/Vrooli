/**
 * Card Component
 *
 * A reusable container component for content sections, panels, and cards.
 * Uses CVA for consistent styling variants across the application.
 *
 * This consolidates the repeated "rounded-xl border border-white/10 bg-slate-800/30"
 * pattern found across 16+ locations in the UI.
 */

import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const cardVariants = cva(
  "rounded-xl border border-white/10 bg-slate-800/30",
  {
    variants: {
      padding: {
        none: "",
        sm: "p-4",
        default: "p-6",
        lg: "p-8",
      },
      interactive: {
        true: "transition hover:border-cyan-500/50 hover:bg-slate-800/50 cursor-pointer",
        false: "",
      },
      centered: {
        true: "text-center",
        false: "",
      },
    },
    defaultVariants: {
      padding: "default",
      interactive: false,
      centered: false,
    },
  }
);

export interface CardProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof cardVariants> {}

/**
 * Card component for consistent content containers.
 *
 * Usage:
 * ```tsx
 * // Basic panel
 * <Card>Content here</Card>
 *
 * // Centered content (for empty states, loading, etc.)
 * <Card padding="lg" centered>
 *   <Icon />
 *   <p>No items found</p>
 * </Card>
 *
 * // Interactive card (for clickable items)
 * <Card padding="sm" interactive>
 *   <h3>Item title</h3>
 * </Card>
 * ```
 */
export function Card({
  className,
  padding,
  interactive,
  centered,
  ...props
}: CardProps) {
  return (
    <div
      className={cn(cardVariants({ padding, interactive, centered, className }))}
      {...props}
    />
  );
}
