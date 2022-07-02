/**
 * Finds items which have been removed from the array.
 * @param original The original array
 * @param updated The updated array
 * @returns The items which have been removed from the array
 */
export const findRemovedItems = <T extends { id: string }>(original: T[], updated: T[]): T[] => {
    const removed: T[] = [];
    for (const originalItem of original) {
        const updatedItem = updated.find(item => item.id === originalItem.id);
        if (!updatedItem) removed.push(originalItem);
    }
    return removed;
}