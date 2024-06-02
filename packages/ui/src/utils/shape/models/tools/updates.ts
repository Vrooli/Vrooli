import { DUMMY_ID, exists, uuid } from "@local/shared";
import { ShapeModel } from "types";
import { PubSub } from "utils/pubsub";
import { hasObjectChanged } from "utils/shape/general";
import { createOwner, createRel, shouldConnect } from "./creates";

type OwnerPrefix = "" | "ownedBy";
type OwnerType = "User" | "Team";

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
    const originalVersions = originalRoot.versions ?? [];
    // Find every version in the updated root that is not in the original root (using the version's id)
    const newVersions = updatedRoot.versions.filter((v) => !originalVersions.find((ov) => ov.id === v.id));
    // Versions in the originalRoot and the updatedRoot, which are different from each other, are updates
    const updatedVersions = updatedRoot.versions.filter((v) => originalVersions.some((ov) => ov.id === v.id && JSON.stringify(ov) !== JSON.stringify(v)));
    // Shape and return version data
    let result: { versionsCreate?: VersionCreateInput[], versionsUpdate?: VersionUpdateInput[] } = {};
    if (newVersions.length > 0) {
        result.versionsCreate = newVersions.map((version) => shape.create({
            ...version,
            root: { id: originalRoot.id },
        })) as VersionCreateInput[];
    }
    if (updatedVersions.length > 0) {
        result.versionsUpdate = updatedVersions.map((version) => shape.update(originalRoot.versions?.find((ov) => ov.id === version.id), {
            ...version,
            root: { id: originalRoot.id },
        })) as VersionUpdateInput[];
    }
    return result;
};

type UpdatePrimsResult<T, K extends keyof T, PK extends keyof T> = ({ [F in K]: Exclude<T[F], null | undefined> } & { [F in PK]: T[F] });

/**
 * Helper function for setting a list of primitive fields of an update 
 * shape. If updated is different from original, return updated,
 * otherwise return just the primary key
 * 
 * NOTE 1: We return the primary key because this is used in conjunction with updateRel. 
 * If there are no primitive field updates, but there are relationship updates, then 
 * we need to have the ID to properly update the object.
 * 
 * NOTE 2: Due to TypeScript limitations, return type assumes that every field 
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
): UpdatePrimsResult<T, K, PK> => {
    const getPrimaryValue = (): string | undefined => {
        if (!primary) return undefined;
        let primaryValue = (original?.[primary] ?? updated?.[primary]) as string | undefined;
        if (primaryValue === DUMMY_ID) primaryValue = uuid();
        return primaryValue;
    }
    const primaryValue = getPrimaryValue();

    // If no updated, return just the primary key (if it exists)
    if (!updated) {
        if (primary && primaryValue) return { [primary]: primaryValue } as UpdatePrimsResult<T, K, PK>;
        return {} as UpdatePrimsResult<T, K, PK>;
    }

    // Find changed fields
    const changedFields = fields.reduce((acc, field) => {
        const key = Array.isArray(field) ? field[0] : field;
        const value = Array.isArray(field) ? field[1](updated[key]) : updated[key];
        const isDifferent = !original ? true : JSON.stringify(value) !== JSON.stringify(original[key]);
        if (isDifferent && value !== undefined) return { ...acc, [key]: value };
        return acc;
    }, {});

    // If no primary, return changed fields
    if (!primary || !primaryValue) return changedFields as UpdatePrimsResult<T, K, PK>;
    // Return changed fields with primary key
    return { ...changedFields, [primary]: primaryValue } as UpdatePrimsResult<T, K, PK>;
};

type UpdateTranslationPrimsResult<T, K extends keyof T, PK extends keyof T> = ({ [F in K]: Exclude<T[F], null | undefined> } & { [F in PK]: T[F] } & { language: string });

/**
 * Like updatePrims, but forces "language" field to be included in the return type 
 * (unless the result is empty)
 * @param original The original object
 * @param updated The updated object
 * @param primary The primary key of the object, which is always returned
 * @param fields The fields to check, which are returned if they are different
 */
export const updateTranslationPrims = <T extends { language: string }, K extends keyof T, PK extends keyof T>(
    original: T | null | undefined,
    updated: T | null | undefined,
    primary: PK | null,
    ...fields: (K | [K, (val: any) => any])[]
): UpdateTranslationPrimsResult<T, K, PK> => {
    const updatePrimsResult = updatePrims(original, updated, primary, ...fields);
    // If the result is empty or only has the primary key, return empty object
    if (Object.keys(updatePrimsResult).length <= 1) return {} as UpdateTranslationPrimsResult<T, K, PK>;
    return {
        ...updatePrimsResult,
        language: original?.language ?? updated?.language ?? "en",
    } as UpdateTranslationPrimsResult<T, K, PK>;
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
            relTypes.filter(x => ["Create", "Connect"].includes(x)) as any[], // Only allow create/connect, since other types don't make sense without an original
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
    // Check connect
    if (filteredRelTypes.includes("Connect")) {
        const shaped: string[] = [];
        for (const updatedItem of (updatedRelation as { id?: string }[])) {
            // Can only connect items that exist and have an ID
            if (!updatedItem || !updatedItem[idField]) continue;
            // Can only connect items that are not in the original array (otherwise, they should be updated or noop)
            const originalItem = (originalRelation as { id?: string }[]).find(item => item[idField] === updatedItem[idField]);
            if (originalItem) continue;
            // Can only connect if the item is not new (i.e. ID is not DUMMY_ID), AND it's not going to be created instead
            // If not, we can connect it if the ID is not DUMMY_ID
            const canConnect = updatedItem.id !== DUMMY_ID && (!filteredRelTypes.includes("Create") || shouldConnect(updatedItem));
            if (canConnect) shaped.push(updatedItem[idField]);
        }
        if (shaped.length > 0) {
            result[`${relation}Connect` as string] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
    }
    // Check create
    if (filteredRelTypes.includes("Create")) {
        if (!shape || !(shape as ShapeModel<object, object, object | null>).create) throw new Error(`shape.create is required for create: ${relation}`);
        let shaped: any[] = [];
        for (const updatedItem of updatedRelation.map(data => preShaper(data, updated))) {
            if (!updatedItem || !updatedItem[idField]) continue;
            const oi = originalRelation.find(item => item?.[idField] === updatedItem[idField]);
            // Add if not found in original, and not a connect-only item
            if (!oi && !shouldConnect(updatedItem)) {
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
            const updatedItem = updatedRelation.find(item => item?.[idField] === originalItem[idField]);
            if (!updatedItem) shaped.push(originalItem[idField as string]);
        }
        if (shaped.length > 0) {
            // Logic is different for one-to-one disconnects, since it expects a boolean instead of an ID. 
            // And if we already have a connect, we can exclude the disconnect altogether
            if (isOneToOne === "one") {
                const hasConnect = typeof result[`${relation}Connect`] === "string";
                if (!hasConnect && shaped && shaped[0]) result[`${relation}Disconnect` as string] = true;
            } else {
                result[`${relation}Disconnect` as string] = shaped;
            }
        }
    }
    // Check delete
    if (filteredRelTypes.includes("Delete")) {
        let shaped: string[] = [];
        // Any item that exists in the original array, but not in the updated array, should be deleted
        for (const originalItem of originalRelation) {
            if (!originalItem || !originalItem[idField]) continue;
            const updatedItem = updatedRelation.find(item => item?.[idField] === originalItem[idField]);
            if (!updatedItem) shaped.push(originalItem[idField as string]);
        }
        // Filter out items which are already disconnected
        if (shaped.length > 0) {
            shaped = shaped?.filter((x) => !asArray(result[`${relation}Disconnect` as string]).includes(x));
        }
        if (shaped.length > 0) {
            // Logic is different for one-to-one deletes, since it expects a boolean instead of an ID.
            if (isOneToOne === "one") {
                result[`${relation}Delete` as string] = true;
            } else {
                result[`${relation}Delete` as string] = shaped;
            }
        }
    }
    // Check update
    if (filteredRelTypes.includes("Update")) {
        if (!shape || !(shape as ShapeModel<object, object | null, object>).update) throw new Error(`shape.update is required for update: ${relation}`);
        const originalDataArray = originalRelation.map(data => preShaper(data, originalRelation));
        const shaped: any[] = [];
        for (const updatedItem of updatedRelation.map(data => preShaper(data, updatedRelation))) {
            if (!updatedItem || !updatedItem[idField]) continue;
            const oi = originalDataArray.find(item => item?.[idField] === updatedItem[idField]);
            if (oi && ((shape as ShapeModel<object, object | null, object>).hasObjectChanged ?? hasObjectChanged)(oi, updatedItem)) {
                const update = (shape as ShapeModel<object, object | null, object>).update(oi, updatedItem);
                if (typeof update === "object" && Object.keys(update).length > 0) shaped.push(update);
            }
        }
        if (shaped.length > 0) {
            result[`${relation}Update` as string] = isOneToOne === "one" ? shaped && shaped[0] : shaped;
        }
    }
    // Return result
    return result;
};
