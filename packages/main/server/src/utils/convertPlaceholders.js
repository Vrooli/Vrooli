import { getLogic } from "../getters";
export const convertPlaceholders = async ({ idsByAction, idsByType, prisma, languages, }) => {
    const updatedIdsByAction = { ...idsByAction };
    const updatedIdsByType = { ...idsByType };
    for (const actionType in idsByAction) {
        const ids = idsByAction[actionType];
        const updatedIds = [];
        for (const id of ids) {
            if (typeof id === "string" && id.includes("|")) {
                const [objectType, path] = id.split("|", 2);
                const [rootId, ...relations] = path.split(".");
                const { delegate } = getLogic(["delegate"], objectType, languages, "convertPlaceholders");
                const queryResult = await delegate(prisma).findUnique({
                    where: { id: rootId },
                    select: relations.reduce((selectObj, relation) => {
                        return { [relation]: selectObj };
                    }, {}),
                });
                let currentObject = queryResult;
                for (const relation of relations) {
                    currentObject = currentObject[relation];
                }
                if (currentObject && currentObject.id) {
                    updatedIds.push(currentObject.id);
                }
            }
            else {
                updatedIds.push(id);
            }
        }
        updatedIdsByAction[actionType] = updatedIds;
    }
    for (const objectType in idsByType) {
        const ids = idsByType[objectType];
        const updatedIds = [];
        for (const id of ids) {
            if (typeof id === "string" && id.includes("|")) {
                const [rootId, ...relations] = id.split("|")[1].split(".");
                const queryResult = await prisma[objectType].findUnique({
                    where: { id: rootId },
                    select: relations.reduce((selectObj, relation) => {
                        return { [relation]: selectObj };
                    }, {}),
                });
                let currentObject = queryResult;
                for (const relation of relations) {
                    currentObject = currentObject[relation];
                }
                if (currentObject && currentObject.id) {
                    updatedIds.push(currentObject.id);
                }
            }
            else {
                updatedIds.push(id);
            }
        }
        updatedIdsByType[objectType] = updatedIds;
    }
    return {
        idsByAction: updatedIdsByAction,
        idsByType: updatedIdsByType,
    };
};
//# sourceMappingURL=convertPlaceholders.js.map