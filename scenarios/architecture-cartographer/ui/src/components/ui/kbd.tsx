import * as React from "react";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";

export type KbdProps = React.HTMLAttributes<HTMLElement>;

export function Kbd({ className, ...props }: KbdProps) {
  return (
    <kbd
      data-testid={selectors.ui.kbd.root}
      className={cn(
        "inline-flex h-5 min-w-[1.25rem] items-center justify-center rounded-control border border-app-border bg-app-surface px-1 font-mono text-[0.75rem] font-medium text-app-foreground shadow-sm",
        className,
      )}
      {...props}
    />
  );
}
