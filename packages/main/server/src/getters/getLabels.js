import { CustomError, logger } from "../events";
import { ObjectMap } from "../models";
export async function getLabels(objects, objectType, prisma, languages, errorTrace) {
    const model = ObjectMap[objectType];
    if (!model) {
        throw new CustomError("0347", "InvalidArgs", languages, { errorTrace, objectType });
    }
    if (objects.length <= 0)
        return [];
    const objectsWithLanguages = typeof objects[0] === "string" ? objects.map(id => ({ id, languages })) : objects;
    let where;
    let select;
    let labelsData;
    try {
        where = { id: { in: objectsWithLanguages.map(x => x.id) } };
        select = typeof model.display.select === "function" ? model.display.select() : model.display.select;
        labelsData = await model.delegate(prisma).findMany({
            where,
            select,
        });
    }
    catch (error) {
        logger.error("readManyHelper: Failed to find searchResults", { trace: "0385", error, objectType, where, select });
        throw new CustomError("0387", "InternalError", languages, { objectType });
    }
    if (!labelsData || labelsData.length <= 0)
        return new Array(objectsWithLanguages.length).fill("");
    const labels = objectsWithLanguages.map(object => {
        const data = labelsData.find(x => x.id === object.id);
        if (!data)
            return "";
        return model.display.label(data, object.languages);
    });
    return labels;
}
//# sourceMappingURL=getLabels.js.map