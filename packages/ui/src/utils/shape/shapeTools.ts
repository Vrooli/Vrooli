import { Pubs } from "utils/consts";

/**
 * Finds objects which have been created, and returns an array of the created objects, formatted for
 * the create mutation
 * @param original The original array
 * @param updated The updated array
 * @param formatForCreate The function to format an object for the create mutation
 * @returns An array of the created objects formatted for the create mutation,
 * or undefined if no objects have been created.
 */
export const findCreatedItems = <
    Input extends { id?: string | null },
    Output
>(
    original: Input[],
    updated: Input[],
    formatForCreate: (item: Input) => Output
): Output[] | undefined => {
    const createdItems: Output[] = [];
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
export const findUpdatedItems = <
    Input extends { id?: string | null },
    Output
>(
    original: Input[],
    updated: Input[],
    hasObjectChanged: (original: Input & { id: string }, updated: Input & { id: string }) => boolean,
    formatForUpdate: (original: Input & { id: string }, updated: Input & { id: string }) => Output
): Output[] | undefined => {
    const updatedItems: Output[] = [];
    for (const updatedItem of updated) {
        if (!updatedItem.id) continue;
        const oi: Input & { id: string } | undefined = original.find(item => item.id && item.id === updatedItem.id) as Input & { id: string } | undefined;
        if (oi && hasObjectChanged(oi, updatedItem as Input & { id: string })) updatedItems.push(formatForUpdate(oi, updatedItem as Input & { id: string }));
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
export const findDeletedItems = <
    Input extends { id?: string | null }
>(
    original: Input[],
    updated: Input[]
): string[] | undefined => {
    const removed: string[] = [];
    for (const originalItem of original) {
        if (!originalItem.id) continue;
        const updatedItem = updated.find(item => item.id === originalItem.id);
        if (!updatedItem) removed.push(originalItem.id);
    }
    return removed.length > 0 ? removed : undefined;
}

/**
 * Finds items which have been added to the array, and connects them to the parent.
 * @param original The original array
 * @param updated The updated array
 * @param the IDs of items which have been added to the array,
 * or undefined if no items have been added.
 */
export const findConnectedItems = <
    Input extends { id?: string | null }
>(
    original: Input[],
    updated: Input[]
): string[] | undefined => {
    const connected: string[] = [];
    for (const updatedItem of updated) {
        if (!updatedItem.id) continue;
        const originalItem = original.find(item => item.id === updatedItem.id);
        if (!originalItem) connected.push(updatedItem.id);
    }
    return connected.length > 0 ? connected : undefined;
}

/**
 * Finds items which have been disconnected from the parent.
 * @param original The original array
 * @param updated The updated array
 * @returns The IDs of items which have been disconnected from the parent,
 * or undefined if no items have been disconnected.
 */
export const findDisconnectedItems = <
    Input extends { id?: string | null }
>(
    original: Input[],
    updated: Input[]
): string[] | undefined => {
    const disconnected: string[] = [];
    for (const originalItem of original) {
        if (!originalItem.id) continue;
        const updatedItem = updated.find(item => item.id === originalItem.id);
        if (!updatedItem) disconnected.push(originalItem.id);
    }
    return disconnected.length > 0 ? disconnected : undefined;
}

type ShapeListCreateField<RelField extends string, Output> = {
    [key in `${RelField}Create`]?: Output[] | undefined;
}
type ShapeListUpdateField<RelField extends string, Output> = {
    [key in `${RelField}Update`]?: Output[] | undefined;
}
type ShapeListDeleteField<RelField extends string> = {
    [key in `${RelField}Delete`]?: string[] | undefined;
}
type ShapeListConnectField<RelField extends string> = {
    [key in `${RelField}Connect`]?: string[] | undefined;
}
type ShapeListDisconnectField<RelField extends string> = {
    [key in `${RelField}Disconnect`]?: string[] | undefined;
}
type ShapeUpdateList<RelField extends string, OutputCreate, OutputUpdate> =
    ShapeListCreateField<RelField, OutputCreate> &
    ShapeListUpdateField<RelField, OutputUpdate> &
    ShapeListDeleteField<RelField> &
    ShapeListConnectField<RelField> &
    ShapeListDisconnectField<RelField>;

/**
 * Helper function for formatting a list of objects for a create mutation
 * @param items Objects to format
 * @param relationshipName The name of the relationship (e.g. 'translations')
 * @param formatForCreate A function for formatting a single object
 * @returns An array of formatted objects
 */
export const shapeCreateList = <
    RelField extends string,
    Input,
    Output extends {}
>(
    items: { [key in RelField]?: Input[] | null | undefined },
    relationshipName: RelField,
    formatForCreate: (item: Input) => Output | undefined
): ShapeListCreateField<RelField, Output> => {
    console.log('shapeCreateList', relationshipName, items);
    const creates: Input[] | null | undefined = items[relationshipName];
    if (!creates) return {};
    const formatted: Output[] = [];
    for (const item of creates) {
        const currFormatted = formatForCreate(item);
        if (currFormatted) formatted.push(currFormatted);
    }
    if (formatted.length > 0) {
        return { [`${relationshipName}Create`]: formatted } as ShapeListCreateField<RelField, Output>;
    }
    return {};
}

/**
 * Helper function for formatting a list of objects for a connect mutation
 * @param items Objects to format
 * @param relationshipName The name of the relationship (e.g. 'translations')
 * @returns An array of formatted objects
 */
export const shapeConnectList = <
    RelField extends string,
    Input extends { id: string }
>(
    items: { [key in RelField]?: Input[] | null | undefined },
    relationshipName: RelField
): ShapeListConnectField<RelField> => {
    const connects: Input[] | null | undefined = items[relationshipName];
    if (!connects) return {};
    const formatted: string[] = [];
    for (const item of connects) {
        if (item.id) formatted.push(item.id);
    }
    if (formatted.length > 0) {
        return { [`${relationshipName}Connect`]: formatted } as ShapeListConnectField<RelField>;
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
 * @param treatLikeConnects If true, use connect instead of create
 * @parma treatLikeDisconnects If true, use disconnect instead of delete
 * @returns An array of formatted objects
 */
export const shapeUpdateList = <
    RelField extends string,
    Input extends { id?: string | null },
    OutputCreate extends {},
    OutputUpdate extends { id: string }
>(
    original: { [key in RelField]?: (Input & { id: string })[] | null | undefined },
    updated: { [key in RelField]?: Input[] | null | undefined },
    relationshipName: RelField,
    hasObjectChanged: (original: Input & { id: string }, updated: Input & { id: string }) => boolean,
    formatForCreate: (item: Input) => OutputCreate,
    formatForUpdate: (original: Input & { id: string }, updated: Input & { id: string }) => OutputUpdate | undefined,
    treatLikeConnects: boolean = false,
    treatLikeDisconnects: boolean = false,
): ShapeUpdateList<RelField, OutputCreate, OutputUpdate> => {
    console.log('shapeupdatelist', original, updated, relationshipName)
    const o = original[relationshipName];
    const u = updated[relationshipName];
    if (!u) return {};
    // If no original items, treat all as created/connected
    if (!o || !Array.isArray(o)) {
        if (treatLikeConnects) {
            // If treating like connects, there must be an ID in every updated item
            if (!u.every(item => item.id)) {
                PubSub.publish(Pubs.Snack, { message: 'Invalid update: missing ID in update items', severity: 'error' });
                return {};
            }
            return shapeConnectList(updated as { [key in RelField]: (Input & { id: string })[] }, relationshipName) as ShapeUpdateList<RelField, OutputCreate, OutputUpdate>;

        } else {
            return shapeCreateList(updated as { [key in RelField]: Input[] }, relationshipName, formatForCreate) as ShapeUpdateList<RelField, OutputCreate, OutputUpdate>;
        }
    }
    if (Array.isArray(u) && u.length > 0) {
        return {
            [`${relationshipName}Update`]: findUpdatedItems(o, u, hasObjectChanged, formatForUpdate),
            [`${relationshipName}Create`]: !treatLikeConnects ? findCreatedItems(o, u, formatForCreate) : undefined,
            [`${relationshipName}Connect`]: treatLikeConnects ? findConnectedItems(o, u) : undefined,
            [`${relationshipName}Delete`]: !treatLikeDisconnects ? findDeletedItems(o, u) : undefined,
            [`${relationshipName}Disconnect`]: treatLikeDisconnects ? findDisconnectedItems(o, u) : undefined,
        } as ShapeUpdateList<RelField, OutputCreate, OutputUpdate>
    }
    return {};
}

/**
 * Helper function for formatting an object for an update mutation
 * @param original The original object
 * @param updated The updated object
 * @param formatForUpdate The function to format the updated object for the update mutation
 */
export const shapeUpdate = <
    Input extends { id: string },
    Output extends {}
>(
    original: Input,
    updated: Input | null | undefined,
    formatForUpdate: (original: Input, updated: Input) => Output | undefined
): Output | undefined => {
    if (!updated?.id) return undefined;
    let result = formatForUpdate(original, updated);
    // Remove every value from the result that is undefined
    if (result) result = Object.fromEntries(Object.entries(result).filter(([, value]) => value !== undefined)) as Output;
    // Return result if it is not empty
    return result && Object.keys(result).length > 0 ? result : undefined;
}