/**
 * BacklogItemsContext — shared kind/name → BacklogItem lookup.
 *
 * Built once per items array via useMemo so consumers (e.g. DependencyIndicator
 * inside every BacklogCard) don't each rebuild their own Map. Eliminates the
 * O(N²) work that used to fan out across the backlog list and command-post feed
 * every render.
 */

import { useMemo, type ReactNode } from "react";
import { BacklogItemsContext, type ItemsByKey } from "./backlog-items-lookup";
import type { BacklogItem } from "../../types";

interface BacklogItemsProviderProps {
  items: readonly BacklogItem[];
  children: ReactNode;
}

export function BacklogItemsProvider({ items, children }: BacklogItemsProviderProps) {
  const itemsByKey = useMemo<ItemsByKey>(
    () => new Map(items.map((item) => [`${item.kind}/${item.name}`, item])),
    [items],
  );
  return <BacklogItemsContext.Provider value={itemsByKey}>{children}</BacklogItemsContext.Provider>;
}
