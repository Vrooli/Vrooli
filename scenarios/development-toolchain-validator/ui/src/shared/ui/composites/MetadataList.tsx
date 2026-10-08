import type { ReactNode } from "react";
import { cn } from "../../lib/utils";

export interface MetadataItem {
  label: ReactNode;
  value: ReactNode;
  /** Whether to render the value in a mono font (paths, hashes). */
  mono?: boolean;
}

export interface MetadataListProps {
  items: readonly MetadataItem[];
  className?: string;
}

/**
 * Responsive `<dl>` of label/value pairs. Desktop: two-column grid.
 * Mobile: stacks rows. Mono values use the design-tokens mono family.
 */
export function MetadataList({ items, className }: MetadataListProps) {
  return (
    <dl
      className={cn(
        "grid gap-x-3 gap-y-1 text-xs sm:grid-cols-[max-content_1fr]",
        className,
      )}
    >
      {items.map((item, idx) => (
        <div key={idx} className="contents">
          <dt className="text-app-muted-foreground">{item.label}</dt>
          <dd className={cn("text-app-foreground", item.mono && "font-mono")}>{item.value}</dd>
        </div>
      ))}
    </dl>
  );
}
