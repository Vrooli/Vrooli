import { mergeDeep } from "./objects";

/**
 * Checks if existingItems (an array or an object) is defined and non-empty.
 * If so, returns it with expected shape (determined by the shape of defaultItems); 
 * otherwise, returns defaultItems.
 *
 * @param existingItems - The items to check.
 * @param defaultItems - The items to return if existingItems is undefined or empty.
 * @return Merged existingItems if non-empty and defined, otherwise defaultItems.
 */
export const orDefault = <T extends Array<any> | { [key: string]: any } | null | undefined>(
    existingItems: T | null | undefined,
    defaultItems: T,
): T => {
    if (Array.isArray(existingItems)) {
        if (existingItems.length === 0) {
            return defaultItems;
        }
        // Ensure defaultItems is an array with at least one element for mapping
        const defaultArrayItem = Array.isArray(defaultItems) && defaultItems.length > 0 ? defaultItems[0] : {} as any;
        return existingItems.map(item => mergeDeep(item, defaultArrayItem)) as T;
    } else if (typeof existingItems === 'object' && existingItems !== null) {
        return Object.keys(existingItems).length > 0
            ? mergeDeep(existingItems, defaultItems)
            : defaultItems;
    }

    return defaultItems;
}
