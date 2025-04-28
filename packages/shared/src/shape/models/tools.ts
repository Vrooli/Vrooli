import { ShapeModel } from "../../consts/commonTypes.js";
import { DUMMY_ID } from "../../id/uuid.js";
import { exists } from "../../utils/exists.js";
import { hasObjectChanged } from "../general/objectTools.js";

type OwnerPrefix = "" | "ownedBy";
type OwnerType = "User" | "Team";

type RelationshipTypeForCreates = "Connect" | "Create";

// Array if isOneToOne is false, otherwise single
type MaybeArray<T extends "object" | "id", IsOneToOne extends "one" | "many"> =
    T extends "object" ?
    IsOneToOne extends "one" ? any : any[] :
    IsOneToOne extends "one" ? string : string[]

// Array if isOneToOne is false, otherwise boolean
type MaybeArrayBoolean<IsOneToOne extends "one" | "many"> =
    IsOneToOne extends "one" ? boolean : string[]

type CreateRelOutput<
    IsOneToOne extends "one" | "many",
    RelTypes extends string,
    FieldName extends string,
> = (
        ({ [x in `${FieldName}Connect`]: "Connect" extends RelTypes ? MaybeArray<"id", IsOneToOne> : never }) &
        ({ [x in `${FieldName}Create`]: "Create" extends RelTypes ? MaybeArray<"object", IsOneToOne> : never })
    )


/**
 * Shapes ownership connect fields for a GraphQL create input. Will only 
 * return 0 or 1 fields - cannot have two owners.
 * @param item The item to shape, with an owner field starting with the specified prefix
 * @returns Ownership connect object
 */
export function createOwner<
    OType extends OwnerType,
    Item extends { owner?: { __typename: OType, id: string } | null | undefined },
    Prefix extends OwnerPrefix & string
>(
    item: Item,
    prefix: Prefix = "" as Prefix,
): { [K in `${Prefix}${OType}Connect`]?: string } {
    // Find owner data in item
    const ownerData = item.owner;
    // If owner data is undefined, or type is not a User or Team return empty object
    if (ownerData === null || ownerData === undefined || (ownerData.__typename !== "User" && ownerData.__typename !== "Team")) return {};
    // Create field name (with first letter lowercase)
    let fieldName = `${prefix}${ownerData.__typename}Connect`;
    fieldName = fieldName.charAt(0).toLowerCase() + fieldName.slice(1);
    // Return shaped field
    return { [fieldName]: ownerData.id } as any;
}

/**
 * Shapes versioning relation data for a GraphQL create input
 * @param root The root object, which contains the version data
 * @param shape The version's shape object
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export function createVersion<
    Root extends { id: string, versions?: Record<string, unknown>[] | null | undefined },
    VersionCreateInput extends object,
>(
    root: Root,
    shape: ShapeModel<any, VersionCreateInput, null>,
): ({ versionsCreate?: VersionCreateInput[] }) {
    // Return empty object if no version data
    if (!Array.isArray(root.versions) || root.versions.length === 0) return {};
    // Shape and return version data, injecting the root ID
    return {
        versionsCreate: root.versions.map((version) => shape.create({
            ...version,
            root: { id: root.id },
        })) as VersionCreateInput[],
    };
}

type CreatePrimsResult<T, K extends keyof T> = { [F in K]: Exclude<T[F], null | undefined> };

/**
 * Helper function for setting a list of primitive fields of a create 
 * shape. Essentially, adds every field that's defined (i.e. not undefined), 
 * and performs any pre-shaping functions.
 * 
 * NOTE 1: Due to TypeScript limitations, return type assumes that every field 
 * will be defined, even if it's undefined.
 * 
 * NOTE 2: If you need to reference the object's ID (which may be converted from a dummy ID 
 * to a real ID) in preShape functions, you should use the results from createPrims instead 
 * of relying in the original object.
 */
export function createPrims<T, K extends keyof T>(
    object: T,
    ...fields: (K | [K, (val: any) => any])[]
): CreatePrimsResult<T, K> {
    if (typeof object !== "object" || object === null) {
        console.error("Invalid input: object must be a non-null object");
        return {} as CreatePrimsResult<T, K>;
    }

    let hasId = false;
    // Create prims
    const prims = fields.reduce((acc, field) => {
        const key = Array.isArray(field) ? field[0] : field;
        if (key === "id") hasId = true;
        const value = Array.isArray(field) ? field[1](object[key]) : object[key];
        if (value !== undefined) return { ...acc, [key]: value };
        return acc;
    }, {}) as CreatePrimsResult<T, K>;


    return prims;
}

/**
 * Checks if an object should be connected or created.
 * @param data The object to check
 * @returns True if the object does not contain any data other than IDs and __typename, 
 * OR if it contains `__connect: true`
 */
export function shouldConnect(data: object) {
    if (typeof data !== "object" || data === null || (data as Record<string, unknown>).id === DUMMY_ID) return false;
    if (data["__connect"] === true) return true;
    const validKeys = Object.keys(data).filter(key => typeof data[key] !== undefined);
    return validKeys.every(k => ["id", "__typename"].includes(k) && typeof data[k] === "string");
}

/**
 * Shapes relationship connect fields for a GraphQL create input
 * @param item The item to shape
 * @param relation The name of the relationship field
 * @param relTypes The allowed operations on the relations (e.g. create, connect)
 * @param isOneToOne "one" if the relationship is one-to-one, and "many" otherwise. This makes the results a single object instead of an array
 * @param shape The relationship's shape object
 * @param preShape A function to convert the item before passing it to the shape function
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export function createRel<
    Item extends (IsOneToOne extends "one" ?
        { [x in FieldName]?: object | null | undefined } :
        { [x in FieldName]?: object[] | null | undefined }),
    FieldName extends string,
    RelTypes extends readonly RelationshipTypeForCreates[],
    // Shape object only required when RelTypes includes 'Create' or 'Update'
    Shape extends ("Create" extends RelTypes[number] ?
        ShapeModel<any, object, null> :
        never),
    IsOneToOne extends "one" | "many",
>(
    item: Item,
    relation: FieldName,
    relTypes: RelTypes,
    isOneToOne: IsOneToOne,
    shape?: Shape,
    preShape?: (item: any) => any,
): CreateRelOutput<IsOneToOne, RelTypes[number], FieldName> {
    // Check if shape is required
    if (relTypes.includes("Create")) {
        if (!shape) throw new Error(`Model is required if relTypes includes "Create": ${relation}`);
    }
    // Find relation data in item
    const relationData = item[relation];
    // If relation data is undefined, return empty object
    if (relationData === undefined) return {} as CreateRelOutput<IsOneToOne, RelTypes[number], FieldName>;
    // If relation data is null, this is only valid for 
    // disconnects, so return empty object (since we don't deal
    // with disconnects here)
    if (relationData === null) return {} as CreateRelOutput<IsOneToOne, RelTypes[number], FieldName>;
    // Initialize result
    const result: { [x: string]: any } = {};
    // Make preShape a function, if not provided 
    const preShaper = preShape ?? ((x: any) => x);
    // Apply preShape function to the relationData
    const shapedRelationData = Array.isArray(relationData)
        ? relationData.map(preShaper)
        : preShaper(relationData);
    // Loop through relation types
    for (const t of relTypes) {
        // If type is connect, add IDs to result
        if (t === "Connect") {
            // If create is an option, ignore items which should be created
            let filteredRelationData = Array.isArray(shapedRelationData) ? shapedRelationData : [shapedRelationData];
            if (relTypes.includes("Create")) {
                filteredRelationData = filteredRelationData.filter((x) => shouldConnect(x));
            }
            if (filteredRelationData.length === 0) continue;
            result[`${relation}${t}`] = isOneToOne === "one" ?
                filteredRelationData[0][shape?.idField ?? "id"] :
                (filteredRelationData as Array<object>).map((x) => x[shape?.idField ?? "id"]);
        }
        else if (t === "Create") {
            // Ignore items which should be connected
            let filteredRelationData = Array.isArray(shapedRelationData) ? shapedRelationData : [shapedRelationData];
            filteredRelationData = filteredRelationData.filter((x) => !shouldConnect(x));
            if (filteredRelationData.length === 0) continue;
            result[`${relation}${t}`] = isOneToOne === "one" ?
                shape!.create(filteredRelationData[0]) :
                (filteredRelationData as any).map((x: any) => shape!.create(x));
        }
    }
    // Return result
    return result as CreateRelOutput<IsOneToOne, RelTypes[number], FieldName>;
}

type RelationshipTypeForUpdates = "Connect" | "Create" | "Delete" | "Disconnect" | "Update";

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
export function updateOwner<
    OType extends OwnerType,
    OriginalItem extends { owner?: { __typename: OwnerType, id: string } | null | undefined },
    UpdatedItem extends { owner?: { __typename: OType, id: string } | null | undefined },
    Prefix extends OwnerPrefix & string
>(
    originalItem: OriginalItem,
    updatedItem: UpdatedItem,
    prefix: Prefix = "" as Prefix,
): { [K in `${Prefix}${OType}Connect`]?: string } {
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
}

/**
 * Shapes versioning relation data for a GraphQL update input
 * @param root The root object, which contains the version data
 * @param shape The version's shape object
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export function updateVersion<
    Root extends { id: string, versions?: Record<string, unknown>[] | null | undefined },
    VersionCreateInput extends object,
    VersionUpdateInput extends object,
>(
    originalRoot: Root,
    updatedRoot: Root,
    shape: ShapeModel<any, VersionCreateInput, VersionUpdateInput>,
): ({ versionsCreate?: VersionCreateInput[], versionsUpdate?: VersionUpdateInput[] }) {
    // Return empty object if no updated version data. We don't handle deletes here
    if (!updatedRoot.versions) return {};
    const originalVersions = originalRoot.versions ?? [];
    // Find every version in the updated root that is not in the original root (using the version's id)
    const newVersions = updatedRoot.versions.filter((v) => !originalVersions.find((ov) => ov.id === v.id));
    // Versions in the originalRoot and the updatedRoot, which are different from each other, are updates
    const updatedVersions = updatedRoot.versions.filter((v) => originalVersions.some((ov) => ov.id === v.id && JSON.stringify(ov) !== JSON.stringify(v)));
    // Shape and return version data
    const result: { versionsCreate?: VersionCreateInput[], versionsUpdate?: VersionUpdateInput[] } = {};
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
}

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
export function updatePrims<T, K extends keyof T, PK extends keyof T>(
    original: T | null | undefined,
    updated: T | null | undefined,
    primary: PK | null,
    ...fields: (K | [K, (val: any) => any])[]
): UpdatePrimsResult<T, K, PK> {
    function getPrimaryValue(): string | undefined {
        if (!primary) return undefined;
        const primaryValue = original?.[primary] ?? updated?.[primary];
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

    // Return changed fields and primary key
    return { ...changedFields, [primary]: primaryValue } as UpdatePrimsResult<T, K, PK>;
}

type UpdateTranslationPrimsResult<T, K extends keyof T, PK extends keyof T> = ({ [F in K]: Exclude<T[F], null | undefined> } & { [F in PK]: T[F] } & { language: string });

/**
 * Like updatePrims, but forces "language" field to be included in the return type 
 * (unless the result is empty)
 * @param original The original object
 * @param updated The updated object
 * @param primary The primary key of the object, which is always returned
 * @param fields The fields to check, which are returned if they are different
 */
export function updateTranslationPrims<T extends { language: string }, K extends keyof T, PK extends keyof T>(
    original: T | null | undefined,
    updated: T | null | undefined,
    primary: PK | null,
    ...fields: (K | [K, (val: any) => any])[]
): UpdateTranslationPrimsResult<T, K, PK> {
    const updatePrimsResult = updatePrims(original, updated, primary, ...fields);
    // If the result is empty or only has the primary key, return empty object
    if (Object.keys(updatePrimsResult).length <= 1) return {} as UpdateTranslationPrimsResult<T, K, PK>;
    return {
        ...updatePrimsResult,
        language: original?.language ?? updated?.language ?? "en",
    } as UpdateTranslationPrimsResult<T, K, PK>;
}

/**
 * Helper function for formatting an object for an update mutation
 * @param updated The updated object
 * @param shape Shapes the updated object for the update mutation
 */
export function shapeUpdate<
    Input extends object,
    Output extends object,
>(
    updated: Input | null | undefined,
    shape: Output | (() => Output),
): Output | undefined {
    if (!updated) return undefined;
    let result = typeof shape === "function" ? (shape as (updated?: Input) => Output)(updated) : shape;
    // Remove every value from the result that is undefined
    if (result) result = Object.fromEntries(Object.entries(result).filter(([, value]) => value !== undefined)) as Output;
    return result;
}

function asArray<T>(value: T | T[]): T[] {
    if (Array.isArray(value)) return value;
    return [value];
}

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
export function updateRel<
    Item extends (IsOneToOne extends "one" ?
        { [x in FieldName]?: object | null | undefined } :
        { [x in FieldName]?: object[] | null | undefined }),
    FieldName extends string,
    RelTypes extends readonly RelationshipTypeForUpdates[],
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
): UpdateRelOutput<IsOneToOne, RelTypes[number], FieldName> {
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
}

function applyPrimaryKey(
    original: object[],
    updated: object[],
    primaryKey: string,
): object[] {
    // Create a lookup map for the original objects based on the primary key.
    const originalMap = new Map<any, any>();
    original.forEach(obj => {
        originalMap.set(obj[primaryKey], obj);
    });

    // Iterate through the updated list and update the id if a match is found.
    updated.forEach(obj => {
        const matchingOriginal = originalMap.get(obj[primaryKey]);
        if (matchingOriginal) {
            (obj as { id?: string }).id = matchingOriginal.id;
        }
    });

    return updated;
}

/**
 * Like updateRel, but specifically for translations. 
 * These are unique by language, so we treat the language as the primary key.
 */
export function updateTransRel<
    Item extends { translations?: object[] | null | undefined },
    Shape extends ShapeModel<any, any, any>,
>(
    original: Item,
    updated: Item,
    shape: Shape,
): UpdateRelOutput<"many", "Create" | "Update" | "Delete", "translations"> {
    applyPrimaryKey((original as { translations?: object[] }).translations ?? [], (updated as { translations?: object[] }).translations ?? [], "language");
    return updateRel(original, updated, "translations", ["Create", "Update", "Delete"], "many", shape);
}

/**
 * Like updateRel, but specifically for tags.
 * These are unique by tag, so we treat the tag as the primary key.
 */
export function updateTagsRel<
    Item extends { tags?: object[] | null | undefined },
    Shape extends ShapeModel<any, any, any>,
>(
    original: Item,
    updated: Item,
    shape: Shape,
): UpdateRelOutput<"many", "Create" | "Update" | "Delete", "tags"> {
    applyPrimaryKey((original as { tags?: object[] }).tags ?? [], (updated as { tags?: object[] }).tags ?? [], "tag");
    return updateRel(original, updated, "tags", ["Create", "Update", "Delete"], "many", shape);
}

/**
 * Prepares date string for the database
 * @param dateStr Date string to shape
 * @param minDate Minimum date allowed
 * @param maxDate Maximum date allowed
 * @returns Shaped date string, or null if date is invalid
 */
export function shapeDate(
    dateStr: string,
    minDate: Date = new Date("2023-01-01"),
    maxDate: Date = new Date("2100-01-01"),
): string | null {
    // Create a new Date object from the local date string
    const date = new Date(dateStr);

    // Check if date is Invalid Date
    if (date.toString() === "Invalid Date") {
        return null;
    }

    // Check if date is before minDate
    if (date < minDate) {
        return null;
    }

    // Check if date is after maxDate
    if (date > maxDate) {
        return null;
    }

    // Return the date string in the format 'YYYY-MM-DDTHH:MM:SS.SSSZ'
    return date.toISOString();
}
