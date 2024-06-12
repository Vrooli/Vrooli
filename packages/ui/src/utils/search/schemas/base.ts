import { FormSchema } from "forms/types";

export type SearchParams = {
    /** Schema to display advanced search dialog */
    advancedSearchSchema: FormSchema | null;
    /** What sort option to use by default */
    defaultSortBy: any;
    /** Endpoint to use for findMany */
    findManyEndpoint: string;
    /** Endpoint to use for findOne */
    findOneEndpoint: string | undefined;
    /** All possible sort options */
    sortByOptions: any;
}

/**
 * Converts shorthand search info to SearchParams object
 */
export const toParams = (
    advancedSearchSchema: FormSchema,
    findManyPair: { endpoint: string },
    findOnePair: { endpoint: string } | undefined,
    sortByOptions: { [key: string]: string },
    defaultSortBy: string,
): SearchParams => ({
    advancedSearchSchema,
    defaultSortBy,
    findManyEndpoint: findManyPair.endpoint,
    findOneEndpoint: findOnePair?.endpoint,
    sortByOptions,
});
