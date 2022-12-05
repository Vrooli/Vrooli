import { SessionUser } from "../endpoints/types";
import { PrismaType } from "../types";
import { addSupplementalFields } from "./addSupplementalFields";
import { PartialGraphQLInfo } from "./types";
import pkg from 'lodash';
const { flatten } = pkg;

/**
 * Combines addSupplementalFields calls for multiple object types
 * @param data Array of arrays, where each array is a list of the same object type queried from the database
 * @param partialInfos Array of PartialGraphQLInfo objects, in the same order as data arrays
 * @param keys Keys to associate with each data array
 * @param userData Requesting user's data
 * @param prisma Prisma client
 * @returns Object with keys equal to objectTypes, and values equal to arrays of objects with supplemental fields added
 */
export const addSupplementalFieldsMultiTypes = async (
    data: { [x: string]: any }[][],
    partialInfos: PartialGraphQLInfo[],
    keys: string[],
    userData: SessionUser | null,
    prisma: PrismaType,
): Promise<{ [x: string]: any[] }> => {
    // Flatten data array
    const combinedData = flatten(data);
    // Create an array of partials, that match the data array
    let combinedPartialInfo: PartialGraphQLInfo[] = [];
    for (let i = 0; i < data.length; i++) {
        const currPartial = partialInfos[i];
        // Push partial for each data array
        for (let j = 0; j < data[i].length; j++) {
            combinedPartialInfo.push(currPartial);
        }
    }
    // Call addSupplementalFields
    const combinedResult = await addSupplementalFields(prisma, userData, combinedData, combinedPartialInfo);
    // Convert combinedResult into object with keys equal to objectTypes, and values equal to arrays of those types
    const formatted: { [y: string]: any[] } = {};
    let start = 0;
    for (let i = 0; i < keys.length; i++) {
        const currKey = keys[i];
        const end = start + data[i].length;
        formatted[currKey] = combinedResult.slice(start, end);
        start = end;
    }
    return formatted;
}