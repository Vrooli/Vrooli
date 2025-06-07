import { validatePK } from "@vrooli/shared";

/**
 * Filters out any invalid handles from an array of handles.
 * Handles start with a $ and have 3 to 16 characters.
 * @param handles - array of handles to filter
 * @returns Array of valid handles
 */
export function onlyValidHandles(handles: (string | null | undefined)[]): string[] {
    return handles.filter(handle => typeof handle === "string" && handle.match(/^\$[a-zA-Z0-9]{3,16}$/)) as string[];
}

/**
 * Filters out any invalid IDs from an array of IDs.
 * @param ids - array of IDs to filter
 * @returns Array of valid IDs
 */
export function onlyValidIds(ids: (string | null | undefined)[]): string[] {
    return ids.filter(id => validatePK(id)) as string[];
}
