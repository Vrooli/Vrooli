import { ShapeModel } from 'types';

type RelationshipType = 'Connect' | 'Create';

// Array if isOneToOne is false, otherwise single
type MaybeArray<T extends 'object' | 'id', IsOneToOne extends 'one' | 'many'> =
    T extends 'object' ?
    IsOneToOne extends 'one' ? any : any[] :
    IsOneToOne extends 'one' ? string : string[]

type CreateRelOutput<
    IsOneToOne extends 'one' | 'many',
    RelTypes extends string,
    FieldName extends string,
> = (
        ({ [x in `${FieldName}Connect`]: 'Connect' extends RelTypes ? MaybeArray<'id', IsOneToOne> : never }) &
        ({ [x in `${FieldName}Create`]: 'Create' extends RelTypes ? MaybeArray<'object', IsOneToOne> : never })
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
    Item extends (IsOneToOne extends 'one' ?
        { [x in FieldName]?: {} | null | undefined } :
        { [x in FieldName]?: {}[] | null | undefined }),
    FieldName extends string,
    RelTypes extends readonly RelationshipType[],
    // Shape object only required when RelTypes includes 'Create' or 'Update'
    Shape extends ('Create' extends RelTypes[number] ?
        ShapeModel<any, {}, null> :
        never),
    IsOneToOne extends 'one' | 'many',
>(
    item: Item,
    relation: FieldName,
    relTypes: RelTypes,
    isOneToOne: IsOneToOne,
    shape?: Shape,
    preShape?: (item: any) => any,
): CreateRelOutput<IsOneToOne, RelTypes[number], FieldName> => {
    // Check if shape is required
    if (relTypes.includes('Create')) {
        if (!shape) throw new Error('Model is required if relTypes includes "Create"');
    }
    // Find relation data in item
    const relationData = item[relation];
    // If relation data is undefined, return empty object
    if (relationData === undefined) return {} as any;
    // Initialize result
    const result: { [x: string]: any } = {};
    // Make preShape a function, if not provided 
    const preShaper = preShape ?? ((x: any) => x);
    // Loop through relation types
    for (const t of relTypes) {
        // If type is connect, add IDs to result
        if (t === 'Connect') {
            result[`${relation}${t}`] = isOneToOne === 'one' ?
                (relationData as any)[shape!.idField ?? 'id'] :
                (relationData as any).map((x: any) => x[shape!.idField ?? 'id']);
        }
        else if (t === 'Create') {
            // Ignore items which only have an ID, since they must be connects instead
            let filteredRelationData = Array.isArray(relationData) ? relationData : [relationData];
            filteredRelationData = filteredRelationData.filter((x) => Object.values(x).length > 1);
            if (filteredRelationData.length === 0) continue;
            result[`${relation}${t}`] = isOneToOne === 'one' ?
                shape!.create(preShaper(relationData)) :
                (relationData as any).map((x: any) => shape!.create(preShaper(x)));
        }
    }
    // Return result
    return result as any;
};