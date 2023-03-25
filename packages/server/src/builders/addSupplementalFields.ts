import { GqlModelType, SessionUser } from '@shared/consts';
import pkg from 'lodash';
import { getLogic } from "../getters";
import { ObjectMap } from "../models";
import { PrismaType, SingleOrArray } from "../types";
import { addSupplementalFieldsHelper } from "./addSupplementalFieldsHelper";
import { combineSupplements } from "./combineSupplements";
import { groupPrismaData } from "./groupPrismaData";
import { PartialGraphQLInfo } from "./types";
const { merge } = pkg;

/**
 * Adds supplemental fields to the select object, and all of its relationships (and their relationships, etc.)
 * Groups objects types together, so database is called only once for each type.
 * @param prisma Prisma client
 * @param userData Requesting user's data
 * @param data Array of GraphQL-shaped data, where each object contains at least an ID
 * @param partialInfo PartialGraphQLInfo object
 * @returns data array with supplemental fields added to each object
 */
export const addSupplementalFields = async (
    prisma: PrismaType,
    userData: SessionUser | null,
    data: ({ [x: string]: any } | null | undefined)[],
    partialInfo: SingleOrArray<PartialGraphQLInfo>,
): Promise<{ [x: string]: any }[]> => {
    if (data.length === 0) return [];
    // Group data into dictionaries, which will make later operations easier
    let { objectTypesIdsDict, selectFieldsDict, objectIdsDataDict } = groupPrismaData(data, partialInfo);
    // Dictionary to store supplemental data
    const supplementsByObjectId: { [x: string]: any } = {};

    // Loop through each type in objectTypesIdsDict
    for (const [type, ids] of Object.entries(objectTypesIdsDict)) {
        // Find the supplemental data for each object id in ids
        const objectData = ids.map((id: string) => objectIdsDataDict[id]);
        const { format } = getLogic(['format'], type as keyof typeof ObjectMap, userData?.languages ?? ['en'], 'addSupplementalFields');
        const valuesWithSupplements = format?.supplemental ?
            await addSupplementalFieldsHelper({ languages: userData?.languages ?? ['en'], objects: objectData, objectType: type as GqlModelType, partial: selectFieldsDict[type], prisma, userData }) :
            objectData;
        // Supplements are calculated for an array of objects, so we must loop through 
        // Add each value to supplementsByObjectId
        for (const v of valuesWithSupplements) {
            supplementsByObjectId[v.id] = v;
            // Also add the type to the object, which can be used 
            // by our union resolver to determine which __typename to use
            supplementsByObjectId[v.id].__typename = type;
        }
    }
    // Convert supplementsByObjectId dictionary back into shape of data
    let result = data.map(d => (d === null || d === undefined) ? d : combineSupplements(d, supplementsByObjectId));
    return result
}