import { ObjectMap } from "../models";
import { SessionUser } from '@shared/consts';
import { PrismaType, RecursivePartial } from "../types";
import { PartialGraphQLInfo } from "./types";
import { GraphQLModelType, SupplementalConverter } from "../models/types";

/**
 * Sets value in object using dot notation
 * @param obj Object to set value in
 * @param path Path to set value in
 * @param value Value to set
 * @returns Object with value set
 */
const setDotNotationValue = (obj: { [x: string]: any }, path: string, value: any): { [x: string]: any } => {
    const keys = path.split('.');
    const lastKey = keys.pop();
    const lastObj = keys.reduce((obj, key) => obj[key] = obj[key] || {}, obj);
    if (lastKey) lastObj[lastKey] = value;
    return obj;
}

/**
 * Adds supplemental fields data to the given objects
 */
export const addSupplementalFieldsHelper = async <GraphQLModel extends { [x: string]: any }>({ languages, objects, objectType, partial, prisma, userData }: {
    languages: string[],
    objects: ({ id: string } & { [x: string]: any })[],
    objectType: GraphQLModelType,
    partial: PartialGraphQLInfo,
    prisma: PrismaType,
    userData: SessionUser | null,
}): Promise<RecursivePartial<GraphQLModel>[]> => {
    if (!objects || objects.length === 0) return [];
    // Get supplemental info for object
    const supplementer: SupplementalConverter<any> | undefined = ObjectMap[objectType]?.format?.supplemental;
    if (!supplementer) return objects;
    // Get IDs from objects
    const ids = objects.map(({ id }) => id);
    // Get supplemental data by field
    const supplementalData = await supplementer.toGraphQL({ ids, languages, objects, partial, prisma, userData });
    // Loop through each field in supplemental data
    for (const [field, data] of Object.entries(supplementalData)) {
        // Each value is an array of data for each object, in the same order as the objects
        // Set the value for each object
        objects = objects.map((x, i) => setDotNotationValue(x, field, data[i])) as any[];
    }
    return objects;
}