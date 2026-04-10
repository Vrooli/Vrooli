// DOC: docs/internal/COHERENCE-NOTES.md#styling-patterns
import React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center rounded-full text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--btn-focus-ring)] disabled:pointer-events-none disabled:opacity-60",
  {
    variants: {
      variant: {
        default: "bg-[var(--btn-default-bg)] text-[var(--btn-default-text)] hover:bg-[var(--btn-default-hover-bg)]",
        outline: "border border-[var(--btn-outline-border)] text-[var(--btn-outline-text)] hover:bg-[var(--btn-outline-hover-bg)]"
      },
      size: {
        default: "h-11 px-5",
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

export function Button({ className, variant, size, asChild = false, ...props }: ButtonProps) {
  const Comp = asChild ? Slot : "button";

  return <Comp className={cn(buttonVariants({ variant, size, className }))} {...props} />;
}
