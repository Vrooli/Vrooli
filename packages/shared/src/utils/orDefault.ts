/**
 * Checks if existingItems (an array or an object) is defined and non-empty.
 * If so, returns it; otherwise, returns defaultItems.
 *
 * @param existingItems - The items to check.
 * @param defaultItems - The items to return if existingItems is undefined or empty.
 * @return existingItems if non-empty and defined, otherwise defaultItems.
 */
export function orDefault<T extends Array<any> | { [key: string]: any } | null | undefined>(
    existingItems: T | null | undefined,
    defaultItems: T,
): T {
    if (Array.isArray(existingItems)) {
        return (existingItems && existingItems.length) ? existingItems : defaultItems;
    } else if (typeof existingItems === "object" && existingItems !== null) {
        return (existingItems && Object.keys(existingItems).length > 0) ? existingItems : defaultItems;
    }

    return defaultItems;
}
