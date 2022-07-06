/**
 * Wrapper type for helping convert objects in the shape of a query result, 
 * to a create/update input object.
 */
export type ShapeWrapper<T> = Partial<Omit<T, '__typename'>> & { __typename?: string };

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
    formatForUpdate: (original: T, updated: T) => R
): R[] | undefined => {
    const updatedItems: R[] = [];
    for (const updatedItem of updated) {
        const oi = original.find(item => item.id === updatedItem.id);
        if (oi && hasObjectChanged(oi, updatedItem)) updatedItems.push(formatForUpdate(oi, updatedItem));
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
export const findRemovedItems = <T extends { id: string }>(original: T[], updated: T[]): string[] | undefined => {
    const removed: string[] = [];
    for (const originalItem of original) {
        const updatedItem = updated.find(item => item.id === originalItem.id);
        if (!updatedItem) removed.push(originalItem.id);
    }
    return removed.length > 0 ? removed : undefined;
}

/**
 * Helper function for formatting a list of objects for a create mutation
 * @param items Objects to format
 * @param formatForCreate A function for formatting a single object
 * @returns An array of formatted objects
 */
export const shapeCreateList = <T, R>(
    items: T[] | null | undefined,
    formatForCreate: (item: T) => R | undefined
): R[] | undefined => {
    if (!items) return undefined;
    const formatted: R[] = [];
    for (const item of items) {
        const currFormatted = formatForCreate(item);
        if (currFormatted) formatted.push(currFormatted);
    }
    return formatted.length > 0 ? formatted : undefined;
}

type ShapeUpdateListCreate<N extends string, RC> = {
    [key in `${N}Create`]?: RC[] | undefined;
}
type ShapeUpdateListUpdate<N extends string, RU> = {
    [key in `${N}Update`]?: RU[] | undefined;
}
type ShapeUpdateListRemove<N extends string> = {
    [key in `${N}Remove`]?: string[] | undefined;
}
type ShapeUpdateList<N extends string, RC, RU> =
    ShapeUpdateListCreate<N, RC> &
    ShapeUpdateListUpdate<N, RU> &
    ShapeUpdateListRemove<N>;
/**
 * Helper function for formatting a list of objects for an update mutation
 * @param original The original array
 * @param updated The updated array
 * @param relationshipName The name of the relationship (e.g. 'translations')
 * @param shapeCreateList A function for formatting a list of create objects
 * @param hasObjectChanged A function which returns true if the object has changed
 * @param formatForUpdate The function to format an updated object for the update mutation
 * @returns An array of formatted objects
 */
export const shapeUpdateList = <N extends string, T extends { id?: string | null }, RC extends {}, RU extends { id: string }>(
    original: T[] | null | undefined,
    updated: T[] | null | undefined,
    relationshipName: N,
    shapeCreateList: (items: T[]) => RC[] | undefined,
    hasObjectChanged: (original: T, updated: T) => boolean,
    formatForUpdate: (original: T, updated: T) => RU | undefined
): ShapeUpdateList<N, RC, RU> => {
    if (!updated) return {};
    const filteredUpdated: (T & { id: string })[] = updated.filter(updated => Boolean(updated.id)) as (T & { id: string })[];
    if (!original || !Array.isArray(original)) {
        return { [`${relationshipName}Create`]: shapeCreateList(filteredUpdated) } as ShapeUpdateList<N, RC, RU>;
    }
    const filteredOriginal: (T & { id: string })[] = original.filter(original => Boolean(original.id)) as (T & { id: string })[];
    if (Array.isArray(updated) && updated.length > 0) {
        return {
            [`${relationshipName}Create`]: findCreatedItems(filteredOriginal, filteredUpdated, shapeCreateList),
            [`${relationshipName}Update`]: findUpdatedItems(filteredOriginal, filteredUpdated, hasObjectChanged, formatForUpdate),
            [`${relationshipName}Delete`]: findRemovedItems(filteredOriginal, filteredUpdated),
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
    const result = formatForUpdate(original, updated);
    // Only return result if it is not undefined, as has a value BESIDES the id field that is not undefined
    return (result && Object.entries(result).some(([key, value]) => key !== 'id' && value !== undefined)) ? result : undefined;
}