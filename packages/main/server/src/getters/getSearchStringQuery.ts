import { GqlModelType } from ":local/consts";
import { getLogic } from ".";
import { isRelationshipObject } from "../builders";
import { SearchStringQuery, SearchStringQueryParams } from "../models/types";
import { SearchStringMap } from "../utils";

/**
 * @param queryParams Data required to replace keys
 * @param query The query object to convert
 * @returns Fully-converted Prisma query, ready to be passed into prisma.findMany()
 */
const getSearchStringQueryHelper = <Where extends { [x: string]: any }>(
    queryParams: SearchStringQueryParams,
    query: SearchStringQuery<Where>,
): Where => {
    // Loop through [key, value] pairs in query
    const where: Where = {} as Where;
    for (const [key, value] of Object.entries(query)) {
        // If value is an array, recursively convert each element
        if (Array.isArray(value)) {
            where[key as keyof Where] = value.map((v) => {
                if (typeof v === "string" && SearchStringMap[v]) {
                    return SearchStringMap[v](queryParams);
                } else if (typeof v === "object") {
                    return getSearchStringQueryHelper(queryParams, v);
                } else {
                    return v;
                }
            }) as any;
        }
        // If value is a key of SearchStringMap, convert it to a Prisma query
        else if (SearchStringMap[key]) {
            where[key as keyof Where] = SearchStringMap[key](queryParams);
        }
        // If value is an object, recursively convert it
        else if (isRelationshipObject(value)) {
            where[key as keyof Where] = getSearchStringQueryHelper(queryParams, value);
        }
        // If value is a string, convert it to a Prisma query
        else if (typeof value === "string" && SearchStringMap[value]) {
            (where as any)[key as keyof Where] = SearchStringMap[value](queryParams);
        }
        // Otherwise, just copy the value
        else {
            where[key as keyof Where] = value;
        }
    }
    return where;
};

/**
 * Converts a searchStringQuery object into a Prisma search query. 
 * This is accomplished by recursively converting any keys in the searchStringQuery object 
 * to their corresponding Prisma query (stored in SearchStringMap).
 * @returns Fully-converted Prisma query, ready to be passed into prisma.findMany()
 */
export function getSearchStringQuery<Where extends { [x: string]: any }>({
    languages,
    objectType,
    searchString,
}: {
    languages?: string[] | undefined;
    objectType: `${GqlModelType}`;
    searchString: string;
}): Where {
    if (searchString.length === 0) return {} as Where;
    // Get searcher
    const { search } = getLogic(["search"], objectType, languages ?? ["en"], "getSearchStringQuery");
    const insensitive = ({ contains: searchString.trim(), mode: "insensitive" as const });
    return getSearchStringQueryHelper({ insensitive, languages, searchString }, search.searchStringQuery());
}
