import { getLastUrlPart, parseSearchParams } from "@shared/route";
import { uuidValidate } from "@shared/uuid";
import { adaHandleRegex } from "@shared/validation";
import { PubSub } from "utils/pubsub";



/**
 * Converts a string to a BigInt
 * @param value String to convert
 * @param radix Radix (base) to use
 * @returns 
 */
function toBigInt(value: string, radix: number) {
    return [...value.toString()]
        .reduce((r, v) => r * BigInt(radix) + BigInt(parseInt(v, radix)), 0n);
}

/**
 * Converts a UUID into a shorter, base 36 string without dashes. 
 * Useful for displaying UUIDs in a more compact format, such as in a URL.
 * @param uuid v4 UUID to convert
 * @returns base 36 string without dashes
 */
export const uuidToBase36 = (uuid: string): string => {
    try {
        const base36 = toBigInt(uuid.replace(/-/g, ''), 16).toString(36);
        return base36 === '0' ? '' : base36;
    } catch (error) {
        PubSub.get().publishSnack({ messageKey: 'CouldNotConvertId', severity: 'Error', data: { uuid } });
        return '';
    }
}

/**
 * Converts a base 36 string without dashes into a UUID.
 * @param base36 base 36 string without dashes
 * @param showError Whether to show an error snack if the conversion fails
 * @returns v4 UUID
 */
export const base36ToUuid = (base36: string, showError = true): string => {
    try {
        // Convert to base 16. If the ID is less than 32 characters, pad start with 0s. 
        // Then, insert dashes
        const uuid = toBigInt(base36, 36).toString(16).padStart(32, '0').replace(/(.{8})(.{4})(.{4})(.{4})(.{12})/, '$1-$2-$3-$4-$5');
        return uuid === '0' ? '' : uuid;
    } catch (error) {
        if (showError) PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: 'Error', data: { base36 } });
        return '';
    }
}

/**
 * Finds information in the URL to query for a specific item. 
 * There are multiple ways to specify an item in the URL. 
 * 
 * For non-versioned items, they can be queried by ID, or sometimes by handle. 
 * This is specified as site.com/item/id or site.com/item/handle.
 * 
 * For versioned items, they can be queried by ID, ID of the root item, or handle of the root item.
 * This is specified as site.com/item/id, site.com/item?id=id, site.com/item?root=id, or site.com/item?handle=handle.
 */
export const parseSingleItemUrl = (): { 
    handleRoot?: string,
    handle?: string,
    idRoot?: string,
    id?: string,
} => {
    // Get the last part of the URL
    const lastPart = getLastUrlPart();
    // If this matches the handle or ID regex, return it
    if (adaHandleRegex.test(lastPart)) return { handle: lastPart };
    if (uuidValidate(base36ToUuid(lastPart, false))) return { id: base36ToUuid(lastPart) };
    // Otherwise, parse the search params
    const searchParams = parseSearchParams();
    // If there is a handle, return it
    if (typeof searchParams.handle === 'string' && adaHandleRegex.test(searchParams.handle)) return { handleRoot: searchParams.handle };
    // If there is an ID, return it
    if (searchParams.id && uuidValidate(base36ToUuid(searchParams.id as any, false))) return { idRoot: base36ToUuid(searchParams.id as any, false) };
    // Otherwise, return nothing
    return {};
}