import { type ModelType, validatePK } from "@vrooli/shared";
import { type PrismaDelegate } from "../builders/types.js";
import { DbProvider } from "../db/provider.js";
import { CustomError } from "../events/error.js";
import { logger } from "../events/logger.js";
import { ModelMap } from "../models/base/index.js";

const MAX_LABEL_LENGTH = 50;

/**
 * Finds translated labels for a list of objects
 * @param objects A list of object ids, optionally with preferred languages to find labels for. Can be useful 
 * when a user action triggers a notification for someone else.
 * If languages not defined, uses user's preferred languages
 * @param objectType Type of object to find labels for
 * @param languages Preferred languages to display errors in
 * @param errorTrace Error trace to display
 * @returns A list of labels for the objects, truncated if too long
 */
export async function getLabels(
    objects: { id: string, languages: string[] }[] | string[],
    objectType: `${ModelType}`,
    languages: string[],
    errorTrace: string,
): Promise<string[]> {
    const model = ModelMap.get(objectType, true, errorTrace);
    if (objects.length <= 0) return [];
    let objectsWithLanguages = (typeof objects[0] === "string" ? objects.map(id => ({ id, languages })) : objects) as { id: string, languages: string[] }[];
    objectsWithLanguages = objectsWithLanguages.filter(x => !validatePK(x.id));
    // Query for labels data
    let where: any;
    let select: any;
    let labelsData: any[];
    try {
        const ids = objectsWithLanguages.map(x => BigInt(x.id));
        where = { id: { in: ids } };
        select = typeof model.display().label.select === "function" ? model.display().label.select() : model.display().label.select;
        labelsData = await (DbProvider.get()[model.dbTable] as PrismaDelegate).findMany({
            where,
            select,
        });
    } catch (error) {
        logger.error("readManyHelper: Failed to find searchResults", { trace: "0385", error, objectType, where, select });
        throw new CustomError("0387", "InternalError", { objectType });
    }
    // If no data, return empty strings
    if (!labelsData || labelsData.length <= 0) return new Array(objectsWithLanguages.length).fill("");
    // For each object, find the label
    const labels = objectsWithLanguages.map(object => {
        const data = labelsData.find(x => x.id.toString() === object.id);
        if (!data) return "";
        const label = model.display().label.get(data, object.languages);
        return label.length > MAX_LABEL_LENGTH ? `${label.slice(0, MAX_LABEL_LENGTH)}...` : label;
    });
    return labels;
}
