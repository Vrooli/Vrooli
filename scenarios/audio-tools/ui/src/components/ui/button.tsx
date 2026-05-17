import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 rounded-control text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-focus/60 disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        default: "bg-app-primary text-app-primary-foreground hover:brightness-95",
        outline: "border border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
        ghost: "bg-transparent text-app-foreground hover:bg-app-surface-muted",
        subtle: "bg-app-surface-muted text-app-foreground hover:bg-app-surface-muted/70",
        destructive: "bg-app-danger text-app-danger-foreground hover:brightness-95",
        link: "text-app-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-10 px-4",
        sm: "h-8 px-3 text-xs",
        lg: "h-12 px-6 text-base",
        icon: "h-9 w-9 p-0",
      },
    },
    defaultVariants: { variant: "default", size: "default" },
  },
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

// `buttonVariants` is the cva instance the <Button> component uses to
// compute its class names. It is intentionally exported from the same
// module so consumers (Link-styled-as-button, asChild slots, etc.) can
// reuse the exact same variant table without indirection. The Fast
// Refresh limitation is acceptable: edits to this file remount Button
// users, which is fine for a primitive design-token boundary.
// eslint-disable-next-line react-refresh/only-export-components
export { buttonVariants };
