import { RelationshipMap } from "../models/types";
import { isRelationshipObject } from "./isRelationshipObject";
// TODO error in here somewhere. Not deconstructing unions properly
/**
 * Deconstructs a GraphQL object's relationship fields into database fields. It's the opposite of constructRelationships
 * @param data - GraphQL-shaped object
 * @param relationshipMap - Mapping of relationship names to their transform shapes
 * @returns DB-shaped object
 */
export const deconstructRelationships = <GraphQLModel>(data: { [x: string]: any }, relationshipMap: RelationshipMap<GraphQLModel>): { [x: string]: any } => {
    // Create result object
    let result: { [x: string]: any } = data;
    // Filter out all fields in the relationshipMap that don't have an object value
    const relationshipFields: [string, { [key: string]: any }][] = Object.entries(relationshipMap).filter(([key, value]) => isRelationshipObject(value)) as any[];
    // For each relationship field
    for (const [key, value] of relationshipFields) {
        // If it's not in data, continue
        if (!data[key]) continue;
        // Get data in union field
        let unionData = data[key];
        // Remove the union field from the result
        delete result[key];
        // If not an object, skip
        if (!isRelationshipObject(unionData)) continue;
        // Determine if data should be wrapped in a "root" field
        const isWrapped = Object.keys(value).length === 1 && Object.keys(value)[0] === 'root';
        const unionMap: { [key: string]: string } = isWrapped ? value.root : value;
        // unionMap is an object where the keys are possible types of the union object, and values are the db field associated with that type
        // Iterate over the possible types
        for (const [type, dbField] of Object.entries(unionMap)) {
            // If the type is in the union data, add the db field to the result. 
            // Don't forget to handle "root" field
            if (unionData[type]) {
                if (isWrapped) {
                    result.root = isRelationshipObject(result.root) ? { ...result.root, [dbField]: unionData[type] } : { [dbField]: unionData[type] };
                } else {
                    result[dbField] = unionData[type];
                }
            }
        }
    }
    return result;
}