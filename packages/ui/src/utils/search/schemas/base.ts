import { FormSchema } from "forms/types";
import { DocumentNode } from "graphql";

export type SearchParams = {
    advancedSearchSchema: FormSchema | null;
    defaultSortBy: any;
    sortByOptions: any;
    query: DocumentNode;
}

/**
 * Converts shorthand search info to SearchParams object
 */
export const toParams = (
    advancedSearchSchema: FormSchema,
    query: DocumentNode,
    sortByOptions: { [key: string]: string },
    defaultSortBy: string,
): SearchParams => ({
    advancedSearchSchema,
    defaultSortBy,
    query,
    sortByOptions,
});
