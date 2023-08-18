import { ShapeModel } from "types";

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
    // Helper function that checks if an object only has data for connecting
    const hasOnlyConnectData = (data: object) => {
        const validKeys = Object.keys(data).filter(key => data[key] !== undefined);
        return validKeys.every(k => ["id", "__typename"].includes(k));
    };
    // Loop through relation types
    for (const t of relTypes) {
        // If type is connect, add IDs to result
        if (t === "Connect") {
            // If create is an option, ignore items which have more than just connect data, since they must be creates instead
            let filteredRelationData = Array.isArray(relationData) ? relationData : [relationData];
            if (relTypes.includes("Create")) {
                filteredRelationData = filteredRelationData.filter((x) => hasOnlyConnectData(x));
            }
            if (filteredRelationData.length === 0) continue;
            console.log("adding to result", `${relation}${t}`);
            result[`${relation}${t}`] = isOneToOne === "one" ?
                relationData[shape?.idField ?? "id"] :
                (relationData as Array<object>).map((x) => x[shape?.idField ?? "id"]);
        }
        else if (t === "Create") {
            // Ignore items which only have connect data
            let filteredRelationData = Array.isArray(relationData) ? relationData : [relationData];
            filteredRelationData = filteredRelationData.filter((x) => !hasOnlyConnectData(x));
            if (filteredRelationData.length === 0) continue;
            result[`${relation}${t}`] = isOneToOne === "one" ?
                shape!.create(preShaper(relationData)) :
                (relationData as any).map((x: any) => shape!.create(preShaper(x)));
        }
    }
    // Return result
    return result as CreateRelOutput<IsOneToOne, RelTypes[number], FieldName>;
};
