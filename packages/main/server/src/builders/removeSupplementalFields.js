import { omit } from "@local/utils";
import pkg from "lodash";
import { ObjectMap } from "../models";
const { merge } = pkg;
export const removeSupplementalFields = (objectType, partial) => {
    const supplementer = ObjectMap[objectType]?.format?.supplemental;
    if (!supplementer)
        return partial;
    const withoutGqlSupp = omit(partial, supplementer.graphqlFields);
    if (supplementer.dbFields) {
        const dbSupp = supplementer.dbFields.reduce((acc, curr) => {
            acc[curr] = true;
            return acc;
        }, {});
        return merge(withoutGqlSupp, dbSupp);
    }
    return withoutGqlSupp;
};
//# sourceMappingURL=removeSupplementalFields.js.map