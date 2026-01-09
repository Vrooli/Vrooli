import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full px-3 py-1 text-xs font-medium transition-colors",
  {
    variants: {
      variant: {
        default: "bg-slate-700 text-slate-50",
        success: "bg-green-900/50 text-green-200 border border-green-700",
        warning: "bg-yellow-900/50 text-yellow-200 border border-yellow-700",
        error: "bg-red-900/50 text-red-200 border border-red-700",
        info: "bg-blue-900/50 text-blue-200 border border-blue-700"
      }
    },
    defaultVariants: {
      variant: "default"
    }
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant }), className)} {...props} />;
}
