import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";

const codeVariants = cva("font-mono", {
  variants: {
    variant: {
      inline:
        "rounded-control bg-app-surface-muted px-1.5 py-0.5 text-[0.85em] text-app-foreground border border-app-border",
      block:
        "block w-full overflow-x-auto rounded-panel border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground",
    },
  },
  defaultVariants: { variant: "inline" },
});

export interface CodeProps
  extends React.HTMLAttributes<HTMLElement>,
    VariantProps<typeof codeVariants> {}

export function Code({ className, variant, ...props }: CodeProps) {
  if (variant === "block") {
    return (
      <pre
        data-testid={selectors.ui.code.root}
        className={cn(codeVariants({ variant }), className)}
        {...props}
      />
    );
  }
  return (
    <code
      data-testid={selectors.ui.code.root}
      className={cn(codeVariants({ variant }), className)}
      {...props}
    />
  );
}
