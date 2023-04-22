/**
 * Filters out any invalid handles from an array of handles.
 * Handles start with a $ and have 3 to 16 characters.
 * @param handles - array of handles to filter
 * @returns Array of valid handles
 */
export const onlyValidHandles = (handles: (string | null | undefined)[]): string[] => handles.filter(handle => typeof handle === "string" && handle.match(/^\$[a-zA-Z0-9]{3,16}$/)) as string[];
