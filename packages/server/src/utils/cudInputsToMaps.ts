import { GqlModelType } from "@shared/consts";
import pkg from 'lodash';
import { PrismaUpdate } from "../builders/types";
import { getLogic } from "../getters";
import { PrismaType } from "../types";
import { convertPlaceholders } from "./convertPlaceholders";
import { inputToMapWithPartials } from "./inputToMapWithPartials";
import { IdsByAction, IdsByType, InputsByType, QueryAction } from "./types";
const { merge } = pkg;

/**
 * Calls objectToIds on each object in an array, and merges the results. Then, 
 * creates a new inputsByType object
 * @returns Three maps. One to map types to ids, one to map actions to ids, and one to map ids to inputs
 */
export const cudInputsToMaps = async ({
    createMany,
    updateMany,
    deleteMany,
    objectType,
    prisma,
    languages,
}: {
    createMany: { [x: string]: any }[] | null | undefined,
    deleteMany: string[] | null | undefined,
    updateMany: {
        where: { [x: string]: any },
        data: { [x: string]: any },
    }[] | null | undefined,
    objectType: `${GqlModelType}`,
    prisma: PrismaType,
    languages: string[]
}): Promise<{
    idsByAction: IdsByAction,
    idsByType: IdsByType,
    inputsByType: InputsByType,
}> => {
    // Initialize return objects
    let idsByType: IdsByType = {};
    let idsByAction: IdsByAction = {};
    let inputsByType: InputsByType = {};
    // Find validator for this (root) object type
    const { format } = getLogic(['format'], objectType, languages, 'getAuthenticatedIds');
    // Combine createMany, updateMany, and deleteMany into one array, with an action property
    const manyList = [
        ...((createMany ?? []).map(data => ({ actionType: 'Create', data }))),
        ...((updateMany ?? []).map(data => ({ actionType: 'Update', data }))),
        ...((deleteMany ?? []).map(id => ({ actionType: 'Delete', data: id }))),
    ]
    // Filter out objects that are strings (i.e. Deletes), since the rest of the function only works with objects
    const inputs = manyList.filter(object => {
        const isString = typeof object.data === 'string';
        // If string, add to idsByType and idsByAction. Deletes don't have an input, so we can ignore the inputsByType map
        if (isString) {
            idsByType[objectType] = [...(idsByType[objectType] ?? []), object.data as string];
            idsByAction['Delete'] = [...(idsByAction['Delete'] ?? []), object.data as string];
        }
        return !isString;
    });
    // For each input object
    inputs.forEach(object => {
        // Call objectToIds to get ids of all objects requiring authentication. 
        // For implicit IDs (see function for explanation), this will return placeholders
        // that we can use to query for the actual ids.
        const {
            idsByAction: childIdsByAction,
            idsByType: childIdsByType,
            inputsByType: childInputsByType,
        } = inputToMapWithPartials(object.actionType as QueryAction, format.prismaRelMap, object.data as PrismaUpdate, languages);
        // Merge idsByAction and idsByType with childIdsByAction and childIdsByType
        idsByAction = merge(idsByAction, childIdsByAction);
        idsByType = merge(idsByType, childIdsByType);
        inputsByType = merge(inputsByType, childInputsByType);
        // Add input to inputsByType
        if (!inputsByType[objectType]) inputsByType[objectType] = { Create: [], Update: [], Delete: [] };
        inputsByType[objectType]![object.actionType].push(object.data);
    });
    // Remove placeholder ids from idsByType and idsByAction
    const withoutPlaceholders = await convertPlaceholders({
        idsByAction,
        idsByType,
        prisma,
        languages,
    });
    // Return the three maps
    return {
        idsByType: withoutPlaceholders.idsByType,
        idsByAction: withoutPlaceholders.idsByAction,
        inputsByType,
    };
}