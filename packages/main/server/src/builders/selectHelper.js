import { isObject } from "@local/utils";
import { removeTypenames } from "./removeTypenames";
import { selPad } from "./selPad";
import { toPartialPrismaSelect } from "./toPartialPrismaSelect";
export const selectHelper = (partial) => {
    let modified = toPartialPrismaSelect(partial);
    if (!isObject(modified))
        return undefined;
    modified = removeTypenames(modified);
    modified = selPad(modified);
    return modified;
};
//# sourceMappingURL=selectHelper.js.map