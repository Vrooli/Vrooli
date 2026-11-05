import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const dotVariants = cva("inline-block h-2 w-2 rounded-full", {
  variants: {
    tone: {
      neutral: "bg-app-muted-foreground",
      success: "bg-app-success",
      warning: "bg-app-warning",
      danger: "bg-app-danger",
      info: "bg-app-info",
      primary: "bg-app-primary",
    },
    pulse: { true: "animate-pulse", false: "" },
  },
  defaultVariants: { tone: "neutral", pulse: false },
});

export interface StatusDotProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof dotVariants> {
  /** Accessible label paired with the dot so the signal is not color-only. */
  label: string;
}

export function StatusDot({ tone, pulse, label, className, ...props }: StatusDotProps) {
  return (
    <span className={cn("inline-flex items-center gap-2 text-xs text-app-foreground", className)} {...props}>
      <span className={cn(dotVariants({ tone, pulse }))} aria-hidden="true" />
      <span>{label}</span>
    </span>
  );
}
