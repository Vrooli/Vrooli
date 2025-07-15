import { type FormSchema } from "@vrooli/shared";

// AI_CHECK: TYPE_SAFETY=fixed-search-schemas-types | LAST: 2025-06-28

export type SearchParams = {
    /** Schema to display advanced search dialog */
    advancedSearchSchema: FormSchema | null;
    /** What sort option to use by default */
    defaultSortBy: string;
    /** Endpoint to use for findMany */
    findManyEndpoint: string;
    /** Endpoint to use for findOne */
    findOneEndpoint: string | undefined;
    /** All possible sort options */
    sortByOptions: Record<string, string>;
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
