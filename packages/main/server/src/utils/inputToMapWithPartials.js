import { getLogic } from "../getters";
export const inputToMapWithPartials = (actionType, relMap, object, languages) => {
    const idsByAction = {};
    const idsByType = {};
    const inputsByType = {};
    if (["Create", "Update", "Delete"].includes(actionType)) {
        inputsByType[relMap.__typename] = { Create: [], Update: [], Delete: [], [actionType]: [object] };
    }
    if (["Connect", "Disconnect", "Delete"].includes(actionType)) {
        idsByAction[actionType] = [object];
        idsByType[relMap.__typename] = [object];
        return { idsByAction, idsByType, inputsByType };
    }
    if (actionType !== "Create") {
        const objectId = actionType === "Update" ? object.where.id : object.id;
        idsByAction[actionType] = [objectId];
        idsByType[relMap.__typename] = [objectId];
    }
    Object.keys(relMap).forEach((key) => {
        if (key === "__typename")
            return;
        const relType = relMap[key];
        const processRelationshipAction = (action, relationshipAction, relationshipType) => {
            const { format } = getLogic(["format"], relationshipType, languages, "inputToMapWithPartials loop");
            const childRelMap = format.gqlRelMap;
            if (object.hasOwnProperty(relationshipAction)) {
                const nestedObjects = Array.isArray(object[relationshipAction])
                    ? object[relationshipAction]
                    : [object[relationshipAction]];
                nestedObjects.forEach((nestedObject) => {
                    if (action === "Disconnect" && typeof nestedObject === "boolean") {
                        nestedObject = `${relationshipType}|${key}`;
                    }
                    const { idsByAction: nestedIdsByAction, idsByType: nestedIdsByType, inputsByType: nestedInputsByType } = inputToMapWithPartials(action, childRelMap, nestedObject, languages);
                    Object.assign(idsByAction, nestedIdsByAction);
                    Object.assign(idsByType, nestedIdsByType);
                    Object.assign(inputsByType, nestedInputsByType);
                });
            }
        };
        ["Connect", "Disconnect", "Create", "Update", "Delete"].forEach((action) => {
            if (typeof relType === "object") {
                Object.keys(relType).forEach((unionKey) => {
                    const unionRelType = relType[unionKey];
                    const relationshipAction = `${unionKey}${action}`;
                    processRelationshipAction(action, relationshipAction, unionRelType);
                });
            }
            else {
                const relationshipAction = `${key}${action}`;
                processRelationshipAction(action, relationshipAction, relType);
            }
        });
    });
    return { idsByAction, idsByType, inputsByType };
};
//# sourceMappingURL=inputToMapWithPartials.js.map