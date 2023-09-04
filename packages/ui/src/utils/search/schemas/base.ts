import { FormSchema } from "forms/types";

export type SearchParams = {
    advancedSearchSchema: FormSchema | null;
    defaultSortBy: any;
    findManyEndpoint: string;
    findOneEndpoint: string | undefined;
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
