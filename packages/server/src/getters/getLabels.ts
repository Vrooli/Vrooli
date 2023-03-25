import { GqlModelType } from "@shared/consts";
import { CustomError, logger } from "../events";
import { ObjectMap } from "../models";
import { PrismaType } from "../types";

/**
 * Finds translated labels for a list of objects
 * @param objects A list of object ids, optionally with preferred languages to find labels for. Can be useful 
 * when a user action triggers a notification for someone else.
 * If languages not defined, uses user's preferred languages
 * @param objectType Type of object to find labels for
 * @param prisma Prisma client
 * @param languages Preferred languages to display errors in
 * @param errorTrace Error trace to display
 */
export async function getLabels(
    objects: { id: string, languages: string[] }[] | string[],
    objectType: `${GqlModelType}`,
    prisma: PrismaType,
    languages: string[],
    errorTrace: string,
): Promise<string[]> {
    const model = ObjectMap[objectType]
    if (!model) {
        throw new CustomError('0347', 'InvalidArgs', languages, { errorTrace, objectType });
    }
    if (objects.length <= 0) return [];
    const objectsWithLanguages = typeof objects[0] === 'string' ? objects.map(id => ({ id, languages })) : objects;
    // Query for labels data
    let where: any;
    let select: any;
    let labelsData: any[];
    try {
        where = { id: { in: objectsWithLanguages.map(x => x.id) } };
        select = typeof model.display.select === 'function' ? model.display.select() : model.display.select,
            labelsData = await model.delegate(prisma).findMany({
                where,
                select,
            })
    } catch (error) {
        logger.error('readManyHelper: Failed to find searchResults', { trace: '0385', error, objectType, where, select });
        throw new CustomError('0387', 'InternalError', languages, { objectType });
    }
    // If no data, return empty strings
    if (!labelsData || labelsData.length <= 0) return new Array(objectsWithLanguages.length).fill('');
    // For each object, find the label
    const labels = objectsWithLanguages.map(object => {
        const data = labelsData.find(x => x.id === object.id);
        if (!data) return '';
        return model.display.label(data, object.languages);
    })
    return labels;
}