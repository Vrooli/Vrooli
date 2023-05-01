import { uuidValidate } from "@local/shared";

/**
 * Filters out any invalid IDs from an array of IDs.
 * @param ids - array of IDs to filter
 * @returns Array of valid IDs
 */
export const onlyValidIds = (ids: (string | null | undefined)[]): string[] => ids.filter(id => typeof id === 'string' && uuidValidate(id)) as string[];
