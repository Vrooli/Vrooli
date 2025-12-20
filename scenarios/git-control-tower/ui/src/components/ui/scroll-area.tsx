import { forwardRef } from "react";
import { cn } from "../../lib/utils";

export interface ScrollAreaProps extends React.HTMLAttributes<HTMLDivElement> {
  maxHeight?: string;
}

export const ScrollArea = forwardRef<HTMLDivElement, ScrollAreaProps>(
  ({ className, maxHeight = "100%", style, ...props }, ref) => {
    return (
      <div
        ref={ref}
        className={cn(
          "overflow-auto scrollbar-thin scrollbar-thumb-slate-700 scrollbar-track-transparent",
          className
        )}
        style={{ maxHeight, ...style }}
        {...props}
      />
    );
  }
);

ScrollArea.displayName = "ScrollArea";
