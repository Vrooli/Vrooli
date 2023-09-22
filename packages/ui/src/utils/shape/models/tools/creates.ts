import { DUMMY_ID, uuid } from "@local/shared";
import { ShapeModel } from "types";

type OwnerPrefix = "" | "ownedBy";
type OwnerType = "User" | "Organization";

type RelationshipType = "Connect" | "Create";

// Array if isOneToOne is false, otherwise single
type MaybeArray<T extends "object" | "id", IsOneToOne extends "one" | "many"> =
    T extends "object" ?
    IsOneToOne extends "one" ? any : any[] :
    IsOneToOne extends "one" ? string : string[]

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
export const createOwner = <
    OType extends OwnerType,
    Item extends { owner?: { __typename: OType, id: string } | null | undefined },
    Prefix extends OwnerPrefix & string
>(
    item: Item,
    prefix: Prefix = "" as Prefix,
): { [K in `${Prefix}${OType}Connect`]?: string } => {
    // Find owner data in item
    const ownerData = item.owner;
    // If owner data is undefined, or type is not a User or Organization return empty object
    if (ownerData === null || ownerData === undefined || (ownerData.__typename !== "User" && ownerData.__typename !== "Organization")) return {};
    // Create field name (with first letter lowercase)
    let fieldName = `${prefix}${ownerData.__typename}Connect`;
    fieldName = fieldName.charAt(0).toLowerCase() + fieldName.slice(1);
    // Return shaped field
    return { [fieldName]: ownerData.id } as any;
};

/**
 * Shapes versioning relation data for a GraphQL create input
 * @param root The root object, which contains the version data
 * @param shape The version's shape object
 * @returns The shaped object, ready to be passed to the mutation endpoint
 */
export const createVersion = <
    Root extends { id: string, versions?: Record<string, unknown>[] | null | undefined },
    VersionCreateInput extends object,
>(
    root: Root,
    shape: ShapeModel<any, VersionCreateInput, null>,
): ({ versionsCreate?: VersionCreateInput[] }) => {
    // Return empty object if no version data
    if (!root.versions) return {};
    // Shape and return version data, injecting the root ID
    return {
        versionsCreate: root.versions.map((version) => shape.create({
            ...version,
            root: { id: root.id },
        })) as VersionCreateInput[],
    };
};

/**
 * Helper function for setting a list of primitive fields of a create 
 * shape. Essentially, adds every field that's defined, and converts nulls to 
 * undefined
 * 
 * NOTE 1: Due to TypeScript limitations, return type assumes that every field 
 * will be defined, even if it's undefined.
 * 
 * NOTE 2: If you need to reference the object's ID (which may be converted from a dummy ID 
 * to a real ID) in preShape functions, you should use the results from createPrims instead 
 * of relying in the original object.
 */
export const createPrims = <T, K extends keyof T>(
    object: T,
    ...fields: (K | [K, (val: any) => any])[]
): { [F in K]: Exclude<T[F], null | undefined> } => {
    // Create prims
    const prims = fields.reduce((acc, field) => {
        const key = Array.isArray(field) ? field[0] : field;
        const value = Array.isArray(field) ? field[1](object[key]) : object[key];
        return { ...acc, [key]: value !== null ? value : undefined };
    }, {}) as any;

    // If "id" is defined in fields, make sure it's not DUMMY_ID
    if (!fields.some(field => Array.isArray(field) ? field[0] === "id" : field === "id")) return prims;

    return { ...prims, id: prims.id === DUMMY_ID ? uuid() : prims.id };
};

/**
 * Helper function that checks if an object only has data for connecting
 * @param data The object to check
 * @returns True if the object does not contain any data other than IDs and __typename
 */
export const hasOnlyConnectData = (data: object) => {
    const validKeys = Object.keys(data).filter(key => data[key] !== undefined);
    return validKeys.every(k => ["id", "__typename"].includes(k));
};

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
export const createRel = <
    Item extends (IsOneToOne extends "one" ?
        { [x in FieldName]?: object | null | undefined } :
        { [x in FieldName]?: object[] | null | undefined }),
    FieldName extends string,
    RelTypes extends readonly RelationshipType[],
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
): CreateRelOutput<IsOneToOne, RelTypes[number], FieldName> => {
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
            // If create is an option, ignore items which have more than just connect data, since they must be creates instead
            let filteredRelationData = Array.isArray(shapedRelationData) ? shapedRelationData : [shapedRelationData];
            if (relTypes.includes("Create")) {
                filteredRelationData = filteredRelationData.filter((x) => hasOnlyConnectData(x));
            }
            if (filteredRelationData.length === 0) continue;
            result[`${relation}${t}`] = isOneToOne === "one" ?
                filteredRelationData[0][shape?.idField ?? "id"] :
                (filteredRelationData as Array<object>).map((x) => x[shape?.idField ?? "id"]);
        }
        else if (t === "Create") {
            // Ignore items which only have connect data
            let filteredRelationData = Array.isArray(shapedRelationData) ? shapedRelationData : [shapedRelationData];
            filteredRelationData = filteredRelationData.filter((x) => !hasOnlyConnectData(x));
            if (filteredRelationData.length === 0) continue;
            result[`${relation}${t}`] = isOneToOne === "one" ?
                shape!.create(filteredRelationData[0]) :
                (filteredRelationData as any).map((x: any) => shape!.create(x));
        }
    }
    // Return result
    return result as CreateRelOutput<IsOneToOne, RelTypes[number], FieldName>;
};
