import { SearchStringQueryParams } from "../models/types"
import { SearchStringMap } from "../utils"

/**
 * Using keys of the SearchStringMap, builds an array of Prisma queries 
 * to use for searching an object with text
 * @param keys The keys of the SearchStringMap to use
 * @param params The params to use for the search
 * @returns An array of Prisma queries
 */
export const searchStringBuilder = <
    T extends (keyof typeof SearchStringMap)[]
>(keys: T, params: SearchStringQueryParams): ReturnType<typeof SearchStringMap[T[number]]>[] => {
    const queries = keys.map((key) => {
        return SearchStringMap[key](params);
    })
    return queries as any;
}