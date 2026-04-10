import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

// ─────────────────────────────────────────────────────────────────────────────
// Badge Primitive
// [REQ:P0-001] Reference Scenario Registry - Semantic status badges
// ─────────────────────────────────────────────────────────────────────────────
//
// A semantic badge component for displaying status, counts, and metadata.
// Uses CVA for consistent variant styling across the application.
// ─────────────────────────────────────────────────────────────────────────────

const badgeVariants = cva(
  "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium",
  {
    variants: {
      variant: {
        default: "bg-slate-500/20 text-slate-300",
        primary: "bg-indigo-500/20 text-indigo-300",
        success: "bg-emerald-500/20 text-emerald-300",
        warning: "bg-amber-500/20 text-amber-300",
        danger: "bg-red-500/20 text-red-300"
      },
      size: {
        default: "px-2.5 py-1 text-xs",
        sm: "px-2 py-0.5 text-xs",
        lg: "px-3 py-1.5 text-sm"
      }
    },
    defaultVariants: {
      variant: "default",
      size: "default"
    }
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, size, ...props }: BadgeProps) {
  return (
    <span
      className={cn(badgeVariants({ variant, size, className }))}
      {...props}
    />
  );
}
