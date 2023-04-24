import { GqlModelType } from "@local/shared";
import { GqlRelMap } from "../models/types";
import { isRelationshipObject } from "./isRelationshipObject";

/**
 * Deconstructs a GraphQL object's relationship fields into database fields. It's the opposite of constructRelationships
 * @param data - GraphQL-shaped object
 * @param gqlRelMap - Mapping of relationship names to their transform shapes
 * @returns DB-shaped object
 */
export const deconstructUnions = <
    GQLObject extends { [x: string]: any },
    PrismaObject extends { [x: string]: any }
>(data: { [x: string]: any }, gqlRelMap: GqlRelMap<GQLObject, PrismaObject>): { [x: string]: any } => {
    // Create result object
    const result: { [x: string]: any } = data;
    // Any value in the gqlRelMap which is an object is a union. 
    // All other values can be ignored.
    const unionFields: [string, { [x: string]: GqlModelType }][] = Object.entries(gqlRelMap).filter(([_, value]) => isRelationshipObject(value)) as any[];
    // For each union field
    for (const [key, value] of unionFields) {
        // If it's not in data, continue
        if (!data[key]) continue;
        // Store data from the union field
        const unionData = data[key];
        // Remove the union field from the result
        delete result[key];
        // If not an object, skip
        if (!isRelationshipObject(unionData)) continue;
        // Each value in "value" 
        // Iterate over the possible types
        for (const [prismaField, type] of Object.entries(value)) {
            // If the type is in the union data, add the db field to the result. 
            if (unionData[type]) {
                result[prismaField] = unionData[type];
            }
        }
    }
    return result;
};
