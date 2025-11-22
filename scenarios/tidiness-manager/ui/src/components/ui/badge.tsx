import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-white/50 focus:ring-offset-2",
  {
    variants: {
      variant: {
        default: "bg-slate-700 text-slate-100 border border-slate-600",
        active: "bg-blue-700 text-white border border-blue-600",
        success: "bg-green-700 text-white border border-green-600",
        warning: "bg-yellow-700 text-white border border-yellow-600",
        error: "bg-red-700 text-white border border-red-600",
        paused: "bg-orange-700 text-white border border-orange-600",
        outline: "border border-white/30 text-slate-200",
        ghost: "bg-white/5 text-slate-300 hover:bg-white/10",
      },
      size: {
        default: "text-xs",
        sm: "text-[10px] px-2 py-0.5",
        lg: "text-sm px-3 py-1",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, size, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant, size }), className)} {...props} />
  );
}
