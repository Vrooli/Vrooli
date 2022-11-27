import { getSearcher } from "./getSearcher";
import { GetSearchStringProps } from "./types";

/**
 * Helper function for searchStringQuery
 * @returns GraphQL search query object
 */
export function getSearchString<Where extends { [x: string]: any }>({
    languages,
    objectType,
    searchString,
}: GetSearchStringProps): Where {
    if (searchString.length === 0) return {} as Where;
    const searcher = getSearcher<any, any, any, Where>(objectType, languages ?? ['en'], 'getSearchString');
    const insensitive = ({ contains: searchString.trim(), mode: 'insensitive' as const });
    return searcher.searchStringQuery({ insensitive, languages, searchString })
}