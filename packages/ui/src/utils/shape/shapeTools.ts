
/**
 * Finds objects which have been created, and returns an array of the created objects, formatted for
 * the create mutation
 * @param original The original array
 * @param updated The updated array
 * @param formatForCreate The function to format an object for the create mutation
 * @returns An array of the created objects formatted for the create mutation,
 * or undefined if no objects have been created.
 */
export const findCreatedItems = <T extends { id?: string | null }, R>(
    original: T[],
    updated: T[],
    formatForCreate: (item: T) => R
): R[] | undefined => {
    const createdItems: R[] = [];
    for (const updatedItem of updated) {
        const oi = original.find(item => item.id === updatedItem.id);
        if (!oi) createdItems.push(formatForCreate(updatedItem));
    }
    return createdItems.length > 0 ? createdItems : undefined;
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
export const findUpdatedItems = <T extends { id?: string | null }, R>(
    original: T[],
    updated: T[],
    hasObjectChanged: (original: T & { id: string }, updated: T & { id: string }) => boolean,
    formatForUpdate: (original: T & { id: string }, updated: T & { id: string }) => R
): R[] | undefined => {
    const updatedItems: R[] = [];
    for (const updatedItem of updated) {
        if (!updatedItem.id) continue;
        const oi: T & { id: string } | undefined = original.find(item => item.id && item.id === updatedItem.id) as T & { id: string } | undefined;
        if (oi && hasObjectChanged(oi, updatedItem as T & { id: string })) updatedItems.push(formatForUpdate(oi, updatedItem as T & { id: string }));
    }
    return updatedItems.length > 0 ? updatedItems : undefined;
}

/**
 * Finds items which have been removed from the array.
 * @param original The original array
 * @param updated The updated array
 * @returns The IDs of items which have been removed from the array, 
 * or undefined if no items have been removed.
 */
export const findRemovedItems = <T extends { id?: string | null }>(original: T[], updated: T[]): string[] | undefined => {
    const removed: string[] = [];
    for (const originalItem of original) {
        if (!originalItem.id) continue;
        const updatedItem = updated.find(item => item.id === originalItem.id);
        if (!updatedItem) removed.push(originalItem.id);
    }
    return removed.length > 0 ? removed : undefined;
}

type ShapeListCreateField<N extends string, RC> = {
    [key in `${N}Create`]?: RC[] | undefined;
}
type ShapeListUpdateField<N extends string, RU> = {
    [key in `${N}Update`]?: RU[] | undefined;
}
type ShapeListDeleteField<N extends string> = {
    [key in `${N}Delete`]?: string[] | undefined;
}
type ShapeUpdateList<N extends string, RC, RU> =
    ShapeListCreateField<N, RC> &
    ShapeListUpdateField<N, RU> &
    ShapeListDeleteField<N>;

/**
 * Helper function for formatting a list of objects for a create mutation
 * @param items Objects to format
 * @param relationshipName The name of the relationship (e.g. 'translations')
 * @param formatForCreate A function for formatting a single object
 * @returns An array of formatted objects
 */
export const shapeCreateList = <N extends string, T, RC extends {}>(
    items: T[] | null | undefined,
    relationshipName: N,
    formatForCreate: (item: T) => RC | undefined
): ShapeListCreateField<N, RC> => {
    if (!items) return {};
    const formatted: RC[] = [];
    for (const item of items) {
        const currFormatted = formatForCreate(item);
        if (currFormatted) formatted.push(currFormatted);
    }
    if (formatted.length > 0) {
        return { [`${relationshipName}Create`]: formatted } as ShapeListCreateField<N, RC>;
    }
    return {};
}

/**
 * Helper function for formatting a list of objects for an update mutation
 * @param original The original array
 * @param updated The updated array
 * @param relationshipName The name of the relationship (e.g. 'translations')
 * @param hasObjectChanged A function which returns true if the object has changed
 * @param formatForCreate The function to format an object for the create mutation
 * @param formatForUpdate The function to format an object for the update mutation
 * @returns An array of formatted objects
 */
export const shapeUpdateList = <N extends string, T extends { id?: string | null }, RC extends {}, RU extends { id: string }>(
    original: (T & { id: string })[] | null | undefined,
    updated: T[] | null | undefined,
    relationshipName: N,
    hasObjectChanged: (original: T & { id: string }, updated: T & { id: string }) => boolean,
    formatForCreate: (item: T) => RC,
    formatForUpdate: (original: T & { id: string }, updated: T & { id: string }) => RU | undefined
): ShapeUpdateList<N, RC, RU> => {
    if (!updated) return {};
    // If no original items, treat all as created
    if (!original || !Array.isArray(original)) {
        return shapeCreateList(updated ?? [], relationshipName, formatForCreate) as ShapeUpdateList<N, RC, RU>;
    }
    if (Array.isArray(updated) && updated.length > 0) {
        return {
            [`${relationshipName}Create`]: findCreatedItems(original, updated, formatForCreate),
            [`${relationshipName}Update`]: findUpdatedItems(original, updated, hasObjectChanged, formatForUpdate),
            [`${relationshipName}Delete`]: findRemovedItems(original, updated),
        } as ShapeUpdateList<N, RC, RU>
    }
    return {};
}

/**
 * Helper function for formatting an object for an update mutation
 * @param original The original object
 * @param updated The updated object
 * @param formatForUpdate The function to format the updated object for the update mutation
 */
export const shapeUpdate = <T extends { id: string }, R extends {}>(
    original: T,
    updated: T | null | undefined,
    formatForUpdate: (original: T, updated: T) => R | undefined
): R | undefined => {
    if (!updated?.id) return undefined;
    let result = formatForUpdate(original, updated);
    // Remove every value from the result that is undefined
    if (result) result = Object.fromEntries(Object.entries(result).filter(([, value]) => value !== undefined)) as R;
    // Return result if it is not empty
    return result && Object.keys(result).length > 0 ? result : undefined;
}