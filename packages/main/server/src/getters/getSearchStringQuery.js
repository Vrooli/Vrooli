import { getLogic } from ".";
import { isRelationshipObject } from "../builders";
import { SearchStringMap } from "../utils";
const getSearchStringQueryHelper = (queryParams, query) => {
    const where = {};
    for (const [key, value] of Object.entries(query)) {
        if (Array.isArray(value)) {
            where[key] = value.map((v) => {
                if (typeof v === "string" && SearchStringMap[v]) {
                    return SearchStringMap[v](queryParams);
                }
                else if (typeof v === "object") {
                    return getSearchStringQueryHelper(queryParams, v);
                }
                else {
                    return v;
                }
            });
        }
        else if (SearchStringMap[key]) {
            where[key] = SearchStringMap[key](queryParams);
        }
        else if (isRelationshipObject(value)) {
            where[key] = getSearchStringQueryHelper(queryParams, value);
        }
        else if (typeof value === "string" && SearchStringMap[value]) {
            where[key] = SearchStringMap[value](queryParams);
        }
        else {
            where[key] = value;
        }
    }
    return where;
};
export function getSearchStringQuery({ languages, objectType, searchString, }) {
    if (searchString.length === 0)
        return {};
    const { search } = getLogic(["search"], objectType, languages ?? ["en"], "getSearchStringQuery");
    const insensitive = ({ contains: searchString.trim(), mode: "insensitive" });
    return getSearchStringQueryHelper({ insensitive, languages, searchString }, search.searchStringQuery());
}
//# sourceMappingURL=getSearchStringQuery.js.map