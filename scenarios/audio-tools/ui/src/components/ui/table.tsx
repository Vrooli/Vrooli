import * as React from "react";
import { cn } from "../../lib/utils";

export function Table({ className, ...props }: React.HTMLAttributes<HTMLTableElement>) {
  return (
    <div className="w-full overflow-x-auto">
      <table className={cn("w-full table-fixed border-collapse text-sm", className)} {...props} />
    </div>
  );
}
export function THead({ className, ...props }: React.HTMLAttributes<HTMLTableSectionElement>) {
  return (
    <thead
      className={cn(
        "bg-app-surface-muted text-xs uppercase tracking-wide text-app-muted-foreground",
        className,
      )}
      {...props}
    />
  );
}
export function TBody({ className, ...props }: React.HTMLAttributes<HTMLTableSectionElement>) {
  return <tbody className={cn("divide-y divide-app-border", className)} {...props} />;
}
export function TR({ className, ...props }: React.HTMLAttributes<HTMLTableRowElement>) {
  return <tr className={cn("hover:bg-app-surface-muted/50", className)} {...props} />;
}
export function TH({ className, ...props }: React.ThHTMLAttributes<HTMLTableCellElement>) {
  return (
    <th
      scope="col"
      className={cn("px-3 py-2 text-left font-medium", className)}
      {...props}
    />
  );
}
export function TD({ className, ...props }: React.TdHTMLAttributes<HTMLTableCellElement>) {
  return <td className={cn("px-3 py-2 align-middle text-app-foreground", className)} {...props} />;
}
