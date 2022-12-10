import { omit } from "@shared/utils";
import { ObjectMap } from "../models";
import { GraphQLModelType, SupplementalConverter } from "../models/types";
import { PartialGraphQLInfo, PartialPrismaSelect } from "./types";
import pkg from 'lodash';
const { merge } = pkg;


/**
 * Removes supplemental fields (i.e. fields that cannot be calculated in the main query), and also may 
 * add additional fields to calculate the supplemental fields
 * @param objectType Type of object to get supplemental fields for
 * @param partial Select fields object
 * @returns partial with supplemental fields removed, and maybe additional fields added
 */
export const removeSupplementalFields = (objectType: GraphQLModelType, partial: PartialGraphQLInfo | PartialPrismaSelect) => {
    // Get supplemental info for object
    const supplementer: SupplementalConverter<any> | undefined = ObjectMap[objectType]?.format?.supplemental;
    if (!supplementer) return partial;
    // Remove graphQL supplemental fields
    console.log('before removeSupplementalFields omit', supplementer.graphqlFields, JSON.stringify(partial), '\n\n');
    const withoutGqlSupp = omit(partial, supplementer.graphqlFields);
    console.log('after removeSupplementalFields omit', JSON.stringify(withoutGqlSupp), '\n\n');
    // Add db supplemental fields
    if (supplementer.dbFields) {
        // For each db supplemental field, add it to the select object with value true
        const dbSupp = supplementer.dbFields.reduce((acc, curr) => {
            acc[curr] = true;
            return acc;
        }, {} as PartialPrismaSelect);
        // Merge db supplemental fields with select object
        return merge(withoutGqlSupp, dbSupp);
    }
    return withoutGqlSupp;
}