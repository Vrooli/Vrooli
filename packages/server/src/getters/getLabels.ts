import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { GraphQLModelType } from "../models/types";
import { PrismaType } from "../types";

/**
 * Finds translated labels for a list of objects
 * @param objects A list of object ids and preferred languages to find labels for
 * @param objectType Type of object to find labels for
 * @param prisma Prisma client
 * @param languages Preferred languages to display errors in
 * @param errorTrace Error trace to display
 */
export async function getLabels(
    objects: { id: string, languages: string[] }[],
    objectType: GraphQLModelType,
    prisma: PrismaType, 
    languages: string[],
    errorTrace: string,
): Promise<string[]> {
    const model = ObjectMap[objectType]
    if (!model) {
        throw new CustomError('0347', 'InvalidArgs', languages, { errorTrace, objectType });
    }
    // Query for labels data
    const labelsData = await model.delegate(prisma).findMany({
        where: { id: { in: objects.map(x => x.id) } },
        select: model.display.select,
    })
    // If no data, return empty strings
    if (!labelsData || labelsData.length <= 0) return new Array(objects.length).fill('');
    // For each object, find the label
    const labels = objects.map(object => {
        const data = labelsData.find(x => x.id === object.id);
        if (!data) return '';
        return model.display.label(data, object.languages);
    })
    return labels;
}