import { RelationshipMap } from "../models/types";
import { isRelationshipObject } from "./isRelationshipObject";

/**
 * Constructs a GraphQL object's relationship fields from database fields. It's the opposite of deconstructRelationshipsHelper
 * @param partialInfo - Partial info object
 * @param relationshipMap - Mapping of GraphQL union field names to Prisma object field names
 * @returns partialInfo object with union fields added
 */
export const constructRelationshipsHelper = <GraphQLModel>(partialInfo: { [x: string]: any }, relationshipMap: RelationshipMap<GraphQLModel>): { [x: string]: any } => {
    // Create result object
    let result: { [x: string]: any } = partialInfo;
    // Filter out all fields in the relationshipMap that don't have an object value
    const relationshipFields: [string, { [key: string]: any }][] = Object.entries(relationshipMap).filter(([key, value]) => isRelationshipObject(value)) as any[];
    // For each relationship field
    for (const [key, value] of relationshipFields) {
        // Determine if data should be unwrapped from a "root" field
        const isWrapped = Object.keys(value).length === 1 && Object.keys(value)[0] === 'root';
        const unionMap: { [key: string]: string } = isWrapped ? value.root : value;
        // For each type, dbField pair
        for (const [_, dbField] of Object.entries(unionMap)) {
            // If the dbField is in the partialInfo
            const isInPartialInfo = isWrapped ? result.root && result.root[dbField] !== undefined : result[dbField] !== undefined;
            if (isInPartialInfo) {
                // Set the union field to the dbField
                if (isWrapped) {
                    result.root = isRelationshipObject(result.root) ? { ...result.root, [key]: result.root[dbField] } : { [key]: result.root[dbField] };
                } else {
                    result[key] = result[dbField];
                }
                // Delete the dbField from the result
                if (isWrapped) {
                    delete result.root[dbField];
                } else {
                    delete result[dbField];
                }
            }
        }
    }
    return result;
}