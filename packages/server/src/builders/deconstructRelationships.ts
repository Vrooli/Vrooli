import { GraphQLModelType, RelationshipMap } from "../models/types";
import { isRelationshipObject } from "./isRelationshipObject";

/**
 * Deconstructs a GraphQL object's relationship fields into database fields. It's the opposite of constructRelationships
 * @param data - GraphQL-shaped object
 * @param relationshipMap - Mapping of relationship names to their transform shapes
 * @returns DB-shaped object
 */
export const deconstructRelationships = <GraphQLModel>(data: { [x: string]: any }, relationshipMap: RelationshipMap<GraphQLModel>): { [x: string]: any } => {
    // Create result object
    let result: { [x: string]: any } = data;
    // Any value in the relationshipMap which is an array is a union. 
    // All other values can be ignored.
    const unionFields: [string, GraphQLModelType[]][] = Object.entries(relationshipMap).filter(([_, value]) => Array.isArray(value)) as any[];
    // For each union field
    for (const [key, value] of unionFields) {
        // If it's not in data, continue
        if (!data[key]) continue;
        // Store data from the union field
        let unionData = data[key];
        // Remove the union field from the result
        delete result[key];
        // If not an object, skip
        if (!isRelationshipObject(unionData)) continue;
        // value is an array of possible types of the union object
        // Iterate over the possible types
        for (const type of value) {
            // If the type is in the union data, add the db field to the result. 
            if (unionData[type]) {
                result[type] = unionData[type];
            }
        }
    }
    return result;
}