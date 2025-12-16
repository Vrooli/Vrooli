import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium transition-colors",
  {
    variants: {
      variant: {
        default: "bg-slate-800 text-slate-200 border border-slate-700",
        success: "bg-emerald-950 text-emerald-400 border border-emerald-800",
        warning: "bg-amber-950 text-amber-400 border border-amber-800",
        error: "bg-red-950 text-red-400 border border-red-800",
        info: "bg-blue-950 text-blue-400 border border-blue-800",
        staged: "bg-emerald-950 text-emerald-400 border border-emerald-800",
        unstaged: "bg-amber-950 text-amber-400 border border-amber-800",
        untracked: "bg-slate-800 text-slate-400 border border-slate-700",
        conflict: "bg-red-950 text-red-400 border border-red-800"
      }
    },
    defaultVariants: {
      variant: "default"
    }
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant, className }))} {...props} />;
}
