import { getLogic } from "../getters";
import { GraphQLModelType } from "../models/types";

/**
 * Given an object of fields mapped to GraphQLModelTypes, creates a Prisma 
 * select object that combines all of the select objects from each type's validators
 */
export const permissionsSelectHelper = <PrismaSelect extends { [x: string]: any }>(
    map: { [key in keyof PrismaSelect]?: GraphQLModelType },
    userId: string | null,
    languages: string[]
): { [key in keyof PrismaSelect]?: { select: any } } => {
    // Initialize result
    const result: { [key in keyof PrismaSelect]?: { select: any } } = {};
    // Iterate through map
    for (const [field, type] of Object.entries(map)) {
        if (!type) continue;
        // Get validator
        const { validate } = getLogic(['validate'], type, languages, 'permissionsSelectHelper');
        result[field as keyof PrismaSelect] = { select: validate.permissionsSelect(userId, languages) };
    }
    return result as PrismaSelect;
}