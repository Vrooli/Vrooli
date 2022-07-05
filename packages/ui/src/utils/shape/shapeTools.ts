/**
 * Finds items which have been removed from the array.
 * @param original The original array
 * @param updated The updated array
 * @returns The IDs of items which have been removed from the array, 
 * or undefined if no items have been removed.
 */
export const findRemovedItems = <T extends { id: string }>(original: T[], updated: T[]): string[] | undefined => {
    const removed: string[] = [];
    for (const originalItem of original) {
        const updatedItem = updated.find(item => item.id === originalItem.id);
        if (!updatedItem) removed.push(originalItem.id);
    }
    return removed.length > 0 ? removed : undefined;
}

/**
 * Find objects which have been updated, and returns an array of the updated objects, formatted for 
 * the update mutation
 * @param original The original array
 * @param updated The updated array
 * @param hasObjectChanged A function which returns true if the object has changed
 * @param formatForUpdate The function to format the updated object for the update mutation
 * @returns An array of the updated objects formatted for the update mutation, 
 * or undefined if no objects have been updated.
 */
export const findUpdatedItems = <T extends { id: string }, R>(
    original: T[],
    updated: T[],
    hasObjectChanged: (original: T, updated: T) => boolean,
    formatForUpdate: (original: T, updated: T) => any
): R[] | undefined => {
    const updatedItems: R[] = [];
    for (const updatedItem of updated) {
        const oi = original.find(item => item.id === updatedItem.id);
        if (oi && hasObjectChanged(oi, updatedItem)) updatedItems.push(formatForUpdate(oi, updatedItem));
    }
    return updatedItems.length > 0 ? updatedItems : undefined;
}

/**
 * Finds objects which have been created, and returns an array of the created objects, formatted for
 * the create mutation
 * @param original The original array
 * @param updated The updated array
 * @param formatForCreate The function to format a list of created objects for the create mutation
 * @returns An array of the created objects formatted for the create mutation,
 * or undefined if no objects have been created.
 */
export const findCreatedItems = <T extends { id: string }, R>(
    original: T[],
    updated: T[],
    formatForCreate: (items: T[]) => R[] | null | undefined
): R[] | undefined => {
    const createdItems: T[] = [];
    for (const updatedItem of updated) {
        const oi = original.find(item => item.id === updatedItem.id);
        if (!oi) createdItems.push(updatedItem);
    }
    return createdItems.length > 0 ? formatForCreate(createdItems) as R[] : undefined;
}