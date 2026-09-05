import * as React from "react";

import { cn } from "../../lib/utils";

export const Skeleton = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(
  function Skeleton({ className, ...props }, ref) {
    return (
      <div
        ref={ref}
        aria-hidden
        className={cn("animate-pulse rounded-control bg-app-surface-muted", className)}
        {...props}
      />
    );
  },
);
