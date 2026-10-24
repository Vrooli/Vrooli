// Shared listing primitives — types both InventoryFilters and
// ScenarioFilters use. Lives in features/listing/ specifically because
// both list pages collapse to the same shape: search + filter slot +
// sort. Three files total: types.ts, useListState.ts, ListToolbar.tsx.
export type SortDir = "asc" | "desc";

export interface SortState<TKey extends string> {
  key: TKey;
  dir: SortDir;
}

export function flipDir(d: SortDir): SortDir {
  return d === "asc" ? "desc" : "asc";
}
