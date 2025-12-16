import { cn } from "../../lib/utils";

export interface ScrollAreaProps extends React.HTMLAttributes<HTMLDivElement> {
  maxHeight?: string;
}

export function ScrollArea({ className, maxHeight = "100%", style, ...props }: ScrollAreaProps) {
  return (
    <div
      className={cn(
        "overflow-auto scrollbar-thin scrollbar-thumb-slate-700 scrollbar-track-transparent",
        className
      )}
      style={{ maxHeight, ...style }}
      {...props}
    />
  );
}
