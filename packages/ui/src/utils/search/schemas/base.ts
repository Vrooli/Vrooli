import { FormSchema } from "@local/shared";

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
export function toParams(
    advancedSearchSchema: FormSchema,
    endpoints: { findOne?: { endpoint: string } | undefined, findMany: { endpoint: string } },
    sortByOptions: { [key: string]: string },
    defaultSortBy: string,
): SearchParams {
    return {
        advancedSearchSchema,
        defaultSortBy,
        findManyEndpoint: endpoints.findMany.endpoint,
        findOneEndpoint: endpoints.findOne?.endpoint,
        sortByOptions,
    };
}
