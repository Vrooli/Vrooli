import type { ReactNode } from "react";

import { cn } from "../lib/utils";

/** Labelled facts about one record: a muted label beside its value, one row each. */
export function Facts({ items, className }: { items: readonly { label: string; value: ReactNode; mono?: boolean }[]; className?: string }) {
  return (
    <dl className={cn("grid grid-cols-[minmax(6rem,9rem)_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm", className)}>
      {items.map((item) => (
        <div key={item.label} className="col-span-2 grid grid-cols-subgrid">
          <dt className="text-app-muted-foreground">{item.label}</dt>
          <dd className={cn("min-w-0 [overflow-wrap:anywhere]", item.mono && "font-mono text-xs")}>{item.value}</dd>
        </div>
      ))}
    </dl>
  );
}
