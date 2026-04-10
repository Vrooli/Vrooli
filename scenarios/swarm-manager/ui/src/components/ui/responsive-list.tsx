import * as React from "react";
import { cn } from "../../lib/utils";

interface ResponsiveListProps extends React.HTMLAttributes<HTMLDivElement> {
  columns?: string;
}

export function ResponsiveList({
  columns = "md:grid-cols-2 lg:grid-cols-3",
  className,
  ...props
}: ResponsiveListProps) {
  return (
    <div
      className={cn(
        "flex flex-col divide-y divide-white/10 md:divide-none md:grid md:gap-4",
        columns,
        className
      )}
      {...props}
    />
  );
}

type ResponsiveListItemProps<E extends React.ElementType = "div"> = {
  as?: E;
  interactive?: boolean;
  className?: string;
} & Omit<React.ComponentPropsWithoutRef<E>, "as" | "className">;

export function ResponsiveListItem<E extends React.ElementType = "div">({
  as,
  interactive = false,
  className,
  ...props
}: ResponsiveListItemProps<E>) {
  const Component = as ?? "div";

  return (
    <Component
      className={cn(
        "py-3 md:py-0 md:rounded-xl md:border md:border-white/10 md:bg-slate-800/30 md:p-4",
        interactive &&
          "md:transition md:hover:border-cyan-500/50 md:hover:bg-slate-800/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-cyan-500/40",
        className
      )}
      {...props}
    />
  );
}
