import { DUMMY_ID, exists, uuid } from "@local/shared";
import { ShapeModel } from "types";
import { PubSub } from "utils/pubsub";
import { hasObjectChanged } from "utils/shape/general";
import { createOwner, createRel, hasOnlyConnectData } from "./creates";

type OwnerPrefix = "" | "ownedBy";
type OwnerType = "User" | "Organization";

type RelationshipType = "Connect" | "Create" | "Delete" | "Disconnect" | "Update";

// Array if isOneToOne is false, otherwise single
type MaybeArray<T extends "object" | "id", IsOneToOne extends "one" | "many"> =
    T extends "object" ?
    IsOneToOne extends "one" ? any : any[] :
    IsOneToOne extends "one" ? string : string[]

// Array if isOneToOne is false, otherwise boolean
type MaybeArrayBoolean<IsOneToOne extends "one" | "many"> =
    IsOneToOne extends "one" ? boolean : string[]

type UpdateRelOutput<
    IsOneToOne extends "one" | "many",
    RelTypes extends string,
    FieldName extends string,
> = (
        ({ [x in `${FieldName}Connect`]: "Connect" extends RelTypes ? MaybeArray<"id", IsOneToOne> : never }) &
        ({ [x in `${FieldName}Create`]: "Create" extends RelTypes ? MaybeArray<"object", IsOneToOne> : never }) &
        ({ [x in `${FieldName}Delete`]: "Delete" extends RelTypes ? MaybeArrayBoolean<IsOneToOne> : never }) &
        ({ [x in `${FieldName}Disconnect`]: "Disconnect" extends RelTypes ? MaybeArrayBoolean<IsOneToOne> : never }) &
        ({ [x in `${FieldName}Update`]: "Update" extends RelTypes ? MaybeArray<"object", IsOneToOne> : never })
    )

/**
 * Shapes ownership connect fields for a GraphQL update input
 * @param item The item to shape, with an "owner" field
 * @returns Ownership connect object
 */
export const updateOwner = <
    OType extends OwnerType,
    OriginalItem extends { owner?: { __typename: OwnerType, id: string } | null | undefined },
    UpdatedItem extends { owner?: { __typename: OType, id: string } | null | undefined },
    Prefix extends OwnerPrefix & string
>(
    originalItem: OriginalItem,
    updatedItem: UpdatedItem,
    prefix: Prefix = "" as Prefix,
): { [K in `${Prefix}${OType}Connect`]?: string } => {
    // Find owner data in item
    const originalOwnerData = originalItem.owner;
    const updatedOwnerData = updatedItem.owner;
    // If original and updated missing
    if (
        (originalOwnerData === null || originalOwnerData === undefined) &&
        (updatedOwnerData === null || updatedOwnerData === undefined)
    ) return {};
    // If original and updated match, return empty object
    if (originalOwnerData?.id === updatedOwnerData?.id) return {};
    // If updated missing, disconnect
    // TODO disabled for now
    // Treat as create
    return createOwner(updatedItem as any, prefix);
};

/**
 * Shapes versioning relation data for a GraphQL update input
 * @param root The root object, which contains the version data
 * @param shape The version's shape object
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export const updateVersion = <
    Root extends { id: string, versions?: Record<string, unknown>[] | null | undefined },
    VersionCreateInput extends object,
    VersionUpdateInput extends object,
>(
    originalRoot: Root,
    updatedRoot: Root,
    shape: ShapeModel<any, VersionCreateInput, VersionUpdateInput>,
): ({ versionsCreate?: VersionCreateInput[], versionsUpdate?: VersionUpdateInput[] }) => {
    // Return empty object if no updated version data. We don't handle deletes here
    if (!updatedRoot.versions) return {};
    // Find every version in the updated root that is not in the original root (using the version's id)
    const newVersions = updatedRoot.versions.filter((v) => !originalRoot.versions?.find((ov) => ov.id === v.id));
    // Every other version in the updated root is an update
    const updatedVersions = updatedRoot.versions.filter((v) => !newVersions.find((nv) => nv.id === v.id));
    // Shape and return version data
    return {
        versionsCreate: newVersions.map((version) => shape.create({
            ...version,
            root: { id: originalRoot.id },
        })) as VersionCreateInput[],
        versionsUpdate: updatedVersions.map((version) => shape.update(originalRoot.versions?.find((ov) => ov.id === version.id), {
            ...version,
            root: { id: originalRoot.id },
        })) as VersionUpdateInput[],
    };
};

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
export const updatePrims = <T, K extends keyof T, PK extends keyof T>(
    original: T | null | undefined,
    updated: T | null | undefined,
    primary: PK | null,
    ...fields: (K | [K, (val: any) => any])[]
): ({ [F in K]: Exclude<T[F], null | undefined> } & { [F in PK]: T[F] }) => {
    // If no original or updated, return undefined for all fields
    if (!updated || !original) {
        return fields.reduce((acc, field) => {
            const key = Array.isArray(field) ? field[0] : field;
            return { ...acc, [key]: undefined };
        }, {}) as any;
    }

    // Create prims
    const changedFields = fields.reduce((acc, field) => {
        const key = Array.isArray(field) ? field[0] : field;
        const value = Array.isArray(field) ? field[1](updated[key]) : updated[key];
        return { ...acc, [key]: value !== original[key] ? value : undefined };
    }, {});

    // If no primary key, return changed fields
    if (!primary) return changedFields as any;

    // If primary key is not an ID, return changed fields with primary key
    if (primary !== "id") return { ...changedFields, [primary]: original[primary] } as any;

    // If primary key is an ID, return changed fields with primary key, and make sure it's not DUMMY_ID
    return { ...changedFields, [primary]: original[primary] === DUMMY_ID ? uuid() : original[primary] } as any;
};

/**
 * Like updatePrims, but forces "language" field to be included in the return type
 * @param original The original object
 * @param updated The updated object
 * @param primary The primary key of the object, which is always returned
 * @param fields The fields to check, which are returned if they are different
 */
export const updateTranslationPrims = <T extends { language: string }, K extends keyof T, PK extends keyof T>(
    original: T | null | undefined,
    updated: T | null | undefined,
    primary: PK | null,
    ...fields: K[]
): ({ [F in K]: Exclude<T[F], null | undefined> } & { [F in PK]: T[F] } & { language: string }) => {
    const updatePrimsResult = updatePrims(original, updated, primary, ...fields);
    return { ...updatePrimsResult, language: original?.language ?? updated?.language } as any;
};

/**
 * Helper function for formatting an object for an update mutation
 * @param updated The updated object
 * @param shape Shapes the updated object for the update mutation
 * @param assertHasUpdate Asserts that the updated object has at least one value
 */
export const shapeUpdate = <
    Input extends object,
    Output extends object,
    AssertHasUpdate extends boolean = false
>(
    updated: Input | null | undefined,
    shape: Output | (() => Output),
    assertHasUpdate: AssertHasUpdate = false as AssertHasUpdate,
): Output | undefined => {
    if (!updated) return undefined;
    let result = typeof shape === "function" ? (shape as () => Output)() : shape;
    // Remove every value from the result that is undefined
    if (result) result = Object.fromEntries(Object.entries(result).filter(([, value]) => value !== undefined)) as Output;
    // If assertHasUpdate is true, make sure that the result has at least one value
    if (assertHasUpdate && (!result || Object.keys(result).length === 0)) {
        PubSub.get().publishSnack({ messageKey: "NothingToUpdate", severity: "Error" });
        return undefined;
    }
    return result;
};

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
    idField: IDField = "id" as IDField,
): string | string[] | undefined => {
    const connected: string[] = [];
    for (const updatedItem of (updated as any)) {
        if (!updatedItem || !updatedItem[idField]) continue;
        const originalItem = (original as any).find(item => item[idField] === updatedItem[idField]);
        if (!originalItem) connected.push(updatedItem[idField]);
    }
    return connected.length > 0 ? connected : undefined;
};

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
    Shape extends ShapeModel<any, object, any>,
    Output extends object
>(
    original: Item,
    updated: Item,
    relation: FieldName,
    shape: Shape,
    preShape?: (item: any, originalOrUpdated: Item) => any,
): Output[] | undefined => {
    const idField = shape.idField ?? "id";
    const preShaper = preShape ?? ((x: any) => x);
    const originalDataArray = asArray(original[relation]);
    const updatedDataArray = asArray(updated[relation]).map(data => preShaper(data, updated));
    const createdItems: Output[] = [];
    for (const updatedItem of updatedDataArray) {
        if (!updatedItem || !updatedItem[idField]) continue;
        const oi = originalDataArray.find(item => item[idField] === updatedItem[idField]);
        // Add if not found in original, and not a connect-only item
        if (!oi && !hasOnlyConnectData(updatedItem)) {
            createdItems.push(shape.create(updatedItem) as Output);
        }
    }
    return createdItems.length > 0 ? createdItems : undefined;
};

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
    Shape extends ShapeModel<any, null, object>,
    Output extends object
>(
    original: Item,
    updated: Item,
    relation: FieldName,
    shape: Shape,
    preShape?: (item: any, originalOrUpdated: Item) => any,
): Output[] | undefined => {
    const idField = shape.idField ?? "id";
    const preShaper = preShape ?? ((x: any) => x);
    const originalDataArray = asArray(original[relation]).map(data => preShaper(data, original));
    const updatedDataArray = asArray(updated[relation]).map(data => preShaper(data, updated));
    const updatedItems: Output[] = [];
    for (const updatedItem of updatedDataArray) {
        if (!updatedItem || !updatedItem[idField]) continue;
        const oi = originalDataArray.find(item => item && item[idField] && item[idField] === updatedItem[idField]);
        if (oi && (shape.hasObjectChanged ?? hasObjectChanged)(oi, updatedItem)) {
            updatedItems.push(shape.update(oi, updatedItem) as Output);
        }
    }
    return updatedItems.length > 0 ? updatedItems : undefined;
};

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
    idField: IDField = "id" as IDField,
): string[] | undefined => {
    const removed: string[] = [];
    for (const originalItem of original) {
        if (!originalItem || !originalItem[idField]) continue;
        const updatedItem = updated.find(item => item[idField] === originalItem[idField]);
        if (!updatedItem) removed.push(originalItem[idField as string]);
    }
    return removed.length > 0 ? removed : undefined;
};

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
    idField: IDField = "id" as IDField,
): string[] | undefined => {
    const disconnected: string[] = [];
    for (const originalItem of original) {
        if (!original || !originalItem[idField]) continue;
        const updatedItem = updated.find(item => item[idField] === originalItem[idField]);
        if (!updatedItem) disconnected.push(originalItem[idField as string]);
    }
    return disconnected.length > 0 ? disconnected : undefined;
};

const asArray = <T>(value: T | T[]): T[] => {
    if (Array.isArray(value)) return value;
    return [value];
};

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
    Item extends (IsOneToOne extends "one" ?
        { [x in FieldName]?: object | null | undefined } :
        { [x in FieldName]?: object[] | null | undefined }),
    FieldName extends string,
    RelTypes extends readonly RelationshipType[],
    // Shape object only required when RelTypes includes 'Create' or 'Update'
    Shape extends ("Create" extends RelTypes[number] ?
        "Update" extends RelTypes[number] ?
        ShapeModel<any, object, object> :
        ShapeModel<any, object, null> :
        "Update" extends RelTypes[number] ?
        ShapeModel<any, null, object> :
        never),
    IsOneToOne extends "one" | "many",
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
    if (relTypes.includes("Create") || relTypes.includes("Update")) {
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
            relTypes.filter(x => x === "Create" || x === "Connect") as any,
            isOneToOne,
            shape as any,
        ) as any;
    }
    // If updated if undefined, return empty object
    if (updatedRelationData === undefined) return {} as any;
    // If updated is null, treat as delete/disconnect. 
    // We do this by removing connect/create/update from relTypes
    const filteredRelTypes = updatedRelationData === null ?
        relTypes.filter(x => x !== "Create" && x !== "Connect" && x !== "Update") as any :
        relTypes;
    // Initialize result
    const result: { [x: string]: any } = {};
    const idField = (shape as ShapeModel<any, any, any> | undefined)?.idField ?? "id";
    // Loop through relation types
    for (const t of filteredRelTypes) {
        // If type is connect, add IDs to result
        if (t === "Connect") {
            const shaped = findConnectedItems(
                asArray(originalRelationData as any),
                asArray(updatedRelationData as any),
                idField);
            result[`${relation}${t}`] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
        else if (t === "Create") {
            const shaped = findCreatedItems(
                original as any,
                updated as any,
                relation,
                shape as ShapeModel<any, object, null>,
                preShape as any);
            result[`${relation}${t}`] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
        else if (t === "Delete") {
            const shaped = findDeletedItems(
                asArray(originalRelationData as any),
                asArray(updatedRelationData as any),
                idField);
            result[`${relation}${t}`] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
        else if (t === "Disconnect") {
            const shaped = findDisconnectedItems(
                asArray(originalRelationData as any),
                asArray(updatedRelationData as any),
                idField);
            // Logic is different for one-to-one disconnects, since it expects a boolean instead of an ID. 
            // And if we already have a connect, we can exclude the disconnect altogether
            if (isOneToOne === "one") {
                const hasConnect = typeof result[`${relation}Connect`] === "string";
                if (!hasConnect) result[`${relation}${t}`] = shaped && shaped[0] ? true : undefined;
            } else {
                result[`${relation}${t}`] = shaped;
            }
        }
        else if (t === "Update") {
            const shaped = findUpdatedItems(
                original as any,
                updated as any,
                relation,
                shape as ShapeModel<any, null, object>,
                preShape as any);
            result[`${relation}${t}`] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
    }
    // Return result
    return result as any;
};
