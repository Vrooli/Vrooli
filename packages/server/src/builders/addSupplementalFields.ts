import { ObjectMap } from "../models";
import { Formatter } from "../models/types";
import { GqlModelType, SessionUser } from '@shared/consts';
import { PrismaType, SingleOrArray } from "../types";
import { addSupplementalFieldsHelper } from "./addSupplementalFieldsHelper";
import { combineSupplements } from "./combineSupplements";
import { groupIdsByType } from "./groupIdsByType";
import { pickObjectById } from "./pickObjectById";
import { PartialGraphQLInfo } from "./types";
import pkg from 'lodash';
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
    console.log('addSupplementalFields start');
    if (data.length === 0) return [];
    // Group data IDs and select fields by type. This is needed to reduce the number of times 
    // the database is called, as we can query all objects of the same type at once
    let objectIdsDict: { [x: string]: string[] } = {};
    let selectFieldsDict: { [x: string]: { [x: string]: any } } = {};
    for (let i = 0; i < data.length; i++) {
        console.log('addSupplementalFields loop a', i, data.length)
        const currData = data[i];
        const currPartialInfo = Array.isArray(partialInfo) ? partialInfo[i] : partialInfo;
        if (!currData || !currPartialInfo) continue;
        const [childObjectIdsDict, childSelectFieldsDict] = groupIdsByType(currData, currPartialInfo);
        console.log('addSupplementalFields loop b')
        // Merge each array in childObjectIdsDict into objectIdsDict
        for (const [childType, childObjects] of Object.entries(childObjectIdsDict)) {
            objectIdsDict[childType] = objectIdsDict[childType] ?? [];
            objectIdsDict[childType].push(...childObjects);
        }
        console.log('addSupplementalFields loop c')
        // Merge each object in childSelectFieldsDict into selectFieldsDict
        selectFieldsDict = merge(selectFieldsDict, childSelectFieldsDict);
        console.log('addSupplementalFields loop d')
    }
    console.log('addSupplementalFields 2')
    // Dictionary to store objects by ID, instead of type. This is needed to combineSupplements
    const objectsById: { [x: string]: any } = {};

    // Loop through each type in objectIdsDict
    for (const [type, ids] of Object.entries(objectIdsDict)) {
        console.log('addSupplementalFields loop2 a', type, ids)
        // Find the data for each id in ids. Since the data parameter is an array,
        // we must loop through each element in it and call pickObjectById
        const objectData = ids.map((id: string) => pickObjectById(data, id));
        console.log('addSupplementalFields loop2 b')
        // Now that we have the data for each object, we can add the supplemental fields
        const formatter: Formatter<any, any> | undefined = typeof type === 'string' ? ObjectMap[type as keyof typeof ObjectMap]?.format : undefined;
        console.log('addSupplementalFields loop2 c')
        const valuesWithSupplements = formatter?.supplemental ?
            await addSupplementalFieldsHelper({ languages: userData?.languages ?? ['en'], objects: objectData, objectType: type as GqlModelType, partial: selectFieldsDict[type], prisma, userData }) :
            objectData;
        console.log('addSupplementalFields loop2 d')
        // Add each value to objectsById
        for (const v of valuesWithSupplements) {
            objectsById[v.id] = v;
            // Also add the type to the object, which can be used 
            // by our union resolver to determine which __typename to use
            objectsById[v.id].__typename = type;
        }
        console.log('addSupplementalFields loop2 e')
    }
    console.log('addSupplementalFields 3')
    // Convert objectsById dictionary back into shape of data
    let result = data.map(d => (d === null || d === undefined) ? d : combineSupplements(d, objectsById));
    console.log('addSupplementalFields end')
    return result
}