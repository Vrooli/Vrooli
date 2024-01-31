import { exists, GqlModelType, isObject } from "@local/shared";
import { GqlRelMap, ModelLogicType } from "../models/types";
import { isRelationshipObject } from "./isOfType";

/**
 * Constructs a GraphQL object's relationship fields from database fields. It's the opposite of deconstructRelationships
 * @param data - Data object
 * @param partialInfo - Partial info object
 * @param gqlRelMap - GraphQL relationship map. Typically used for GraphQL-related 
 * operations, but unions in this object are defined with Prisma fields
 * @returns partialInfo object with union fields added
 */
export const constructUnions = <
    Typename extends `${GqlModelType}`,
    GQLModel extends ModelLogicType["GqlModel"],
    PrismaModel extends ModelLogicType["PrismaModel"],
>(
    data: { [x: string]: any },
    partialInfo: { [x: string]: any },
    gqlRelMap: GqlRelMap<Typename, GQLModel, PrismaModel>,
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

/**
 * Deconstructs a GraphQL object's relationship fields into database fields. It's the opposite of constructRelationships
 * @param data - GraphQL-shaped object
 * @param gqlRelMap - Mapping of relationship names to their transform shapes
 * @returns DB-shaped object
 */
export const deconstructUnions = <
    Typename extends `${GqlModelType}`,
    GQLModel extends ModelLogicType["GqlModel"],
    PrismaModel extends ModelLogicType["PrismaModel"],
>(data: { [x: string]: any }, gqlRelMap: GqlRelMap<Typename, GQLModel, PrismaModel>): { [x: string]: any } => {
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
