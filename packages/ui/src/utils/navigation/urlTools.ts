import { adaHandleRegex, getLastUrlPart, LINKS, uuidValidate } from "@local/shared";
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
        const base36 = toBigInt(uuid.replace(/-/g, ""), 16).toString(36);
        return base36 === "0" ? "" : base36;
    } catch (error) {
        PubSub.get().publishSnack({ messageKey: "CouldNotConvertId", severity: "Error", data: { uuid } });
        return "";
    }
};

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
        const uuid = toBigInt(base36, 36).toString(16).padStart(32, "0").replace(/(.{8})(.{4})(.{4})(.{4})(.{12})/, "$1-$2-$3-$4-$5");
        return uuid === "0" ? "" : uuid;
    } catch (error) {
        if (showError) PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error", data: { base36 } });
        return "";
    }
};

export type SingleItemUrl = {
    handleRoot?: string,
    handle?: string,
    idRoot?: string,
    id?: string,
}

/**
 * Finds information in the URL to query for a specific item. 
 * There are multiple ways to specify an item in the URL. 
 * 
 * For non-versioned items, they can be queried by ID, or sometimes by handle. 
 * This is specified as site.com/item/id or site.com/item/handle.
 * 
 * For versioned items, they can be queried by ID, ID of the root item, or handle of the root item.
 * This is specified as site.com/item/rootId, site.com/item/rootId/id, site.com/handle, or site.com/handle/id.
 * 
 * NOTE: This function may sometimes be used for deeper navigation within a single item, 
 * such as site.com/reports/id or site.com/comments/id. In this case, the logic is still the same.
 */
export const parseSingleItemUrl = (): SingleItemUrl => {
    // Initialize the return object
    const returnObject: SingleItemUrl = {};
    // Get the last 2 parts of the URL
    const lastPart = getLastUrlPart();
    const secondLastPart = getLastUrlPart(1);
    // First check the second last part. If it matches the handle or ID regex, then 
    // we know this is a versioned item
    if (adaHandleRegex.test(secondLastPart)) {
        returnObject.handleRoot = secondLastPart;
    } else if (uuidValidate(base36ToUuid(secondLastPart, false))) {
        returnObject.idRoot = base36ToUuid(secondLastPart);
    }
    // Otherwise, this still might be a versioned item. Just with only the 
    // root ID or handle defined. To check, we must see if any part of the url 
    // contains the name of a versioned object
    const objectsWithVersions = [
        LINKS.Api,
        LINKS.Note,
        LINKS.Project,
        LINKS.Routine,
        LINKS.SmartContract,
        LINKS.Standard,
    ].map(link => link.split("/").pop());
    const allUrlParts = window.location.pathname.split("/");
    const isVersioned = allUrlParts.some(part => objectsWithVersions.includes(part));
    // Now check the last part
    if (adaHandleRegex.test(lastPart)) {
        if (isVersioned) returnObject.handleRoot = lastPart;
        else returnObject.handle = lastPart;
    } else if (uuidValidate(base36ToUuid(lastPart, false))) {
        if (isVersioned) returnObject.idRoot = base36ToUuid(lastPart, false);
        else returnObject.id = base36ToUuid(lastPart, false);
    }
    console.log("parseSingleItemUrl RESULT", returnObject);
    // Return the object
    return returnObject;
};
