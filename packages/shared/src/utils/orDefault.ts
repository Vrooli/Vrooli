/**
 * Returns the existing items if they are defined and non-empty; otherwise, returns the default items.
 */
export function orDefault<T>(
    existingItems: Array<T> | null | undefined,
    defaultItems: Array<T>
): Array<T> {
    return existingItems && existingItems.length
        ? existingItems
        : defaultItems;
}