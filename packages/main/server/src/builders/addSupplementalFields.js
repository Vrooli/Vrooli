import pkg from "lodash";
import { getLogic } from "../getters";
import { addSupplementalFieldsHelper } from "./addSupplementalFieldsHelper";
import { combineSupplements } from "./combineSupplements";
import { groupPrismaData } from "./groupPrismaData";
const { merge } = pkg;
export const addSupplementalFields = async (prisma, userData, data, partialInfo) => {
    if (data.length === 0)
        return [];
    const { objectTypesIdsDict, selectFieldsDict, objectIdsDataDict } = groupPrismaData(data, partialInfo);
    const supplementsByObjectId = {};
    for (const [type, ids] of Object.entries(objectTypesIdsDict)) {
        const objectData = ids.map((id) => objectIdsDataDict[id]);
        const { format } = getLogic(["format"], type, userData?.languages ?? ["en"], "addSupplementalFields");
        const valuesWithSupplements = format?.supplemental ?
            await addSupplementalFieldsHelper({ languages: userData?.languages ?? ["en"], objects: objectData, objectType: type, partial: selectFieldsDict[type], prisma, userData }) :
            objectData;
        for (const v of valuesWithSupplements) {
            supplementsByObjectId[v.id] = v;
            supplementsByObjectId[v.id].__typename = type;
        }
    }
    const result = data.map(d => (d === null || d === undefined) ? d : combineSupplements(d, supplementsByObjectId));
    return result;
};
//# sourceMappingURL=addSupplementalFields.js.map