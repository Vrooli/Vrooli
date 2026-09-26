import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { Loader2 } from "lucide-react";
import * as React from "react";

import { cn } from "../../lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 rounded-control text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus focus-visible:ring-offset-2 focus-visible:ring-offset-app-background disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        primary: "bg-app-primary text-app-primary-foreground hover:brightness-95",
        secondary: "bg-app-surface-muted text-app-foreground hover:brightness-95",
        ghost: "bg-transparent text-app-foreground hover:bg-app-surface-muted",
        outline: "border border-app-border text-app-foreground hover:bg-app-surface-muted",
        danger: "bg-app-danger text-white hover:brightness-95",
      },
      size: {
        sm: "h-9 px-3 min-h-touch",
        md: "h-11 px-5 min-h-touch",
        lg: "h-12 px-6 min-h-touch text-base",
        icon: "h-11 w-11 min-h-touch min-w-touch p-0",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "md",
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  loading?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { className, variant, size, asChild = false, loading = false, disabled, children, ...props },
  ref,
) {
  const Comp = asChild ? Slot : "button";
  // When asChild, Slot expects a single child element — skip the spinner
  // injection. asChild + loading is an unusual combo; callers that need
  // both should compose loading manually inside the slotted element.
  const content = asChild ? (
    children
  ) : (
    <>
      {loading ? <Loader2 aria-hidden className="h-4 w-4 animate-spin" /> : null}
      {children}
    </>
  );
  return (
    <Comp
      ref={ref}
      className={cn(buttonVariants({ variant, size, className }))}
      aria-busy={loading || undefined}
      disabled={disabled || loading}
      {...props}
    >
      {content}
    </Comp>
  );
});
