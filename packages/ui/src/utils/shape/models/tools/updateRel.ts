import { exists } from '@shared/utils';
import { ShapeModel } from 'types';
import { hasObjectChanged } from 'utils/shape/general';
import { createRel } from './createRel';

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
 * @returns Created objects formatted for the create mutation,
 * or undefined if no objects have been created.
 */
const findCreatedItems = <
    Item extends { [x in FieldName]: Record<string, any> | Record<string, any>[] },
    FieldName extends string,
    Shape extends ShapeModel<any, {}, any>,
    Output extends {}
>(
    original: Item,
    updated: Item,
    relation: FieldName,
    shape: Shape,
    preShape?: (item: any, originalOrUpdated: Item) => any,
): Output[] | undefined => {
    const idField = shape.idField ?? 'id';
    const preShaper = preShape ?? ((x: any) => x);
    const originalDataArray = asArray(original[relation]);
    const updatedDataArray = asArray(updated[relation]);
    const createdItems: Output[] = [];
    for (const updatedItem of updatedDataArray) {
        if (!updatedItem || !updatedItem[idField]) continue;
        const oi = originalDataArray.find(item => item[idField] === updatedItem[idField]);
        // Add if not found in original, and has more than just the id field (if only ID, must be a connect)
        if (!oi && Object.values(updatedItem).length > 1) {
            createdItems.push(shape.create(preShaper(updatedItem, updated)) as Output);
        }
    }
    return createdItems.length > 0 ? createdItems : undefined;
}

/**
 * Find objects which have been updated, and returns an array of the updated objects, formatted for 
 * the update mutation
 * @param original The original array
 * @param updated The updated array
 * @returns An array of the updated objects formatted for the update mutation, 
 * or undefined if no objects have been updated.
 */
const findUpdatedItems = <
    Item extends { [x in FieldName]: Record<string, any> | Record<string, any>[] },
    FieldName extends string,
    Shape extends ShapeModel<any, null, {}>,
    Output extends {}
>(
    original: Item,
    updated: Item,
    relation: FieldName,
    shape: Shape,
    preShape?: (item: any, originalOrUpdated: Item) => any,
): Output[] | undefined => {
    const idField = shape.idField ?? 'id';
    const preShaper = preShape ?? ((x: any) => x);
    const originalDataArray = asArray(original[relation]);
    const updatedDataArray = asArray(updated[relation]);
    const updatedItems: Output[] = [];
    for (const updatedItem of updatedDataArray) {
        if (!updatedItem || !updatedItem[idField]) continue;
        const oi = originalDataArray.find(item => item && item[idField] && item[idField] === updatedItem[idField]);
        if (oi && (shape.hasObjectChanged ?? hasObjectChanged)(preShaper(oi, original), preShaper(updatedItem, updated))) {
            updatedItems.push(shape.update(preShaper(oi, original), preShaper(updatedItem, updated)) as Output);
        }
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
    IsOneToOne extends 'one' ? any : any[] :
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
 * @param preShape A function to convert the item before passing it to the shape function
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export const updateRel = <
    Item extends (IsOneToOne extends 'one' ?
        { [x in FieldName]?: {} | null | undefined } :
        { [x in FieldName]?: {}[] | null | undefined }),
    FieldName extends string,
    RelTypes extends readonly RelationshipType[],
    // Shape object only required when RelTypes includes 'Create' or 'Update'
    Shape extends ('Create' extends RelTypes[number] ?
        'Update' extends RelTypes[number] ?
        ShapeModel<any, {}, {}> :
        ShapeModel<any, {}, null> :
        'Update' extends RelTypes[number] ?
        ShapeModel<any, null, {}> :
        never),
    IsOneToOne extends 'one' | 'many',
>(
    original: Item,
    updated: Item,
    relation: FieldName,
    relTypes: RelTypes,
    isOneToOne: IsOneToOne,
    shape?: Shape,
    preShape?: (item: Record<string, any>, originalOrUpdated: Record<string, any>) => any,
): UpdateRelOutput<IsOneToOne, RelTypes[number], FieldName> => {
    // Check if shape is required
    if (relTypes.includes('Create') || relTypes.includes('Update')) {
        if (!shape) throw new Error(`Model is required if relTypes includes "Create" or "Update": ${relation}`);
    }
    // Find relation data in item
    const originalRelationData = original[relation];
    const updatedRelationData = updated[relation];
    // If no original, treat as create/connect
    if (!exists(originalRelationData)) {
        return createRel(
            updated,
            relation,
            relTypes.filter(x => x === 'Create' || x === 'Connect') as any,
            isOneToOne,
            shape as any,
        ) as any;
    }
    // If updated if undefined, return empty object
    if (updatedRelationData === undefined) return {} as any;
    // If updated is null, treat as delete/disconnect. 
    // We do this by removing connect/create/update from relTypes
    const filteredRelTypes = updatedRelationData === null ?
        relTypes.filter(x => x !== 'Create' && x !== 'Connect' && x !== 'Update') as any :
        relTypes;
    // Initialize result
    const result: { [x: string]: any } = {};
    const idField = (shape as ShapeModel<any, any, any> | undefined)?.idField ?? 'id';
    // Loop through relation types
    for (const t of filteredRelTypes) {
        // If type is connect, add IDs to result
        if (t === 'Connect') {
            const shaped = findConnectedItems(
                asArray(originalRelationData as any),
                asArray(updatedRelationData as any),
                idField);
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
        else if (t === 'Create') {
            const shaped = findCreatedItems(
                original as any,
                updated as any,
                relation,
                shape as ShapeModel<any, {}, null>,
                preShape as any);
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
        else if (t === 'Delete') {
            const shaped = findDeletedItems(
                asArray(originalRelationData as any),
                asArray(updatedRelationData as any),
                idField);
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
        else if (t === 'Disconnect') {
            const shaped = findDisconnectedItems(
                asArray(originalRelationData as any),
                asArray(updatedRelationData as any),
                idField);
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
        else if (t === 'Update') {
            const shaped = findUpdatedItems(
                original as any,
                updated as any,
                relation,
                shape as ShapeModel<any, null, {}>,
                preShape as any);
            result[`${relation}${t}`] = isOneToOne === 'one' ? shaped && shaped[0] : shaped;
        }
    }
    // Return result
    return result as any;
};