import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/50 disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        default: "bg-slate-50 text-slate-900 hover:bg-white",
        outline: "border border-slate-700 text-slate-200 hover:bg-slate-800",
        ghost: "text-slate-200 hover:bg-slate-800",
        success: "bg-emerald-600 text-white hover:bg-emerald-500",
        destructive: "bg-red-600 text-white hover:bg-red-500",
        secondary: "bg-slate-700 text-slate-100 hover:bg-slate-600"
      },
      size: {
        default: "h-10 px-4",
        sm: "h-8 px-3 text-xs",
        lg: "h-12 px-6",
        icon: "h-9 w-9"
      }
    },
    defaultVariants: {
      variant: "default",
      size: "default"
    }
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

export function Button({ className, variant, size, asChild = false, ...props }: ButtonProps) {
  const Comp = asChild ? Slot : "button";

  return <Comp className={cn(buttonVariants({ variant, size, className }))} {...props} />;
}
