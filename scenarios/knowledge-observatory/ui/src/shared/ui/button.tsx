// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import { Slot } from "@radix-ui/react-slot";
import { Button as RclButton } from "@vrooli/react-component-library/Button/2.2.14";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-full font-medium transition-colors ko-focus-ring disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        primary: "ko-button-primary",
        secondary: "ko-button-secondary",
        outline: "ko-button-outline",
        danger: "ko-button-danger",
      },
      size: {
        default: "h-11 px-5 text-sm",
        sm: "h-9 px-4 text-sm",
        compact: "ko-button-compact",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "default",
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

export function Button({ className, variant, size, asChild = false, ...props }: ButtonProps) {
  const mergedClassName = cn(buttonVariants({ variant, size, className }));
  if (asChild) return <Slot className={mergedClassName} {...props} />;
  const rclVariant = variant === "danger" ? "danger" : variant === "secondary" || variant === "outline" ? "secondary" : "primary";
  const rclSize = size === "sm" || size === "compact" ? "sm" : "md";
  return <RclButton variant={rclVariant} size={rclSize} className={mergedClassName} {...props} />;
}
