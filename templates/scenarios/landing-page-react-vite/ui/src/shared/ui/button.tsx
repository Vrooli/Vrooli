import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-full text-sm font-semibold transition-colors duration-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:ring-offset-slate-900 disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        default: "bg-[#F97316] text-white hover:bg-[#fb8c35] focus-visible:ring-[#F97316]",
        outline: "border border-white/20 text-white hover:border-white/60 focus-visible:ring-white/40",
        ghost: "text-slate-200 hover:text-white focus-visible:ring-white/30",
        muted: "bg-white/5 text-white hover:bg-white/10 focus-visible:ring-white/20"
      },
      size: {
        default: "h-12 px-6",
        sm: "h-9 px-4"
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

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, type, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    const componentProps = asChild
      ? props
      : {
          type: type ?? "button",
          ...props,
        };

    return (
      <Comp
        className={cn(buttonVariants({ variant, size }), className)}
        ref={ref}
        {...componentProps}
      />
    );
  }
);

Button.displayName = "Button";

export { buttonVariants };
