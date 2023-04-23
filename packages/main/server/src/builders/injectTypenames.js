import { ObjectMap } from "../models";
import { isRelationshipObject } from "./isRelationshipObject";
export const injectTypenames = (select, parentRelationshipMap) => {
    const result = {};
    for (const [selectKey, selectValue] of Object.entries(select)) {
        if (["type", "__typename"].includes(selectKey))
            continue;
        const nestedValue = parentRelationshipMap[selectKey];
        if (typeof selectValue !== "object") {
            result[selectKey] = selectValue;
            continue;
        }
        if (!nestedValue) {
            result[selectKey] = injectTypenames(selectValue, {});
            continue;
        }
        if (typeof nestedValue === "string") {
            if (selectValue && ObjectMap[nestedValue]) {
                result[selectKey] = injectTypenames(selectValue, ObjectMap[nestedValue].format.gqlRelMap);
            }
        }
        else if (isRelationshipObject(nestedValue)) {
            for (const [_, type] of Object.entries(nestedValue)) {
                if (selectValue[type] && ObjectMap[type]) {
                    if (!result[selectKey])
                        result[selectKey] = {};
                    result[selectKey][type] = injectTypenames(selectValue[type], ObjectMap[type].format.gqlRelMap);
                }
            }
        }
    }
    if (parentRelationshipMap.__typename)
        result.__typename = parentRelationshipMap.__typename;
    return result;
};
//# sourceMappingURL=injectTypenames.js.map