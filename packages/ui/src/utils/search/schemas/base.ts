import { FormSchema } from "forms/types";

export type SearchParams = {
    advancedSearchSchema: FormSchema | null;
    defaultSortBy: any;
    endpoint: string;
    sortByOptions: any;
}

/**
 * Converts shorthand search info to SearchParams object
 */
export const toParams = (
    advancedSearchSchema: FormSchema,
    pair: { endpoint: string },
    sortByOptions: { [key: string]: string },
    defaultSortBy: string,
): SearchParams => ({
    advancedSearchSchema,
    defaultSortBy,
    endpoint: pair.endpoint,
    sortByOptions,
});
