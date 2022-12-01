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
    const supplementer: SupplementalConverter<any, any> | undefined = ObjectMap[objectType]?.format?.supplemental;
    if (!supplementer) return partial;
    // Remove graphQL supplemental fields
    const withoutGqlSupp = omit(partial, supplementer.graphqlFields);
    // Add db supplemental fields
    const withDbSupp = merge(withoutGqlSupp, supplementer.dbFields);
    // Return result
    return withDbSupp;
}