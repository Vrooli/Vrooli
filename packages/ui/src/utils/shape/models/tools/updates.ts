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
        PubSub.get().publish("snack", { messageKey: "NothingToUpdate", severity: "Error" });
        return undefined;
    }
    return result;
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
    Shape extends ShapeModel<any, any, any>,
    IsOneToOne extends "one" | "many",
>(
    original: Item,
    updated: Item,
    relation: FieldName,
    relTypes: RelTypes,
    isOneToOne: IsOneToOne,
    shape?: Shape,
    preShape?: (item: any, originalOrUpdated: any) => any,
): UpdateRelOutput<IsOneToOne, RelTypes[number], FieldName> => {
    // Initialize result
    const result: UpdateRelOutput<IsOneToOne, RelTypes[number], FieldName> = {} as UpdateRelOutput<IsOneToOne, RelTypes[number], FieldName>;
    // Check if shape is required
    if (relTypes.includes("Create") || relTypes.includes("Update")) {
        if (!shape) throw new Error(`Model is required if relTypes includes "Create" or "Update": ${relation}`);
    }
    // If no original, treat as create/connect
    if (!exists(original[relation])) {
        return createRel(
            updated,
            relation,
            relTypes.filter(x => ["Create", "Connect"].includes(x)) as any[],
            isOneToOne,
            shape as any,
        ) as UpdateRelOutput<IsOneToOne, RelTypes[number], FieldName>;
    }
    // If updated if undefined, return empty object
    if (updated[relation] === undefined) return result;
    // If updated is null, treat as delete/disconnect. 
    // We do this by removing connect/create/update from relTypes
    const filteredRelTypes = updated[relation] === null ? relTypes.filter(x => !["Create", "Connect", "Update"].includes(x)) : relTypes;
    // Find relation data in item
    const originalRelation = asArray(original[relation] as any);
    const updatedRelation = asArray(updated[relation] as any);
    const idField = shape?.idField ?? "id";
    const preShaper = preShape ?? ((x) => x);
    // Check connect before create, so that we can exclude items in the create array which are being connected
    if (filteredRelTypes.includes("Connect")) {
        const shaped: string[] = [];
        for (const updatedItem of (updatedRelation as { id?: string }[])) {
            // Can only connect items that exist and have an ID. 
            // We must use "id" and not the idField, because custom unique fields don't have an equivalent to 
            // DUMMY_ID to indicate that they are for creating instead of connecting
            if (!updatedItem || !updatedItem.id) continue;
            // Check if an item with the same ID exists in the original array
            const originalItem = (originalRelation as { id?: string }[]).find(item => item.id === updatedItem.id);
            // If not, we can connect it if the ID is not DUMMY_ID
            if (!originalItem && updatedItem.id !== DUMMY_ID) shaped.push(updatedItem.id);
        }
        result[`${relation}Connect` as string] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
    }
    // Check create
    if (filteredRelTypes.includes("Create")) {
        if (!shape || !(shape as ShapeModel<object, object, object | null>).create) throw new Error(`shape.create is required for create: ${relation}`);
        let shaped: any[] = [];
        for (const updatedItem of updatedRelation.map(data => preShaper(data, updated))) {
            if (!updatedItem || !updatedItem[idField]) continue;
            const oi = originalRelation.find(item => item[idField] === updatedItem[idField]);
            // Add if not found in original, and not a connect-only item
            if (!oi && !hasOnlyConnectData(updatedItem)) {
                shaped.push((shape as ShapeModel<object, object, object | null>).create(updatedItem));
            }
        }
        if (shaped.length > 0) {
            // Filter out items which are already connected
            shaped = shaped?.filter((x) => x.id === DUMMY_ID || !asArray(result[`${relation}Connect` as string]).includes(x.id));
            result[`${relation}Create` as string] = isOneToOne === "one" ? shaped[0] : shaped;
        }
    }
    // Check disconnect before delete, so that we can exclude items in the delete array which are being disconnected
    if (filteredRelTypes.includes("Disconnect")) {
        const shaped: string[] = [];
        // Any item that exists in the original array, but not in the updated array, should be disconnected
        for (const originalItem of originalRelation) {
            if (!originalItem || !originalItem[idField]) continue;
            const updatedItem = updatedRelation.find(item => item[idField] === originalItem[idField]);
            if (!updatedItem) shaped.push(originalItem[idField as string]);
        }
        // Logic is different for one-to-one disconnects, since it expects a boolean instead of an ID. 
        // And if we already have a connect, we can exclude the disconnect altogether
        if (isOneToOne === "one") {
            const hasConnect = typeof result[`${relation}Connect`] === "string";
            if (!hasConnect) result[`${relation}Disconnect` as string] = shaped && shaped[0] ? true : undefined;
        } else {
            result[`${relation}Disconnect` as string] = shaped;
        }
    }
    // Check delete
    if (filteredRelTypes.includes("Delete")) {
        let shaped: string[] = [];
        // Any item that exists in the original array, but not in the updated array, should be deleted
        for (const originalItem of originalRelation) {
            if (!originalItem || !originalItem[idField]) continue;
            const updatedItem = updatedRelation.find(item => item[idField] === originalItem[idField]);
            if (!updatedItem) shaped.push(originalItem[idField as string]);
        }
        // Filter out items which are already disconnected
        if (shaped.length > 0) {
            shaped = shaped?.filter((x) => !asArray(result[`${relation}Disconnect` as string]).includes(x));
            result[`${relation}Delete` as string] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
    }
    // Check update
    if (filteredRelTypes.includes("Update")) {
        if (!shape || !(shape as ShapeModel<object, object | null, object>).update) throw new Error(`shape.create is required for create: ${relation}`);
        const originalDataArray = originalRelation.map(data => preShaper(data, originalRelation));
        const shaped: any[] = [];
        for (const updatedItem of updatedRelation.map(data => preShaper(data, updatedRelation))) {
            if (!updatedItem || !updatedItem[idField]) continue;
            const oi = originalDataArray.find(item => item && item[idField] && item[idField] === updatedItem[idField]);
            if (oi && ((shape as ShapeModel<object, object | null, object>).hasObjectChanged ?? hasObjectChanged)(oi, updatedItem)) {
                shaped.push((shape as ShapeModel<object, object | null, object>).update(oi, updatedItem));
            }
        }
        result[`${relation}Update` as string] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
    }
    // Return result
    return result;
};
