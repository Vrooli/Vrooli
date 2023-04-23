import * as yup from "yup";
import { rel } from "./rel";
export const yupObj = (fields, rels, excludePairs, omitRels) => {
    let relObjects = {};
    rels.forEach((params) => {
        if (omitRels && (typeof omitRels === "string" ? params[0] === omitRels : omitRels.includes(params[0]))) {
            return;
        }
        relObjects = { ...relObjects, ...rel(...params) };
    });
    const obj = yup.object().shape({
        ...fields,
        ...relObjects,
    }, excludePairs);
    return obj;
};
//# sourceMappingURL=yupObj.js.map