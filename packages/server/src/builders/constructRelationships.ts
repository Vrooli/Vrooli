import { GraphQLModelType, RelationshipMap } from "../models/types";

/**
 * Constructs a GraphQL object's relationship fields from database fields. It's the opposite of deconstructRelationships
 * @param partialInfo - Partial info object
 * @param relationshipMap - Mapping of GraphQL union field names to Prisma object field names
 * @returns partialInfo object with union fields added
 */
export const constructRelationships = <GraphQLModel>(partialInfo: { [x: string]: any }, relationshipMap: RelationshipMap<GraphQLModel>): { [x: string]: any } => {
    // Create result object
    let result: { [x: string]: any } = partialInfo;
    // Any value in the relationshipMap which is an array is a union. 
    // All other values can be ignored.
    const unionFields: [string, GraphQLModelType[]][] = Object.entries(relationshipMap).filter(([_, value]) => Array.isArray(value)) as any[];
    // For each union field
    for (const [key, value] of unionFields) {
        // For each GraphQL type in the union
        for (const type of value) {
            // If the type is in the partialInfo
            const isInPartialInfo = result[type] !== undefined;
            if (isInPartialInfo) {
                // Set the union field to the type
                result[key] = result[type];
                // Delete the type from the result
                delete result[type];
            }
        }
    }
    return result;
}