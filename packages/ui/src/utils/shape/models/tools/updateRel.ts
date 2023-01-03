import { ShapeModel } from 'types';
import { createRel } from './createRel';
import { hasObjectChanged } from 'utils/shape/general';

/**
 * Finds items which have been added to the array, and connects them to the parent.
 * @param original The original array
 * @param updated The updated array
 * @param idField The name of the id field in the relationship (defaults to "id")
 * @returns IDs of items which have been added to the array,
 * or undefined if no items have been added.
 */
const findConnectedItems = <
    Input extends { [x in IDField]: string }[],
    IDField extends string,
>(
    original: Input[],
    updated: Input[],
    idField: IDField = 'id' as IDField,
): string | string[] | undefined => {
    const connected: string[] = [];
    for (const updatedItem of (updated as any)) {
        if (!updatedItem || !updatedItem[idField]) continue;
        const originalItem = (original as any).find(item => item[idField] === updatedItem[idField]);
        if (!originalItem) connected.push(updatedItem[idField]);
    }
    return connected.length > 0 ? connected : undefined;
}

/**
 * Finds objects which have been created, and returns an array of the created objects, formatted for
 * the create mutation
 * @param original The original array
 * @param updated The updated array
 * @param formatForCreate The function to format an object for the create mutation
 * @returns Created objects formatted for the create mutation,
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

const asArray = <T>(value: T | T[]): T[] => {
    if (Array.isArray(value)) return value;
    return [value];
}

type RelationshipType = 'Connect' | 'Create' | 'Delete' | 'Disconnect' | 'Update';

// Array if isOneToOne is false, otherwise single
type MaybeArray<T extends 'object' | 'id', IsOneToOne extends 'one' | 'many'> =
    T extends 'object' ?
    IsOneToOne extends 'one' ? object : object[] :
    IsOneToOne extends 'one' ? string : string[]

// Array if isOneToOne is false, otherwise boolean
type MaybeArrayBoolean<IsOneToOne extends 'one' | 'many'> =
    IsOneToOne extends 'one' ? boolean : string[]

type UpdateRelOutput<
    IsOneToOne extends 'one' | 'many',
    RelTypes extends string,
    FieldName extends string,
> = (
        ({ [x in `${FieldName}Connect`]: 'Connect' extends RelTypes ? MaybeArray<'id', IsOneToOne> : never }) &
        ({ [x in `${FieldName}Create`]: 'Create' extends RelTypes ? MaybeArray<'object', IsOneToOne> : never }) &
        ({ [x in `${FieldName}Delete`]: 'Delete' extends RelTypes ? MaybeArrayBoolean<IsOneToOne> : never }) &
        ({ [x in `${FieldName}Disconnect`]: 'Disconnect' extends RelTypes ? MaybeArrayBoolean<IsOneToOne> : never }) &
        ({ [x in `${FieldName}Update`]: 'Update' extends RelTypes ? MaybeArray<'object', IsOneToOne> : never })
    )

/**
 * Shapes relationship connect fields for a GraphQL update input
 * @param item The item to shape
 * @param relation The name of the relationship field
 * @param relTypes The allowed operations on the relations (e.g. create, connect)
 * @param isOneToOne "one" if the relationship is one-to-one, and "many" otherwise. This makes the results a single object instead of an array
 * @param shape The relationship's shape object
 * @param idField The name of the id field in the relationship (defaults to "id")
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export const updateRel = <
    Item extends (IsOneToOne extends 'one' ?
        { [x in FieldName]?: { [y in IdFieldName]: string } | null | undefined } :
        { [x in FieldName]?: { [y in IdFieldName]: string }[] | null | undefined }),
    FieldName extends string,
    RelTypes extends readonly RelationshipType[],
    // Shape object only required when RelTypes includes 'Create' or 'Update'
    Shape extends ('Create' extends RelTypes[number] ?
        'Update' extends RelTypes[number] ?
        ShapeModel<any, any, any> :
        ShapeModel<any, any, null> :
        'Update' extends RelTypes[number] ?
        ShapeModel<any, null, any> :
        never),
    IsOneToOne extends 'one' | 'many',
    IdFieldName extends string = 'id'
>(
    originalItem: Item,
    updatedItem: Item,
    relation: FieldName,
    relTypes: RelTypes,
    isOneToOne: IsOneToOne,
    shape?: Shape,
    idField?: IdFieldName,
): UpdateRelOutput<IsOneToOne, RelTypes[number], FieldName> => {
    // Check if shape is required
    if (relTypes.includes('Create') || relTypes.includes('Update')) {
        if (!shape) throw new Error('Model is required if relTypes includes "Create" or "Update"');
    }
    // Find relation data in item
    const originalRelationData = originalItem[relation];
    const updatedRelationData = updatedItem[relation];
    // If not original or updated, return empty object
    if (originalRelationData === undefined && updatedRelationData === undefined) return {} as any;
    // If no original, treat as create/connect
    if (originalRelationData === undefined) {
        return createRel(updatedItem, relation, relTypes.filter(x => x === 'Create' || x === 'Connect') as any, isOneToOne, shape as any, idField) as any;
    }
    // Initialize result
    const result: { [x: string]: any } = {};
    // Loop through relation types
    for (const t of relTypes) {
        // If type is connect, add IDs to result
        if (t === 'Connect') {
            const shaped = findConnectedItems(
                asArray(originalRelationData as any), 
                asArray(updatedRelationData as any), 
                idField ?? 'id');
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
        else if (t === 'Create') {
            const shaped = findCreatedItems(
                asArray(originalRelationData as object), 
                asArray(updatedRelationData as object), 
                (shape as ShapeModel<any, true, any>).create,
                idField ?? 'id');
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
        else if (t === 'Delete') {
            const shaped = findDeletedItems(
                asArray(originalRelationData as any),
                asArray(updatedRelationData as any),
                idField ?? 'id');
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
        else if (t === 'Disconnect') {
            const shaped = findDisconnectedItems(
                asArray(originalRelationData as any),
                asArray(updatedRelationData as any),
                idField ?? 'id');
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
        else if (t === 'Update') {
            const shaped = findUpdatedItems(
                asArray(originalRelationData as any),
                asArray(updatedRelationData as any),
                (shape as ShapeModel<any, any, true>).hasObjectChanged ?? hasObjectChanged,
                (shape as ShapeModel<any, any, true>).update,
                idField ?? 'id');
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
    }
    // Return result
    return result as any;
};