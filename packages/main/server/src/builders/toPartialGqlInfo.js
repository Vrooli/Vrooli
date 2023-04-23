import { CustomError } from "../events";
import { resolveGraphQLInfo } from "../utils";
import { injectTypenames } from "./injectTypenames";
export const toPartialGqlInfo = (info, gqlRelMap, languages, throwIfNotPartial = false) => {
    if (!info) {
        if (throwIfNotPartial)
            throw new CustomError("0345", "InternalError", languages);
        return undefined;
    }
    let select;
    const isGraphQLResolveInfo = info.hasOwnProperty("fieldNodes") && info.hasOwnProperty("returnType");
    if (isGraphQLResolveInfo) {
        select = resolveGraphQLInfo(JSON.parse(JSON.stringify(info)));
    }
    else {
        select = info;
    }
    if (select.hasOwnProperty("pageInfo") && select.hasOwnProperty("edges")) {
        select = select.edges.node;
    }
    else if (select.hasOwnProperty("endCursor") && select.hasOwnProperty("totalThreads") && select.hasOwnProperty("threads")) {
        select = select.threads.comment;
    }
    select = injectTypenames(select, gqlRelMap);
    if (!select)
        throw new CustomError("0346", "InternalError", languages);
    return select;
};
//# sourceMappingURL=toPartialGqlInfo.js.map