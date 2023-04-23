import pkg from "lodash";
import { isRelationshipObject } from "./isRelationshipObject";
const { merge } = pkg;
export const combineSupplements = (data, objectsById) => {
    const result = {};
    for (const [key, value] of Object.entries(data)) {
        if (Array.isArray(value)) {
            result[key] = data[key].map((v) => combineSupplements(v, objectsById));
        }
        else if (isRelationshipObject(value)) {
            result[key] = combineSupplements(value, objectsById);
        }
    }
    return merge(result, objectsById[data.id]);
};
//# sourceMappingURL=combineSupplements.js.map