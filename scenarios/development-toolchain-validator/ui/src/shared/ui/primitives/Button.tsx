import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { forwardRef } from "react";
import { cn } from "../../lib/utils";

/**
 * Button primitive — CVA variants for visual intent + size.
 *
 * Variants:
 *   - default: cyan accent (primary actions)
 *   - secondary: muted surface (neutral actions)
 *   - outline: bordered, transparent background
 *   - ghost: text-only, hover surface
 *   - danger: destructive actions
 *
 */
const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 rounded-control text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-accent/60 disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        default:
          "bg-app-accent text-app-primary-foreground hover:bg-app-accent-hover",
        secondary:
          "bg-app-surface-muted text-app-foreground hover:bg-app-surface-raised",
        outline:
          "border border-app-border text-app-foreground hover:bg-app-surface-muted",
        ghost: "text-app-foreground hover:bg-app-surface-muted",
        danger:
          "bg-status-unexpected-bg text-status-unexpected hover:brightness-110",
      },
      size: {
        default: "h-11 px-5",
        sm: "h-9 px-4",
        icon: "h-9 w-9 p-0",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        ref={ref}
        data-variant={variant ?? "default"}
        className={cn(buttonVariants({ variant, size, className }))}
        {...props}
      />
    );
  },
);
Button.displayName = "Button";
