import { getValidator } from "../getters";
import { GraphQLModelType } from "../models/types";

/**
 * Given a list of fields and GraphQLModels which have validators, creates a Prisma 
 * select object that combines all of the select objects from the validators
 */
export const permissionsSelectHelper = <PrismaSelect extends { [x: string]: any }>(
    list: [keyof PrismaSelect, GraphQLModelType][],
    userId: string | null,
    languages: string[]
): PrismaSelect => {
    // Initialize result
    const result: Partial<PrismaSelect> = {};
    // Iterate through list
    for (const [field, model] of list) {
        // Get validator
        const validator = getValidator(model, languages, 'permissionsSelectHelper');
        result[field] = { select: validator.permissionsSelect(userId, languages) } as any;
    }
    return result as PrismaSelect;
}