import { CODE } from "@shared/consts";
import { CustomError, genErrorCode } from "../../events";
import { ObjectMap } from "../builder";
import { GraphQLModelType, Searcher } from "../types";

/**
 * Finds all permissions for the given object ids
 */
export function getSearcher<
    SearchInput,
    SortBy extends string,
    OrderBy extends { [x: string]: any }, Where extends { [x: string]: any }
>(
    objectType: GraphQLModelType,
    errorTrace: string,
): Searcher<SearchInput, SortBy, OrderBy, Where>{
    // Find searcher and prisma delegate for this object type
    const searcher: Searcher<SearchInput, SortBy, OrderBy, Where> | undefined = ObjectMap[objectType]?.search;
    if (!searcher) {
        throw new CustomError(CODE.InvalidArgs, `Invalid object type in ${errorTrace}: ${objectType}`, { code: genErrorCode('0296') });
    }
    return searcher;
}