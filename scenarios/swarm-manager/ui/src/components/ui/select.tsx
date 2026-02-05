import { forwardRef } from "react";
import { ChevronDown } from "lucide-react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const selectVariants = cva(
  "w-full appearance-none rounded-lg border text-slate-100 focus:outline-none transition-colors disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        default:
          "border-white/10 bg-slate-800/50 px-4 py-2 text-sm focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500",
        filter: "border-white/10 bg-slate-700 px-3 py-1.5 text-sm text-slate-200 focus:border-cyan-500",
        compact:
          "border-white/10 bg-slate-900 px-2 py-1 text-xs text-slate-200 focus:border-cyan-500 focus:ring-1 focus:ring-cyan-500",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

export interface SelectProps
  extends React.SelectHTMLAttributes<HTMLSelectElement>,
    VariantProps<typeof selectVariants> {
  withChevron?: boolean;
  wrapperClassName?: string;
}

export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, variant, withChevron = false, wrapperClassName, ...props }, ref) => {
    const select = (
      <select
        ref={ref}
        className={cn(selectVariants({ variant, className }), withChevron ? "pr-8" : "")}
        {...props}
      />
    );

    if (!withChevron) {
      return select;
    }

    return (
      <div className={cn("relative", wrapperClassName)}>
        {select}
        <ChevronDown className="pointer-events-none absolute right-2 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
      </div>
    );
  }
);

Select.displayName = "Select";
