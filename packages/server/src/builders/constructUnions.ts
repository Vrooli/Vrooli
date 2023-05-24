import { exists, GqlModelType, isObject } from "@local/shared";
import { GqlRelMap } from "../models/types";

/**
 * Constructs a GraphQL object's relationship fields from database fields. It's the opposite of deconstructRelationships
 * @param data - Data object
 * @param partialInfo - Partial info object
 * @param gqlRelMap - GraphQL relationship map. Typically used for GraphQL-related 
 * operations, but unions in this object are defined with Prisma fields
 * @returns partialInfo object with union fields added
 */
export const constructUnions = <
    GQLObject extends { [x: string]: any },
    PrismaObject extends { [x: string]: any }
>(
    data: { [x: string]: any },
    partialInfo: { [x: string]: any },
    gqlRelMap: GqlRelMap<GQLObject, PrismaObject>,
): { data: { [x: string]: any }, partialInfo: { [x: string]: any } } => {
    // Create result objects
    const resultData: { [x: string]: any } = data;
    const resultPartialInfo: { [x: string]: any } = { ...partialInfo };
    // Any value in the gqlRelMap which is an object is a union.
    // All other values can be ignored.
    const unionFields: [string, { [x: string]: GqlModelType }][] = Object.entries(gqlRelMap).filter(([, value]) => typeof value === "object") as any[];
    // For each union field
    for (const [gqlField, unionData] of unionFields) {
        // For each entry in the union
        for (const [dbField, type] of Object.entries(unionData)) {
            // If the current field is in the partial info, use it as the union data
            const isInPartialInfo = exists(resultData[dbField]);
            if (isInPartialInfo) {
                // Set the union field to the type
                resultData[gqlField] = { ...resultData[dbField], __typename: type };
                // If the union hasn't been converted in partialInfo, convert it
                if (isObject(resultPartialInfo[gqlField]) && isObject(resultPartialInfo[gqlField][type])) {
                    resultPartialInfo[gqlField] = resultPartialInfo[gqlField][type];
                }
            }
            // Delete the dbField from resultData
            delete resultData[dbField];
        }
        // If no union data was found, set the union field to null
        if (!exists(resultData[gqlField])) {
            resultData[gqlField] = null;
        }
    }
    return { data: resultData, partialInfo: resultPartialInfo };
};
