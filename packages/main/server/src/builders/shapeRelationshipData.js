import { isObject } from "@local/utils";
import { filterFields } from "./filterFields";
export const shapeRelationshipData = (data, excludes = [], isOneToOne = false) => {
    const shapeAsMany = (data) => {
        if (Array.isArray(data)) {
            return data.map(e => {
                if (isObject(e)) {
                    return filterFields(e, excludes);
                }
                else {
                    return { id: e };
                }
            });
        }
        else if (isObject(data)) {
            return [filterFields(data, excludes)];
        }
        else {
            return [{ id: data }];
        }
    };
    let result = shapeAsMany(data);
    if (isOneToOne) {
        if (result.length > 0) {
            result = result[0];
        }
        else {
            result = {};
        }
    }
    return result;
};
//# sourceMappingURL=shapeRelationshipData.js.map