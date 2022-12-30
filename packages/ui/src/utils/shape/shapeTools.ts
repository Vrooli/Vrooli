import { SnackSeverity } from "components";
import { fi } from "date-fns/locale";
import { PubSub } from "utils/pubsub";

/**
 * Finds objects which have been created, and returns an array of the created objects, formatted for
 * the create mutation
 * @param original The original array
 * @param updated The updated array
 * @param formatForCreate The function to format an object for the create mutation
 * @returns An array of the created objects formatted for the create mutation,
 * or undefined if no objects have been created.
 */
const findCreatedItems = <
    IDField extends string,
    Input extends { [key in IDField]?: string | null },
    Output
>(
    original: Input[],
    updated: Input[],
    formatForCreate: (item: Input) => Output,
    idField: IDField = 'id' as IDField,
): Output[] | undefined => {
    const createdItems: Output[] = [];
    for (const updatedItem of updated) {
        const oi = original.find(item => item[idField] === updatedItem[idField]);
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
const findUpdatedItems = <
    IDField extends string,
    Input extends { [key in IDField]?: string | null },
    Output
>(
    original: Input[],
    updated: Input[],
    hasObjectChanged: (original: Input & { [key in IDField]: string }, updated: Input & { [key in IDField]: string }) => boolean,
    formatForUpdate: (original: Input & { [key in IDField]: string }, updated: Input & { [key in IDField]: string }) => Output,
    idField: IDField = 'id' as IDField,
): Output[] | undefined => {
    const updatedItems: Output[] = [];
    for (const updatedItem of updated) {
        if (!updatedItem || !updatedItem[idField]) continue;
        const oi: Input & { [key in IDField]: string } | undefined = original.find(item => item && item[idField] && item[idField] === updatedItem[idField]) as Input & { [key in IDField]: string } | undefined;
        if (oi && hasObjectChanged(oi, updatedItem as Input & { [key in IDField]: string })) updatedItems.push(formatForUpdate(oi, updatedItem as Input & { [key in IDField]: string }));
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
const findDeletedItems = <
    IDField extends string,
    Input extends { [key in IDField]?: string | null }
>(
    original: Input[],
    updated: Input[],
    idField: IDField = 'id' as IDField,
): string[] | undefined => {
    const removed: string[] = [];
    for (const originalItem of original) {
        if (!originalItem || !originalItem[idField]) continue;
        const updatedItem = updated.find(item => item[idField] === originalItem[idField]);
        if (!updatedItem) removed.push(originalItem[idField as string]);
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
const findConnectedItems = <
    IDField extends string,
    Input extends { [key in IDField]?: string | null }
>(
    original: Input[],
    updated: Input[],
    idField: IDField = 'id' as IDField,
): string[] | undefined => {
    const connected: string[] = [];
    for (const updatedItem of updated) {
        if (!updatedItem || !updatedItem[idField]) continue;
        const originalItem = original.find(item => item[idField] === updatedItem[idField]);
        if (!originalItem) connected.push(updatedItem[idField as string]);
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
const findDisconnectedItems = <
    IDField extends string,
    Input extends { [key in IDField]?: string | null }
>(
    original: Input[],
    updated: Input[],
    idField: IDField = 'id' as IDField,
): string[] | undefined => {
    const disconnected: string[] = [];
    for (const originalItem of original) {
        if (!original || !originalItem[idField]) continue;
        const updatedItem = updated.find(item => item[idField] === originalItem[idField]);
        if (!updatedItem) disconnected.push(originalItem[idField as string]);
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
 * @param relationshipField The name of the relationship (e.g. 'translations')
 * @param formatForCreate A function for formatting a single object
 * @returns An array of formatted objects
 */
export const shapeCreateList = <
    RelField extends string,
    Input,
    Output extends {}
>(
    items: { [key in RelField]?: Input[] | null | undefined },
    relationshipField: RelField,
    formatForCreate: (item: Input) => Output | undefined
): ShapeListCreateField<RelField, Output> => {
    const creates: Input[] | null | undefined = items[relationshipField];
    if (!creates) return {};
    const formatted: Output[] = [];
    for (const item of creates) {
        const currFormatted = formatForCreate(item);
        if (currFormatted) formatted.push(currFormatted);
    }
    if (formatted.length > 0) {
        return { [`${relationshipField}Create`]: formatted } as ShapeListCreateField<RelField, Output>;
    }
    return {};
}

/**
 * Helper function for formatting a list of objects for a connect mutation
 * @param items Objects to format
 * @param relationshipField The name of the relationship (e.g. 'translations')
 * @returns An array of formatted objects
 */
export const shapeConnectList = <
    IDField extends string,
    RelField extends string,
    Input extends { [key in IDField]: string }
>(
    items: { [key in RelField]?: Input[] | null | undefined },
    relationshipField: RelField,
    idField: IDField = 'id' as IDField,
): ShapeListConnectField<RelField> => {
    const connects: Input[] | null | undefined = items[relationshipField];
    if (!connects) return {};
    const formatted: string[] = [];
    for (const item of connects) {
        if (item[idField]) formatted.push(item[idField]);
    }
    if (formatted.length > 0) {
        return { [`${relationshipField}Connect`]: formatted } as ShapeListConnectField<RelField>;
    }
    return {};
}

/**
 * Helper function for formatting a list of objects for an update mutation
 * @param original The original array
 * @param updated The updated array
 * @param relationshipField The name of the relationship (e.g. 'translations')
 * @param hasObjectChanged A function which returns true if the object has changed
 * @param formatForCreate The function to format an object for the create mutation
 * @param formatForUpdate The function to format an object for the update mutation
 * @param idField The name of the ID field. Defaults to 'id'.
 * @param treatLikeConnects If true, use connect instead of create
 * @param treatLikeDisconnects If true, use disconnect instead of delete
 * @returns An array of formatted objects
 */
export const shapeUpdateList = <
    RelField extends string,
    IDField extends string,
    Input extends { [key in IDField]?: string | null },
    OutputCreate extends {},
    OutputUpdate extends { [key in IDField]: string }
>(
    original: { [key in RelField]?: (Input & { [key in IDField]: string })[] | null | undefined },
    updated: { [key in RelField]?: Input[] | null | undefined },
    relationshipField: RelField,
    hasObjectChanged: (original: Input & { [key in IDField]: string }, updated: Input & { [key in IDField]: string }) => boolean,
    formatForCreate: (item: Input) => OutputCreate,
    formatForUpdate: (original: Input & { [key in IDField]: string }, updated: Input & { [key in IDField]: string }) => OutputUpdate | undefined,
    idField: IDField = 'id' as IDField,
    treatLikeConnects: boolean = false,
    treatLikeDisconnects: boolean = false,
): ShapeUpdateList<RelField, OutputCreate, OutputUpdate> => {
    const o = original[relationshipField];
    const u = updated[relationshipField];
    if (!u) return {};
    // If no original items, treat all as created/connected
    if (!o || !Array.isArray(o)) {
        if (treatLikeConnects) {
            // If treating like connects, there must be an ID in every updated item
            if (!u.every(item => item && item[idField as string])) {
                PubSub.get().publishSnack({ messageKey: 'ErrorUnknown', severity: SnackSeverity.Error });
                return {};
            }
            return shapeConnectList(updated as { [key in RelField]: (Input & { [key in IDField]: string })[] }, relationshipField, idField) as ShapeUpdateList<RelField, OutputCreate, OutputUpdate>;

        } else {
            return shapeCreateList(updated as { [key in RelField]: Input[] }, relationshipField, formatForCreate) as ShapeUpdateList<RelField, OutputCreate, OutputUpdate>;
        }
    }
    if (Array.isArray(u)) {
        return {
            [`${relationshipField}Update`]: findUpdatedItems(o, u, hasObjectChanged, formatForUpdate, idField),
            [`${relationshipField}Create`]: !treatLikeConnects ? findCreatedItems(o, u, formatForCreate, idField) : undefined,
            [`${relationshipField}Connect`]: treatLikeConnects ? findConnectedItems(o, u, idField) : undefined,
            [`${relationshipField}Delete`]: !treatLikeDisconnects ? findDeletedItems(o, u, idField) : undefined,
            [`${relationshipField}Disconnect`]: treatLikeDisconnects ? findDisconnectedItems(o, u, idField) : undefined,
        } as ShapeUpdateList<RelField, OutputCreate, OutputUpdate>
    }
    return {};
}

/**
 * Helper function for formatting an object for an update mutation
 * @param updated The updated object
 * @param shape Shapes the updated object for the update mutation
 */
export const shapeUpdate = <
    Input extends {},
    Output extends {}
>(
    updated: Input | null | undefined,
    shape: Output | (() => Output),
): Output | undefined => {
    if (!updated) return undefined;
    let result = typeof shape === 'function' ? (shape as () => Output)() : shape;
    // Remove every value from the result that is undefined
    if (result) result = Object.fromEntries(Object.entries(result).filter(([, value]) => value !== undefined)) as Output;
    // Return result if it is not empty
    return result && Object.keys(result).length > 0 ? result : undefined;
}

/**
 * Helper function for setting a list of primitive fields of an update 
 * shape. If updated is different from original, return updated,
 * otherwise return undefined
 * 
 * NOTE: Due to TypeScript limitations, return type assumes that every field 
 * will be defined, even if it's undefined
 * @param original The original object
 * @param updated The updated object
 * @param primary The primary key of the object, which is always returned
 * @param fields The fields to check, which are returned if they are different
 */
export const shapeUpdatePrims = <T, K extends keyof T, PK extends keyof T>(
    original: T | null | undefined,
    updated: T | null | undefined,
    primary: PK | null
    ...fields: K[]
): ({ [F in K]: Exclude<T[F], null | undefined> } & { [F in PK]: T[F] }) => {
    if (!updated || !original) return fields.reduce((acc, field) => ({ ...acc, [field]: undefined }), {}) as any;
    const changedFields = fields.reduce((acc, field) => ({ ...acc, [field]: updated[field] !== original[field] ? updated[field] : undefined }), {});
    if (!primary) return changedFields as any;
    return { ...changedFields, [primary]: original[primary] } as any;
}

/**
 * Helper function for setting a list of primitive fields of a create 
 * shape. Essentially, adds every field that's defined, and converts nulls to 
 * undefined
 * 
 * NOTE: Due to TypeScript limitations, return type assumes that every field 
 * will be defined, even if it's undefined
 */
export const shapeCreatePrims = <T, K extends keyof T>(
    object: T,
    ...fields: K[]
): { [F in K]: Exclude<T[F], null | undefined> } => {
    return fields.reduce((acc, field) => ({ ...acc, [field]: object[field] !== null ? object[field] : undefined }), {}) as any;
}