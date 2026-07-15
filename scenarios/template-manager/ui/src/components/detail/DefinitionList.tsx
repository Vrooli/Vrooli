import type { ReactNode } from "react";

export interface DefinitionItem {
  label: string;
  value: ReactNode;
  /** Force the item to span the full row on wide layouts (e.g. long paths). */
  full?: boolean;
}

export interface DefinitionListProps {
  items: DefinitionItem[];
  testId?: string;
}

/**
 * A responsive term/description grid for the key facts in an Overview section.
 * Values that are empty render an em dash so the layout never collapses.
 */
export function DefinitionList({ items, testId }: DefinitionListProps) {
  return (
    <dl data-testid={testId} className="grid gap-3 sm:grid-cols-2">
      {items.map((item) => (
        <div
          key={item.label}
          className={`min-w-0 rounded-panel border border-app-border px-3 py-2 ${item.full ? "sm:col-span-2" : ""}`}
        >
          <dt className="text-xs font-medium text-app-muted-foreground">{item.label}</dt>
          <dd className="mt-0.5 min-w-0 break-words text-sm font-semibold text-app-foreground">
            {isEmpty(item.value) ? "—" : item.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}

function isEmpty(value: ReactNode): boolean {
  return value === null || value === undefined || value === "";
}
