import { CustomError } from "../../events";
import { ObjectMap } from "../builder";
import { GraphQLModelType, Searcher } from "../types";

export function getSearcher<
    SearchInput,
    SortBy extends string,
    OrderBy extends { [x: string]: any }, Where extends { [x: string]: any }
>(
    objectType: GraphQLModelType,
    languages: string[],
    errorTrace: string,
): Searcher<SearchInput, SortBy, OrderBy, Where>{
    const searcher: Searcher<SearchInput, SortBy, OrderBy, Where> | undefined = ObjectMap[objectType]?.search;
    if (!searcher) {
        throw new CustomError('0296', 'InternalError', languages, { errorTrace, objectType });
    }
    return searcher;
}