import { JoinMap } from "../models/types";
import { isRelationshipArray, isRelationshipObject } from "./isOfType";
import { PartialGraphQLInfo } from "./types";

/**
 * Idempotent helper function for adding join tables between 
 * many-to-many relationship parents and children
 * @param partialInfo - GraphQL-shaped object
 * @param map - Mapping of many-to-many relationship names to join table names
 */
export const addJoinTables = (partialInfo: PartialGraphQLInfo, map: JoinMap | undefined): any => {
    if (!map) return partialInfo;
    // Create result object
    const result: any = {};
    // Iterate over join map
    for (const [key, value] of Object.entries(map)) {
        // If the key is in the object, 
        if (partialInfo[key]) {
            // Skip if already padded with join table name
            if (isRelationshipArray(partialInfo[key])) {
                if ((partialInfo[key] as any).every((o: any) => isRelationshipObject(o) && Object.keys(o).length === 1 && Object.keys(o)[0] !== "id")) {
                    result[key] = partialInfo[key];
                    continue;
                }
            } else if (isRelationshipObject(partialInfo[key])) {
                if (Object.keys(partialInfo[key] as any).length === 1 && Object.keys(partialInfo[key] as any)[0] !== "id") {
                    result[key] = partialInfo[key];
                    continue;
                }
            }
            // Otherwise, pad with the join table name
            result[key] = { [value]: partialInfo[key] };
        }
    }
    return {
        ...partialInfo,
        ...result,
    };
};
