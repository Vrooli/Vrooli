import type { CatalogEntry } from "./api";

export type CatalogChange = { catalog: string; entry: CatalogEntry };
export type CatalogSaveBatchResult =
  | { saved: CatalogChange[]; failed: null }
  | { saved: CatalogChange[]; failed: { change: CatalogChange; error: unknown } };

/** Save in order and retain the exact successful prefix when a later write fails. */
export async function saveCatalogBatch(
  changes: readonly CatalogChange[],
  save: (catalog: string, entry: CatalogEntry) => Promise<unknown>,
): Promise<CatalogSaveBatchResult> {
  const saved: CatalogChange[] = [];
  for (const change of changes) {
    try {
      await save(change.catalog, change.entry);
      saved.push(change);
    } catch (error) {
      return { saved, failed: { change, error } };
    }
  }
  return { saved, failed: null };
}

/** Remove only drafts whose current value still matches the successfully saved snapshot. */
export function removeSavedDrafts<T extends CatalogEntry>(
  drafts: Record<string, T>,
  saved: readonly CatalogChange[],
): Record<string, T> {
  const savedByKey = new Map(saved.map(({ catalog, entry }) => [`${catalog}:${entry.id}`, JSON.stringify(entry)]));
  return Object.fromEntries(Object.entries(drafts).filter(([key, entry]) => savedByKey.get(key) !== JSON.stringify(entry)));
}

export function matchesSavedDraft(
  current: { catalog: string; entry: CatalogEntry } | null,
  saved: readonly CatalogChange[],
): boolean {
  if (!current) return false;
  const snapshot = saved.find(({ catalog, entry }) => catalog === current.catalog && entry.id === current.entry.id);
  return snapshot !== undefined && JSON.stringify(snapshot.entry) === JSON.stringify(current.entry);
}
