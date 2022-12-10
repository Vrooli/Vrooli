import { CustomError } from "../events";
import { ObjectMap } from "../models";
import { GraphQLModelType, Searcher } from "../models/types";

export function getSearcher<
    SearchInput,
    SortBy extends string,
    Where extends { [x: string]: any }
>(
    objectType: GraphQLModelType,
    languages: string[],
    errorTrace: string,
): Searcher<SearchInput, SortBy, Where>{
    const searcher: Searcher<SearchInput, SortBy, Where> | undefined = ObjectMap[objectType]?.search;
    if (!searcher) {
        throw new CustomError('0296', 'InternalError', languages, { errorTrace, objectType });
    }
    return searcher;
}