import pkg from "lodash";
import { getLogic } from "../getters";
import { convertPlaceholders } from "./convertPlaceholders";
import { inputToMapWithPartials } from "./inputToMapWithPartials";
const { merge } = pkg;
export const cudInputsToMaps = async ({ createMany, updateMany, deleteMany, objectType, prisma, languages, }) => {
    let idsByType = {};
    let idsByAction = {};
    let inputsByType = {};
    const { format } = getLogic(["format"], objectType, languages, "getAuthenticatedIds");
    const manyList = [
        ...((createMany ?? []).map(data => ({ actionType: "Create", data }))),
        ...((updateMany ?? []).map(data => ({ actionType: "Update", data }))),
        ...((deleteMany ?? []).map(id => ({ actionType: "Delete", data: id }))),
    ];
    const inputs = manyList.filter(object => {
        const isString = typeof object.data === "string";
        if (isString) {
            idsByType[objectType] = [...(idsByType[objectType] ?? []), object.data];
            idsByAction["Delete"] = [...(idsByAction["Delete"] ?? []), object.data];
        }
        return !isString;
    });
    inputs.forEach(object => {
        const { idsByAction: childIdsByAction, idsByType: childIdsByType, inputsByType: childInputsByType, } = inputToMapWithPartials(object.actionType, format.prismaRelMap, object.data, languages);
        idsByAction = merge(idsByAction, childIdsByAction);
        idsByType = merge(idsByType, childIdsByType);
        inputsByType = merge(inputsByType, childInputsByType);
    });
    const withoutPlaceholders = await convertPlaceholders({
        idsByAction,
        idsByType,
        prisma,
        languages,
    });
    return {
        idsByType: withoutPlaceholders.idsByType,
        idsByAction: withoutPlaceholders.idsByAction,
        inputsByType,
    };
};
//# sourceMappingURL=cudInputsToMaps.js.map