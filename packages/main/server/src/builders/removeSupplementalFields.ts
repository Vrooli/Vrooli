import { GqlModelType, omit } from "@local/shared";
import pkg from "lodash";
import { ObjectMap } from "../models";
import { SupplementalConverter } from "../models/types";
import { PartialGraphQLInfo, PartialPrismaSelect } from "./types";

const { merge } = pkg;


/**
 * Removes supplemental fields (i.e. fields that cannot be calculated in the main query), and also may 
 * add additional fields to calculate the supplemental fields
 * @param objectType Type of object to get supplemental fields for
 * @param partial Select fields object
 * @returns partial with supplemental fields removed, and maybe additional fields added
 */
export const removeSupplementalFields = (objectType: `${GqlModelType}`, partial: PartialGraphQLInfo | PartialPrismaSelect) => {
    // Get supplemental info for object
    const supplementer: SupplementalConverter<any> | undefined = ObjectMap[objectType]?.format?.supplemental;
    if (!supplementer) return partial;
    // Remove graphQL supplemental fields
    const withoutGqlSupp = omit(partial, supplementer.graphqlFields);
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
};
